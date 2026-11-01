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

type npmProjectCLIService struct{ bindings npmProjectBindings }

func (s npmProjectCLIService) ApplyNPM(ctx context.Context, request cli.NPMProjectRequest) (outcome cli.NPMProjectOutcome, resultErr error) {
	service, err := newNPMProjectService(s.bindings, request.Intent)
	if err != nil {
		return outcome, err
	}
	outcome.Verified, err = service.Apply(ctx, request.Install)
	outcome.RootChecked, outcome.RootRequired, outcome.RootInspection = service.rootChecked, service.rootRequired, service.rootResult
	if err != nil {
		return outcome, err
	}
	id, manifest, sbom, ok := service.promoter.PublishedRecords()
	if !ok {
		return outcome, errors.New("npm completed without verified publication records")
	}
	outcome.ManifestID, outcome.ManifestHandle, outcome.SBOMHandle = id, manifest, sbom
	return outcome, nil
}

func bindNPMProjectClient(command *cobra.Command, executor sandbox.TrustedExecutor, observer sandbox.TraceObserver) error {
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	root := filepath.Join(cache, "heliopause")
	intake, evidenceRoot, cacheRoot, stateRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "npm-project-cache"), filepath.Join(root, "npm-projects")
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
	native, err := sandbox.NewLinuxNPMResolverWithExecutorAndPolicy(executor, observer, policy)
	if err != nil {
		return err
	}
	backend, err := sandbox.NewLinuxBackendWithExecutor(intake, executor, observer)
	if err != nil {
		return err
	}
	return cli.AddNPMProject(command, npmProjectCLIService{npmProjectBindings{intake, evidenceRoot, cacheRoot, stateRoot, source, store, static, native, backend}})
}
