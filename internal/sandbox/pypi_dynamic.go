package sandbox

import (
	"context"
	"errors"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	pythonDynamicTimeout = 45 * time.Second
	pythonSitePath       = "/tmp/haa-site"
)

var pythonImportName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*$`)

// PythonWheelRunner is the narrow adapter boundary consumed by PyPI dynamic
// inspection. Import names are normalized static-inspection output, not caller
// supplied module names.
type PythonWheelRunner interface {
	InspectWheel(context.Context, domain.AcquiredArtifact, []string) (domain.SandboxResult, error)
}

type discardCommandRunner interface {
	RunDiscard(context.Context, string, ...string) error
}

// PythonArtifactIntroducer streams one verified wheel from the HAA intake root
// into a running container. It never puts the Host path into a Docker command.
type PythonArtifactIntroducer struct {
	intakeRoot string
	runner     CommandRunner
}

func NewPythonArtifactIntroducer(intakeRoot string, runner CommandRunner) (*PythonArtifactIntroducer, error) {
	if !filepath.IsAbs(intakeRoot) || runner == nil {
		return nil, errors.New("python wheel introducer is not configured")
	}
	return &PythonArtifactIntroducer{intakeRoot: filepath.Clean(intakeRoot), runner: runner}, nil
}

func (i *PythonArtifactIntroducer) IntroduceWheel(ctx context.Context, containerID string, artifact domain.AcquiredArtifact) error {
	if !pythonWheelVariant(artifact.Identity().Variant()) {
		return errors.New("python wheel introduction request is invalid")
	}
	return i.introduce(ctx, containerID, artifact, pythonWheelPath(artifact), artifact.Identity().Variant())
}

func pythonWheelVariant(variant string) bool { return variant == "wheel" || variant == "derived-wheel" }

func pythonWheelPath(artifact domain.AcquiredArtifact) string {
	name := strings.ReplaceAll(artifact.Identity().Name(), "-", "_")
	version := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' {
			return r
		}
		return '_'
	}, artifact.Identity().Version())
	return "/tmp/haa-" + name + "-" + version + "-py3-none-any.whl"
}

// PythonDynamicBackend creates one network-isolated runsc-trace container for
// importing a statically declared wheel surface. It returns observations only;
// it never creates Findings, Evidence, Policy or Promotion state.
type PythonDynamicBackend struct {
	runner       CommandRunner
	introducer   *PythonArtifactIntroducer
	observer     TraceObserver
	probe        func(context.Context) (PythonCapability, error)
	newSessionID func() (domain.SandboxSessionID, error)
	timeout      time.Duration
	cleanupWait  time.Duration
}

func NewPythonDynamicBackend(runner CommandRunner, introducer *PythonArtifactIntroducer, observer TraceObserver, probe func(context.Context) (PythonCapability, error)) (*PythonDynamicBackend, error) {
	if runner == nil || introducer == nil || observer == nil || probe == nil {
		return nil, errors.New("python dynamic backend is not configured")
	}
	if _, ok := runner.(discardCommandRunner); !ok {
		return nil, errors.New("python dynamic runner must discard command output")
	}
	return &PythonDynamicBackend{runner: runner, introducer: introducer, observer: observer, probe: probe, newSessionID: domain.NewSandboxSessionID, timeout: pythonDynamicTimeout, cleanupWait: cleanupTimeout}, nil
}

func (b *PythonDynamicBackend) InspectWheel(ctx context.Context, artifact domain.AcquiredArtifact, imports []string) (domain.SandboxResult, error) {
	if b == nil || b.runner == nil || b.introducer == nil || b.observer == nil || b.probe == nil || b.newSessionID == nil || ctx == nil || !validImportSurface(imports) {
		return domain.SandboxResult{}, errors.New("python dynamic inspection request is invalid")
	}
	sessionID, err := b.newSessionID()
	if err != nil {
		return domain.SandboxResult{}, err
	}
	capability, err := b.probe(ctx)
	if err != nil || !capability.Available || capability.Runtime != PinnedPythonRuntime() {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_RUNTIME_UNAVAILABLE")
	}
	created, err := b.runner.Output(ctx, "docker", pythonDynamicCreateArguments(sessionID)...)
	if err != nil || !containerIDPattern.MatchString(strings.TrimSpace(string(created))) {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_SETUP_FAILED")
	}
	containerID := strings.TrimSpace(string(created))
	trace, err := b.observer.Start(ctx, containerID)
	if err != nil {
		if b.remove(containerID) != nil {
			return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_CLEANUP_FAILED")
		}
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_OBSERVER_FAILED")
	}
	runCtx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()
	if err := discardCommand(runCtx, b.runner, "docker", "start", containerID); err != nil {
		return b.finishIncomplete(sessionID, containerID, trace, "M5_PYPI_DYNAMIC_SETUP_FAILED")
	}
	wheelPath := pythonWheelPath(artifact)
	if err := b.introducer.IntroduceWheel(runCtx, containerID, artifact); err != nil {
		return b.finishIncomplete(sessionID, containerID, trace, "M5_PYPI_DYNAMIC_INTRODUCTION_FAILED")
	}
	if err := discardCommand(runCtx, b.runner, "docker", "exec", containerID, "python", "-I", "-m", "pip", "install", "--no-index", "--no-deps", "--no-compile", "--disable-pip-version-check", "--target", pythonSitePath, wheelPath); err != nil {
		return b.finishIncomplete(sessionID, containerID, trace, "M5_PYPI_DYNAMIC_INSTALL_FAILED")
	}
	arguments := append([]string{"exec", containerID, "python", "-I", "-c", pythonImportScript}, imports...)
	if err := discardCommand(runCtx, b.runner, "docker", arguments...); err != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return b.finishIncomplete(sessionID, containerID, trace, "M5_PYPI_DYNAMIC_TIMEOUT")
		}
		return b.finishIncomplete(sessionID, containerID, trace, "M5_PYPI_DYNAMIC_IMPORT_FAILED")
	}
	observations, limitation := b.disposeAndCollect(containerID, trace)
	if limitation != "" {
		return pythonIncomplete(sessionID, limitation)
	}
	completed, err := domain.NewSandboxObservation(domain.ObservationProcess, "python-import-completed")
	if err != nil {
		return domain.SandboxResult{}, err
	}
	return domain.NewSandboxResult(sessionID, domain.SandboxCompleted, "", append(observations, completed))
}

func (b *PythonDynamicBackend) finishIncomplete(sessionID domain.SandboxSessionID, containerID string, trace TraceReader, limitation string) (domain.SandboxResult, error) {
	_, collectLimitation := b.disposeAndCollect(containerID, trace)
	if collectLimitation == "M5_PYPI_DYNAMIC_CLEANUP_FAILED" {
		limitation = collectLimitation
	}
	return pythonIncomplete(sessionID, limitation)
}
func (b *PythonDynamicBackend) disposeAndCollect(containerID string, trace TraceReader) ([]domain.SandboxObservation, string) {
	if b.remove(containerID) != nil {
		return nil, "M5_PYPI_DYNAMIC_CLEANUP_FAILED"
	}
	ctx, cancel := context.WithTimeout(context.Background(), b.cleanupWait)
	defer cancel()
	observations, limitation := collectTrace(ctx, trace)
	if limitation != "" {
		return nil, "M5_PYPI_DYNAMIC_OBSERVATION_INCOMPLETE"
	}
	return observations, ""
}
func (b *PythonDynamicBackend) remove(containerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), b.cleanupWait)
	defer cancel()
	return discardCommand(ctx, b.runner, "docker", "rm", "--force", containerID)
}

func discardCommand(ctx context.Context, runner CommandRunner, binary string, arguments ...string) error {
	discarder, ok := runner.(discardCommandRunner)
	if !ok {
		return errors.New("sandbox command runner must discard command output")
	}
	return discarder.RunDiscard(ctx, binary, arguments...)
}
func pythonIncomplete(sessionID domain.SandboxSessionID, code string) (domain.SandboxResult, error) {
	return domain.NewSandboxResult(sessionID, domain.SandboxIncomplete, code, nil)
}
func validImportSurface(imports []string) bool {
	if len(imports) == 0 || len(imports) > 32 {
		return false
	}
	seen := map[string]bool{}
	for _, name := range imports {
		if !pythonImportName.MatchString(name) || seen[name] {
			return false
		}
		seen[name] = true
	}
	return true
}
func pythonDynamicCreateArguments(sessionID domain.SandboxSessionID) []string {
	return []string{"create", "--pull", "never", "--runtime", gVisorRuntimeName, "--user", "1000:1000", "--network", "none", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--pids-limit", "64", "--memory", "512m", "--cpus", "1", "--ulimit", "cpu=30:30", "--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=256m,uid=1000,gid=1000,mode=0700", "--name", "heliopause-pypi-" + sessionID.String(), pythonImageReference, "sleep", "infinity"}
}

const pythonImportScript = "import importlib,sys\nsys.path.insert(0,'/tmp/haa-site')\nfor name in sys.argv[1:]: importlib.import_module(name)\n"
