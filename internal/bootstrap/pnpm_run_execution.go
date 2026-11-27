package bootstrap

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationnpm "github.com/rahoney/heliopause/internal/verification/npm"
)

type pnpmRunExecutionWorkflow struct {
	native  *sandbox.PNPMResolver
	source  *artifactnpm.Resolver
	static  *inspectionnpm.Inspector
	prepare *sandbox.PNPMInspectionPreparer
	backend *sandbox.Backend
	store   *local.Store
	staging *promotion.LocalStaging
}

func (r pnpmRunExecutionWorkflow) Run(ctx context.Context, request application.ExecutionPreparationRequest) (application.ExecutionResult, error) {
	if ctx == nil || r.native == nil || r.source == nil || r.static == nil || r.prepare == nil || r.backend == nil || r.store == nil || r.staging == nil {
		return application.ExecutionResult{}, errors.New("pnpm current execution requires its actual source, inspection and runtime consumers")
	}
	if err := artifactnpm.ValidatePNPMRunReference(request.Reference()); err != nil {
		return application.ExecutionResult{}, err
	}
	frozen, err := r.native.ResolveToolRun(ctx, request.Reference(), r.source)
	if err != nil {
		return application.ExecutionResult{}, err
	}
	inspector, err := inspectionnpm.NewPNPMExecutionInspector(frozen, r.static, r.prepare, r.backend)
	if err != nil {
		return application.ExecutionResult{}, err
	}
	fresh, err := application.NewExecutionInspectService(r.source, verificationnpm.IntegrityVerifier{}, inspector, r.store, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		return application.ExecutionResult{}, err
	}
	preparation, err := application.NewExecutionPreparationService(pnpmRunFrozenResolver{request.Reference(), frozen}, fresh, pnpmRunEntryBinder{r.static, frozen}, evidence.Generator{}, r.staging)
	if err != nil {
		return application.ExecutionResult{}, err
	}
	service, err := application.NewExecutionService(preparation, pnpmRunExecutionRuntime{r.backend, r.static, inspector, frozen})
	if err != nil {
		return application.ExecutionResult{}, err
	}
	return service.Run(ctx, request)
}

type pnpmRunFrozenResolver struct {
	reference domain.ArtifactReference
	frozen    artifactnpm.PNPMInspectionContext
}

func (r pnpmRunFrozenResolver) ResolveExecutionDependencies(ctx context.Context, reference domain.ArtifactReference) (domain.DependencyResolution, error) {
	if ctx == nil || ctx.Err() != nil || reference != r.reference || !r.frozen.MatchesExecutionResolution(r.frozen.ExecutionResolution()) {
		return domain.DependencyResolution{}, errors.New("pnpm current request differs from its native source selection")
	}
	return r.frozen.ExecutionResolution(), nil
}

type pnpmRunEntryBinder struct {
	static *inspectionnpm.Inspector
	frozen artifactnpm.PNPMInspectionContext
}

func (b pnpmRunEntryBinder) BindExecutionEntry(ctx context.Context, inspection application.ExecutionInspection, command string) (domain.ExecutionContext, error) {
	if !b.frozen.MatchesExecutionResolution(inspection.Resolution()) {
		return domain.ExecutionContext{}, errors.New("pnpm entry binding differs from current native whole inspection")
	}
	return bindNodeExecutionEntry(ctx, b.static, inspection, command, artifactnpm.SelectPNPMExecutionEntry)
}

type pnpmRunExecutionRuntime struct {
	backend   *sandbox.Backend
	static    *inspectionnpm.Inspector
	inspector *inspectionnpm.PNPMExecutionInspector
	frozen    artifactnpm.PNPMInspectionContext
}

func (r pnpmRunExecutionRuntime) Execute(ctx context.Context, execution domain.ExecutionContext) (application.ExecutionRuntimeOutcome, error) {
	if ctx == nil || r.backend == nil || r.static == nil || r.inspector == nil || !r.frozen.MatchesExecutionResolution(execution.Resolution()) {
		return nil, errors.New("pnpm runtime requires the same complete current inspection")
	}
	acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
	for _, inspection := range execution.Set().Inspected().Inspections() {
		acquired[inspection.Node()] = inspection.Artifact()
	}
	prepared, err := r.inspector.PreparedLayoutFor(execution.Resolution(), acquired)
	if err != nil {
		return nil, err
	}
	inventory, report, err := r.static.InspectArchive(ctx, execution.EntryArtifact())
	if err != nil || !inventory.Valid() || len(report.Findings()) != 0 {
		return nil, errors.Join(errors.New("pnpm current primary has no complete measured original"), err)
	}
	layout, err := artifactnpm.BindPNPMNodeExecutionLayout(ctx, execution, prepared, inventory)
	if err != nil {
		return nil, err
	}
	return r.backend.ExecuteNodeExecution(ctx, execution, layout)
}
