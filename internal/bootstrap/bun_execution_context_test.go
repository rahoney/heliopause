package bootstrap

import (
	"testing"

	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestBunProgramContextUsesSameApprovedOriginalWithDistinctRuntime(t *testing.T) {
	operation, resolution, set, proof := nodeContextFixture(t)
	node, err := newNodeExecutionContext(operation, resolution, set, proof, nil)
	if err != nil {
		t.Fatal(err)
	}
	bun, err := newJavaScriptExecutionContext(operation, resolution, set, proof, nil, sandbox.BunExecutionRuntimeIdentity())
	if err != nil || !bun.Valid() || bun.RuntimeIdentity() == node.RuntimeIdentity() || bun.EntryArtifact() != node.EntryArtifact() || bun.EntryPath() != node.EntryPath() || bun.EntryDigest() != node.EntryDigest() || bun.EntrySize() != node.EntrySize() || bun.SourceFingerprint() != node.SourceFingerprint() || bun.OperationID() != node.OperationID() {
		t.Fatal("Bun context replaced source facts or reused the Node runtime identity", err)
	}
}
