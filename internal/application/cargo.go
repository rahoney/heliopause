package application

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

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

// CargoUpdateService quarantines the whole selected lock graph. Enrollment
// intent is supplied by the per-request mutation adapter before selection.
type CargoUpdateService struct {
	resolver   ports.ProjectLockUpdateResolver
	promoter   ports.ProjectMutation
	inspection ProjectInspection
	staging    ports.ProjectCacheStaging
}

func NewCargoUpdateService(resolver ports.ProjectLockUpdateResolver, promoter ports.ProjectMutation, inspection ProjectInspection, staging ports.ProjectCacheStaging) (*CargoUpdateService, error) {
	if resolver == nil || promoter == nil || inspection == nil || staging == nil {
		return nil, errors.New("cargo update requires guarded lock selection, inspection and cache staging")
	}
	return &CargoUpdateService{resolver, promoter, inspection, staging}, nil
}
func (s *CargoUpdateService) Update(ctx context.Context, install domain.InstallContext, selector string) (domain.ProjectVerifiedSet, error) {
	if s == nil {
		return domain.ProjectVerifiedSet{}, errors.New("cargo update service is unavailable")
	}
	selected, err := executeGuardedProjectUpdate(ctx, install, s.promoter, s.inspection, s.staging, "crates-io", "Cargo",
		func(controls []domain.ProjectControlFile) (domain.ProjectDependencyUpdate, error) {
			update, err := s.resolver.ResolveProjectLockUpdate(ctx, install, controls, selector)
			if err == nil && update.HasRequestedResolution() {
				return domain.ProjectDependencyUpdate{}, errors.New("cargo update requires complete-project selection without a requested primary")
			}
			return update, err
		})
	return selected.verified, err
}
