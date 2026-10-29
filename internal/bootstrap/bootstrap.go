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
	"github.com/rahoney/heliopause/internal/releasebootstrap"
	"github.com/rahoney/heliopause/internal/releaseinstall"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
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
	if err := cli.SetPNPMClientVersion(command, runtimeidentity.PNPMVersion); err != nil {
		return err
	}
	if err := cli.SetBunClientVersion(command, runtimeidentity.BunVersion); err != nil {
		return err
	}
	defer func() {
		if resultErr != nil {
			resultErr = cli.PresentRuntimeError(command, resultErr)
		}
	}()
	if err := cli.SetLanguageDetector(command, func() ([]string, string, error) {
		return hosttool.PreferredDisplayLanguages(ctx)
	}); err != nil {
		return err
	}
	command.SetArgs(args)
	source, err := cli.PreparePresentation(command, args)
	if err != nil {
		return err
	}
	if helpRequested(args) && cli.SelectedPackageClient(command) != cli.NPXPackageClient {
		return command.ExecuteContext(ctx)
	}
	if source == "" {
		// Binding is inert. The launcher alone decides whether to request a
		// diagnosis; opening help/settings must not resolve or prepare a runtime.
		if err := cli.AddDoctor(command, doctorService{}); err != nil {
			return err
		}
		return command.ExecuteContext(ctx)
	}
	if source == "config" {
		return command.ExecuteContext(ctx)
	}
	if source == "doctor" {
		root, rootErr := releaseinstall.DefaultRoot()
		if rootErr != nil {
			return rootErr
		}
		if err := cli.AddDoctor(command, doctorService{installRoot: root}); err != nil {
			return err
		}
		return command.ExecuteContext(ctx)
	}
	if source == "bootstrap" {
		if err := cli.AddBootstrapPreparation(command, releasebootstrap.New()); err != nil {
			return err
		}
		return command.ExecuteContext(ctx)
	}
	if source == "self-update" {
		if err := selfUpdatePlatform(); err != nil {
			return err
		}
	}
	if cli.SelectedPackageClient(command) == cli.NPXPackageClient && (source != "npm" || runtime.GOOS != "linux" || runtime.GOARCH != "amd64") {
		return errors.New("npx execution requires the public npm source and Linux amd64")
	}
	var pythonClient pythonPackageClient
	if (source == "pypi" || source == "pip") && cli.SelectedPackageClient(command) != cli.UVProjectPackageClient {
		pythonClient, err = pythonPackageClientFor(cli.SelectedPackageClient(command))
		if err != nil {
			return err
		}
		if err := pythonClient.checkPlatform(runtime.GOOS, runtime.GOARCH); err != nil {
			return err
		}
	}
	if cli.SelectedPackageClient(command) == cli.PNPMPackageClient {
		if source != "npm" {
			return errors.New("PNPM client requires the public npm source")
		}
		if err := pnpmProjectPlatform(runtime.GOOS, runtime.GOARCH); err != nil {
			return err
		}
	}
	if cli.SelectedPackageClient(command) == cli.BunPackageClient {
		if source != "npm" {
			return errors.New("bun client requires the public npm source")
		}
		if err := bunProjectPlatform(runtime.GOOS, runtime.GOARCH); err != nil {
			return err
		}
	}
	if cli.SelectedPackageClient(command) == cli.UVProjectPackageClient {
		if source != "pypi" {
			return errors.New("UV project requires the public PyPI source")
		}
		if err := uvPackageClientPlatform(runtime.GOOS, runtime.GOARCH); err != nil {
			return err
		}
	}
	if source == "npm" || source == "pypi" || source == "pip" || source == "github" || source == "go" || source == "cargo" || source == "terraform" || source == "self-update" {
		var closeRuntime func() error
		ctx, closeRuntime, err = commandRuntime(ctx)
		if err != nil {
			if source == "self-update" {
				return selfUpdateFailure(err)
			}
			return err
		}
		if closeRuntime != nil {
			defer func() {
				cleanupErr := closeRuntime()
				cli.PresentFinalizationError(command, cleanupErr)
				resultErr = errors.Join(resultErr, cleanupErr)
			}()
		}
	}
	var trustedExecutor *hosttool.Executor
	if source == "self-update" {
		if err := selfUpdateRuntime(ctx); err != nil {
			return err
		}
	}
	var observerSupervisor *sandbox.ObserverSupervisor
	var processObserver sandbox.TraceObserver
	if runtime.GOOS == "linux" && source != "" && (source == "npm" || source == "pypi" || source == "pip" || source == "github" || source == "go" || source == "cargo" || source == "terraform" || source == "self-update") {
		trustedExecutor, err = hosttool.NewSystem(ctx)
		if err != nil {
			return err
		}
		defer func() {
			cleanupErr := trustedExecutor.Close()
			cli.PresentFinalizationError(command, cleanupErr)
			resultErr = errors.Join(resultErr, cleanupErr)
		}()
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
		observerSupervisor.SetDiagnosticWriter(cli.DiagnosticWriter(command))
		processObserver = observerSupervisor.Observer()
		defer func() {
			cleanupErr := observerSupervisor.Close()
			cli.PresentFinalizationError(command, cleanupErr)
			resultErr = errors.Join(resultErr, cleanupErr)
		}()
	}
	if source != "" && source == "go" {
		if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
			return errors.New("automatic Go Module resolution requires Linux amd64")
		}
		resolverPolicy, policyErr := newSystemResolverPolicyAdapter()
		if policyErr != nil {
			return policyErr
		}
		isolatedRunner, runnerErr := sandbox.NewLinuxGoModuleRunner(trustedExecutor, processObserver, resolverPolicy)
		if runnerErr != nil {
			return runnerErr
		}
		resolver, resolverErr := sandbox.NewGoModuleResolver(isolatedRunner)
		if resolverErr != nil {
			return resolverErr
		}
		service, projectService, upgradeService, serviceErr := newGoModuleServices(resolver)
		if serviceErr != nil {
			return serviceErr
		}
		if err := cli.AddGoModuleGet(command, service); err != nil {
			return err
		}
		if err := cli.AddGoModuleUpgrade(command, upgradeService); err != nil {
			return err
		}
		if err := cli.AddGoModuleDownload(command, projectService); err != nil {
			return err
		}
		buildService, serviceErr := newGoModuleBuildService(trustedExecutor, processObserver)
		if serviceErr != nil {
			return serviceErr
		}
		if err := cli.AddGoModuleBuild(command, buildService); err != nil {
			return err
		}
	}
	if source != "" && source == "cargo" {
		if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
			return errors.New("automatic Cargo resolution requires Linux amd64")
		}
		resolverPolicy, policyErr := newSystemResolverPolicyAdapter()
		if policyErr != nil {
			return policyErr
		}
		isolatedRunner, runnerErr := sandbox.NewLinuxCargoRunner(trustedExecutor, processObserver, resolverPolicy)
		if runnerErr != nil {
			return runnerErr
		}
		resolver, resolverErr := sandbox.NewCargoResolver(isolatedRunner)
		if resolverErr != nil {
			return resolverErr
		}
		service, buildService, updateService, serviceErr := newCargoServices(resolver, trustedExecutor, processObserver)
		if serviceErr != nil {
			return serviceErr
		}
		if err := cli.AddCargoAdd(command, service); err != nil {
			return err
		}
		if err := cli.AddCargoBuild(command, buildService); err != nil {
			return err
		}
		if err := cli.AddCargoUpdate(command, updateService); err != nil {
			return err
		}
	}
	if source != "" && source == "terraform" {
		if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
			return errors.New("automatic Terraform Provider installation requires Linux amd64")
		}
		service, upgradeService, serviceErr := newTerraformServices(trustedExecutor, processObserver)
		if serviceErr != nil {
			return serviceErr
		}
		if err := cli.AddTerraformInit(command, service); err != nil {
			return err
		}
		if err := cli.AddTerraformUpgrade(command, upgradeService); err != nil {
			return err
		}
	}
	if source == "npm" && cli.SelectedPackageClient(command) == cli.PNPMPackageClient {
		if err := bindPNPMProjectClient(command, trustedExecutor, processObserver); err != nil {
			return err
		}
		return command.ExecuteContext(ctx)
	}
	if source == "npm" && cli.SelectedPackageClient(command) == cli.BunPackageClient {
		if err := bindBunProjectClient(command, trustedExecutor, processObserver); err != nil {
			return err
		}
		return command.ExecuteContext(ctx)
	}
	if source == "pypi" && cli.SelectedPackageClient(command) == cli.UVProjectPackageClient {
		closeClient, err := bindUVProjectClient(command, trustedExecutor, processObserver)
		if err != nil {
			return err
		}
		defer func() {
			cleanupErr := closeClient()
			cli.PresentFinalizationError(command, cleanupErr)
			resultErr = errors.Join(resultErr, cleanupErr)
		}()
		return command.ExecuteContext(ctx)
	}
	if source != "" && source == "npm" {
		if cli.SelectedPackageClient(command) == cli.NPXPackageClient {
			return runNPMExecution(ctx, command, trustedExecutor, processObserver)
		}
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
		updateResolver, ok := dependencyResolver.(ports.DependencyUpdateResolver)
		if !ok {
			return errors.New("npm resolver lacks guarded update selection")
		}
		npmPromoter, ok := promoter.(*promotion.NPMPromotion)
		if !ok {
			return errors.New("npm Promotion lacks guarded update preparation")
		}
		preparation, err := promotion.NewNPMUpdatePreparation(npmPromoter, updateResolver)
		if err != nil {
			return err
		}
		updater, err := application.NewInstallUpdateService(installer, preparation)
		if err != nil {
			return err
		}
		if err := cli.AddNPMUpdate(command, updater); err != nil {
			return err
		}
	}
	if source != "" && (source == "pypi" || source == "pip") {
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
			backend, factoryErr := sandbox.NewLinuxPyPIDynamicBackendWithExecutor(filepath.Join(root, "intake"), trustedExecutor, processObserver, newObservationResourceAdapter())
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
		resolver, err := pythonClient.resolver(runtime.GOOS, runtime.GOARCH, trustedExecutor, processObserver)
		if err != nil {
			return err
		}
		if pythonClient.close != nil {
			ownedResolver := resolver
			defer func() {
				cleanupErr := pythonClient.close(ownedResolver)
				cli.PresentFinalizationError(command, cleanupErr)
				resultErr = errors.Join(resultErr, cleanupErr)
			}()
		}
		installCommand, err := cli.PackageClientInstallCommand(command, pythonClient.id)
		if err != nil {
			return err
		}
		readPrerequisite := func() (string, error) {
			if installCommand.Flags().Lookup("inspection-prerequisites") == nil {
				return "", nil
			}
			return installCommand.Flags().GetString("inspection-prerequisites")
		}
		nativeLookupResolver := resolver
		inspection.WithInspectionPrerequisites(inspectionPrerequisiteLoader(readPrerequisite, resolver, intake))
		resolver = prerequisiteCheckingResolver{resolver: resolver, read: readPrerequisite}
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
		if runtime.GOOS == "linux" && pythonClient.derivation {
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
		if err := pythonClient.bind(command, pythonClientBindings{installer: installer, promoter: promoter, resolver: resolver, nativeLookup: nativeLookupResolver, intake: intake, intakeRoot: filepath.Join(root, "intake"), evidenceRoot: filepath.Join(root, "evidence"), stagingRoot: filepath.Join(root, "staging"), inspector: inspection, evidence: evidence}); err != nil {
			return err
		}
	}
	if source != "" && source == "github" {
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
		promoter, err := promotion.NewRecordedGitHubReleasePromotion(filepath.Join(root, "staging"), filepath.Join(root, "evidence"), filepath.Join(root, "github-installations"), evidence)
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
		preparation, err := promotion.NewGitHubReleaseUpdatePreparation(promoter, artifact)
		if err != nil {
			return err
		}
		updater, err := application.NewInstallUpdateService(installer, preparation)
		if err != nil {
			return err
		}
		if err := cli.AddGitHubReleaseReplacement(command, updater); err != nil {
			return err
		}
	}
	if source == "self-update" {
		updater, err := newSelfUpdater(ctx, trustedExecutor, observerSupervisor, processObserver)
		if err != nil {
			return err
		}
		if err := cli.AddSelfUpdate(command, updater); err != nil {
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
		for _, profile := range artifactpypi.AllSourceProfiles() {
			if !artifactpypi.IsPyTorchSource(profile.Source()) {
				continue
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

// pyTorchResourceInstaller selects one immutable root resource policy before
// the existing generic install workflow begins. Node identities remain owned
// by the locked dependency graph and are not rewritten here.
type pyTorchResourceInstaller struct {
	installer cli.Installer
	profile   artifactpypi.SourceProfile
}

func (i pyTorchResourceInstaller) Install(ctx context.Context, request application.InstallRequest) (application.InstallOutcome, error) {
	if i.installer == nil || request.Reference().Source() != i.profile.Source() {
		return application.InstallOutcome{}, errors.New("PyTorch resource installer request is invalid")
	}
	resourceContext, err := artifactpypi.ContextWithResourcePolicy(ctx, i.profile)
	if err != nil {
		return application.InstallOutcome{}, err
	}
	timedContext, cancel := context.WithTimeout(resourceContext, i.profile.ResourcePolicy().Duration())
	defer cancel()
	return i.installer.Install(timedContext, request)
}

func (r pythonSourceResolver) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, installContext domain.InstallContext) (domain.DependencyResolution, error) {
	if resolver := r.routes[reference.Source().String()]; resolver != nil {
		return resolver.ResolveDependencies(ctx, reference, installContext)
	}
	return domain.DependencyResolution{}, errors.New("selected Python source profile is unsupported")
}

func (r pythonSourceResolver) ResolveDependencyUpdate(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext, controls []domain.ProjectControlFile) (domain.DependencyResolution, error) {
	resolver, ok := r.routes[reference.Source().String()].(ports.DependencyUpdateResolver)
	if !ok {
		return domain.DependencyResolution{}, errors.New("selected Python source does not support guarded upgrades")
	}
	return resolver.ResolveDependencyUpdate(ctx, reference, install, controls)
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

func (unsupportedInstallResolver) ResolveDependencyUpdate(context.Context, domain.ArtifactReference, domain.InstallContext, []domain.ProjectControlFile) (domain.DependencyResolution, error) {
	return domain.DependencyResolution{}, errors.New("automatic npm update requires Linux amd64")
}

func (unsupportedInstallResolver) ResolveDependencies(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error) {
	return domain.DependencyResolution{}, errors.New("automatic npm install requires Linux amd64")
}

type unsupportedPyPIInstallResolver struct{}

func (unsupportedPyPIInstallResolver) ResolveDependencyUpdate(context.Context, domain.ArtifactReference, domain.InstallContext, []domain.ProjectControlFile) (domain.DependencyResolution, error) {
	return domain.DependencyResolution{}, errors.New("automatic PyPI upgrade requires Linux amd64")
}

func (unsupportedPyPIInstallResolver) ResolveDependencies(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error) {
	return domain.DependencyResolution{}, errors.New("automatic PyPI install requires Linux amd64")
}

type doctorService struct{ installRoot string }

func (s doctorService) Diagnose(ctx context.Context) cli.DoctorReport {
	root := s.installRoot
	if root == "" {
		// Missing user configuration is an unavailable installation check,
		// not a reason to prevent the interactive launcher from opening.
		if resolved, err := releaseinstall.DefaultRoot(); err == nil {
			root = resolved
		}
	}
	checks := installationAndHostChecks(ctx, root)
	healthy := true
	for _, check := range checks {
		healthy = healthy && check.Healthy
	}
	return cli.DoctorReport{Healthy: healthy, Checks: checks}
}
