package sandbox

import (
	"context"
	"errors"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// InspectPipxRuntime observes acquired HAA release bytes with the existing
// controller ledger. It neither installs a wheel nor creates Domain Evidence.
func (b *PythonDynamicBackend) InspectPipxRuntime(ctx context.Context, artifact domain.AcquiredArtifact, plan artifactpypi.PipxRuntimePlan) (PythonObservationResult, error) {
	if b == nil || ctx == nil || b.runner == nil || b.observer == nil || b.probe == nil || b.newSessionID == nil {
		return PythonObservationResult{}, errors.New("pipx runtime backend is not configured")
	}
	input, err := preparePipxObservationInput(ctx, artifact, plan)
	if err != nil {
		return PythonObservationResult{}, err
	}
	session, err := b.newSessionID()
	if err != nil {
		return PythonObservationResult{}, err
	}
	incomplete := func(code string) (PythonObservationResult, error) {
		result, err := pythonIncomplete(session, code)
		return PythonObservationResult{SandboxResult: result}, err
	}
	if b.resources == nil {
		return incomplete("P4_PIPX_ACCOUNTING_UNAVAILABLE")
	}
	if _, ok := b.observer.(pipxRuntimeObserver); !ok {
		return incomplete("P4_PIPX_OBSERVER_UNAVAILABLE")
	}
	capability, err := b.probe(ctx)
	if err != nil || !capability.Available || capability.Runtime != PinnedPythonRuntime() {
		return incomplete("P4_PIPX_RUNTIME_UNAVAILABLE")
	}
	var commands []PythonCommandObservation
	result, err := b.executeObservationTransaction(ctx, session, pythonObservationSpec{
		artifact: input.artifact, policy: artifactpypi.PublicPyPIProfile().ResourcePolicy(), profile: pipxObserverProfile, resourceClass: "pypi", units: input.units, pipx: &input,
	}, &commands)
	return PythonObservationResult{SandboxResult: result, Commands: commands}, err
}
