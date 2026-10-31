package hosttool

import (
	"context"
	"errors"
	"slices"
	"time"
)

const (
	maximumExecutionStreamBytes = 1 << 20
	executionPipeWaitDelay      = 2 * time.Second
)

// ErrExecutionOutputLimit reports incomplete capture, never successful truncation.
var ErrExecutionOutputLimit = errors.New("execution output exceeded its stream limit")

// CapturedOutput contains untrusted process-local bytes, not guest completion
// authority. Callers must separately establish admission, lifecycle and cleanup.
type CapturedOutput struct {
	stdout, stderr                 []byte
	stdoutOverflow, stderrOverflow bool
}

func (o CapturedOutput) Stdout() []byte       { return slices.Clone(o.stdout) }
func (o CapturedOutput) Stderr() []byte       { return slices.Clone(o.stderr) }
func (o CapturedOutput) StdoutOverflow() bool { return o.stdoutOverflow }
func (o CapturedOutput) StderrOverflow() bool { return o.stderrOverflow }

func (CapturedOutput) String() string   { return "private execution output" }
func (CapturedOutput) GoString() string { return "private execution output" }

func (CapturedOutput) MarshalJSON() ([]byte, error) {
	return nil, errors.New("execution output is private memory data")
}

// RunCaptured retains separate bounded streams from an already registered Host
// tool. Overflow cancels this Host process; it does not prove a Docker guest or
// descendant stopped. The original process error and cancellation cause survive.
func (e *Executor) RunCaptured(ctx context.Context, name string, arguments ...string) (CapturedOutput, error) {
	if ctx == nil {
		return CapturedOutput{}, errors.New("trusted command context is required")
	}
	if ctx.Err() != nil {
		return CapturedOutput{}, errors.Join(ctx.Err(), context.Cause(ctx))
	}
	runCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	command, err := e.command(runCtx, name, arguments...)
	if err != nil {
		return CapturedOutput{}, err
	}
	stdout := executionStream{cancel: cancel}
	stderr := executionStream{cancel: cancel}
	command.Stdout, command.Stderr = &stdout, &stderr
	command.WaitDelay = executionPipeWaitDelay
	err = command.Run()
	output := CapturedOutput{stdout.body, stderr.body, stdout.overflow, stderr.overflow}
	if output.stdoutOverflow || output.stderrOverflow {
		err = errors.Join(err, ErrExecutionOutputLimit)
	}
	return output, errors.Join(err, ctx.Err(), context.Cause(runCtx))
}

type executionStream struct {
	body     []byte
	overflow bool
	cancel   context.CancelCauseFunc
}

func (s *executionStream) Write(body []byte) (int, error) {
	remaining := maximumExecutionStreamBytes - len(s.body)
	accepted := min(remaining, len(body))
	if accepted > 0 {
		if s.body == nil {
			s.body = make([]byte, 0, maximumExecutionStreamBytes)
		}
		s.body = append(s.body, body[:accepted]...)
	}
	if accepted < len(body) {
		s.overflow = true
		s.cancel(ErrExecutionOutputLimit)
	}
	// Drain until process cancellation or WaitDelay closes the pipe. Returning
	// a short write here would obscure the explicit overflow/cancellation cause.
	return len(body), nil
}
