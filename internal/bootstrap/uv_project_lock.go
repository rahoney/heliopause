package bootstrap

import (
	"context"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	verificationpypi "github.com/rahoney/heliopause/internal/verification/pypi"
)

type uvProjectLockService struct {
	workflow *application.UVProjectService
	promoter *promotion.UVProjectLockPromotion
}

func (s *uvProjectLockService) Apply(ctx context.Context, install domain.InstallContext) (domain.ProjectVerifiedSet, error) {
	return s.workflow.Apply(ctx, install)
}

func newUVProjectLockService(b uvProjectBindings, root string) (*uvProjectLockService, error) {
	operation, err := domain.NewOperationID()
	if err != nil {
		return nil, err
	}
	cache, err := promotion.NewPythonProjectCache(b.intakeRoot, b.evidenceRoot, b.cacheRoot, b.evidence, evidence.Generator{}, operation)
	if err != nil {
		return nil, err
	}
	promoter, err := promotion.NewUVProjectLockPromotion(root, b.resolver, cache)
	if err != nil {
		return nil, err
	}
	inspection, err := application.NewProjectInspectService(b.intake, verificationpypi.IntegrityVerifier{}, b.inspector, b.evidence, policy.M3{}, policy.M4{}, func() (domain.OperationID, error) { return operation, nil }, domain.NewRunID)
	if err != nil {
		return nil, err
	}
	workflow, err := application.NewUVProjectService(promoter, promoter, inspection, promoter)
	if err != nil {
		return nil, err
	}
	return &uvProjectLockService{workflow, promoter}, nil
}
