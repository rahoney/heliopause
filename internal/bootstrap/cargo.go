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
	inspectionbuild "github.com/rahoney/heliopause/internal/inspection/projectbuild"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationcargo "github.com/rahoney/heliopause/internal/verification/cargo"
	verificationbuild "github.com/rahoney/heliopause/internal/verification/projectbuild"
)

type cargoProjectMutationAdapter struct {
	promoter *promotion.CargoProjectPromotion
}

func (a cargoProjectMutationAdapter) Begin(ctx context.Context, install domain.InstallContext) (ports.ProjectMutationGuard, error) {
	return a.promoter.Begin(ctx, install)
}

func (a cargoProjectMutationAdapter) BeginBuild(ctx context.Context, install domain.InstallContext) (ports.ProjectBuildGuard, error) {
	return a.promoter.Begin(ctx, install)
}

func newCargoServices(resolver *sandbox.CargoResolver, executor sandbox.TrustedExecutor, observer sandbox.TraceObserver) (add *application.CargoAddService, build *application.CargoBuildService, resultErr error) {
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return nil, nil, err
	}
	root := filepath.Join(cacheRoot, "heliopause")
	intakeRoot, evidenceRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence")
	artifact, err := artifactcargo.NewPublicClient(intakeRoot)
	if err != nil {
		return nil, nil, err
	}
	verifier, err := verificationcargo.NewIntegrityVerifier(intakeRoot)
	if err != nil {
		return nil, nil, err
	}
	inspector, err := inspectioncargo.NewStaticInspector(intakeRoot)
	if err != nil {
		return nil, nil, err
	}
	evidence, err := local.NewStore(evidenceRoot)
	if err != nil {
		return nil, nil, err
	}
	inspection, err := application.NewProjectInspectService(artifact, verifier, inspector, evidence, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		return nil, nil, err
	}
	cache, err := promotion.NewCargoVerifiedCache(intakeRoot, evidenceRoot, filepath.Join(root, "cargo-verified-cache"), evidence)
	if err != nil {
		return nil, nil, err
	}
	promoter, err := promotion.NewCargoProjectPromotion(cache, filepath.Join(root, "cargo-projects"))
	if err != nil {
		return nil, nil, err
	}
	add, err = application.NewCargoAddService(resolver, cargoProjectMutationAdapter{promoter}, inspection, cache)
	if err != nil {
		return nil, nil, err
	}
	reader, err := sandbox.NewCargoBuildSourceReader(intakeRoot)
	if err != nil {
		return nil, nil, err
	}
	buildVerifier, err := verificationbuild.NewSourceVerifier(reader, "cargo")
	if err != nil {
		return nil, nil, err
	}
	backend, err := sandbox.NewLinuxObservedCargoBuilder(intakeRoot, executor, observer)
	if err != nil {
		return nil, nil, err
	}
	buildInspector, err := inspectionbuild.NewInspector(backend, "cargo")
	if err != nil {
		return nil, nil, err
	}
	observed, err := application.NewProjectBuildService(cargoProjectMutationAdapter{promoter}, buildVerifier, buildInspector, evidence, policy.M3{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		return nil, nil, err
	}
	build, err = application.NewCargoBuildService(resolver, cargoProjectMutationAdapter{promoter}, inspection, cache, observed)
	return add, build, err
}
