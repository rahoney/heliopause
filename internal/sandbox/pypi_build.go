package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	pythonBuildInput  = "/tmp/haa-build-input"
	pythonDerivedPath = "/tmp/haa-derived"
	derivedWheelLimit = 64 << 20
)

// DerivedWheel identifies bytes produced by a PEP 517 backend without
// equating them to the source sdist. The caller must statically and dynamically
// re-inspect these bytes before they can enter a Verified Set.
type DerivedWheel struct {
	Artifact                domain.AcquiredArtifact
	Filename                string
	SourceDigest            domain.ContentDigest
	BuildRequirementDigests []domain.ContentDigest
	BuildBackend            string
	BuildConfigSHA256       string
}

// PythonSdistBuilder runs one PEP 517 build only after static inspection has
// supplied the source recipe and every exact build wheel. It has no network,
// no Host mounts and no backend-controlled command surface.
type PythonSdistBuilder struct {
	runner       CommandRunner
	introducer   *PythonArtifactIntroducer
	observer     TraceObserver
	probe        func(context.Context) (PythonCapability, error)
	newSessionID func() (domain.SandboxSessionID, error)
	timeout      time.Duration
	cleanupWait  time.Duration
}

func NewPythonSdistBuilder(runner CommandRunner, introducer *PythonArtifactIntroducer, observer TraceObserver, probe func(context.Context) (PythonCapability, error)) (*PythonSdistBuilder, error) {
	if runner == nil || introducer == nil || observer == nil || probe == nil {
		return nil, errors.New("python sdist builder is not configured")
	}
	if _, ok := runner.(discardCommandRunner); !ok {
		return nil, errors.New("python sdist runner must discard command output")
	}
	return &PythonSdistBuilder{runner: runner, introducer: introducer, observer: observer, probe: probe, newSessionID: domain.NewSandboxSessionID, timeout: pythonDynamicTimeout, cleanupWait: cleanupTimeout}, nil
}

