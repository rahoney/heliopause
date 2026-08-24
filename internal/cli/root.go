// Package cli owns Heliopause command parsing and presentation.
package cli

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

// New returns a new Heliopause root command.
func New(stdout, stderr io.Writer) (*cobra.Command, error) {
	if stdout == nil {
		return nil, errors.New("cli: stdout writer is nil")
	}
	if stderr == nil {
		return nil, errors.New("cli: stderr writer is nil")
	}

	command := &cobra.Command{
		Use:           "helox",
		Short:         "Inspect software artifacts before trusted promotion",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	command.RunE = func(command *cobra.Command, _ []string) error {
		if err := command.Context().Err(); err != nil {
			return err
		}

		return command.Help()
	}
	command.SetOut(stdout)
	command.SetErr(stderr)

	return command, nil
}

type ReferenceParser func(string) (domain.ArtifactReference, error)

func AddGitHubReleaseInspect(root *cobra.Command, parser ReferenceParser, inspector Inspector) error {
	if root == nil || parser == nil || inspector == nil {
		return errors.New("github inspect command requires root and use case")
	}
	command := ensureGitHubCommand(root)
	inspect := &cobra.Command{Use: "inspect <owner>/<repo>@<tag>#<asset>", Args: cobra.ExactArgs(1), RunE: func(command *cobra.Command, args []string) error {
		reference, err := parser(args[0])
		if err != nil {
			return err
		}
		request, err := application.NewInspectRequest(reference)
		if err != nil {
			return err
		}
		code, err := ExecuteInspect(contextOrBackground(command.Context()), inspector, request, true, command.OutOrStdout())
		if err != nil {
			return err
		}
		if code != 0 {
			return ExitError{Code: code}
		}
		return nil
	}}
	command.AddCommand(inspect)
	return nil
}

func AddGitHubReleaseInstall(root *cobra.Command, parser ReferenceParser, installer Installer) error {
	if root == nil || parser == nil || installer == nil {
		return errors.New("github install command requires root and use case")
	}
	command := ensureGitHubCommand(root)
	var target string
	install := &cobra.Command{Use: "install <owner>/<repo>@<tag>#<asset>", Args: cobra.ExactArgs(1), RunE: func(command *cobra.Command, args []string) error {
		reference, err := parser(args[0])
		if err != nil {
			return err
		}
		targetValue, err := domain.NewInstallTarget(target)
		if err != nil {
			return err
		}
		installContext, err := domain.NewInstallContext(targetValue)
		if err != nil {
			return err
		}
		request, err := application.NewInstallRequest(reference, installContext)
		if err != nil {
			return err
		}
		code, err := ExecuteInstall(contextOrBackground(command.Context()), installer, request, true, command.OutOrStdout())
		if err != nil {
			return err
		}
		if code != 0 {
			return ExitError{Code: code}
		}
		return nil
	}}
	install.Flags().StringVar(&target, "target", "", "new absolute installation target (required)")
	_ = install.MarkFlagRequired("target")
	command.AddCommand(install)
	return nil
}

// AddPyPIInstall adds the injected PyPI Install/Promotion use case. The
// generic JSON/human result contract is intentionally shared with npm.
func AddPyPIInstall(root *cobra.Command, installer Installer) error {
	if root == nil || installer == nil {
		return errors.New("pypi install command requires root and use case")
	}
	pypiCommand := ensurePyPICommand(root)
	var target string
	installCommand := &cobra.Command{Use: "install <project>[@<version>]", Args: cobra.ExactArgs(1), RunE: func(command *cobra.Command, args []string) error {
		reference, err := artifactpypi.ParseReference(args[0])
		if err != nil {
			return err
		}
		installTarget, err := domain.NewInstallTarget(target)
		if err != nil {
			return err
		}
		installContext, err := domain.NewInstallContext(installTarget)
		if err != nil {
			return err
		}
		request, err := application.NewInstallRequest(reference, installContext)
		if err != nil {
			return err
		}
		exitCode, operationErr := ExecuteInstall(contextOrBackground(command.Context()), installer, request, true, command.OutOrStdout())
		if operationErr != nil {
			return operationErr
		}
		if exitCode != 0 {
			return ExitError{Code: exitCode}
		}
		return nil
	}}
	installCommand.Flags().StringVar(&target, "target", "", "new absolute installation target (required)")
	_ = installCommand.MarkFlagRequired("target")
	pypiCommand.AddCommand(installCommand)
	return nil
}

func npmInstallContext(target string) (domain.InstallContext, error) {
	if target != "" {
		value, err := domain.NewInstallTarget(target)
		if err != nil {
			return domain.InstallContext{}, err
		}
		return domain.NewInstallContext(value)
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return domain.InstallContext{}, errors.New("resolve npm project directory")
	}
	project, err := domain.NewInstallTarget(workingDirectory)
	if err != nil {
		return domain.InstallContext{}, errors.New("npm project directory is unsupported")
	}
	return domain.NewNPMProjectInstallContext(project)
}

// AddPyPIInspect adds the isolated PyPI primary-distribution inspect path.
func AddPyPIInspect(root *cobra.Command, inspector Inspector) error {
	if root == nil || inspector == nil {
		return errors.New("pypi inspect command requires root and use case")
	}
	pypiCommand := ensurePyPICommand(root)
	inspectCommand := &cobra.Command{Use: "inspect <project>[@<version>]", Args: cobra.ExactArgs(1), RunE: func(command *cobra.Command, args []string) error {
		reference, err := artifactpypi.ParseReference(args[0])
		if err != nil {
			return err
		}
		request, err := application.NewInspectRequest(reference)
		if err != nil {
			return err
		}
		exitCode, operationErr := ExecuteInspect(contextOrBackground(command.Context()), inspector, request, true, command.OutOrStdout())
		if operationErr != nil {
			return operationErr
		}
		if exitCode != 0 {
			return ExitError{Code: exitCode}
		}
		return nil
	}}
	pypiCommand.AddCommand(inspectCommand)
	return nil
}

// AddNPMInspect adds the injected npm Inspect use case to a root command.
func AddNPMInspect(root *cobra.Command, inspector Inspector) error {
	if root == nil || inspector == nil {
		return errors.New("npm inspect command requires root and use case")
	}
	npmCommand := ensureNPMCommand(root)
	inspectCommand := &cobra.Command{Use: "inspect <package-reference>", Args: cobra.ExactArgs(1), RunE: func(command *cobra.Command, args []string) error {
		reference, err := artifactnpm.ParseReference(args[0])
		if err != nil {
			return err
		}
		request, err := application.NewInspectRequest(reference)
		if err != nil {
			return err
		}
		exitCode, operationErr := ExecuteInspect(contextOrBackground(command.Context()), inspector, request, true, command.OutOrStdout())
		if operationErr != nil {
			return operationErr
		}
		if exitCode != 0 {
			return ExitError{Code: exitCode}
		}
		return nil
	}}
	npmCommand.AddCommand(inspectCommand)
	return nil
}

// AddNPMInstall adds the injected npm Install/Promotion use case.
func AddNPMInstall(root *cobra.Command, installer Installer) error {
	if root == nil || installer == nil {
		return errors.New("npm install command requires root and use case")
	}
	npmCommand := ensureNPMCommand(root)
	var target string
	installCommand := &cobra.Command{Use: "install <package-reference>", Args: cobra.ExactArgs(1), RunE: func(command *cobra.Command, args []string) error {
		reference, err := artifactnpm.ParseReference(args[0])
		if err != nil {
			return err
		}
		installContext, err := npmInstallContext(target)
		if err != nil {
			return err
		}
		request, err := application.NewInstallRequest(reference, installContext)
		if err != nil {
			return err
		}
		exitCode, operationErr := ExecuteInstall(contextOrBackground(command.Context()), installer, request, true, command.OutOrStdout())
		if operationErr != nil {
			return operationErr
		}
		if exitCode != 0 {
			return ExitError{Code: exitCode}
		}
		return nil
	}}
	installCommand.Flags().StringVar(&target, "target", "", "advanced new absolute installation target")
	npmCommand.AddCommand(installCommand)
	return nil
}

func ensureNPMCommand(root *cobra.Command) *cobra.Command {
	for _, command := range root.Commands() {
		if command.Name() == "npm" {
			return command
		}
	}
	command := &cobra.Command{Use: "npm", Short: "Inspect and install npm registry packages"}
	root.AddCommand(command)
	return command
}

func ensurePyPICommand(root *cobra.Command) *cobra.Command {
	for _, command := range root.Commands() {
		if command.Name() == "pypi" {
			return command
		}
	}
	command := &cobra.Command{Use: "pypi", Short: "Inspect and install PyPI distributions"}
	root.AddCommand(command)
	return command
}

func ensureGitHubCommand(root *cobra.Command) *cobra.Command {
	for _, command := range root.Commands() {
		if command.Name() == "github" {
			return command
		}
	}
	command := &cobra.Command{Use: "github", Short: "Inspect and install public GitHub Release assets"}
	root.AddCommand(command)
	return command
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
