package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// GoModuleResolutionService is the application boundary for the first M12 Go
// phase. It has no Host-tool knowledge: the composition root supplies the
// isolated, source-pinned dependency resolver.
type GoModuleResolutionService struct{ resolver ports.DependencyResolver }

type ProjectInspection interface {
	InspectProject(context.Context, domain.ProjectDependencySnapshot) (domain.ProjectVerifiedSet, error)
}

// GoModuleGetService holds the original guard before selection, inspects the
// complete frozen project, and publishes that selection without another get.
type GoModuleGetService struct {
	resolver   ports.ProjectDependencyUpdateResolver
	promoter   ports.ProjectMutation
	inspection ProjectInspection
	staging    ports.ProjectCacheStaging
}

func NewGoModuleGetService(resolver ports.ProjectDependencyUpdateResolver, promoter ports.ProjectMutation, inspection ProjectInspection, staging ports.ProjectCacheStaging) (*GoModuleGetService, error) {
	if resolver == nil || promoter == nil || inspection == nil || staging == nil {
		return nil, errors.New("go get service requires guarded selection, inspection and cache staging")
	}
	return &GoModuleGetService{resolver, promoter, inspection, staging}, nil
}

func (s *GoModuleGetService) Get(ctx context.Context, reference domain.ArtifactReference, installContext domain.InstallContext) (resolution domain.DependencyResolution, resultErr error) {
	if s == nil || s.resolver == nil || s.promoter == nil || ctx == nil || reference.Source().String() != "go-proxy" || !installContext.Valid() {
		return domain.DependencyResolution{}, errors.New("valid Go get request is required")
	}
	guard, err := s.promoter.Begin(ctx, installContext)
	if err != nil {
		return resolution, fmt.Errorf("guard Go project: %w", err)
	}
	if guard == nil {
		return resolution, errors.New("go project mutation returned no guard")
	}
	defer func() {
		resultErr = errors.Join(resultErr, guard.Close())
		if resultErr != nil {
			resolution = domain.DependencyResolution{}
		}
	}()
	update, err := s.resolver.ResolveProjectDependencyUpdate(ctx, reference, installContext, guard.Controls())
	if err != nil {
		return domain.DependencyResolution{}, fmt.Errorf("resolve exact Go module graph: %w", err)
	}
	if !update.Valid() || update.Snapshot().Context() != installContext || update.Snapshot().Source() != reference.Source() {
		return resolution, errors.New("go resolver returned an invalid project update")
	}
	if err := guard.VerifyUnchanged(ctx); err != nil {
		return resolution, err
	}
	verified, err := s.inspection.InspectProject(ctx, update.Snapshot())
	if err != nil {
		return resolution, fmt.Errorf("inspect complete selected Go project: %w", err)
	}
	if err := guard.VerifyUnchanged(ctx); err != nil {
		return resolution, err
	}
	staged, err := s.staging.StageProject(ctx, verified)
	if err != nil {
		return resolution, fmt.Errorf("stage approved Go project cache: %w", err)
	}
	if err := guard.Commit(ctx, update, staged); err != nil {
		return domain.DependencyResolution{}, fmt.Errorf("promote exact Go module graph: %w", err)
	}
	return update.Resolution(), nil
}

// GoModuleProjectResolutionService is the application boundary for commands
// that operate on the complete current project rather than one requested
// module.
type GoModuleProjectResolutionService struct {
	resolver   ports.ProjectDependencyResolver
	promoter   ports.ProjectMutation
	inspection ProjectInspection
	staging    ports.ProjectCacheStaging
}

func NewGoModuleProjectResolutionService(resolver ports.ProjectDependencyResolver, promoter ports.ProjectMutation, inspection ProjectInspection, staging ports.ProjectCacheStaging) (*GoModuleProjectResolutionService, error) {
	if resolver == nil || promoter == nil || inspection == nil || staging == nil {
		return nil, errors.New("go project download requires guarded resolution, inspection and cache staging")
	}
	return &GoModuleProjectResolutionService{resolver, promoter, inspection, staging}, nil
}

func (s *GoModuleProjectResolutionService) Resolve(ctx context.Context, installContext domain.InstallContext) (snapshot domain.ProjectDependencySnapshot, resultErr error) {
	if s == nil || s.resolver == nil || ctx == nil || !installContext.Valid() {
		return domain.ProjectDependencySnapshot{}, errors.New("valid Go project resolution request is required")
	}
	guard, err := s.promoter.Begin(ctx, installContext)
	if err != nil {
		return snapshot, fmt.Errorf("guard Go project download: %w", err)
	}
	if guard == nil {
		return snapshot, errors.New("go project download returned no guard")
	}
	defer func() {
		resultErr = errors.Join(resultErr, guard.Close())
		if resultErr != nil {
			snapshot = domain.ProjectDependencySnapshot{}
		}
	}()
	snapshot, err = s.resolver.ResolveProjectDependencies(ctx, installContext)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, fmt.Errorf("resolve complete Go project graph: %w", err)
	}
	if !snapshot.Valid() || snapshot.Context() != installContext || snapshot.Source().String() != "go-proxy" {
		return domain.ProjectDependencySnapshot{}, errors.New("go project resolver returned an invalid snapshot")
	}
	controls := guard.Controls()
	if len(controls) != len(snapshot.ControlDigests()) {
		return domain.ProjectDependencySnapshot{}, errors.New("go project download control coverage differs from guard")
	}
	for i, control := range snapshot.ControlDigests() {
		if control.Name() != controls[i].Name() || control.Digest() != controls[i].Digest() {
			return domain.ProjectDependencySnapshot{}, errors.New("go project download controls differ from guard")
		}
	}
	if err := guard.VerifyUnchanged(ctx); err != nil {
		return snapshot, err
	}
	verified, err := s.inspection.InspectProject(ctx, snapshot)
	if err != nil {
		return snapshot, fmt.Errorf("inspect complete Go download: %w", err)
	}
	if err := guard.VerifyUnchanged(ctx); err != nil {
		return snapshot, err
	}
	staged, err := s.staging.StageProject(ctx, verified)
	if err != nil {
		return snapshot, fmt.Errorf("stage approved Go download: %w", err)
	}
	if err := guard.CommitSnapshot(ctx, snapshot, staged); err != nil {
		return snapshot, fmt.Errorf("retain approved Go download: %w", err)
	}
	return snapshot, nil
}

func NewGoModuleResolutionService(resolver ports.DependencyResolver) (*GoModuleResolutionService, error) {
	if resolver == nil {
		return nil, errors.New("go module resolution service requires a dependency resolver")
	}
	return &GoModuleResolutionService{resolver: resolver}, nil
}

func (s *GoModuleResolutionService) Resolve(ctx context.Context, reference domain.ArtifactReference, installContext domain.InstallContext) (domain.DependencyResolution, error) {
	if s == nil || s.resolver == nil || ctx == nil || reference.Source().String() != "go-proxy" || !installContext.Valid() {
		return domain.DependencyResolution{}, errors.New("valid Go module resolution request is required")
	}
	resolution, err := s.resolver.ResolveDependencies(ctx, reference, installContext)
	if err != nil {
		return domain.DependencyResolution{}, fmt.Errorf("resolve exact Go module graph: %w", err)
	}
	return resolution, nil
}
