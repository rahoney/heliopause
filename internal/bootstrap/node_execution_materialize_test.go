package bootstrap

import (
	"context"
	"errors"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestNodeMaterializationAdapterRejectsBeforeSourceIO(t *testing.T) {
	t.Parallel()
	operation, resolution, set, proof := nodeContextFixture(t)
	execution, err := newNodeExecutionContext(operation, resolution, set, proof, nil)
	if err != nil {
		t.Fatal(err)
	}
	static, err := inspectionnpm.NewInspector(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	backend := &sandbox.Backend{}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("fixture stopped before source access")
	cancel(cause)
	wrong, err := domain.NewExecutionContext(domain.ExecutionContextData{
		OperationID: operation, Resolution: resolution, Set: set, EntryArtifact: execution.EntryArtifact(), Command: execution.Command(), EntryPath: execution.EntryPath(), EntryDigest: execution.EntryDigest(), EntrySize: execution.EntrySize(), RuntimeIdentity: "different-runtime",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []struct {
		name      string
		ctx       context.Context
		backend   *sandbox.Backend
		static    *inspectionnpm.Inspector
		execution domain.ExecutionContext
		canceled  bool
	}{
		{"nil context", nil, backend, static, execution, false},
		{"nil backend", context.Background(), nil, static, execution, false},
		{"nil inspector", context.Background(), backend, nil, execution, false},
		{"zero context", context.Background(), backend, static, domain.ExecutionContext{}, false},
		{"different runtime", context.Background(), backend, static, wrong, false},
		{"canceled", ctx, backend, static, execution, true},
	} {
		t.Run(input.name, func(t *testing.T) {
			layout, err := materializeNodeExecution(input.ctx, input.backend, input.static, input.execution)
			if err == nil || layout.Valid() {
				t.Fatal("invalid input reached source/runtime")
			}
			if input.canceled && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
				t.Fatal("original cancellation cause lost", err)
			}
		})
	}
}