// Build creates exactly one derived wheel in a fresh gVisor session. Any
// unknown requirement, observer loss, output ambiguity or cleanup failure
// returns no derived bytes.
func (b *PythonSdistBuilder) Build(ctx context.Context, source domain.AcquiredArtifact, recipe artifactpypi.SdistInspection, buildWheels []domain.AcquiredArtifact) (DerivedWheel, domain.SandboxResult, error) {
	if b == nil || b.runner == nil || b.introducer == nil || b.observer == nil || b.probe == nil || b.newSessionID == nil || ctx == nil || !validSdistBuildInput(source, recipe, buildWheels) {
		return DerivedWheel{}, domain.SandboxResult{}, errors.New("python sdist build request is invalid")
	}
	sessionID, err := b.newSessionID()
	if err != nil {
		return DerivedWheel{}, domain.SandboxResult{}, err
	}
	capability, err := b.probe(ctx)
	if err != nil || !capability.Available || capability.Runtime != PinnedPythonRuntime() {
		result, resultErr := pythonIncomplete(sessionID, "M5_PYPI_BUILD_RUNTIME_UNAVAILABLE")
		return DerivedWheel{}, result, resultErr
	}
	created, err := b.runner.Output(ctx, "docker", pythonDynamicCreateArguments(sessionID)...)
	if err != nil || !containerIDPattern.MatchString(strings.TrimSpace(string(created))) {
		result, resultErr := pythonIncomplete(sessionID, "M5_PYPI_BUILD_SETUP_FAILED")
		return DerivedWheel{}, result, resultErr
	}
	containerID := strings.TrimSpace(string(created))
	trace, err := b.observer.Start(ctx, containerID)
	if err != nil {
		if b.remove(containerID) != nil {
			result, resultErr := pythonIncomplete(sessionID, "M5_PYPI_BUILD_CLEANUP_FAILED")
			return DerivedWheel{}, result, resultErr
		}
		result, resultErr := pythonIncomplete(sessionID, "M5_PYPI_BUILD_OBSERVER_FAILED")
		return DerivedWheel{}, result, resultErr
	}
	runCtx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()
	fail := func(code string) (DerivedWheel, domain.SandboxResult, error) {
		_, limitation := b.disposeAndCollect(containerID, trace)
		if limitation == "M5_PYPI_DYNAMIC_CLEANUP_FAILED" {
			code = "M5_PYPI_BUILD_CLEANUP_FAILED"
		}
		result, resultErr := pythonIncomplete(sessionID, code)
		return DerivedWheel{}, result, resultErr
	}
	if err := discardCommand(runCtx, b.runner, "docker", "start", containerID); err != nil {
		return fail("M5_PYPI_BUILD_SETUP_FAILED")
	}
	sdistPath := pythonSdistPath(source)
	if err := b.introducer.introduce(runCtx, containerID, source, sdistPath, "sdist"); err != nil {
		return fail("M5_PYPI_BUILD_INTRODUCTION_FAILED")
	}
	wheelPaths := make([]string, 0, len(buildWheels))
	for index, wheel := range buildWheels {
		destination := fmt.Sprintf("%s/%02d.whl", pythonBuildInput, index)
		if err := b.introducer.introduce(runCtx, containerID, wheel, destination, "wheel"); err != nil {
			return fail("M5_PYPI_BUILD_INTRODUCTION_FAILED")
		}
		wheelPaths = append(wheelPaths, destination)
	}
	if err := discardCommand(runCtx, b.runner, "docker", "exec", containerID, "python", "-I", "-m", "venv", "/tmp/haa-buildenv"); err != nil {
		return fail("M5_PYPI_BUILD_ENVIRONMENT_FAILED")
	}
	installArgs := append([]string{"exec", containerID, "/tmp/haa-buildenv/bin/python", "-I", "-m", "pip", "install", "--no-index", "--no-deps", "--no-compile", "--disable-pip-version-check"}, wheelPaths...)
	if err := discardCommand(runCtx, b.runner, "docker", installArgs...); err != nil {
		return fail("M5_PYPI_BUILD_REQUIREMENTS_FAILED")
	}
	if err := discardCommand(runCtx, b.runner, "docker", "exec", containerID, "/tmp/haa-buildenv/bin/python", "-I", "-m", "pip", "wheel", "--no-index", "--no-deps", "--no-build-isolation", "--wheel-dir", pythonDerivedPath, sdistPath); err != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return fail("M5_PYPI_BUILD_TIMEOUT")
		}
		return fail("M5_PYPI_BUILD_FAILED")
	}
	filenameBytes, err := b.runner.Output(runCtx, "docker", "exec", containerID, "python", "-I", "-c", pythonSingleWheelNameScript, pythonDerivedPath)
	filename := strings.TrimSpace(string(filenameBytes))
	if err != nil || len(filename) > 256 {
		return fail("M5_PYPI_BUILD_OUTPUT_AMBIGUOUS")
	}
	if project, version, _, _, _, parseErr := artifactpypi.ParseWheelFilename(filename); parseErr != nil || project != recipe.Project || version != recipe.Version {
		return fail("M5_PYPI_BUILD_OUTPUT_AMBIGUOUS")
	}
	derived, streamErr := b.streamDerivedWheel(runCtx, containerID, source, filename, recipe, buildWheels)
	if streamErr != nil {
		return fail("M5_PYPI_BUILD_OUTPUT_FAILED")
	}
	observations, limitation := b.disposeAndCollect(containerID, trace)
	if limitation != "" {
		result, resultErr := pythonIncomplete(sessionID, "M5_PYPI_BUILD_OBSERVATION_INCOMPLETE")
		return DerivedWheel{}, result, resultErr
	}
	completed, _ := domain.NewSandboxObservation(domain.ObservationProcess, "pep517-build-completed")
	result, resultErr := domain.NewSandboxResult(sessionID, domain.SandboxCompleted, "", append(observations, completed))
	if resultErr != nil {
		return DerivedWheel{}, domain.SandboxResult{}, resultErr
	}
	return derived, result, nil
}

