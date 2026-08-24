package sandbox

import (
	"context"
	"runtime"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// NewLinuxPyPIDynamicBackendWithExecutor uses the composition-root validated
// Host executor for every Docker operation.
func NewLinuxPyPIDynamicBackendWithExecutor(intakeRoot string, executor TrustedExecutor) (*PythonDynamicBackend, func() error, error) {
	return newLinuxPyPIDynamicBackend(intakeRoot, executor)
}

func newLinuxPyPIDynamicBackend(intakeRoot string, executor TrustedExecutor) (*PythonDynamicBackend, func() error, error) {
	observer, err := NewSharedObserver(ObserverOutputEndpoint)
	if err != nil {
		return nil, nil, err
	}
	introducer, err := NewPythonArtifactIntroducer(intakeRoot, executor)
	if err != nil {
		_ = observer.Close()
		return nil, nil, err
	}
	capabilityProbe := func(ctx context.Context) (PythonCapability, error) {
		return probePython(ctx, runtime.GOOS, runtime.GOARCH, executor)
	}
	backend, err := NewPythonDynamicBackend(executor, introducer, observer, capabilityProbe)
	if err != nil {
		_ = observer.Close()
		return nil, nil, err
	}
	return backend, observer.Close, nil
}

// NewLinuxPyPISdistBuilderWithExecutor uses the composition-root validated
// Host executor for the build lifecycle.
func NewLinuxPyPISdistBuilderWithExecutor(intakeRoot string, executor TrustedExecutor) (*PythonSdistBuilder, func() error, error) {
	return newLinuxPyPISdistBuilder(intakeRoot, executor)
}

func newLinuxPyPISdistBuilder(intakeRoot string, executor TrustedExecutor) (*PythonSdistBuilder, func() error, error) {
	observer, err := NewSharedObserver(ObserverOutputEndpoint)
	if err != nil {
		return nil, nil, err
	}
	introducer, err := NewPythonArtifactIntroducer(intakeRoot, executor)
	if err != nil {
		_ = observer.Close()
		return nil, nil, err
	}
	capabilityProbe := func(ctx context.Context) (PythonCapability, error) {
		return probePython(ctx, runtime.GOOS, runtime.GOARCH, executor)
	}
	builder, err := NewPythonSdistBuilder(executor, introducer, observer, capabilityProbe)
	if err != nil {
		_ = observer.Close()
		return nil, nil, err
	}
	return builder, observer.Close, nil
}

// UnavailablePythonWheelRunner makes unsupported hosts explicit incomplete
// inspection results without attempting Docker, Python, or Host execution.
type UnavailablePythonWheelRunner struct{}

func (UnavailablePythonWheelRunner) InspectWheel(context.Context, domain.AcquiredArtifact, []string) (domain.SandboxResult, error) {
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		return domain.SandboxResult{}, err
	}
	return domain.NewSandboxResult(session, domain.SandboxIncomplete, "M5_PYPI_DYNAMIC_RUNTIME_UNAVAILABLE", nil)
}
