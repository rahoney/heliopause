// Package bootstrap owns the Heliopause composition root.
package bootstrap

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/rahoney/heliopause/internal/application"
	artifactgithub "github.com/rahoney/heliopause/internal/artifact/githubrelease"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	evidencerecord "github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/hosttool"
	inspectiongithub "github.com/rahoney/heliopause/internal/inspection/githubrelease"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	inspectionpypi "github.com/rahoney/heliopause/internal/inspection/pypi"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationgithub "github.com/rahoney/heliopause/internal/verification/githubrelease"
	verificationnpm "github.com/rahoney/heliopause/internal/verification/npm"
	verificationpypi "github.com/rahoney/heliopause/internal/verification/pypi"
)

// Run constructs and executes the CLI with process-owned dependencies.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) (resultErr error) {
	command, err := cli.New(stdout, stderr)
	if err != nil {
		return err
	}
	var trustedExecutor *hosttool.Executor
	if runtime.GOOS == "linux" && len(args) > 0 && (args[0] == "npm" || args[0] == "pypi" || args[0] == "github") {
		trustedExecutor, err = hosttool.NewSystem(ctx)
		if err != nil {
			return err
		}
		defer func() { resultErr = errors.Join(resultErr, trustedExecutor.Close()) }()
	}
	if len(args) > 0 && args[0] == "npm" {
		cacheRoot, err := os.UserCacheDir()
		if err != nil {
			return err
		}
		root := filepath.Join(cacheRoot, "heliopause")
		artifact, err := artifactnpm.NewPublicResolver(filepath.Join(root, "intake"))
		if err != nil {
			return err
		}
		staticInspection, err := inspectionnpm.NewInspector(filepath.Join(root, "intake"))
		if err != nil {
			return err
		}
		var dynamicSandbox ports.Sandbox
		if runtime.GOOS == "linux" {
			backend, observer, err := sandbox.NewLinuxBackendWithExecutor(filepath.Join(root, "intake"), trustedExecutor)
			if err != nil {
				return err
			}
			defer observer.Close()
			dynamicSandbox = backend
		} else {
			probedSandbox, err := sandbox.NewProbedSandbox(sandbox.Probe)
			if err != nil {
				return err
			}
			dynamicSandbox = probedSandbox
		}
		dynamicInspection, err := inspectionnpm.NewDynamicInspector(dynamicSandbox)
		if err != nil {
			return err
		}
		inspection, err := inspectionnpm.NewCompositeInspector(staticInspection, dynamicInspection)
		if err != nil {
			return err
		}
		evidence, err := local.NewStore(filepath.Join(root, "evidence"))
		if err != nil {
			return err
		}
		service, err := application.NewInspectService(artifact, verificationnpm.IntegrityVerifier{}, inspection, evidence, policy.M3{}, domain.NewOperationID, domain.NewRunID)
		if err != nil {
			return err
		}
		if err := cli.AddNPMInspect(command, service); err != nil {
			return err
		}
		dependencyResolver, err := installDependencyResolver(runtime.GOOS, runtime.GOARCH, trustedExecutor)
		if err != nil {
			return err
		}
		installInspection, err := application.NewInstallInspectService(dependencyResolver, artifact, verificationnpm.IntegrityVerifier{}, inspection, evidence, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
		if err != nil {
			return err
		}
		staging, err := promotion.NewLocalStaging(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "staging"))
		if err != nil {
			return err
		}
		var promoter ports.Promotion
		if runtime.GOOS == "linux" {
			promoter, err = promotion.NewNPMPromotionWithRunner(filepath.Join(root, "staging"), trustedExecutor)
		} else {
			promoter, err = promotion.NewNPMPromotion(filepath.Join(root, "staging"))
		}
		if err != nil {
			return err
		}
		installer, err := application.NewInstallService(installInspection, evidencerecord.Generator{}, staging, promoter)
		if err != nil {
			return err
		}
		if err := cli.AddNPMInstall(command, installer); err != nil {
			return err
		}
	}
	if len(args) > 0 && args[0] == "pypi" {
		cacheRoot, err := os.UserCacheDir()
		if err != nil {
			return err
		}
		root := filepath.Join(cacheRoot, "heliopause")
		intake, err := artifactpypi.NewPublicIntake(filepath.Join(root, "intake"))
		if err != nil {
			return err
		}
		runtimeIdentity := sandbox.PinnedPythonRuntime()
		static, err := inspectionpypi.NewStaticInspector(filepath.Join(root, "intake"), artifactpypi.WheelTarget{Python: runtimeIdentity.InterpreterTag, ABI: runtimeIdentity.ABITag, Platform: runtimeIdentity.PlatformTag})
		if err != nil {
			return err
		}
		var wheelRunner sandbox.PythonWheelRunner = sandbox.UnavailablePythonWheelRunner{}
		if runtime.GOOS == "linux" {
			backend, closeObserver, factoryErr := sandbox.NewLinuxPyPIDynamicBackendWithExecutor(filepath.Join(root, "intake"), trustedExecutor)
			if factoryErr != nil {
				return factoryErr
			}
			defer closeObserver()
			wheelRunner = backend
		}
		dynamic, err := inspectionpypi.NewDynamicInspector(wheelRunner)
		if err != nil {
			return err
		}
		inspection, err := inspectionpypi.NewCompositeInspector(static, dynamic)
		if err != nil {
			return err
		}
		evidence, err := local.NewStore(filepath.Join(root, "evidence"))
		if err != nil {
			return err
		}
		resolver, closeResolver, err := pypiInstallDependencyResolver(runtime.GOOS, runtime.GOARCH, trustedExecutor)
		if err != nil {
			return err
		}
		defer closeResolver()
		directArtifact, err := artifactpypi.NewGraphArtifact(resolver, intake)
		if err != nil {
			return err
		}
		directInspect, err := application.NewInspectService(directArtifact, verificationpypi.IntegrityVerifier{}, inspection, evidence, policy.M3{}, domain.NewOperationID, domain.NewRunID)
		if err != nil {
			return err
		}
		if err := cli.AddPyPIInspect(command, directInspect); err != nil {
			return err
		}
		installInspection, err := application.NewInstallInspectService(resolver, intake, verificationpypi.IntegrityVerifier{}, inspection, evidence, policy.M3{}, policy.M5{}, domain.NewOperationID, domain.NewRunID)
		if err != nil {
			return err
		}
		if runtime.GOOS == "linux" {
			builder, closeBuilderObserver, factoryErr := sandbox.NewLinuxPyPISdistBuilderWithExecutor(filepath.Join(root, "intake"), trustedExecutor)
			if factoryErr != nil {
				return factoryErr
			}
			defer closeBuilderObserver()
			deriver, deriveErr := inspectionpypi.NewDeriver(static, builder)
			if deriveErr != nil {
				return deriveErr
			}
			installInspection.WithDerivation(deriver)
		}
		staging, err := promotion.NewLocalStaging(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "staging"))
		if err != nil {
			return err
		}
		var promoter ports.Promotion
		if runtime.GOOS == "linux" {
			promoter, err = promotion.NewPyPIPromotionWithRunner(filepath.Join(root, "staging"), trustedExecutor)
		} else {
			promoter, err = promotion.NewPyPIPromotion(filepath.Join(root, "staging"))
		}
		if err != nil {
			return err
		}
		installer, err := application.NewInstallService(installInspection, evidencerecord.Generator{}, staging, promoter)
		if err != nil {
			return err
		}
		if err := cli.AddPyPIInstall(command, installer); err != nil {
			return err
		}
	}
	if len(args) > 0 && args[0] == "github" {
		cacheRoot, err := os.UserCacheDir()
		if err != nil {
			return err
		}
		root := filepath.Join(cacheRoot, "heliopause")
		artifact, err := artifactgithub.NewPublicClient(filepath.Join(root, "intake"))
		if err != nil {
			return err
		}
		static, err := inspectiongithub.NewStaticInspector(filepath.Join(root, "intake"))
		if err != nil {
			return err
		}
		var dynamicSandbox ports.Sandbox
		if runtime.GOOS == "linux" {
			backend, closeObserver, err := sandbox.NewLinuxGitHubELFBackendWithExecutor(filepath.Join(root, "intake"), trustedExecutor)
			if err != nil {
				return err
			}
			defer closeObserver()
			dynamicSandbox = backend
		} else {
			unavailable, err := sandbox.NewProbedSandbox(sandbox.Probe)
			if err != nil {
				return err
			}
			dynamicSandbox = unavailable
		}
		dynamic, err := inspectiongithub.NewDynamicInspector(dynamicSandbox)
		if err != nil {
			return err
		}
		inspection, err := inspectiongithub.NewCompositeInspector(static, dynamic)
		if err != nil {
			return err
		}
		evidence, err := local.NewStore(filepath.Join(root, "evidence"))
		if err != nil {
			return err
		}
		inspect, err := application.NewInspectService(artifact, verificationgithub.IntegrityVerifier{}, inspection, evidence, policy.M6{}, domain.NewOperationID, domain.NewRunID)
		if err != nil {
			return err
		}
		if err := cli.AddGitHubReleaseInspect(command, artifactgithub.ParseReference, inspect); err != nil {
			return err
		}
		installInspect, err := application.NewInstallInspectService(artifact, artifact, verificationgithub.IntegrityVerifier{}, inspection, evidence, policy.M6{}, policy.M6{}, domain.NewOperationID, domain.NewRunID)
		if err != nil {
			return err
		}
		staging, err := promotion.NewLocalStaging(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "staging"))
		if err != nil {
			return err
		}
		promoter, err := promotion.NewGitHubReleasePromotion(filepath.Join(root, "staging"))
		if err != nil {
			return err
		}
		installer, err := application.NewInstallService(installInspect, evidencerecord.Generator{}, staging, promoter)
		if err != nil {
			return err
		}
		if err := cli.AddGitHubReleaseInstall(command, artifactgithub.ParseReference, installer); err != nil {
			return err
		}
	}
	command.SetArgs(args)

	return command.ExecuteContext(ctx)
}

func pypiInstallDependencyResolver(goos, goarch string, executor sandbox.TrustedExecutor) (ports.DependencyResolver, func() error, error) {
	if goos == "linux" && goarch == "amd64" {
		resolver, err := sandbox.NewLinuxPyPIResolverWithExecutor(executor)
		if err != nil {
			return nil, nil, err
		}
		return resolver, resolver.Close, nil
	}
	return unsupportedPyPIInstallResolver{}, func() error { return nil }, nil
}

func installDependencyResolver(goos, goarch string, executor sandbox.TrustedExecutor) (ports.DependencyResolver, error) {
	if goos == "linux" && goarch == "amd64" {
		return sandbox.NewLinuxNPMResolverWithExecutor(executor)
	}
	return unsupportedInstallResolver{}, nil
}

type unsupportedInstallResolver struct{}

func (unsupportedInstallResolver) ResolveDependencies(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error) {
	return domain.DependencyResolution{}, errors.New("automatic npm install requires Linux amd64")
}

type unsupportedPyPIInstallResolver struct{}

func (unsupportedPyPIInstallResolver) ResolveDependencies(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error) {
	return domain.DependencyResolution{}, errors.New("automatic PyPI install requires Linux amd64")
}
