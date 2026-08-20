package sandbox

import (
	"context"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// NewLinuxPyPIDynamicBackend composes the PyPI wheel observer boundary without
// leaking the process runner or gVisor types into bootstrap/Application.
func NewLinuxPyPIDynamicBackend(intakeRoot string) (*PythonDynamicBackend, func() error, error) {
	observer, err := NewSharedObserver(ObserverOutputEndpoint)
	if err != nil {
		return nil, nil, err
	}
	introducer, err := NewPythonArtifactIntroducer(intakeRoot, systemExecutor{})
	if err != nil {
		_ = observer.Close()
		return nil, nil, err
	}
	backend, err := NewPythonDynamicBackend(systemExecutor{}, introducer, observer, ProbePython)
	if err != nil {
		_ = observer.Close()
		return nil, nil, err
	}
	return backend, observer.Close, nil
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
