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
	"github.com/rahoney/heliopause/internal/releaseinstall"
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
	command.SetArgs(args)
	if helpRequested(args) {
		return command.ExecuteContext(ctx)
	}
	if len(args) > 0 && args[0] == "doctor" {
		root, rootErr := releaseinstall.DefaultRoot()
		if rootErr != nil {
			return rootErr
		}
		if err := cli.AddDoctor(command, doctorService{installRoot: root}); err != nil {
			return err
		}
		return command.ExecuteContext(ctx)
	}
	var trustedExecutor *hosttool.Executor
	var observerSupervisor *sandbox.ObserverSupervisor
	var processObserver sandbox.TraceObserver
	if runtime.GOOS == "linux" && len(args) > 0 && (args[0] == "npm" || args[0] == "pypi" || args[0] == "pip" || args[0] == "github") {
		trustedExecutor, err = hosttool.NewSystem(ctx)
		if err != nil {
			return err
		}
		defer func() { resultErr = errors.Join(resultErr, trustedExecutor.Close()) }()
		launcher, launcherErr := hosttool.NewSystemObserverLauncher()
		if launcherErr != nil {
			return launcherErr
		}
		observerSupervisor, err = sandbox.NewObserverSupervisor(ctx, func(startContext context.Context, remoteEndpoint, outputEndpoint string) (sandbox.ObserverProcess, error) {
			return launcher.StartObserver(startContext, remoteEndpoint, outputEndpoint)
		})
		if err != nil {
			return err
		}
		processObserver = observerSupervisor.Observer()
		defer func() { resultErr = errors.Join(resultErr, observerSupervisor.Close()) }()
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
			backend, err := sandbox.NewLinuxBackendWithExecutor(filepath.Join(root, "intake"), trustedExecutor, processObserver)
			if err != nil {
				return err
			}
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
		dependencyResolver, err := installDependencyResolver(runtime.GOOS, runtime.GOARCH, trustedExecutor, processObserver)
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
	if len(args) > 0 && (args[0] == "pypi" || args[0] == "pip") {
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
			backend, factoryErr := sandbox.NewLinuxPyPIDynamicBackendWithExecutor(filepath.Join(root, "intake"), trustedExecutor, processObserver)
			if factoryErr != nil {
				return factoryErr
			}
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
		resolver, err := pypiInstallDependencyResolver(runtime.GOOS, runtime.GOARCH, trustedExecutor, processObserver)
		if err != nil {
			return err
		}
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
			builder, factoryErr := sandbox.NewLinuxPyPISdistBuilderWithExecutor(filepath.Join(root, "intake"), trustedExecutor, processObserver)
			if factoryErr != nil {
				return factoryErr
			}
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
		installers := map[string]cli.Installer{"pypi": installer}
		for _, profileName := range []string{"cpu", "cu126", "cu128"} {
			if profile, ok := artifactpypi.PyTorchProfile(profileName); ok {
				installers[profile.Source().String()] = installer
			}
		}
		if err := cli.AddPyPIInstallSources(command, installers); err != nil {
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
			backend, err := sandbox.NewLinuxGitHubELFBackendWithExecutor(filepath.Join(root, "intake"), trustedExecutor, processObserver)
			if err != nil {
				return err
			}
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
	return command.ExecuteContext(ctx)
}

func helpRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func pypiInstallDependencyResolver(goos, goarch string, executor sandbox.TrustedExecutor, observer sandbox.TraceObserver) (ports.DependencyResolver, error) {
	if goos == "linux" && goarch == "amd64" {
		policyService, err := newSystemResolverPolicyAdapter()
		if err != nil {
			return nil, err
		}
		resolver, err := sandbox.NewLinuxPyPIResolverWithExecutorAndPolicy(executor, observer, policyService)
		if err != nil {
			return nil, err
		}
		routes := map[string]ports.DependencyResolver{artifactpypi.PublicPyPIProfile().Source().String(): resolver}
		for _, profileName := range []string{"cpu", "cu126", "cu128"} {
			profile, ok := artifactpypi.PyTorchProfile(profileName)
			if !ok {
				return nil, errors.New("PyTorch source profile is missing")
			}
			pytorchResolver, resolverErr := sandbox.NewLinuxPyTorchResolverWithExecutorAndPolicy(executor, observer, policyService, profile)
			if resolverErr != nil {
				return nil, resolverErr
			}
			routes[profile.Source().String()] = pytorchResolver
		}
		return pythonSourceResolver{routes: routes}, nil
	}
	return unsupportedPyPIInstallResolver{}, nil
}

type pythonSourceResolver struct {
	routes map[string]ports.DependencyResolver
}

func (r pythonSourceResolver) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, installContext domain.InstallContext) (domain.DependencyResolution, error) {
	if resolver := r.routes[reference.Source().String()]; resolver != nil {
		return resolver.ResolveDependencies(ctx, reference, installContext)
	}
	return domain.DependencyResolution{}, errors.New("selected Python source profile is unsupported")
}

func installDependencyResolver(goos, goarch string, executor sandbox.TrustedExecutor, observer sandbox.TraceObserver) (ports.DependencyResolver, error) {
	if goos == "linux" && goarch == "amd64" {
		policyService, err := newSystemResolverPolicyAdapter()
		if err != nil {
			return nil, err
		}
		return sandbox.NewLinuxNPMResolverWithExecutorAndPolicy(executor, observer, policyService)
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

type doctorService struct{ installRoot string }

func (s doctorService) Diagnose(ctx context.Context) cli.DoctorReport {
	installation := releaseinstall.CheckInstallation(s.installRoot)
	checks := append([]cli.DoctorCheck{{Name: installation.Name, Healthy: installation.Healthy, Detail: installation.Detail}}, doctorHostChecks(ctx)...)
	healthy := true
	for _, check := range checks {
		healthy = healthy && check.Healthy
	}
	return cli.DoctorReport{Healthy: healthy, Checks: checks}
}
