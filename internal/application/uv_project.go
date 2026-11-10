package application

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// UVProjectService uses the existing complete guarded workflow. Native project
// grammar, original ownership and atomic file publication stay in adapters.
type UVProjectService struct {
	resolver   ports.ProjectLockUpdateResolver
	promoter   ports.ProjectMutation
	inspection ProjectInspection
	staging    ports.ProjectCacheStaging
}

func NewUVProjectService(resolver ports.ProjectLockUpdateResolver, promoter ports.ProjectMutation, inspection ProjectInspection, staging ports.ProjectCacheStaging) (*UVProjectService, error) {
	if resolver == nil || promoter == nil || inspection == nil || staging == nil {
		return nil, errors.New("UV project requires guarded selection, complete inspection and approved staging")
	}
	return &UVProjectService{resolver, promoter, inspection, staging}, nil
}

func (s *UVProjectService) Apply(ctx context.Context, install domain.InstallContext) (domain.ProjectVerifiedSet, error) {
	if s == nil || !install.Valid() || (install.Mode() != domain.InstallPythonVenv && install.Mode() != domain.InstallPythonProjectControls) {
		return domain.ProjectVerifiedSet{}, errors.New("UV project requires an existing Python venv or explicit project-controls context")
	}
	selected, err := executeGuardedProjectUpdate(ctx, install, s.promoter, s.inspection, s.staging, "pypi", "UV project", func(controls []domain.ProjectControlFile) (domain.ProjectDependencyUpdate, error) {
		update, err := s.resolver.ResolveProjectLockUpdate(ctx, install, controls, "")
		if err == nil && update.HasRequestedResolution() {
			return domain.ProjectDependencyUpdate{}, errors.New("UV project requires a complete selection without a primary")
		}
		return update, err
	})
	return selected.verified, err
}
