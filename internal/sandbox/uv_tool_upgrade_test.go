package sandbox

import (
	"context"
	"errors"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

func TestUVToolUpgradeRefusesInvalidOriginalBeforeInfrastructure(t *testing.T) {
	for _, name := range []string{"empty", "missing WHEEL", "changed metadata", "canceled", "nil context"} {
		t.Run(name, func(t *testing.T) {
			input, wheels := toolReplayFixture(t, false)
			_, runner := uvReaderFixture(t)
			endpoints := &uvCountingEndpoints{}
			resolver, err := NewUVResolver(runner, endpoints, &recordingObserver{}, availablePythonProbe, &recordingResolverPolicyService{})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch name {
			case "empty":
				input = artifactpypi.UVToolUpgradeInput{}
			case "missing WHEEL":
				wheels[0].Wheel = nil
			case "changed metadata":
				wheels[0].Metadata = []byte("changed")
			case "canceled":
				cancel()
			case "nil context":
				ctx = nil
			}
			result, err := resolver.ResolveToolUpgrade(ctx, input, wheels)
			if err == nil || len(result.Controls()) != 0 || len(result.Resolution().Graph().Nodes()) != 0 || endpoints.calls != 0 || len(runner.calls) != 0 || len(runner.inputCalls) != 0 {
				t.Fatal("invalid original reached infrastructure or produced selection", err)
			}
			if name == "canceled" && !errors.Is(err, context.Canceled) {
				t.Fatal("primary cancellation lost", err)
			}
		})
	}
}