func validSdistBuildInput(source domain.AcquiredArtifact, recipe artifactpypi.SdistInspection, wheels []domain.AcquiredArtifact) bool {
	if source.Identity().Source().String() != "pypi" || source.Identity().Variant() != "sdist" || source.Identity().Name() != recipe.Project || source.Identity().Version() != recipe.Version || source.Digest().String() != recipe.ObservedSHA256 || len(recipe.BuildRequirements) == 0 || len(wheels) != len(recipe.BuildRequirements) {
		return false
	}
	seen := map[string]bool{}
	for _, wheel := range wheels {
		if wheel.Identity().Source().String() != "pypi" || wheel.Identity().Variant() != "wheel" || wheel.Digest().String() == "" || seen[wheel.Identity().Name()] {
			return false
		}
		seen[wheel.Identity().Name()] = true
	}
	for _, requirement := range recipe.BuildRequirements {
		if !seen[requirement] {
			return false
		}
	}
	return true
}

func (i *PythonArtifactIntroducer) introduce(ctx context.Context, containerID string, artifact domain.AcquiredArtifact, destination, variant string) error {
	if i == nil || i.runner == nil || ctx == nil || !containerIDPattern.MatchString(containerID) || artifact.Identity().Source().String() != "pypi" || artifact.Identity().Variant() != variant || !strings.HasPrefix(destination, "/tmp/") {
		return errors.New("python artifact introduction request is invalid")
	}
	source, err := i.artifactPath(artifact.ContentHandle(), variant)
	if err != nil {
		return err
	}
	file, err := os.Open(source)
	if err != nil {
		return errors.New("verified Python intake is unavailable")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || uint64(info.Size()) != artifact.SizeBytes() {
		return errors.New("verified Python intake is unavailable")
	}
	input, ok := i.runner.(inputCommandRunner)
	if !ok {
		return errors.New("sandbox artifact stream runner is not configured")
	}
	if err := input.RunInput(ctx, file, "docker", "exec", "-i", containerID, "python", "-I", "-c", pythonCopyArtifactScript, destination); err != nil {
		return fmt.Errorf("introduce verified Python artifact: %w", err)
	}
	return nil
}

func (i *PythonArtifactIntroducer) artifactPath(handle, variant string) (string, error) {
	parts := strings.Split(handle, ":")
	if len(parts) != 3 || parts[0] != "intake" || parts[2] != variant {
		return "", errors.New("python content handle is invalid")
	}
	runID, err := domain.ParseRunID(parts[1])
	if err != nil {
		return "", errors.New("python content handle is invalid")
	}
	name := map[string]string{"wheel": "wheel.whl", "derived-wheel": "derived.whl", "sdist": "sdist.tar.gz"}[variant]
	if name == "" {
		return "", errors.New("python content handle is invalid")
	}
	path := filepath.Join(i.intakeRoot, runID.String(), name)
	relative, err := filepath.Rel(i.intakeRoot, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("python intake path escapes root")
	}
	return path, nil
}

func pythonSdistPath(artifact domain.AcquiredArtifact) string {
	name := strings.ReplaceAll(artifact.Identity().Name(), "-", "_")
	version := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' {
			return r
		}
		return '_'
	}, artifact.Identity().Version())
	return "/tmp/haa-" + name + "-" + version + ".tar.gz"
}

func (b *PythonSdistBuilder) disposeAndCollect(containerID string, trace TraceReader) ([]domain.SandboxObservation, string) {
	if b.remove(containerID) != nil {
		return nil, "M5_PYPI_DYNAMIC_CLEANUP_FAILED"
	}
	ctx, cancel := context.WithTimeout(context.Background(), b.cleanupWait)
	defer cancel()
	observations, limitation := collectTrace(ctx, trace)
	if limitation != "" {
		return nil, "M5_PYPI_BUILD_OBSERVATION_INCOMPLETE"
	}
	return observations, ""
}
func (b *PythonSdistBuilder) remove(containerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), b.cleanupWait)
	defer cancel()
	return discardCommand(ctx, b.runner, "docker", "rm", "--force", containerID)
}

