package bootstrap

import (
	"context"
	"errors"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestPythonExecutionRuntimePreservesCanceledCauseBeforeReader(t *testing.T) {
	cause := errors.New("caller canceled before current console reread")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	outcome, err := (pythonExecutionRuntime{}).Execute(ctx, domain.ExecutionContext{})
	if outcome != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal("early unavailable runtime discarded original caller cause", err)
	}
}

func TestPythonExecutionRuntimeRequiresActualAdapters(t *testing.T) {
	if service, err := newPythonToolExecutionService(nil, nil, nil, nil, nil, nil, nil); service != nil || err == nil {
		t.Fatal("missing real adapters became configured workflow")
	}
	for _, ctx := range []context.Context{nil, context.Background()} {
		if outcome, err := (pythonExecutionRuntime{}).Execute(ctx, domain.ExecutionContext{}); outcome != nil || err == nil {
			t.Fatal("unavailable current runtime became completed data")
		}
	}
}
