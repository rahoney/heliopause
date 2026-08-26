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
