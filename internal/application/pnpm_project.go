package application

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// PNPMProjectService uses the existing guarded complete-selection workflow.
// Native client mode/selector and its peer context stay in the concrete adapter.
type PNPMProjectService struct {
	resolver   ports.ProjectLockUpdateResolver
	promoter   ports.ProjectMutation
	inspection ProjectInspection
	staging    ports.ProjectCacheStaging
}

func NewPNPMProjectService(resolver ports.ProjectLockUpdateResolver, promoter ports.ProjectMutation, inspection ProjectInspection, staging ports.ProjectCacheStaging) (*PNPMProjectService, error) {
	if resolver == nil || promoter == nil || inspection == nil || staging == nil {
		return nil, errors.New("pnpm project requires guarded selection, inspection, staging and promotion")
	}
	return &PNPMProjectService{resolver, promoter, inspection, staging}, nil
}

func (s *PNPMProjectService) Apply(ctx context.Context, install domain.InstallContext) (domain.ProjectVerifiedSet, error) {
	if s == nil || ctx == nil || !install.Valid() || install.Mode() != domain.InstallNPMProject {
		return domain.ProjectVerifiedSet{}, errors.New("pnpm mutation requires an existing single project")
	}
	selected, err := executeGuardedProjectUpdate(ctx, install, s.promoter, s.inspection, s.staging, "npm", "pnpm", func(controls []domain.ProjectControlFile) (domain.ProjectDependencyUpdate, error) {
		update, err := s.resolver.ResolveProjectLockUpdate(ctx, install, controls, "")
		if err == nil && update.HasRequestedResolution() {
			return domain.ProjectDependencyUpdate{}, errors.New("pnpm requires complete project selection without a manufactured primary")
		}
		return update, err
	})
	return selected.verified, err
}
