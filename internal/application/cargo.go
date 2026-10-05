package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// CargoResolutionService is the read-only application boundary for exact public
// crates.io graph resolution. Project mutation remains a separate transaction.
type CargoResolutionService struct{ resolver ports.DependencyResolver }

func NewCargoResolutionService(resolver ports.DependencyResolver) (*CargoResolutionService, error) {
	if resolver == nil {
		return nil, errors.New("cargo resolution service requires a dependency resolver")
	}
	return &CargoResolutionService{resolver: resolver}, nil
}

func (s *CargoResolutionService) Resolve(ctx context.Context, reference domain.ArtifactReference, installContext domain.InstallContext) (domain.DependencyResolution, error) {
	if s == nil || s.resolver == nil || ctx == nil || reference.Source().String() != "crates-io" || !installContext.Valid() {
		return domain.DependencyResolution{}, errors.New("valid Cargo resolution request is required")
	}
	resolution, err := s.resolver.ResolveDependencies(ctx, reference, installContext)
	if err != nil {
		return domain.DependencyResolution{}, fmt.Errorf("resolve exact Cargo graph: %w", err)
	}
	return resolution, nil
}

// CargoAddService inspects the complete selected graph under the original guard
// and publishes those exact controls without running Cargo again.
type CargoAddService struct {
	resolver   ports.ProjectDependencyUpdateResolver
	promoter   ports.ProjectMutation
	inspection ProjectInspection
	staging    ports.ProjectCacheStaging
}

func NewCargoAddService(resolver ports.ProjectDependencyUpdateResolver, promoter ports.ProjectMutation, inspection ProjectInspection, staging ports.ProjectCacheStaging) (*CargoAddService, error) {
	if resolver == nil || promoter == nil || inspection == nil || staging == nil {
		return nil, errors.New("cargo add requires guarded selection, inspection and cache staging")
	}
	return &CargoAddService{resolver, promoter, inspection, staging}, nil
}
func (s *CargoAddService) Resolve(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext) (domain.DependencyResolution, error) {
	if s == nil {
		return domain.DependencyResolution{}, errors.New("cargo add service is unavailable")
	}
	return executeProjectUpdate(ctx, reference, install, s.resolver, s.promoter, s.inspection, s.staging, "crates-io", "Cargo", "add")
}
