package bootstrap

import (
	"context"
	"os"
	"path/filepath"

	"github.com/rahoney/heliopause/internal/application"
	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectioncargo "github.com/rahoney/heliopause/internal/inspection/cargo"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationcargo "github.com/rahoney/heliopause/internal/verification/cargo"
)

type cargoProjectMutationAdapter struct {
	promoter *promotion.CargoProjectPromotion
}

func (a cargoProjectMutationAdapter) Begin(ctx context.Context, install domain.InstallContext) (ports.ProjectMutationGuard, error) {
	return a.promoter.Begin(ctx, install)
}

func newCargoAddService(resolver *sandbox.CargoResolver) (*application.CargoAddService, error) {
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	root := filepath.Join(cacheRoot, "heliopause")
	intakeRoot, evidenceRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence")
	artifact, err := artifactcargo.NewPublicClient(intakeRoot)
	if err != nil {
		return nil, err
	}
	verifier, err := verificationcargo.NewIntegrityVerifier(intakeRoot)
	if err != nil {
		return nil, err
	}
	inspector, err := inspectioncargo.NewStaticInspector(intakeRoot)
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
	cache, err := promotion.NewCargoVerifiedCache(intakeRoot, evidenceRoot, filepath.Join(root, "cargo-verified-cache"), evidence)
	if err != nil {
		return nil, err
	}
	promoter, err := promotion.NewCargoProjectPromotion(cache, filepath.Join(root, "cargo-projects"))
	if err != nil {
		return nil, err
	}
	return application.NewCargoAddService(resolver, cargoProjectMutationAdapter{promoter}, inspection, cache)
}
