package bootstrap

import (
	"context"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	verificationpypi "github.com/rahoney/heliopause/internal/verification/pypi"
)

type uvProjectBindings struct {
	intakeRoot, evidenceRoot, cacheRoot string
	intake                              ports.Artifact
	inspector                           ports.Inspection
	evidence                            pythonClientEvidence
	promoter                            *promotion.PyPIPromotion
	resolver                            ports.ProjectLockUpdateResolver
	nativeLookup                        ports.DependencyResolver
}

type uvProjectService struct {
	workflow *application.UVProjectService
	promoter *promotion.UVProjectPromotion
}

func (s *uvProjectService) Apply(ctx context.Context, install domain.InstallContext) (domain.ProjectVerifiedSet, error) {
	return s.workflow.Apply(ctx, install)
}

func newUVProjectService(b uvProjectBindings, root string, intent artifactpypi.UVProjectIntent, selector string, adopt bool) (*uvProjectService, error) {
	operation, err := domain.NewOperationID()
	if err != nil {
		return nil, err
	}
	cache, err := promotion.NewPythonProjectCache(b.intakeRoot, b.evidenceRoot, b.cacheRoot, b.evidence, evidence.Generator{}, operation)
	if err != nil {
		return nil, err
	}
	preparation, err := promotion.NewUVProjectPreparation(b.promoter, b.resolver, root, intent, selector, adopt, []string{b.intakeRoot, b.evidenceRoot, b.cacheRoot})
	if err != nil {
		return nil, err
	}
	if b.nativeLookup != nil {
		preparation, err = preparation.WithNativeEnrollment(b.nativeLookup, b.intake, b.intakeRoot)
		if err != nil {
			return nil, err
		}
	}
	promoter, err := promotion.NewUVProjectPromotion(preparation, cache)
	if err != nil {
		return nil, err
	}
	inspection, err := application.NewProjectInspectService(b.intake, verificationpypi.IntegrityVerifier{}, b.inspector, b.evidence, policy.M3{}, policy.M4{}, func() (domain.OperationID, error) { return operation, nil }, domain.NewRunID)
	if err != nil {
		return nil, err
	}
	workflow, err := application.NewUVProjectService(preparation, promoter, inspection, promoter)
	if err != nil {
		return nil, err
	}
	return &uvProjectService{workflow, promoter}, nil
}
