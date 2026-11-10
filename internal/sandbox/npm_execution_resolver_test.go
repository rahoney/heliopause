package sandbox

import (
	"context"
	"net/netip"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
)

func TestNPMExecutionResolverUsesExistingIsolatedSelectionWithoutTarget(t *testing.T) {
	runner := &recordingRunner{responses: [][]byte{
		[]byte("0123456789abcdef"), []byte("172.30.0.0/24"), []byte("0123456789ab"), nil, nil, []byte(resolverNPMVersion), nil, []byte(resolverLockJSON()),
	}}
	observer := &sequencedObserver{}
	service := &recordingResolverPolicyService{}
	resolver, err := NewNPMResolverWithObserver(runner, staticEndpoints{addresses: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}, observer, service)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := artifactnpm.ParseReference("primary@1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := resolver.ResolveExecutionDependencies(context.Background(), reference)
	if err != nil || len(resolution.Graph().Nodes()) != 1 || resolution.RuntimeIdentity() != resolverRuntimeIdentity || resolution.LockfileDigest().String() == "" {
		t.Fatal("targetless selection changed existing graph/runtime/lock", err)
	}
	executions := 0
	for _, call := range runner.calls {
		if call.binary == "docker" && len(call.arguments) > 1 && call.arguments[0] == "exec" {
			executions++
			assertBoundaryExecArguments(t, call.arguments, "0123456789ab")
		}
	}
	if executions < 4 || len(runner.inputCalls) != 1 {
		t.Fatal("existing protected selection sequence bypassed")
	}
	assertBoundaryInputExecArguments(t, runner.inputCalls[0].arguments, "0123456789ab")
}
