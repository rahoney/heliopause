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

// GoModuleGetService composes immutable resolution with the separate project
// mutation port. Promotion is unreachable when resolution fails.
type GoModuleGetService struct {
	resolver ports.DependencyResolver
	promoter ports.ProjectDependencyPromoter
}

func NewGoModuleGetService(resolver ports.DependencyResolver, promoter ports.ProjectDependencyPromoter) (*GoModuleGetService, error) {
	if resolver == nil || promoter == nil {
		return nil, errors.New("go get service requires resolver and project promoter")
	}
	return &GoModuleGetService{resolver: resolver, promoter: promoter}, nil
}

func (s *GoModuleGetService) Get(ctx context.Context, reference domain.ArtifactReference, installContext domain.InstallContext) (domain.DependencyResolution, error) {
	if s == nil || s.resolver == nil || s.promoter == nil || ctx == nil || reference.Source().String() != "go-proxy" || !installContext.Valid() {
		return domain.DependencyResolution{}, errors.New("valid Go get request is required")
	}
	resolution, err := s.resolver.ResolveDependencies(ctx, reference, installContext)
	if err != nil {
		return domain.DependencyResolution{}, fmt.Errorf("resolve exact Go module graph: %w", err)
	}
	if err := s.promoter.PromoteProjectDependency(ctx, reference, installContext); err != nil {
		return domain.DependencyResolution{}, fmt.Errorf("promote exact Go module graph: %w", err)
	}
	return resolution, nil
}

// GoModuleProjectResolutionService is the application boundary for commands
// that operate on the complete current project rather than one requested
// module.
type GoModuleProjectResolutionService struct {
	resolver ports.ProjectDependencyResolver
}

func NewGoModuleProjectResolutionService(resolver ports.ProjectDependencyResolver) (*GoModuleProjectResolutionService, error) {
	if resolver == nil {
		return nil, errors.New("go project resolution service requires a dependency resolver")
	}
	return &GoModuleProjectResolutionService{resolver: resolver}, nil
}

func (s *GoModuleProjectResolutionService) Resolve(ctx context.Context, installContext domain.InstallContext) (domain.ProjectDependencySnapshot, error) {
	if s == nil || s.resolver == nil || ctx == nil || !installContext.Valid() {
		return domain.ProjectDependencySnapshot{}, errors.New("valid Go project resolution request is required")
	}
	snapshot, err := s.resolver.ResolveProjectDependencies(ctx, installContext)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, fmt.Errorf("resolve complete Go project graph: %w", err)
	}
	if !snapshot.Valid() || snapshot.Context() != installContext || snapshot.Source().String() != "go-proxy" {
		return domain.ProjectDependencySnapshot{}, errors.New("go project resolver returned an invalid snapshot")
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
