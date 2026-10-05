package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// executeProjectUpdate is the existing single-selection workflow shared by the
// actual Go and Cargo consumers; adapters retain all ecosystem/tool semantics.
func executeProjectUpdate(ctx context.Context, reference domain.ArtifactReference, installContext domain.InstallContext, resolver ports.ProjectDependencyUpdateResolver, promoter ports.ProjectMutation, inspection ProjectInspection, staging ports.ProjectCacheStaging, source, label, operation string) (resolution domain.DependencyResolution, resultErr error) {
	if resolver == nil || promoter == nil || inspection == nil || staging == nil || ctx == nil || reference.Source().String() != source || !installContext.Valid() {
		return domain.DependencyResolution{}, fmt.Errorf("valid %s %s request is required", label, operation)
	}
	if err := ctx.Err(); err != nil {
		return resolution, err
	}
	guard, err := promoter.Begin(ctx, installContext)
	if err != nil {
		return resolution, fmt.Errorf("guard %s project: %w", label, err)
	}
	if guard == nil {
		return resolution, fmt.Errorf("%s project mutation returned no guard", label)
	}
	defer func() {
		resultErr = errors.Join(resultErr, guard.Close())
		if resultErr != nil {
			resolution = domain.DependencyResolution{}
		}
	}()
	update, err := resolver.ResolveProjectDependencyUpdate(ctx, reference, installContext, guard.Controls())
	if err != nil {
		return domain.DependencyResolution{}, fmt.Errorf("resolve exact %s graph: %w", label, err)
	}
	if !update.Valid() || update.Snapshot().Context() != installContext || update.Snapshot().Source() != reference.Source() {
		return resolution, fmt.Errorf("%s resolver returned an invalid project update", label)
	}
	if err := guard.VerifyUnchanged(ctx); err != nil {
		return resolution, err
	}
	verified, err := inspection.InspectProject(ctx, update.Snapshot())
	if err != nil {
		return resolution, fmt.Errorf("inspect complete selected %s project: %w", label, err)
	}
	if err := guard.VerifyUnchanged(ctx); err != nil {
		return resolution, err
	}
	staged, err := staging.StageProject(ctx, verified)
	if err != nil {
		return resolution, fmt.Errorf("stage approved %s project cache: %w", label, err)
	}
	if err := guard.Commit(ctx, update, staged); err != nil {
		return domain.DependencyResolution{}, fmt.Errorf("promote exact %s graph: %w", label, err)
	}
	return update.Resolution(), nil
}
