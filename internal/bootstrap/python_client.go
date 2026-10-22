package bootstrap

import (
	"errors"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
	"github.com/spf13/cobra"
)

// Client selection is composition data; it never changes registry SourceID or
// the inspection/Policy/staging ordering owned by Application.
type pythonPackageClient struct {
	id         cli.PackageClientID
	resolver   func(string, string, sandbox.TrustedExecutor, sandbox.TraceObserver) (ports.DependencyResolver, error)
	derivation bool
	platform   func(string, string) error
	bind       func(*cobra.Command, pythonClientBindings) error
}

type pythonClientBindings struct {
	installer              *application.InstallService
	promoter               ports.Promotion
	resolver, nativeLookup ports.DependencyResolver
	intake                 ports.Artifact
	intakeRoot             string
}

func pythonPackageClientFor(id cli.PackageClientID) (pythonPackageClient, error) {
	if id == "" {
		id = cli.PipPackageClient
	}
	clients := map[cli.PackageClientID]pythonPackageClient{
		cli.PipPackageClient:   {id: cli.PipPackageClient, resolver: pypiInstallDependencyResolver, derivation: true, bind: bindPipPackageClient},
		cli.UVPipPackageClient: {id: cli.UVPipPackageClient, resolver: uvInstallDependencyResolver, platform: uvPackageClientPlatform, bind: bindUVPipPackageClient},
	}
	client, ok := clients[id]
	if !ok {
		return pythonPackageClient{}, errors.New("package client has no reviewed composition")
	}
	return client, nil
}

func (c pythonPackageClient) checkPlatform(goos, goarch string) error {
	if c.platform != nil {
		return c.platform(goos, goarch)
	}
	return nil
}

func uvPackageClientPlatform(goos, goarch string) error {
	if goos != "linux" || goarch != "amd64" {
		return errors.New("UV pip quarantine requires Linux amd64")
	}
	return nil
}

func uvInstallDependencyResolver(goos, goarch string, executor sandbox.TrustedExecutor, observer sandbox.TraceObserver) (ports.DependencyResolver, error) {
	if err := uvPackageClientPlatform(goos, goarch); err != nil {
		return nil, err
	}
	policy, err := newSystemResolverPolicyAdapter()
	if err != nil {
		return nil, err
	}
	return sandbox.NewLinuxUVResolverWithExecutorAndPolicy(executor, observer, policy)
}

func bindUVPipPackageClient(command *cobra.Command, bindings pythonClientBindings) error {
	promoter, ok := bindings.promoter.(*promotion.PyPIPromotion)
	if !ok {
		return errors.New("UV installation requires the existing Python transaction")
	}
	preparation, err := promotion.NewUVInstallPreparation(promoter, bindings.resolver)
	if err != nil {
		return err
	}
	installer, err := application.NewPreparedInstallService(bindings.installer, preparation)
	if err != nil {
		return err
	}
	return cli.AddUVPipInstall(command, installer)
}

func bindPipPackageClient(command *cobra.Command, bindings pythonClientBindings) error {
	installers := map[string]cli.Installer{"pypi": bindings.installer}
	for _, profile := range artifactpypi.AllSourceProfiles() {
		if artifactpypi.IsPyTorchSource(profile.Source()) {
			installers[profile.Source().String()] = pyTorchResourceInstaller{installer: bindings.installer, profile: profile}
		}
	}
	updateResolver, ok := bindings.resolver.(ports.DependencyUpdateResolver)
	if !ok {
		return errors.New("python resolver lacks guarded upgrade selection")
	}
	promoter, ok := bindings.promoter.(*promotion.PyPIPromotion)
	if !ok {
		return errors.New("python Promotion lacks guarded upgrade preparation")
	}
	preparation, err := promotion.NewPyPIUpdatePreparation(promoter, updateResolver)
	if err != nil {
		return err
	}
	preparation, err = preparation.WithNativeEnrollment(bindings.nativeLookup, bindings.intake, bindings.intakeRoot)
	if err != nil {
		return err
	}
	updater, err := application.NewInstallUpdateService(bindings.installer, preparation)
	if err != nil {
		return err
	}
	updaters := map[string]cli.Installer{"pypi": updater}
	for _, profile := range artifactpypi.AllSourceProfiles() {
		if artifactpypi.IsPyTorchSource(profile.Source()) {
			updaters[profile.Source().String()] = pyTorchResourceInstaller{installer: updater, profile: profile}
		}
	}
	return cli.AddPyPIInstallSourcesAndUpdates(command, installers, updaters)
}
