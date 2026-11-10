package application

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

type PythonRequirementsService struct {
	resolver   ports.ProjectLockUpdateResolver
	promoter   ports.ProjectMutation
	inspection ProjectInspection
	staging    ports.ProjectCacheStaging
}

func NewPythonRequirementsService(resolver ports.ProjectLockUpdateResolver, promoter ports.ProjectMutation, inspection ProjectInspection, staging ports.ProjectCacheStaging) (*PythonRequirementsService, error) {
	if resolver == nil || promoter == nil || inspection == nil || staging == nil {
		return nil, errors.New("requirements installation needs guarded selection, whole inspection and approved staging")
	}
	return &PythonRequirementsService{resolver, promoter, inspection, staging}, nil
}

// Install's bool is an explicit no-op. No inspection, staging or publication
// is reported when the frozen request has no active installation roots.
func (s *PythonRequirementsService) Install(ctx context.Context, install domain.InstallContext) (domain.ProjectVerifiedSet, bool, error) {
	if s == nil || !install.Valid() || install.Mode() != domain.InstallPythonVenv {
		return domain.ProjectVerifiedSet{}, false, errors.New("requirements installation needs a Python virtual environment")
	}
	selected, err := executeGuardedProjectUpdate(ctx, install, s.promoter, s.inspection, s.staging, "pypi", "Python requirements", func(controls []domain.ProjectControlFile) (domain.ProjectDependencyUpdate, error) {
		update, err := s.resolver.ResolveProjectLockUpdate(ctx, install, controls, "")
		if err == nil && update.HasRequestedResolution() {
			return domain.ProjectDependencyUpdate{}, errors.New("requirements need a complete selection without a fabricated primary")
		}
		return update, err
	}, true)
	return selected.verified, selected.noop, err
}
