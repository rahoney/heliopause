package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

type TerraformResolutionService struct{ resolver ports.DependencyResolver }

func NewTerraformResolutionService(resolver ports.DependencyResolver) (*TerraformResolutionService, error) {
	if resolver == nil {
		return nil, errors.New("terraform resolution service requires a dependency resolver")
	}
	return &TerraformResolutionService{resolver: resolver}, nil
}

func (s *TerraformResolutionService) Resolve(ctx context.Context, reference domain.ArtifactReference, installContext domain.InstallContext) (domain.DependencyResolution, error) {
	if s == nil || s.resolver == nil || ctx == nil || reference.Source().String() != "terraform-registry" || !installContext.Valid() {
		return domain.DependencyResolution{}, errors.New("valid Terraform resolution request is required")
	}
	resolution, err := s.resolver.ResolveDependencies(ctx, reference, installContext)
	if err != nil {
		return domain.DependencyResolution{}, fmt.Errorf("resolve exact Terraform Provider: %w", err)
	}
	return resolution, nil
}
