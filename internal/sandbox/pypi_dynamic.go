package sandbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	pythonDynamicTimeout = 45 * time.Second
	pythonSitePath       = "/haa-site"
)

var pythonImportName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*$`)

// PythonWheelRunner is the narrow adapter boundary consumed by PyPI dynamic
// inspection. Import names are normalized static-inspection output, not caller
// supplied module names.
type PythonWheelRunner interface {
	InspectWheel(context.Context, domain.AcquiredArtifact, []string) (domain.SandboxResult, error)
}

// DependencyAwarePythonWheelRunner is the optional graph-install capability
// used when a Python node must import with its already-acquired dependency
// closure present. The legacy single-wheel method remains unchanged.
type DependencyAwarePythonWheelRunner interface {
	PythonWheelRunner
	InspectWheelWithClosure(context.Context, domain.AcquiredArtifact, []string, []domain.AcquiredArtifact) (domain.SandboxResult, error)
}

type discardCommandRunner interface {
	RunDiscard(context.Context, string, ...string) error
}

// boundedCommandRunner retains a short process-local diagnostic window. The
// Dynamic backend classifies it before returning, and never exposes it in a
// Sandbox Result, Evidence, or CLI output.
type boundedCommandRunner interface {
	RunBounded(context.Context, string, ...string) ([]byte, error)
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
	destination, err := i.validatedWheelDestination(artifact)
	if err != nil {
		return err
	}
	return i.introduceWheelAt(ctx, containerID, artifact, destination)
}

func pythonWheelVariant(variant string) bool { return variant == "wheel" || variant == "derived-wheel" }

func (i *PythonArtifactIntroducer) validatedWheelFilename(artifact domain.AcquiredArtifact) (string, error) {
	if i == nil || !pythonWheelVariant(artifact.Identity().Variant()) {
		return "", errors.New("python wheel filename request is invalid")
	}
	source, err := i.artifactPath(artifact.ContentHandle(), artifact.Identity().Variant())
	if err != nil {
		return "", err
	}
	recordName := "filename"
	if artifact.Identity().Variant() == "derived-wheel" {
		recordName = "derived-filename"
	}
	recordPath := filepath.Join(filepath.Dir(source), recordName)
	info, err := os.Lstat(recordPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("verified Python intake filename is unavailable")
	}
	filenameBytes, err := os.ReadFile(recordPath)
	if err != nil {
		return "", errors.New("verified Python intake filename is unavailable")
	}
	filename := string(filenameBytes)
	if filename == "" || filename == "." || filename == ".." || filepath.IsAbs(filename) || filepath.Base(filename) != filename || strings.ContainsAny(filename, `/\\`) {
		return "", errors.New("verified Python intake filename is invalid")
	}
	project, version, _, _, _, err := artifactpypi.ParseWheelFilenameForSource(filename, artifact.Identity().Source())
	if err != nil || project != artifact.Identity().Name() || version != artifact.Identity().Version() {
		return "", errors.New("verified Python intake filename does not match Artifact")
	}
	return filename, nil
}

func (i *PythonArtifactIntroducer) validatedWheelDestination(artifact domain.AcquiredArtifact) (string, error) {
	filename, err := i.validatedWheelFilename(artifact)
	if err != nil {
		return "", err
	}
	return "/tmp/" + filename, nil
}

type validatedWheel struct {
	artifact    domain.AcquiredArtifact
	destination string
}

func (i *PythonArtifactIntroducer) validatedWheelDestinations(target domain.AcquiredArtifact, closure []domain.AcquiredArtifact) ([]validatedWheel, error) {
	if len(closure) == 0 {
		return nil, errors.New("python dynamic dependency closure is empty")
	}
	targetSeen := false
	validated := make([]validatedWheel, 0, len(closure))
	seen := make(map[string]bool, len(closure))
	for _, item := range closure {
		if !pythonWheelVariant(item.Identity().Variant()) || item.ContentHandle() == "" || item.Digest().String() == "" {
			return nil, errors.New("python dynamic dependency closure contains unsupported artifact")
		}
		destination, err := i.validatedWheelDestination(item)
		if err != nil {
			return nil, err
		}
		if seen[destination] {
			return nil, errors.New("python dynamic dependency closure contains duplicate artifact path")
		}
		seen[destination] = true
		validated = append(validated, validatedWheel{artifact: item, destination: destination})
		if sameArtifactIdentity(item, target) {
			targetSeen = true
		}
	}
	if !targetSeen {
		return nil, errors.New("python dynamic dependency closure omits target artifact")
	}
	return validated, nil
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
	return b.InspectWheelWithClosure(ctx, artifact, imports, []domain.AcquiredArtifact{artifact})
}

func (b *PythonDynamicBackend) InspectWheelWithClosure(ctx context.Context, artifact domain.AcquiredArtifact, imports []string, closure []domain.AcquiredArtifact) (domain.SandboxResult, error) {
	if b == nil || b.runner == nil || b.introducer == nil || b.observer == nil || b.probe == nil || b.newSessionID == nil || ctx == nil || !validImportSurface(imports) {
		return domain.SandboxResult{}, errors.New("python dynamic inspection request is invalid")
	}
	validated, err := b.introducer.validatedWheelDestinations(artifact, closure)
	if err != nil {
		return domain.SandboxResult{}, err
	}
	wheelPaths := make([]string, 0, len(validated))
	for _, item := range validated {
		wheelPaths = append(wheelPaths, item.destination)
	}
	sessionID, err := b.newSessionID()
	if err != nil {
		return domain.SandboxResult{}, err
	}
	capability, err := b.probe(ctx)
	if err != nil || !capability.Available || capability.Runtime != PinnedPythonRuntime() {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_RUNTIME_UNAVAILABLE")
	}
	observerProfile, err := pythonDynamicObserverProfile(artifactpypi.RootSourceProfileNameFromContext(ctx))
	if err != nil {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_SETUP_FAILED")
	}
	resourcePolicy := artifactpypi.ResourcePolicyFromContext(ctx)
	created, err := b.runner.Output(ctx, "docker", pythonDynamicCreateArguments(sessionID, resourcePolicy)...)
	if err != nil || !containerIDPattern.MatchString(strings.TrimSpace(string(created))) {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_SETUP_FAILED")
	}
	containerID := strings.TrimSpace(string(created))
	trace, err := startTrace(ctx, b.observer, containerID, observerProfile)
	if err != nil {
		if b.remove(containerID) != nil {
			return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_CLEANUP_FAILED")
		}
		return pythonIncomplete(sessionID, dynamicObserverFailureCode("START", classifyDynamicObserverFailure(err)))
	}
	timeout := b.timeout
	if resourcePolicy.Duration() > defaultPyPIDynamicDuration {
		timeout = resourcePolicy.Duration()
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := discardCommand(runCtx, b.runner, "docker", "start", containerID); err != nil {
		return b.finishIncomplete(sessionID, containerID, trace, "M5_PYPI_DYNAMIC_SETUP_FAILED")
	}
	if err := awaitBoundaryHelper(runCtx, b.runner, containerID); err != nil {
		return b.finishIncomplete(sessionID, containerID, trace, "M5_PYPI_DYNAMIC_SETUP_FAILED")
	}
	if err := awaitMountAnchors(runCtx, b.observer, containerID); err != nil {
		return b.finishIncomplete(sessionID, containerID, trace, "M5_PYPI_DYNAMIC_OBSERVER_FAILED")
	}
	for _, item := range validated {
		if err := b.introducer.introduceWheelAt(runCtx, containerID, item.artifact, item.destination); err != nil {
			return b.finishIncomplete(sessionID, containerID, trace, "M5_PYPI_DYNAMIC_INTRODUCTION_FAILED")
		}
	}
	installArguments := append(boundaryExecArguments(containerID, boundaryLaunchMode, "python", "-I", "-B", "-m", "pip", "install", "--no-index", "--no-deps", "--no-compile", "--disable-pip-version-check", "--no-cache-dir", "--target", pythonSitePath), wheelPaths...)
	if failureClass, err := runBoundedCommand(runCtx, b.runner, classifyDynamicInstallFailure, "docker", installArguments...); err != nil {
		return b.finishIncomplete(sessionID, containerID, trace, dynamicInstallFailureCode(failureClass))
	}
	arguments := append(boundaryExecArguments(containerID, boundaryPythonHandoffMode, "python", "-I", "-B", "-c", pythonImportScript), imports...)
	if failureClass, err := runBoundedCommand(runCtx, b.runner, classifyDynamicImportFailure, "docker", arguments...); err != nil {
		return b.finishIncomplete(sessionID, containerID, trace, dynamicImportFailureCode(failureClass))
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

func (i *PythonArtifactIntroducer) introduceWheelAt(ctx context.Context, containerID string, artifact domain.AcquiredArtifact, destination string) error {
	return i.introduce(ctx, containerID, artifact, destination, artifact.Identity().Variant())
}

func sameArtifactIdentity(left, right domain.AcquiredArtifact) bool {
	leftIdentity, rightIdentity := left.Identity(), right.Identity()
	return leftIdentity.Source() == rightIdentity.Source() && leftIdentity.Name() == rightIdentity.Name() && leftIdentity.Version() == rightIdentity.Version() && leftIdentity.Variant() == rightIdentity.Variant() && left.Digest() == right.Digest()
}

const defaultPyPIDynamicDuration = 5 * time.Minute

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

func runBoundedCommand(ctx context.Context, runner CommandRunner, classify func(context.Context, string) string, binary string, arguments ...string) (string, error) {
	if bounded, ok := runner.(boundedCommandRunner); ok {
		output, err := bounded.RunBounded(ctx, binary, arguments...)
		if err != nil {
			return classify(ctx, string(output)), err
		}
		return "", nil
	}
	if err := discardCommand(ctx, runner, binary, arguments...); err != nil {
		return classify(ctx, ""), err
	}
	return "", nil
}

const (
	dynamicInstallFailurePrefix          = "M5_PYPI_DYNAMIC_INSTALL_FAILED_"
	dynamicInstallFailurePipArgument     = "PIP_ARGUMENT_ERROR"
	dynamicInstallFailureWheelPlatform   = "WHEEL_PLATFORM_REJECTED"
	dynamicInstallFailureWheelMetadata   = "WHEEL_METADATA_REJECTED"
	dynamicInstallFailurePackageConflict = "PACKAGE_CONFLICT"
	dynamicInstallFailureDuplicate       = "DUPLICATE_DISTRIBUTION"
	dynamicInstallFailureENOSPC          = "ENOSPC"
	dynamicInstallFailureMemory          = "MEMORY_LIMIT"
	dynamicInstallFailureTimeout         = "TIMEOUT"
	dynamicInstallFailurePermission      = "PERMISSION"
	dynamicInstallFailureSandboxRuntime  = "SANDBOX_RUNTIME"
	dynamicInstallFailureOther           = "OTHER"
	dynamicImportFailurePrefix           = "M5_PYPI_DYNAMIC_IMPORT_FAILED_"
	dynamicImportFailureMissingLibrary   = "MISSING_SHARED_LIBRARY"
	dynamicImportFailureDlopenPermission = "DLOPEN_PERMISSION"
	dynamicImportFailureELFLoader        = "ELF_LOADER"
	dynamicImportFailureSymbolVersion    = "SYMBOL_VERSION"
	dynamicImportFailurePythonABI        = "PYTHON_ABI"
	dynamicImportFailureException        = "IMPORT_EXCEPTION"
	dynamicImportFailureMemory           = "MEMORY_LIMIT"
	dynamicImportFailureCPU              = "CPU_LIMIT"
	dynamicImportFailurePID              = "PID_LIMIT"
	dynamicImportFailureTimeout          = "TIMEOUT"
	dynamicImportFailureSandboxRuntime   = "SANDBOX_RUNTIME"
	dynamicImportFailureOther            = "OTHER"
	dynamicObserverFailurePrefix         = "M5_PYPI_DYNAMIC_OBSERVER_FAILED_"
)

func dynamicInstallFailureCode(class string) string {
	return dynamicInstallFailurePrefix + class
}

func dynamicImportFailureCode(class string) string {
	return dynamicImportFailurePrefix + class
}

func dynamicObserverFailureCode(phase, class string) string {
	return dynamicObserverFailurePrefix + phase + "_" + class
}

func classifyDynamicInstallFailure(ctx context.Context, output string) string {
	if ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return dynamicInstallFailureTimeout
	}
	output = strings.ToLower(output)
	switch {
	case strings.Contains(output, "no space left on device"):
		return dynamicInstallFailureENOSPC
	case strings.Contains(output, "cannot allocate memory"), strings.Contains(output, "out of memory"), strings.Contains(output, "memory limit"):
		return dynamicInstallFailureMemory
	case strings.Contains(output, "permission denied"):
		return dynamicInstallFailurePermission
	case strings.Contains(output, "oci runtime"), strings.Contains(output, "runsc"), strings.Contains(output, "containerd"):
		return dynamicInstallFailureSandboxRuntime
	case strings.Contains(output, "not a supported wheel"), strings.Contains(output, "not supported wheel"):
		return dynamicInstallFailureWheelPlatform
	case strings.Contains(output, "invalid wheel"), strings.Contains(output, "bad wheel filename"), strings.Contains(output, "invalid metadata"):
		return dynamicInstallFailureWheelMetadata
	case strings.Contains(output, "resolutionimpossible"), strings.Contains(output, "conflicting dependencies"):
		return dynamicInstallFailurePackageConflict
	case strings.Contains(output, "already exists"), strings.Contains(output, "duplicate"):
		return dynamicInstallFailureDuplicate
	case strings.Contains(output, "no such option"), strings.Contains(output, "invalid requirement"), strings.Contains(output, "usage: pip"):
		return dynamicInstallFailurePipArgument
	default:
		return dynamicInstallFailureOther
	}
}

func classifyDynamicImportFailure(ctx context.Context, output string) string {
	if ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return dynamicImportFailureTimeout
	}
	output = strings.ToLower(output)
	switch {
	case strings.Contains(output, "no such file or directory"), strings.Contains(output, "cannot open shared object file"):
		return dynamicImportFailureMissingLibrary
	case strings.Contains(output, "failed to map segment"), strings.Contains(output, "permission denied"), strings.Contains(output, "operation not permitted"):
		return dynamicImportFailureDlopenPermission
	case strings.Contains(output, "wrong elf class"), strings.Contains(output, "exec format error"), strings.Contains(output, "invalid elf"):
		return dynamicImportFailureELFLoader
	case strings.Contains(output, "glibc_"), strings.Contains(output, "glibcxx_"), strings.Contains(output, "cxxabi_"):
		return dynamicImportFailureSymbolVersion
	case strings.Contains(output, "python abi"), strings.Contains(output, "abi version"), strings.Contains(output, "pyinit_"):
		return dynamicImportFailurePythonABI
	case strings.Contains(output, "cannot allocate memory"), strings.Contains(output, "out of memory"), strings.Contains(output, "memory limit"):
		return dynamicImportFailureMemory
	case strings.Contains(output, "resource temporarily unavailable"), strings.Contains(output, "pids limit"):
		return dynamicImportFailurePID
	case strings.Contains(output, "cpu time limit"):
		return dynamicImportFailureCPU
	case strings.Contains(output, "oci runtime"), strings.Contains(output, "runsc"), strings.Contains(output, "containerd"):
		return dynamicImportFailureSandboxRuntime
	case strings.Contains(output, "traceback"), strings.Contains(output, "importerror"), strings.Contains(output, "modulenotfounderror"):
		return dynamicImportFailureException
	default:
		return dynamicImportFailureOther
	}
}

func classifyDynamicObserverFailure(err error) string {
	if err == nil {
		return "OTHER"
	}
	var fault traceFault
	if !errors.As(err, &fault) {
		return "OTHER"
	}
	switch fault.TraceFaultReason() {
	case "PREVIOUS_TRACE_NOT_FINALIZED":
		return "PREVIOUS_TRACE_NOT_FINALIZED"
	case "HELPER_UNAVAILABLE":
		return "HELPER_UNAVAILABLE"
	case "HELPER_CRASHED":
		return "HELPER_CRASHED"
	case "EVENT_LIMIT":
		return "EVENT_LIMIT"
	case "BYTE_LIMIT":
		return "BYTE_LIMIT"
	case "CHANNEL_OVERFLOW":
		return "SESSION_LIMIT"
	case "READER_ERROR", "READER_TIMEOUT", "STREAM_FAULT", "FINALIZATION_TIMEOUT":
		return "TRACE_COLLECTION_FAILED"
	case "ATTRIBUTION_FAILURE", "CONTAINER_MISMATCH", "PROFILE_LOOKUP_FAILURE", "UNKNOWN_EVENT_KIND", "LIFECYCLE_ERROR":
		return "LIFECYCLE_ERROR"
	default:
		return "OTHER"
	}
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
func pythonDynamicCreateArguments(sessionID domain.SandboxSessionID, resourcePolicy artifactpypi.ResourcePolicy) []string {
	size := strconv.FormatInt(resourcePolicy.RuntimeTmpfs(), 10)
	return []string{"create", "--pull", "never", "--runtime", gVisorRuntimeName, "--network", "none", "--read-only", "--cap-drop", "ALL", "--cap-add", "SETUID", "--cap-add", "SETGID", "--cap-add", "SETPCAP", "--security-opt", "no-new-privileges", "--pids-limit", "64", "--memory", strconv.FormatInt(resourcePolicy.RuntimeMemory(), 10), "--cpus", "1", "--ulimit", "cpu=30:30", "--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=" + size + ",uid=1000,gid=1000,mode=0700", "--tmpfs", pythonSitePath + ":rw,exec,nosuid,nodev,size=" + size + ",uid=1000,gid=1000,mode=0700", "--tmpfs", boundaryHelperMount, "--name", "heliopause-pypi-" + sessionID.String(), pythonImageReference, "/bin/sh", "-ceu", boundaryContainerCommand()}
}

const pythonImportScript = "import importlib,sys\nsys.path.insert(0,'/haa-site')\nfor name in sys.argv[1:]: importlib.import_module(name)\n"

func pythonDynamicObserverProfile(rootProfileName string) (string, error) {
	switch rootProfileName {
	case "pypi":
		return "pypi-wheel", nil
	case "pytorch:cpu":
		return "pypi-wheel-pytorch-cpu", nil
	case "pytorch:cu126":
		return "pypi-wheel-pytorch-cu126", nil
	default:
		return "", errors.New("unsupported Python root source profile for dynamic observer")
	}
}
