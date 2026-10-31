package bootstrap

import (
	"context"
	"errors"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// This adapter supplies only freshly measured source facts and generated
// controller data. Sandbox owns the disposable runtime and its final cleanup.
func materializeNodeExecution(ctx context.Context, backend *sandbox.Backend, static *inspectionnpm.Inspector, execution domain.ExecutionContext) (artifactnpm.NodeExecutionLayout, error) {
	if ctx == nil || backend == nil || static == nil || !execution.Valid() || execution.RuntimeIdentity() != runtimeidentity.NodeImageReference {
		return artifactnpm.NodeExecutionLayout{}, errors.New("node materialization requires current context and actual adapters")
	}
	if ctx.Err() != nil {
		return artifactnpm.NodeExecutionLayout{}, errors.Join(ctx.Err(), context.Cause(ctx))
	}
	controls, err := promotion.BuildNPMExecutionControls(ctx, execution)
	if err != nil {
		return artifactnpm.NodeExecutionLayout{}, err
	}
	var inventories []artifactnpm.ArchiveInventory
	for _, inspection := range execution.Set().Inspected().Inspections() {
		inventory, report, err := static.InspectArchive(ctx, inspection.Artifact())
		if err != nil {
			return artifactnpm.NodeExecutionLayout{}, errors.Join(err, ctx.Err(), context.Cause(ctx))
		}
		if !inventory.Valid() || len(report.Findings()) != 0 {
			return artifactnpm.NodeExecutionLayout{}, errors.New("node materialization source has no complete current inventory")
		}
		inventories = append(inventories, inventory)
	}
	return backend.MaterializeNodeExecution(ctx, execution, controls, inventories)
}
