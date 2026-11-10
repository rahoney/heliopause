package bootstrap

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type nodeExecutionRuntime struct {
	backend *sandbox.Backend
	static  *inspectionnpm.Inspector
}

func (r nodeExecutionRuntime) Execute(ctx context.Context, execution domain.ExecutionContext) (application.ExecutionRuntimeOutcome, error) {
	layout, err := materializeNodeExecution(ctx, r.backend, r.static, execution)
	if err != nil {
		return nil, err
	}
	return r.backend.ExecuteNodeExecution(ctx, execution, layout)
}

func newNPMExecutionService(resolver *sandbox.NPMResolver, source *artifactnpm.Resolver, static *inspectionnpm.Inspector, backend *sandbox.Backend, store *local.Store, staging *promotion.LocalStaging) (*application.ExecutionService, error) {
	if backend == nil || static == nil {
		return nil, errors.New("npm execution requires actual runtime and inspection adapters")
	}
	preparation, err := newNPMExecutionPreparationService(resolver, source, static, backend, store, staging)
	if err != nil {
		return nil, err
	}
	return application.NewExecutionService(preparation, nodeExecutionRuntime{backend, static})
}
