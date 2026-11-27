package application

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// BunProjectService uses the existing guarded complete-selection workflow.
// Native client mode/selector and its peer context stay in the concrete adapter.
type BunProjectService struct {
	resolver   ports.ProjectLockUpdateResolver
	promoter   ports.ProjectMutation
	inspection ProjectInspection
	staging    ports.ProjectCacheStaging
}

func NewBunProjectService(resolver ports.ProjectLockUpdateResolver, promoter ports.ProjectMutation, inspection ProjectInspection, staging ports.ProjectCacheStaging) (*BunProjectService, error) {
	if resolver == nil || promoter == nil || inspection == nil || staging == nil {
		return nil, errors.New("bun project requires guarded selection, inspection, staging and promotion")
	}
	return &BunProjectService{resolver, promoter, inspection, staging}, nil
}

func (s *BunProjectService) Apply(ctx context.Context, install domain.InstallContext) (domain.ProjectVerifiedSet, error) {
	if s == nil || ctx == nil || !install.Valid() || install.Mode() != domain.InstallNPMProject {
		return domain.ProjectVerifiedSet{}, errors.New("bun mutation requires an existing single project")
	}
	selected, err := executeGuardedProjectUpdate(ctx, install, s.promoter, s.inspection, s.staging, "npm", "bun", func(controls []domain.ProjectControlFile) (domain.ProjectDependencyUpdate, error) {
		update, err := s.resolver.ResolveProjectLockUpdate(ctx, install, controls, "")
		if err == nil && update.HasRequestedResolution() {
			return domain.ProjectDependencyUpdate{}, errors.New("bun requires complete project selection without a manufactured primary")
		}
		return update, err
	})
	return selected.verified, err
}
