package bootstrap

import (
	"context"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestNPMExecutionFactoryAndAdapterRejectMissingRuntime(t *testing.T) {
	if service, err := newNPMExecutionService(nil, nil, nil, nil, nil, nil); service != nil || err == nil {
		t.Fatal("missing actual execution adapters accepted")
	}
	if outcome, err := (nodeExecutionRuntime{}).Execute(context.Background(), domain.ExecutionContext{}); outcome != nil || err == nil {
		t.Fatal("empty execution adapter accepted")
	}
}
