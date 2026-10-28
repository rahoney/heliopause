package hosttool

import (
	"context"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

var _ ports.ExecutionCapture = (*Executor)(nil)

// Preserve the qualified capture implementation and concrete output/error
// identity while exposing only the neutral Core data contract to consumers.
func (e *Executor) CaptureExecutionOutput(ctx context.Context, name string, arguments ...string) (domain.CapturedProcessOutput, error) {
	return e.RunCaptured(ctx, name, arguments...)
}
