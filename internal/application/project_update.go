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
	selected, err := executeGuardedProjectUpdate(ctx, installContext, promoter, inspection, staging, source, label,
		func(controls []domain.ProjectControlFile) (domain.ProjectDependencyUpdate, error) {
			update, err := resolver.ResolveProjectDependencyUpdate(ctx, reference, installContext, controls)
			if err == nil && !update.HasRequestedResolution() {
				return domain.ProjectDependencyUpdate{}, fmt.Errorf("%s resolver returned an invalid project update", label)
			}
			return update, err
		})
	return selected.update.Resolution(), err
}

// ProjectUpdateFailure preserves fresh inspection facts without representing
// failed staging, publication or close as a successful update.
type ProjectUpdateFailure struct {
	set   domain.ProjectVerifiedSet
	cause error
}

func (e *ProjectUpdateFailure) Error() string                         { return e.cause.Error() }
func (e *ProjectUpdateFailure) Unwrap() error                         { return e.cause }
func (e *ProjectUpdateFailure) Inspected() domain.InspectedProjectSet { return e.set.Inspected() }
func (e *ProjectUpdateFailure) Decision() domain.PolicyDecision       { return e.set.Decision() }

type projectUpdateResult struct {
	update   domain.ProjectDependencyUpdate
	verified domain.ProjectVerifiedSet
}

func executeGuardedProjectUpdate(ctx context.Context, installContext domain.InstallContext, promoter ports.ProjectMutation, inspection ProjectInspection, staging ports.ProjectCacheStaging, source, label string, selectUpdate func([]domain.ProjectControlFile) (domain.ProjectDependencyUpdate, error)) (result projectUpdateResult, resultErr error) {
	if ctx == nil || !installContext.Valid() || promoter == nil || inspection == nil || staging == nil || selectUpdate == nil {
		return result, errors.New("guarded project update requires all workflow boundaries")
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	guard, err := promoter.Begin(ctx, installContext)
	if err != nil {
		return result, fmt.Errorf("guard %s project: %w", label, err)
	}
	if guard == nil {
		return result, fmt.Errorf("%s project mutation returned no guard", label)
	}
	defer func() {
		resultErr = errors.Join(resultErr, guard.Close())
		if resultErr != nil {
			if result.verified.Valid() {
				resultErr = &ProjectUpdateFailure{result.verified, resultErr}
			}
			result = projectUpdateResult{}
		}
	}()
	ports.ReportProgress(ctx, ports.Progress{Stage: "resolve"})
	update, err := selectUpdate(guard.Controls())
	if err != nil {
		return result, fmt.Errorf("resolve exact %s graph: %w", label, err)
	}
	if !update.Valid() || update.Snapshot().Context() != installContext || update.Snapshot().Source().String() != source {
		return result, fmt.Errorf("%s resolver returned an invalid project update", label)
	}
	if err := guard.VerifyUnchanged(ctx); err != nil {
		return result, err
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "inspect"})
	verified, err := inspection.InspectProject(ctx, update.Snapshot())
	if err != nil {
		return result, fmt.Errorf("inspect complete selected %s project: %w", label, err)
	}
	if !verified.Valid() || verified.Inspected().Snapshot().Context() != update.Snapshot().Context() || verified.Inspected().Snapshot().Source() != update.Snapshot().Source() || verified.Inspected().Snapshot().GraphDigest() != update.Snapshot().GraphDigest() {
		return result, fmt.Errorf("%s inspection returned approval for a different selected project", label)
	}
	result = projectUpdateResult{update: update, verified: verified}
	if err := guard.VerifyUnchanged(ctx); err != nil {
		return result, err
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "stage"})
	staged, err := staging.StageProject(ctx, verified)
	if err != nil {
		return result, fmt.Errorf("stage approved %s project cache: %w", label, err)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "install"})
	if err := guard.Commit(ctx, update, staged); err != nil {
		return result, fmt.Errorf("promote exact %s graph: %w", label, err)
	}
	return result, nil
}
