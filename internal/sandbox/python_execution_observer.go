package sandbox

import (
	"context"
	"slices"
)

const pythonExecutionProfile = "python-execution"

type pythonExecutionObserver interface {
	StartPythonExecutionProfileWithBudget(context.Context, string, *observationTraceLedger) (TraceReader, error)
}

func (o *SharedObserver) StartPythonExecutionProfileWithBudget(ctx context.Context, container string, budget *observationTraceLedger) (TraceReader, error) {
	topology, ok := observerExpectedTopology(pythonExecutionProfile)
	if !ok || budget == nil {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	return o.startProfileWithBudget(ctx, container, pythonExecutionProfile, topology, budget)
}
func pythonExecutionWorkingDirectory(arguments []string) ([]string, error) {
	image := slices.Index(arguments, pythonImageReference)
	if image < 0 || slices.Contains(arguments, "--workdir") {
		return nil, errPythonExecution
	}
	return append(append(append([]string{}, arguments[:image]...), "--workdir", "/"), arguments[image:]...), nil
}