func (b *PythonSdistBuilder) streamDerivedWheel(ctx context.Context, containerID string, source domain.AcquiredArtifact, filename string, recipe artifactpypi.SdistInspection, buildWheels []domain.AcquiredArtifact) (DerivedWheel, error) {
	output, ok := b.runner.(interface {
		RunOutput(context.Context, io.Writer, string, ...string) error
	})
	if !ok {
		return DerivedWheel{}, errors.New("sandbox artifact output runner is not configured")
	}
	parts := strings.Split(source.ContentHandle(), ":")
	if len(parts) != 3 {
		return DerivedWheel{}, errors.New("source content handle is invalid")
	}
	runID, runErr := domain.ParseRunID(parts[1])
	if runErr != nil {
		return DerivedWheel{}, errors.New("source content handle is invalid")
	}
	directory := filepath.Join(b.introducer.intakeRoot, runID.String())
	if _, existsErr := os.Lstat(filepath.Join(directory, "derived.whl")); existsErr == nil || !errors.Is(existsErr, os.ErrNotExist) {
		return DerivedWheel{}, errors.New("derived wheel destination is unavailable")
	}
	temporary, err := os.OpenFile(filepath.Join(directory, ".derived-wheel.tmp"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return DerivedWheel{}, err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	writer := &limitedFileWriter{writer: temporary, limit: derivedWheelLimit, hasher: sha256.New()}
	err = output.RunOutput(ctx, writer, "docker", "exec", containerID, "python", "-I", "-c", pythonStreamFileScript, pythonDerivedPath+"/"+filename)
	syncErr := temporary.Sync()
	closeErr := temporary.Close()
	if err != nil || writer.exceeded || syncErr != nil || closeErr != nil {
		return DerivedWheel{}, errors.New("derived wheel output is invalid")
	}
	if err := os.Rename(temporaryPath, filepath.Join(directory, "derived.whl")); err != nil {
		return DerivedWheel{}, err
	}
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(writer.hasher.Sum(nil)))
	if err != nil {
		return DerivedWheel{}, err
	}
	identity, err := domain.NewResolvedArtifactIdentity(source.Identity().Source(), recipe.Project, recipe.Version, "derived-wheel")
	if err != nil {
		return DerivedWheel{}, err
	}
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:"+runID.String()+":derived-wheel", uint64(writer.size))
	if err != nil {
		return DerivedWheel{}, err
	}
	requirementDigests := make([]domain.ContentDigest, 0, len(buildWheels))
	for _, wheel := range buildWheels {
		requirementDigests = append(requirementDigests, wheel.Digest())
	}
	sort.Slice(requirementDigests, func(i, j int) bool { return requirementDigests[i].String() < requirementDigests[j].String() })
	return DerivedWheel{Artifact: artifact, Filename: filename, SourceDigest: source.Digest(), BuildRequirementDigests: requirementDigests, BuildBackend: recipe.BuildBackend, BuildConfigSHA256: recipe.BuildConfigSHA256}, nil
}

type limitedFileWriter struct {
	writer      *os.File
	limit, size int64
	exceeded    bool
	hasher      hash.Hash
}

func (w *limitedFileWriter) Write(data []byte) (int, error) {
	if int64(len(data)) > w.limit-w.size {
		w.exceeded = true
		return 0, errors.New("derived wheel exceeds bound")
	}
	n, err := w.writer.Write(data)
	if n > 0 {
		_, _ = w.hasher.Write(data[:n])
		w.size += int64(n)
	}
	return n, err
}

const pythonCopyArtifactScript = "import os,sys\np=sys.argv[1]\nos.makedirs(os.path.dirname(p),mode=0o700,exist_ok=True)\nout=open(p,'xb')\nwhile True:\n chunk=sys.stdin.buffer.read(65536)\n if not chunk: break\n out.write(chunk)\nout.close()\n"
const pythonSingleWheelNameScript = "import os,sys\nfiles=[x for x in os.listdir(sys.argv[1]) if x.endswith('.whl') and os.path.isfile(os.path.join(sys.argv[1],x))]\nif len(files)!=1: raise SystemExit(1)\nprint(files[0])\n"
const pythonStreamFileScript = "import os,stat,sys\ns=os.lstat(sys.argv[1])\nif not stat.S_ISREG(s.st_mode): raise SystemExit(1)\nwith open(sys.argv[1],'rb') as f:\n while True:\n  b=f.read(65536)\n  if not b: break\n  sys.stdout.buffer.write(b)\n"
