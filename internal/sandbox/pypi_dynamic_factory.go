package sandbox

import (
	"context"
	"errors"
	"runtime"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// NewLinuxPyPIDynamicBackendWithExecutor uses the composition-root validated
// Host executor for every Docker operation.
func NewLinuxPyPIDynamicBackendWithExecutor(intakeRoot string, executor TrustedExecutor, observer TraceObserver) (*PythonDynamicBackend, error) {
	return newLinuxPyPIDynamicBackend(intakeRoot, executor, observer)
}

func newLinuxPyPIDynamicBackend(intakeRoot string, executor TrustedExecutor, observer TraceObserver) (*PythonDynamicBackend, error) {
	if observer == nil {
		return nil, errors.New("process-scoped observer is required")
	}
	introducer, err := NewPythonArtifactIntroducer(intakeRoot, executor)
	if err != nil {
		return nil, err
	}
	capabilityProbe := func(ctx context.Context) (PythonCapability, error) {
		return probePython(ctx, runtime.GOOS, runtime.GOARCH, executor)
	}
	backend, err := NewPythonDynamicBackend(executor, introducer, observer, capabilityProbe)
	if err != nil {
		return nil, err
	}
	return backend, nil
}

// NewLinuxPyPISdistBuilderWithExecutor uses the composition-root validated
// Host executor for the build lifecycle.
func NewLinuxPyPISdistBuilderWithExecutor(intakeRoot string, executor TrustedExecutor, observer TraceObserver) (*PythonSdistBuilder, error) {
	return newLinuxPyPISdistBuilder(intakeRoot, executor, observer)
}

func newLinuxPyPISdistBuilder(intakeRoot string, executor TrustedExecutor, observer TraceObserver) (*PythonSdistBuilder, error) {
	if observer == nil {
		return nil, errors.New("process-scoped observer is required")
	}
	introducer, err := NewPythonArtifactIntroducer(intakeRoot, executor)
	if err != nil {
		return nil, err
	}
	capabilityProbe := func(ctx context.Context) (PythonCapability, error) {
		return probePython(ctx, runtime.GOOS, runtime.GOARCH, executor)
	}
	builder, err := NewPythonSdistBuilder(executor, introducer, observer, capabilityProbe)
	if err != nil {
		return nil, err
	}
	return builder, nil
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
