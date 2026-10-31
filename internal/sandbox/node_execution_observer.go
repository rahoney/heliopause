package sandbox

import (
	"context"
	"slices"
)

const nodeExecutionProfile = "node-execution"

type nodeExecutionObserver interface {
	nodeInputObserver
	StartNodeExecutionProfile(context.Context, string) (TraceReader, error)
}

func (o *SharedObserver) StartNodeExecutionProfile(ctx context.Context, container string) (TraceReader, error) {
	topology, ok := observerExpectedTopology(nodeExecutionProfile)
	if !ok {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	return o.startProfile(ctx, container, nodeExecutionProfile, topology)
}

// The initial stdout-only lane starts at the immutable image root. OCI init
// cannot traverse the user-owned private tmpfs before credential demotion.
// The approved Node entry is an absolute path in the complete readonly tree.
func nodeExecutionWorkingDirectory(arguments []string) ([]string, error) {
	image := slices.Index(arguments, nodeImageReference)
	if image < 0 || slices.Contains(arguments, "--workdir") {
		return nil, errNodeExecution
	}
	return append(append(append([]string{}, arguments[:image]...), "--workdir", "/"), arguments[image:]...), nil
}
