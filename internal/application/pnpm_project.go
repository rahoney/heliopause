package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// NPMCompleteProjectService uses the existing guarded complete-selection workflow.
// Native client mode/selector and its peer context stay in the concrete adapter.
type NPMCompleteProjectService struct {
	client     string
	resolver   ports.ProjectLockUpdateResolver
	promoter   ports.ProjectMutation
	inspection ProjectInspection
	staging    ports.ProjectCacheStaging
}

func NewNPMCompleteProjectService(client string, resolver ports.ProjectLockUpdateResolver, promoter ports.ProjectMutation, inspection ProjectInspection, staging ports.ProjectCacheStaging) (*NPMCompleteProjectService, error) {
	if (client != "npm" && client != "pnpm") || resolver == nil || promoter == nil || inspection == nil || staging == nil {
		return nil, errors.New("npm complete project requires a supported client and all guarded workflow boundaries")
	}
	return &NPMCompleteProjectService{client, resolver, promoter, inspection, staging}, nil
}

func (s *NPMCompleteProjectService) Apply(ctx context.Context, install domain.InstallContext) (domain.ProjectVerifiedSet, error) {
	if s == nil || ctx == nil || !install.Valid() || install.Mode() != domain.InstallNPMProject {
		return domain.ProjectVerifiedSet{}, errors.New("npm complete mutation requires an existing single project")
	}
	selected, err := executeGuardedProjectUpdate(ctx, install, s.promoter, s.inspection, s.staging, "npm", s.client, func(controls []domain.ProjectControlFile) (domain.ProjectDependencyUpdate, error) {
		update, err := s.resolver.ResolveProjectLockUpdate(ctx, install, controls, "")
		if err == nil && update.HasRequestedResolution() {
			return domain.ProjectDependencyUpdate{}, fmt.Errorf("%s requires complete project selection without a manufactured primary", s.client)
		}
		return update, err
	})
	return selected.verified, err
}
