package bootstrap

import (
	"errors"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// This composition binds the existing Node byte proof to the canonical runtime.
// It performs no execution and cannot replace fresh whole-set inspection.
func newNodeExecutionContext(operation domain.OperationID, resolution domain.DependencyResolution, set domain.VerifiedSet, node artifactnpm.NodeExecutionEntry, arguments []string) (domain.ExecutionContext, error) {
	return newJavaScriptExecutionContext(operation, resolution, set, node, arguments, runtimeidentity.NodeImageReference)
}

func newJavaScriptExecutionContext(operation domain.OperationID, resolution domain.DependencyResolution, set domain.VerifiedSet, node artifactnpm.NodeExecutionEntry, arguments []string, runtime string) (domain.ExecutionContext, error) {
	if !node.Valid() {
		return domain.ExecutionContext{}, errors.New("node execution requires the selected entry byte proof")
	}
	entry := node.Entry()
	member := entry.Target()
	return domain.NewExecutionContext(domain.ExecutionContextData{
		OperationID: operation, Resolution: resolution, Set: set,
		EntryArtifact: entry.Artifact(), Command: entry.Command(),
		EntryPath: member.Path, EntryDigest: member.Digest, EntrySize: uint64(member.Size),
		RuntimeIdentity: runtime, Arguments: arguments,
	})
}
