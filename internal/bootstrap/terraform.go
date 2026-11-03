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
	promoter                   *promotion.TerraformProjectPromotion
	upgrade, adopt, initialize bool
}

func (a terraformProjectMutationAdapter) Begin(ctx context.Context, install domain.InstallContext) (ports.ProjectMutationGuard, error) {
	if a.initialize {
		return a.promoter.BeginInit(ctx, install)
	}
	if a.upgrade {
		return a.promoter.BeginUpgrade(ctx, install, a.adopt)
	}
	return a.promoter.Begin(ctx, install)
}

func newTerraformServices(executor sandbox.TrustedExecutor, observer sandbox.TraceObserver) (*terraformProjectAdapter, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	root := filepath.Join(base, "heliopause")
	intake, evidenceRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence")
	client, err := artifactterraform.NewPublicClient(intake)
	if err != nil {
		return nil, err
	}
	verifier, err := verificationterraform.NewIntegrityVerifier(intake)
	if err != nil {
		return nil, err
	}
	backend, err := sandbox.NewLinuxTerraformProviderBackend(intake, executor, observer, newObservationResourceAdapter())
	if err != nil {
		return nil, err
	}
	inspector, err := inspectionterraform.NewInspector(intake, backend, verifier)
	if err != nil {
		return nil, err
	}
	checkedInspection, err := newAdvisoryInspection(inspector)
	if err != nil {
		return nil, err
	}
	evidence, err := local.NewStore(evidenceRoot)
	if err != nil {
		return nil, err
	}
	inspection, err := application.NewProjectInspectService(client, verifier, checkedInspection, evidence, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		return nil, err
	}
	cache, err := promotion.NewTerraformVerifiedCache(intake, evidenceRoot, filepath.Join(root, "terraform-verified-cache"), evidence)
	if err != nil {
		return nil, err
	}
	promoter, err := promotion.NewTerraformProjectPromotion(cache, filepath.Join(root, "terraform-projects"))
	if err != nil {
		return nil, err
	}
	initService, err := application.NewTerraformInitService(client, terraformProjectMutationAdapter{promoter: promoter}, inspection, cache)
	if err != nil {
		return nil, err
	}
	operation, err := domain.NewOperationID()
	if err != nil {
		return nil, err
	}
	return &terraformProjectAdapter{client: client, promoter: promoter, inspection: inspection, cache: cache, init: initService, operation: operation}, nil
}

// Enrollment intent belongs to this request, never shared mutable promotion state.
type terraformProjectAdapter struct {
	client     *artifactterraform.Client
	promoter   *promotion.TerraformProjectPromotion
	inspection application.ProjectInspection
	cache      *promotion.TerraformVerifiedCache
	init       *application.TerraformInitService
	operation  domain.OperationID
}

func (a *terraformProjectAdapter) Init(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext) (application.TerraformInitResult, error) {
	return a.init.Init(ports.WithStorageOperation(ctx, a.operation), reference, install)
}

func (a *terraformProjectAdapter) Upgrade(ctx context.Context, install domain.InstallContext, adopt bool) (domain.ProjectVerifiedSet, error) {
	service, err := application.NewTerraformUpgradeService(a.client, terraformProjectMutationAdapter{promoter: a.promoter, upgrade: true, adopt: adopt}, a.inspection, a.cache)
	if err != nil {
		return domain.ProjectVerifiedSet{}, err
	}
	return service.Upgrade(ports.WithStorageOperation(ctx, a.operation), install)
}

func (a *terraformProjectAdapter) InitProject(ctx context.Context, install domain.InstallContext) (domain.ProjectVerifiedSet, error) {
	return application.InitializeTerraformProject(ports.WithStorageOperation(ctx, a.operation), install, a.client, terraformProjectMutationAdapter{promoter: a.promoter, initialize: true}, a.inspection, a.cache)
}
