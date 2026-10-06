package application

import (
	"context"
	"errors"
	"fmt"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// resolveProjectSnapshot shares the complete-source preparation lifecycle.
// A primary artifact, SDK rerun or project marker cannot replace its approval.
func resolveProjectSnapshot(ctx context.Context, installContext domain.InstallContext, resolver ports.ProjectDependencyResolver, promoter ports.ProjectMutation, inspection ProjectInspection, staging ports.ProjectCacheStaging, expectedSource, label string) (snapshot domain.ProjectDependencySnapshot, resultErr error) {
	if resolver == nil || promoter == nil || inspection == nil || staging == nil || ctx == nil || !installContext.Valid() {
		return domain.ProjectDependencySnapshot{}, errors.New("valid project resolution request is required")
	}
	if err := ctx.Err(); err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	guard, err := promoter.Begin(ctx, installContext)
	if err != nil {
		return snapshot, fmt.Errorf("guard "+label+" project inputs: %w", err)
	}
	if guard == nil {
		return snapshot, errors.New("project download returned no guard")
	}
	defer func() {
		resultErr = errors.Join(resultErr, guard.Close())
		if resultErr != nil {
			snapshot = domain.ProjectDependencySnapshot{}
		}
	}()
	snapshot, err = resolver.ResolveProjectDependencies(ctx, installContext)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, fmt.Errorf("resolve complete "+label+" project graph: %w", err)
	}
	if !snapshot.Valid() || snapshot.Context() != installContext || snapshot.Source().String() != expectedSource {
		return domain.ProjectDependencySnapshot{}, errors.New("project resolver returned an invalid snapshot")
	}
	controls := guard.Controls()
	if len(controls) != len(snapshot.ControlDigests()) {
		return domain.ProjectDependencySnapshot{}, errors.New("project download control coverage differs from guard")
	}
	byName := make(map[string]domain.ProjectControlFile, len(controls))
	for _, control := range controls {
		if _, exists := byName[control.Name()]; exists {
			return domain.ProjectDependencySnapshot{}, errors.New("project guard returned repeated controls")
		}
		byName[control.Name()] = control
	}
	for _, control := range snapshot.ControlDigests() {
		file, exists := byName[control.Name()]
		if !exists || control.Digest() != file.Digest() {
			return domain.ProjectDependencySnapshot{}, errors.New("project download controls differ from guard")
		}
	}
	if err := guard.VerifyUnchanged(ctx); err != nil {
		return snapshot, err
	}
	verified, err := inspection.InspectProject(ctx, snapshot)
	if err != nil {
		return snapshot, fmt.Errorf("inspect complete "+label+" project inputs: %w", err)
	}
	if err := guard.VerifyUnchanged(ctx); err != nil {
		return snapshot, err
	}
	staged, err := staging.StageProject(ctx, verified)
	if err != nil {
		return snapshot, fmt.Errorf("stage approved "+label+" project inputs: %w", err)
	}
	if err := guard.CommitSnapshot(ctx, snapshot, staged); err != nil {
		return snapshot, fmt.Errorf("retain approved "+label+" project inputs: %w", err)
	}
	return snapshot, nil
}
