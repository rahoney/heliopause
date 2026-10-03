package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// CargoResolutionService is the application boundary for exact public
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
