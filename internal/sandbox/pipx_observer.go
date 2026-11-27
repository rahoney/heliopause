package sandbox

import "context"

const pipxObserverProfile = "github-pipx-runtime"

// The trusted lifecycle phase owns topology. The acquired ZIP cannot choose a
// mount, inherit wheel installation, or replenish the shared transaction budget.
func pipxObservationExpectedTopology(phase pythonClosurePhase) ([]observerMountExpectation, bool) {
	topology, ok := observerExpectedTopology(pipxObserverProfile)
	if !ok || (phase != pythonClosurePreparation && phase != pythonClosureAnchor && phase != pythonClosureObservation) {
		return nil, false
	}
	for i := range topology {
		if topology[i].Mountpoint == pipxRuntimeRoot {
			topology[i].ReadOnly = phase != pythonClosurePreparation
			return topology, true
		}
	}
	return nil, false
}

func (o *SharedObserver) StartPipxRuntimeProfileWithBudget(ctx context.Context, containerID string, phase pythonClosurePhase, budget *observationTraceLedger) (TraceReader, error) {
	if o == nil || ctx == nil || budget == nil || budget.maxEvents != uint64(defaultTraceBudget.events) || budget.maxBytes != defaultTraceBudget.bytes {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	topology, ok := pipxObservationExpectedTopology(phase)
	if !ok {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	return o.startProfileWithBudget(ctx, containerID, pipxObserverProfile, topology, budget)
}
