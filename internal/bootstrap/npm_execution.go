package bootstrap

import (
	"context"
	"os"
	"path/filepath"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/hosttool"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
	"github.com/spf13/cobra"
)

func runNPMExecution(ctx context.Context, command *cobra.Command, executor *hosttool.Executor, observer sandbox.TraceObserver) error {
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	root := filepath.Join(cache, "heliopause")
	intake := filepath.Join(root, "intake")
	source, err := artifactnpm.NewPublicResolver(intake)
	if err != nil {
		return err
	}
	static, err := inspectionnpm.NewInspector(intake)
	if err != nil {
		return err
	}
	backend, err := sandbox.NewLinuxBackendWithExecutor(intake, executor, observer)
	if err != nil {
		return err
	}
	store, err := local.NewStore(filepath.Join(root, "evidence"))
	if err != nil {
		return err
	}
	staging, err := promotion.NewLocalStaging(intake, filepath.Join(root, "evidence"), filepath.Join(root, "staging"))
	if err != nil {
		return err
	}
	resolverPolicy, err := newSystemResolverPolicyAdapter()
	if err != nil {
		return err
	}
	if cli.SelectedPackageClient(command) == cli.PNPMRunPackageClient {
		native, err := sandbox.NewLinuxPNPMResolverWithExecutorAndPolicy(executor, observer, resolverPolicy)
		if err != nil {
			return err
		}
		prepare, err := sandbox.NewPNPMInspectionPreparer(native, intake)
		if err != nil {
			return err
		}
		if err := cli.AddPNPMRunExecution(command, pnpmRunExecutionWorkflow{native, source, static, prepare, backend, store, staging}); err != nil {
			return err
		}
		return command.ExecuteContext(ctx)
	}
	resolver, err := sandbox.NewLinuxNPMResolverWithExecutorAndPolicy(executor, observer, resolverPolicy)
	if err != nil {
		return err
	}
	service, err := newNPMExecutionService(resolver, source, static, backend, store, staging)
	if err != nil {
		return err
	}
	if err := cli.AddNPMExecution(command, service); err != nil {
		return err
	}
	return command.ExecuteContext(ctx)
}
