package application

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// UVPipSyncService applies a complete inspected selection through the existing
// guarded workflow. Source grammar and ownership remain adapter responsibilities.
type UVPipSyncService struct {
	resolver   ports.ProjectLockUpdateResolver
	promoter   ports.ProjectMutation
	inspection ProjectInspection
	staging    ports.ProjectCacheStaging
}

func NewUVPipSyncService(resolver ports.ProjectLockUpdateResolver, promoter ports.ProjectMutation, inspection ProjectInspection, staging ports.ProjectCacheStaging) (*UVPipSyncService, error) {
	if resolver == nil || promoter == nil || inspection == nil || staging == nil {
		return nil, errors.New("UV pip sync requires guarded selection, complete inspection and approved staging")
	}
	return &UVPipSyncService{resolver, promoter, inspection, staging}, nil
}

func (s *UVPipSyncService) Sync(ctx context.Context, install domain.InstallContext) (domain.ProjectVerifiedSet, error) {
	if s == nil || !install.Valid() || install.Mode() != domain.InstallPythonVenv {
		return domain.ProjectVerifiedSet{}, errors.New("UV pip sync requires a Python virtual environment")
	}
	selected, err := executeGuardedProjectUpdate(ctx, install, s.promoter, s.inspection, s.staging, "pypi", "UV pip",
		func(controls []domain.ProjectControlFile) (domain.ProjectDependencyUpdate, error) {
			update, err := s.resolver.ResolveProjectLockUpdate(ctx, install, controls, "")
			if err == nil && update.HasRequestedResolution() {
				return domain.ProjectDependencyUpdate{}, errors.New("UV pip sync requires complete selection without a primary")
			}
			return update, err
		})
	return selected.verified, err
}
