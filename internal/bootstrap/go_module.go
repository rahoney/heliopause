package bootstrap

import (
	"context"
	"os"
	"path/filepath"

	"github.com/rahoney/heliopause/internal/application"
	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectiongo "github.com/rahoney/heliopause/internal/inspection/gomodule"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationgo "github.com/rahoney/heliopause/internal/verification/gomodule"
)

type goProjectMutationAdapter struct{ promoter *promotion.GoProjectPromotion }

func (a goProjectMutationAdapter) Begin(ctx context.Context, install domain.InstallContext) (ports.ProjectMutationGuard, error) {
	return a.promoter.Begin(ctx, install)
}

func newGoModuleGetService(resolver *sandbox.GoModuleResolver) (*application.GoModuleGetService, error) {
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	root := filepath.Join(cacheRoot, "heliopause")
	intakeRoot, evidenceRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence")
	artifact, err := artifactgo.NewPublicClient(intakeRoot)
	if err != nil {
		return nil, err
	}
	verifier, err := verificationgo.NewIntegrityVerifier(intakeRoot)
	if err != nil {
		return nil, err
	}
	inspector, err := inspectiongo.NewStaticInspector(intakeRoot)
	if err != nil {
		return nil, err
	}
	evidence, err := local.NewStore(evidenceRoot)
	if err != nil {
		return nil, err
	}
	inspection, err := application.NewProjectInspectService(artifact, verifier, inspector, evidence, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		return nil, err
	}
	cache, err := promotion.NewGoVerifiedCache(intakeRoot, evidenceRoot, filepath.Join(root, "go-verified-cache"), evidence)
	if err != nil {
		return nil, err
	}
	promoter, err := promotion.NewGoProjectPromotion(cache, filepath.Join(root, "go-projects"))
	if err != nil {
		return nil, err
	}
	return application.NewGoModuleGetService(resolver, goProjectMutationAdapter{promoter}, inspection, cache)
}
