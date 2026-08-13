package sandbox

import (
	"context"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestProbedSandboxFailsClosedForUnsupportedAndUnwiredRuntime(t *testing.T) {
	for _, test := range []struct {
		name, want string
		probe      CapabilityProbe
	}{
		{"unsupported", "M3_LINUX_ONLY", func(context.Context) (Capability, error) { return Capability{LimitationCode: "M3_LINUX_ONLY"}, nil }},
		{"unwired", "M3_DYNAMIC_RUNTIME_UNWIRED", func(context.Context) (Capability, error) { return Capability{Available: true}, nil }},
	} {
		t.Run(test.name, func(t *testing.T) {
			sandbox, err := NewProbedSandbox(test.probe)
			if err != nil {
				t.Fatal(err)
			}
			result, err := sandbox.Execute(context.Background(), sandboxRequest(t))
			if err != nil || result.Status() != domain.SandboxIncomplete {
				t.Fatalf("Execute() = (%q, %v)", result.Status(), err)
			}
			if code, _ := result.LimitationCode(); code != test.want {
				t.Fatalf("LimitationCode() = %q", code)
			}
		})
	}
}
