package bootstrap

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionpypi "github.com/rahoney/heliopause/internal/inspection/pypi"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type pythonExecutionRuntime struct {
	backend *sandbox.PythonDynamicBackend
	static  *inspectionpypi.StaticInspector
}

func (r pythonExecutionRuntime) Execute(ctx context.Context, execution domain.ExecutionContext) (outcome application.ExecutionRuntimeOutcome, resultErr error) {
	if ctx == nil {
		return nil, errors.New("python runtime requires a caller context")
	}
	defer func() { resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx)) }()
	if r.backend == nil || r.static == nil || ctx.Err() != nil || !execution.Valid() {
		return nil, errors.New("python runtime requires current execution and actual inspection/runtime adapters")
	}
	inventory, err := r.static.ReadConsoleExecutionInventory(ctx, execution.EntryArtifact())
	if err != nil {
		return nil, err
	}
	entry, err := artifactpypi.SelectConsoleExecutionEntry(inventory, execution.Command())
	if err != nil {
		return nil, err
	}
	return r.backend.ExecutePythonExecution(ctx, execution, entry)
}

// This internal factory preserves the actual fresh tool-install selector. It
// does not claim native UVx/run equivalence or expose an execution CLI alias.
func newPythonToolExecutionService(native *sandbox.UVResolver, intake *artifactpypi.Intake, inspector *inspectionpypi.CompositeInspector, static *inspectionpypi.StaticInspector, backend *sandbox.PythonDynamicBackend, store *local.Store, staging *promotion.LocalStaging) (*application.ExecutionService, error) {
	if backend == nil || static == nil {
		return nil, errors.New("python execution requires actual runtime and inspection adapters")
	}
	preparation, err := newPythonExecutionPreparationService(native, intake, inspector, static, store, staging)
	if err != nil {
		return nil, err
	}
	return application.NewExecutionService(preparation, pythonExecutionRuntime{backend, static})
}
