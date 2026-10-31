package application

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// ExecutionRuntimeOutcome is private runtime DATA. Complete includes the
// backend's terminal reconciliation, observation and whole guest disposal;
// it grants no inspection or Policy authority.
type ExecutionRuntimeOutcome interface {
	Complete() bool
	ExitCode() (uint8, bool)
	Signal() (uint8, bool)
	Stdout() []byte
	Stderr() []byte
}

type ExecutionRuntime interface {
	Execute(context.Context, domain.ExecutionContext) (ExecutionRuntimeOutcome, error)
}

// ExecutionResult separates fresh quarantine/storage preparation from the
// program's completed execution. Private request/output data is not persisted.
type ExecutionResult struct {
	preparation    PreparedExecution
	selection      PreparedExecutionSelection
	stdout, stderr []byte
	code, signal   uint8
	exited         bool
	complete       bool
	attempted      bool
	failure        string
}

func (r ExecutionResult) Preparation() PreparedExecution                   { return r.preparation }
func (r ExecutionResult) SelectionPreparation() PreparedExecutionSelection { return r.selection }
func (r ExecutionResult) Complete() bool                                   { return r.complete }
func (r ExecutionResult) FailureCode() string                              { return r.failure }

// RuntimeAttempted reports invocation of the runtime adapter, not guest body entry.
func (r ExecutionResult) RuntimeAttempted() bool { return r.attempted }
func (r ExecutionResult) ExitCode() (uint8, bool) {
	if !r.complete || !r.exited {
		return 0, false
	}
	return r.code, true
}
func (r ExecutionResult) Signal() (uint8, bool) {
	if !r.complete || r.exited {
		return 0, false
	}
	return r.signal, true
}
func (r ExecutionResult) Stdout() []byte {
	return append([]byte(nil), r.stdout...)
}
func (r ExecutionResult) Stderr() []byte {
	return append([]byte(nil), r.stderr...)
}
func (ExecutionResult) String() string     { return "private execution result" }
func (r ExecutionResult) GoString() string { return r.String() }
func (ExecutionResult) MarshalJSON() ([]byte, error) {
	return nil, errors.New("execution result contains private memory data")
}

type ExecutionService struct {
	preparation *ExecutionPreparationService
	runtime     ExecutionRuntime
}

func NewExecutionService(preparation *ExecutionPreparationService, runtime ExecutionRuntime) (*ExecutionService, error) {
	if preparation == nil || runtime == nil {
		return nil, errors.New("execution requires current preparation and a runtime")
	}
	return &ExecutionService{preparation, runtime}, nil
}

func (s *ExecutionService) Run(ctx context.Context, request ExecutionPreparationRequest) (result ExecutionResult, resultErr error) {
	defer func() {
		if ctx != nil && ctx.Err() != nil {
			result.complete = false
			result.failure = "CANCELED"
			resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx))
		}
	}()
	if ctx == nil || s == nil || s.preparation == nil || s.runtime == nil {
		result.failure = "REQUEST_INVALID"
		return result, errors.New("execution requires a context and configured workflow")
	}
	validated, err := NewExecutionPreparationRequest(request.Reference(), request.Command(), request.Arguments())
	if err != nil {
		result.failure = "REQUEST_INVALID"
		return result, err
	}
	result.preparation, err = s.preparation.Prepare(ctx, validated)
	if err != nil || !result.preparation.Valid() {
		result.failure = "PREPARATION_FAILED"
		return result, errors.Join(errors.New("execution preparation did not complete"), err)
	}
	return executePreparedExecution(ctx, result, func() (ExecutionRuntimeOutcome, error) {
		return s.runtime.Execute(ctx, result.preparation.ExecutionContext())
	})
}

// The runtime evidence, output capture and terminal rules are identical for
// connected and explicit-root preparations.
func executePreparedExecution(ctx context.Context, result ExecutionResult, execute func() (ExecutionRuntimeOutcome, error)) (ExecutionResult, error) {
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "execute"})
	result.attempted = true
	outcome, resultErr := execute()
	result.failure = "RUNTIME_INCOMPLETE"
	if outcome == nil {
		return result, errors.Join(errors.New("execution runtime did not complete"), resultErr)
	}
	result.stdout, result.stderr = append([]byte(nil), outcome.Stdout()...), append([]byte(nil), outcome.Stderr()...)
	if !outcome.Complete() {
		return result, errors.Join(errors.New("execution runtime did not complete"), resultErr)
	}
	code, exited := outcome.ExitCode()
	signal, signaled := outcome.Signal()
	if exited == signaled || signaled && signal == 0 || exited && code == 0 && resultErr != nil || (signaled || code != 0) && resultErr == nil {
		return result, errors.Join(errors.New("execution terminal and error evidence disagree"), resultErr)
	}
	result.complete, result.failure = true, ""
	result.code, result.signal, result.exited = code, signal, exited
	return result, resultErr
}
