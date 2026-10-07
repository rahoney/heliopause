package application

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// TerraformInitResult is returned only after the guarded installation and guard
// close succeed. Count comes from the complete inspected project snapshot.
type TerraformInitResult struct {
	Resolution    domain.DependencyResolution
	ProviderCount int
}

type TerraformInitService struct {
	resolver   ports.ProjectDependencyUpdateResolver
	promoter   ports.ProjectMutation
	inspection ProjectInspection
	staging    ports.ProjectCacheStaging
}

func NewTerraformInitService(resolver ports.ProjectDependencyUpdateResolver, promoter ports.ProjectMutation, inspection ProjectInspection, staging ports.ProjectCacheStaging) (*TerraformInitService, error) {
	if resolver == nil || promoter == nil || inspection == nil || staging == nil {
		return nil, errors.New("terraform init requires guarded selection, inspection and verified cache staging")
	}
	return &TerraformInitService{resolver, promoter, inspection, staging}, nil
}

type terraformInitSelection struct {
	resolver ports.ProjectDependencyUpdateResolver
	count    int
}

func (s *terraformInitSelection) ResolveProjectDependencyUpdate(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext, controls []domain.ProjectControlFile) (domain.ProjectDependencyUpdate, error) {
	update, err := s.resolver.ResolveProjectDependencyUpdate(ctx, reference, install, controls)
	if err == nil && update.Valid() {
		s.count = len(update.Snapshot().Dependencies())
	}
	return update, err
}

func (s *TerraformInitService) Init(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext) (TerraformInitResult, error) {
	if s == nil {
		return TerraformInitResult{}, errors.New("terraform init service is unavailable")
	}
	selected := &terraformInitSelection{resolver: s.resolver}
	resolution, err := executeProjectUpdate(ctx, reference, install, selected, s.promoter, s.inspection, s.staging, "terraform-registry", "Terraform", "init")
	if err != nil {
		return TerraformInitResult{}, err
	}
	return TerraformInitResult{Resolution: resolution, ProviderCount: selected.count}, nil
}
