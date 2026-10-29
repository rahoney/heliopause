package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/hosttool"
	inspectionpypi "github.com/rahoney/heliopause/internal/inspection/pypi"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
	"github.com/spf13/cobra"
)

type pythonUVRunCLI struct {
	native    *sandbox.UVResolver
	intake    *artifactpypi.Intake
	inspector *inspectionpypi.CompositeInspector
	static    *inspectionpypi.StaticInspector
	backend   *sandbox.PythonDynamicBackend
	store     *local.Store
	staging   *promotion.LocalStaging
}

func (r pythonUVRunCLI) Run(ctx context.Context, request application.ExecutionPreparationRequest) (result application.ExecutionResult, resultErr error) {
	if ctx == nil {
		return result, errors.New("native CLI execution requires context")
	}
	defer func() { resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx)) }()
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	service, err := newUVRunExecutionService(r.native, request.Command(), r.intake, r.inspector, r.static, r.backend, r.store, r.staging)
	if err != nil {
		return result, err
	}
	return service.Run(ctx, request)
}
func uvRunExecutionPlatform(goos, arch string) error {
	if goos != "linux" || arch != "amd64" {
		return errors.New("native UV execution requires Linux amd64")
	}
	return nil
}
func runUVRunExecution(ctx context.Context, command *cobra.Command, executor *hosttool.Executor, observer sandbox.TraceObserver) (resultErr error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	root := filepath.Join(cache, "heliopause")
	intakeRoot := filepath.Join(root, "intake")
	intake, err := artifactpypi.NewPublicIntake(intakeRoot)
	if err != nil {
		return err
	}
	runtime := sandbox.PinnedPythonRuntime()
	static, err := inspectionpypi.NewStaticInspector(intakeRoot, artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag})
	if err != nil {
		return err
	}
	backend, err := sandbox.NewLinuxPyPIDynamicBackendWithExecutor(intakeRoot, executor, observer, newObservationResourceAdapter())
	if err != nil {
		return err
	}
	dynamic, err := inspectionpypi.NewDynamicInspector(backend)
	if err != nil {
		return err
	}
	inspector, err := inspectionpypi.NewCompositeInspector(static, dynamic)
	if err != nil {
		return err
	}
	store, err := local.NewStore(filepath.Join(root, "evidence"))
	if err != nil {
		return err
	}
	staging, err := promotion.NewLocalStaging(intakeRoot, filepath.Join(root, "evidence"), filepath.Join(root, "staging"))
	if err != nil {
		return err
	}
	policy, err := newSystemResolverPolicyAdapter()
	if err != nil {
		return err
	}
	native, err := sandbox.NewLinuxUVResolverWithExecutorAndPolicy(executor, observer, policy)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, native.Close()) }()
	if err := cli.AddUVRunExecution(command, pythonUVRunCLI{native, intake, inspector, static, backend, store, staging}); err != nil {
		return err
	}
	return command.ExecuteContext(ctx)
}
