package application

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type ExecutionSelectionRuntime interface {
	ExecuteSelection(context.Context, domain.ExecutionSelectionContext) (ExecutionRuntimeOutcome, error)
}

type ExecutionSelectionService struct {
	preparation *ExecutionSelectionPreparationService
	runtime     ExecutionSelectionRuntime
}

func NewExecutionSelectionService(preparation *ExecutionSelectionPreparationService, runtime ExecutionSelectionRuntime) (*ExecutionSelectionService, error) {
	if preparation == nil || runtime == nil {
		return nil, errors.New("execution requires current complete-root preparation and runtime")
	}
	return &ExecutionSelectionService{preparation, runtime}, nil
}
func (s *ExecutionSelectionService) Run(ctx context.Context, request ExecutionPreparationRequest) (result ExecutionResult, resultErr error) {
	defer func() {
		if ctx != nil && ctx.Err() != nil {
			result.complete, result.failure = false, "CANCELED"
			resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx))
		}
	}()
	if ctx == nil || s == nil || s.preparation == nil || s.runtime == nil {
		result.failure = "REQUEST_INVALID"
		return result, errors.New("execution requires a context and configured complete-root workflow")
	}
	validated, err := NewExecutionPreparationRequest(request.Reference(), request.Command(), request.Arguments())
	if err != nil {
		result.failure = "REQUEST_INVALID"
		return result, err
	}
	result.selection, err = s.preparation.Prepare(ctx, validated)
	if err != nil || !result.selection.Valid() {
		result.failure = "PREPARATION_FAILED"
		return result, errors.Join(errors.New("execution preparation did not complete"), err)
	}
	return executePreparedExecution(ctx, result, func() (ExecutionRuntimeOutcome, error) {
		return s.runtime.ExecuteSelection(ctx, result.selection.ExecutionContext())
	})
}
