package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// CargoBuildService prepares the complete existing locked graph independently
// of a requested primary. Its build then reacquires the source/approval guard;
// project changes between phases cannot substitute a graph or retained cache.
type CargoBuildService struct {
	resolver   ports.ProjectDependencyResolver
	promoter   ports.ProjectMutation
	inspection ProjectInspection
	staging    ports.ProjectCacheStaging
	build      *ProjectBuildService
}

func NewCargoBuildService(resolver ports.ProjectDependencyResolver, promoter ports.ProjectMutation, inspection ProjectInspection, staging ports.ProjectCacheStaging, build *ProjectBuildService) (*CargoBuildService, error) {
	if resolver == nil || promoter == nil || inspection == nil || staging == nil || build == nil {
		return nil, errors.New("cargo build requires complete locked inputs and observed build")
	}
	return &CargoBuildService{resolver, promoter, inspection, staging, build}, nil
}

func (s *CargoBuildService) Build(ctx context.Context, install domain.InstallContext, selector string) (domain.OperationResult, domain.PublishedProjectBuild, error) {
	if s == nil || ctx == nil || !install.Valid() || selector != "default" {
		return domain.OperationResult{}, domain.PublishedProjectBuild{}, errors.New("cargo build request is invalid")
	}
	if _, err := resolveProjectSnapshot(ctx, install, s.resolver, s.promoter, s.inspection, s.staging, "crates-io", "Cargo"); err != nil {
		return domain.OperationResult{}, domain.PublishedProjectBuild{}, fmt.Errorf("prepare locked Cargo build inputs: %w", err)
	}
	return s.build.Build(ctx, install, selector)
}
