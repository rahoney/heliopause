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
	promoter *promotion.TerraformProjectPromotion
}

func (a terraformProjectMutationAdapter) Begin(ctx context.Context, install domain.InstallContext) (ports.ProjectMutationGuard, error) {
	return a.promoter.Begin(ctx, install)
}

func newTerraformService(executor sandbox.TrustedExecutor, observer sandbox.TraceObserver) (*application.TerraformInitService, error) {
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
	backend, err := sandbox.NewLinuxTerraformProviderBackend(intake, executor, observer)
	if err != nil {
		return nil, err
	}
	inspector, err := inspectionterraform.NewInspector(intake, backend, verifier)
	if err != nil {
		return nil, err
	}
	evidence, err := local.NewStore(evidenceRoot)
	if err != nil {
		return nil, err
	}
	inspection, err := application.NewProjectInspectService(client, verifier, inspector, evidence, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
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
	return application.NewTerraformInitService(client, terraformProjectMutationAdapter{promoter}, inspection, cache)
}
