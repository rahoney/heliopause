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

type nodeExecutionEntryBinder struct{ static *inspectionnpm.Inspector }

func (b nodeExecutionEntryBinder) BindExecutionEntry(ctx context.Context, inspection application.ExecutionInspection, command string) (domain.ExecutionContext, error) {
	return bindNodeExecutionEntry(ctx, b.static, inspection, command, artifactnpm.SelectExecutionEntry)
}

func bindNodeExecutionEntry(ctx context.Context, static *inspectionnpm.Inspector, inspection application.ExecutionInspection, command string, selectEntry func(artifactnpm.ArchiveInventory, string) (artifactnpm.ExecutionEntry, error)) (domain.ExecutionContext, error) {
	return bindJavaScriptExecutionEntry(ctx, static, inspection, command, selectEntry, "")
}

func bindJavaScriptExecutionEntry(ctx context.Context, static *inspectionnpm.Inspector, inspection application.ExecutionInspection, command string, selectEntry func(artifactnpm.ArchiveInventory, string) (artifactnpm.ExecutionEntry, error), runtime string) (domain.ExecutionContext, error) {
	if ctx == nil || static == nil || !inspection.Valid() {
		return domain.ExecutionContext{}, errors.New("node entry binding requires actual static inspection and current whole approval")
	}
	var primary domain.AcquiredArtifact
	for _, entry := range inspection.Set().Inspected().Inspections() {
		if entry.Node() == inspection.Resolution().Graph().Primary() {
			primary = entry.Artifact()
		}
	}
	inventory, report, err := static.InspectArchive(ctx, primary)
	if err != nil {
		return domain.ExecutionContext{}, err
	}
	if !inventory.Valid() || len(report.Findings()) != 0 {
		return domain.ExecutionContext{}, errors.New("current primary has no complete measured archive inventory")
	}
	selected, err := selectEntry(inventory, command)
	if err != nil {
		return domain.ExecutionContext{}, err
	}
	proof, err := static.ReadNodeExecutionEntry(ctx, selected)
	if err != nil {
		return domain.ExecutionContext{}, err
	}
	if runtime != "" {
		return newJavaScriptExecutionContext(inspection.OperationID(), inspection.Resolution(), inspection.Set(), proof, nil, runtime)
	}
	return newNodeExecutionContext(inspection.OperationID(), inspection.Resolution(), inspection.Set(), proof, nil)
}

// All concrete adapters are supplied by the caller's trusted runtime/storage
// setup. There is no Host installation target or Promotion dependency.
func newNPMExecutionPreparationService(resolver *sandbox.NPMResolver, source *artifactnpm.Resolver, static *inspectionnpm.Inspector, backend *sandbox.Backend, store *local.Store, staging *promotion.LocalStaging) (*application.ExecutionPreparationService, error) {
	if resolver == nil || source == nil || static == nil || backend == nil || store == nil || staging == nil {
		return nil, errors.New("npm execution preparation requires actual resolver, source, inspection and storage adapters")
	}
	dynamic, err := inspectionnpm.NewDynamicInspector(backend)
	if err != nil {
		return nil, err
	}
	inspector, err := inspectionnpm.NewCompositeInspector(static, dynamic)
	if err != nil {
		return nil, err
	}
	fresh, err := application.NewExecutionInspectService(source, verificationnpm.IntegrityVerifier{}, inspector, store, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		return nil, err
	}
	return application.NewExecutionPreparationService(resolver, fresh, nodeExecutionEntryBinder{static}, evidence.Generator{}, staging)
}
