package ports

import (
	"context"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// ExecutionCapture consumes an already registered protected Host tool. The
// caller separately owns guest admission, target/exit facts and full cleanup.
type ExecutionCapture interface {
	CaptureExecutionOutput(context.Context, string, ...string) (domain.CapturedProcessOutput, error)
}
