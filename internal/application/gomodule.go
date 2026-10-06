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
	if s == nil {
		return domain.DependencyResolution{}, errors.New("go get service is unavailable")
	}
	return executeProjectUpdate(ctx, reference, installContext, s.resolver, s.promoter, s.inspection, s.staging, "go-proxy", "Go", "get")
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

func (s *GoModuleProjectResolutionService) Resolve(ctx context.Context, installContext domain.InstallContext) (domain.ProjectDependencySnapshot, error) {
	if s == nil {
		return domain.ProjectDependencySnapshot{}, errors.New("go project download service is unavailable")
	}
	return resolveProjectSnapshot(ctx, installContext, s.resolver, s.promoter, s.inspection, s.staging, "go-proxy", "Go")
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
