package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/sandbox"
	"github.com/spf13/cobra"
)

func pnpmProjectPlatform(goos, goarch string) error {
	if goos != "linux" || goarch != "amd64" {
		return errors.New("PNPM project quarantine requires Linux amd64")
	}
	return nil
}

type pnpmProjectCLIService struct{ bindings pnpmProjectBindings }

func (s pnpmProjectCLIService) ApplyPNPM(ctx context.Context, request cli.PNPMProjectRequest) (outcome cli.PNPMProjectOutcome, resultErr error) {
	service, err := newPNPMProjectService(s.bindings, request.Intent, request.Selector, request.Adopt)
	if err != nil {
		return outcome, err
	}
	outcome.Verified, err = service.Apply(ctx, request.Install)
	if err != nil {
		return outcome, err
	}
	id, manifest, sbom, ok := service.promoter.PublishedRecords()
	if !ok {
		return outcome, errors.New("pnpm completed without verified publication records")
	}
	outcome.ManifestID, outcome.ManifestHandle, outcome.SBOMHandle = id, manifest, sbom
	return outcome, nil
}

func bindPNPMProjectClient(command *cobra.Command, executor sandbox.TrustedExecutor, observer sandbox.TraceObserver) error {
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	root := filepath.Join(cache, "heliopause")
	intake, evidenceRoot, cacheRoot, stateRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "pnpm-project-cache"), filepath.Join(root, "pnpm-projects")
	source, err := artifactnpm.NewPublicResolver(intake)
	if err != nil {
		return err
	}
	store, err := local.NewStore(evidenceRoot)
	if err != nil {
		return err
	}
	static, err := inspectionnpm.NewInspector(intake)
	if err != nil {
		return err
	}
	policy, err := newSystemResolverPolicyAdapter()
	if err != nil {
		return err
	}
	native, err := sandbox.NewLinuxPNPMResolverWithExecutorAndPolicy(executor, observer, policy)
	if err != nil {
		return err
	}
	prepare, err := sandbox.NewPNPMInspectionPreparer(native, intake)
	if err != nil {
		return err
	}
	backend, err := sandbox.NewLinuxBackendWithExecutor(intake, executor, observer)
	if err != nil {
		return err
	}
	return cli.AddPNPMProject(command, pnpmProjectCLIService{pnpmProjectBindings{intake, evidenceRoot, cacheRoot, stateRoot, source, store, static, native, prepare, backend}})
}
