package bootstrap

import (
	"context"
	"os"
	"path/filepath"

	"github.com/rahoney/heliopause/internal/application"
	artifactterraform "github.com/rahoney/heliopause/internal/artifact/terraformprovider"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionterraform "github.com/rahoney/heliopause/internal/inspection/terraformprovider"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationterraform "github.com/rahoney/heliopause/internal/verification/terraformprovider"
)

type terraformProjectMutationAdapter struct {
	promoter       *promotion.TerraformProjectPromotion
	upgrade, adopt bool
}

func (a terraformProjectMutationAdapter) Begin(ctx context.Context, install domain.InstallContext) (ports.ProjectMutationGuard, error) {
	if a.upgrade {
		return a.promoter.BeginUpgrade(ctx, install, a.adopt)
	}
	return a.promoter.Begin(ctx, install)
}

func newTerraformServices(executor sandbox.TrustedExecutor, observer sandbox.TraceObserver) (*application.TerraformInitService, *terraformUpgradeAdapter, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return nil, nil, err
	}
	root := filepath.Join(base, "heliopause")
	intake, evidenceRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence")
	client, err := artifactterraform.NewPublicClient(intake)
	if err != nil {
		return nil, nil, err
	}
	verifier, err := verificationterraform.NewIntegrityVerifier(intake)
	if err != nil {
		return nil, nil, err
	}
	backend, err := sandbox.NewLinuxTerraformProviderBackend(intake, executor, observer)
	if err != nil {
		return nil, nil, err
	}
	inspector, err := inspectionterraform.NewInspector(intake, backend, verifier)
	if err != nil {
		return nil, nil, err
	}
	evidence, err := local.NewStore(evidenceRoot)
	if err != nil {
		return nil, nil, err
	}
	inspection, err := application.NewProjectInspectService(client, verifier, inspector, evidence, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		return nil, nil, err
	}
	cache, err := promotion.NewTerraformVerifiedCache(intake, evidenceRoot, filepath.Join(root, "terraform-verified-cache"), evidence)
	if err != nil {
		return nil, nil, err
	}
	promoter, err := promotion.NewTerraformProjectPromotion(cache, filepath.Join(root, "terraform-projects"))
	if err != nil {
		return nil, nil, err
	}
	initService, err := application.NewTerraformInitService(client, terraformProjectMutationAdapter{promoter: promoter}, inspection, cache)
	if err != nil {
		return nil, nil, err
	}
	return initService, &terraformUpgradeAdapter{client, promoter, inspection, cache}, nil
}

// Enrollment intent belongs to this request, never shared mutable promotion state.
type terraformUpgradeAdapter struct {
	client     *artifactterraform.Client
	promoter   *promotion.TerraformProjectPromotion
	inspection application.ProjectInspection
	cache      *promotion.TerraformVerifiedCache
}

func (a *terraformUpgradeAdapter) Upgrade(ctx context.Context, install domain.InstallContext, adopt bool) (domain.ProjectVerifiedSet, error) {
	service, err := application.NewTerraformUpgradeService(a.client, terraformProjectMutationAdapter{promoter: a.promoter, upgrade: true, adopt: adopt}, a.inspection, a.cache)
	if err != nil {
		return domain.ProjectVerifiedSet{}, err
	}
	return service.Upgrade(ctx, install)
}
