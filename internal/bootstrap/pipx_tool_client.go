package bootstrap

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/sandbox"
	"github.com/spf13/cobra"
)

// The base resolver serves the unchanged intake/prerequisite pipeline. Tool
// selection must use the distinct pipx producer, never UV tool selection.
type pipxRuntimeResolver struct {
	*sandbox.UVResolver
	tool *sandbox.PipxResolver
}

func (r *pipxRuntimeResolver) Close() error { return errors.Join(r.tool.Close(), r.UVResolver.Close()) }
func pipxToolClientPlatform(goos, goarch string) error {
	if goos != "linux" || goarch != "amd64" {
		return errors.New("pipx quarantine requires Linux amd64")
	}
	return nil
}
func pipxInstallDependencyResolver(goos, goarch string, executor sandbox.TrustedExecutor, observer sandbox.TraceObserver) (ports.DependencyResolver, error) {
	if err := pipxToolClientPlatform(goos, goarch); err != nil {
		return nil, err
	}
	policy, err := newSystemResolverPolicyAdapter()
	if err != nil {
		return nil, err
	}
	base, err := sandbox.NewLinuxUVResolverWithExecutorAndPolicy(executor, observer, policy)
	if err != nil {
		return nil, err
	}
	tool, err := sandbox.NewLinuxPipxResolverWithExecutorAndPolicy(executor, observer, policy, newObservationResourceAdapter())
	if err != nil {
		return nil, errors.Join(err, base.Close())
	}
	return &pipxRuntimeResolver{base, tool}, nil
}
func closePipxDependencyResolver(resolver ports.DependencyResolver) error {
	owned, ok := resolver.(*pipxRuntimeResolver)
	if !ok || owned == nil {
		return errors.New("pipx resolver lifetime owner changed")
	}
	return owned.Close()
}

type pipxToolCLIInstaller struct {
	bindings    pythonClientBindings
	native      *sandbox.PipxResolver
	readBinRoot func() (string, error)
}

func (s pipxToolCLIInstaller) Install(ctx context.Context, request application.InstallRequest) (application.InstallOutcome, error) {
	if s.readBinRoot == nil || s.native == nil {
		return application.InstallOutcome{}, errors.New("pipx request paths or selector unavailable")
	}
	bin, err := s.readBinRoot()
	if err != nil {
		return application.InstallOutcome{}, err
	}
	if request.Operation() == domain.OperationUpdate {
		service, err := newPipxToolUpdateService(s.bindings, s.native, bin)
		if err != nil {
			return application.InstallOutcome{}, err
		}
		return service.Install(ctx, request)
	}
	service, err := newPipxToolInstallService(s.bindings, s.native, bin)
	if err != nil {
		return application.InstallOutcome{}, err
	}
	return service.Install(ctx, request)
}
func bindPipxToolClient(root *cobra.Command, b pythonClientBindings) error {
	runtime, ok := b.nativeLookup.(*pipxRuntimeResolver)
	if !ok || runtime == nil || runtime.tool == nil {
		return errors.New("pipx client requires its distinct native selector")
	}
	if cli.SelectedPackageClientOperation(root) == "upgrade" {
		command, err := cli.PipxToolUpgradeCommand(root)
		if err != nil {
			return err
		}
		return cli.AddPipxToolUpgrade(root, pipxToolCLIInstaller{b, runtime.tool, func() (string, error) { return command.Flags().GetString("bin-dir") }})
	}
	command, err := cli.PackageClientInstallCommand(root, cli.PipxPackageClient)
	if err != nil {
		return err
	}
	return cli.AddPipxToolInstall(root, pipxToolCLIInstaller{b, runtime.tool, func() (string, error) { return command.Flags().GetString("bin-dir") }})
}
