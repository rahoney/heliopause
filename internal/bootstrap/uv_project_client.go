package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionpypi "github.com/rahoney/heliopause/internal/inspection/pypi"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
	"github.com/spf13/cobra"
)

type uvProjectCLIService struct{ bindings uvProjectBindings }

func (s uvProjectCLIService) ApplyUVProject(ctx context.Context, request cli.UVProjectRequest) (outcome cli.UVProjectOutcome, resultErr error) {
	if request.Intent == artifactpypi.UVProjectLockUpgrade {
		service, err := newUVProjectLockService(s.bindings, request.ProjectRoot)
		if err != nil {
			return outcome, err
		}
		outcome.Verified, err = service.Apply(ctx, request.Install)
		if err != nil {
			return outcome, err
		}
		id, manifest, sbom, ok := service.promoter.PublishedRecords()
		if !ok {
			return outcome, errors.New("UV lock completed without committed publication records")
		}
		outcome.ManifestID, outcome.ManifestHandle, outcome.SBOMHandle = id, manifest, sbom
		return outcome, nil
	}
	service, err := newUVProjectService(s.bindings, request.ProjectRoot, request.Intent, request.Selector, request.Adopt)
	if err != nil {
		return outcome, err
	}
	outcome.Verified, err = service.Apply(ctx, request.Install)
	if err != nil {
		return outcome, err
	}
	id, manifest, sbom, ok := service.promoter.PublishedRecords()
	if !ok {
		return outcome, errors.New("UV project completed without committed publication records")
	}
	outcome.ManifestID, outcome.ManifestHandle, outcome.SBOMHandle = id, manifest, sbom
	return outcome, nil
}

type uvProjectExecutor interface {
	sandbox.TrustedExecutor
	promotion.DockerRunner
}

func bindUVProjectClient(command *cobra.Command, executor uvProjectExecutor, observer sandbox.TraceObserver) (func() error, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	root := filepath.Join(cache, "heliopause")
	intakeRoot, evidenceRoot, cacheRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "staging")
	intake, err := artifactpypi.NewPublicIntake(intakeRoot)
	if err != nil {
		return nil, err
	}
	runtime := sandbox.PinnedPythonRuntime()
	static, err := inspectionpypi.NewStaticInspector(intakeRoot, artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag})
	if err != nil {
		return nil, err
	}
	backend, err := sandbox.NewLinuxPyPIDynamicBackendWithExecutor(intakeRoot, executor, observer, newObservationResourceAdapter())
	if err != nil {
		return nil, err
	}
	dynamic, err := inspectionpypi.NewDynamicInspector(backend)
	if err != nil {
		return nil, err
	}
	inspector, err := inspectionpypi.NewCompositeInspector(static, dynamic)
	if err != nil {
		return nil, err
	}
	store, err := local.NewStore(evidenceRoot)
	if err != nil {
		return nil, err
	}
	policy, err := newSystemResolverPolicyAdapter()
	if err != nil {
		return nil, err
	}
	native, err := sandbox.NewLinuxUVResolverWithExecutorAndPolicy(executor, observer, policy)
	if err != nil {
		return nil, err
	}
	resolver, err := sandbox.NewUVProjectResolver(native)
	if err != nil {
		return nil, err
	}
	promoter, err := promotion.NewPyPIPromotionWithRunner(cacheRoot, executor)
	if err != nil {
		return nil, errors.Join(err, resolver.Close())
	}
	if err := cli.AddUVProject(command, uvProjectCLIService{uvProjectBindings{intakeRoot, evidenceRoot, cacheRoot, intake, inspector, store, promoter, resolver, native}}); err != nil {
		return nil, errors.Join(err, resolver.Close())
	}
	return resolver.Close, nil
}
