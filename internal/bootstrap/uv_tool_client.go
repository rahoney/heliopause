package bootstrap

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

type uvToolCLIInstaller struct {
	bindings    pythonClientBindings
	readBinRoot func() (string, error)
}

func (s uvToolCLIInstaller) Install(ctx context.Context, request application.InstallRequest) (application.InstallOutcome, error) {
	if s.readBinRoot == nil {
		return application.InstallOutcome{}, errors.New("UV tool request paths are unavailable")
	}
	binRoot, err := s.readBinRoot()
	if err != nil {
		return application.InstallOutcome{}, err
	}
	if request.Operation() == domain.OperationUpdate {
		service, err := newUVToolUpdateService(s.bindings, binRoot)
		if err != nil {
			return application.InstallOutcome{}, err
		}
		return service.Install(ctx, request)
	}
	service, err := newUVToolInstallService(s.bindings, binRoot)
	if err != nil {
		return application.InstallOutcome{}, err
	}
	return service.Install(ctx, request)
}

func bindUVToolClient(root *cobra.Command, bindings pythonClientBindings) error {
	if cli.SelectedPackageClientOperation(root) == "upgrade" {
		command, err := cli.UVToolUpgradeCommand(root)
		if err != nil {
			return err
		}
		return cli.AddUVToolUpgrade(root, uvToolCLIInstaller{bindings, func() (string, error) { return command.Flags().GetString("bin-dir") }})
	}
	command, err := cli.PackageClientInstallCommand(root, cli.UVToolPackageClient)
	if err != nil {
		return err
	}
	return cli.AddUVToolInstall(root, uvToolCLIInstaller{bindings, func() (string, error) { return command.Flags().GetString("bin-dir") }})
}

func uvToolClientPlatform(goos, goarch string) error {
	if goos != "linux" || goarch != "amd64" {
		return errors.New("UV tool install requires Linux amd64")
	}
	return nil
}
