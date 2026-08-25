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
	initializeCommandTree(command)

	return command, nil
}

type ReferenceParser func(string) (domain.ArtifactReference, error)

func AddGitHubReleaseInspect(root *cobra.Command, parser ReferenceParser, inspector Inspector) error {
	if root == nil || parser == nil || inspector == nil {
		return errors.New("github inspect command requires root and use case")
	}
	if command := findLeaf(root, "github", "inspect"); command != nil {
		command.RunE = func(command *cobra.Command, args []string) error {
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
		}
		return nil
	}
	return errors.New("github inspect command is not registered")
}

func AddGitHubReleaseInstall(root *cobra.Command, parser ReferenceParser, installer Installer) error {
	if root == nil || parser == nil || installer == nil {
		return errors.New("github install command requires root and use case")
	}
	if install := findLeaf(root, "github", "install"); install != nil {
		install.RunE = func(command *cobra.Command, args []string) error {
			reference, err := parser(args[0])
			if err != nil {
				return err
			}
			target, err := command.Flags().GetString("target")
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
		}
		return nil
	}
	return errors.New("github install command is not registered")
}

// AddPyPIInstall adds the injected PyPI Install/Promotion use case. The
// generic JSON/human result contract is intentionally shared with npm.
func AddPyPIInstall(root *cobra.Command, installer Installer) error {
	if root == nil || installer == nil {
		return errors.New("pypi install command requires root and use case")
	}
	if installCommand := findLeaf(root, "pip", "install"); installCommand != nil {
		installCommand.RunE = func(command *cobra.Command, args []string) error {
			reference, err := artifactpypi.ParseReference(args[0])
			if err != nil {
				return err
			}
			target, err := command.Flags().GetString("target")
			if err != nil {
				return err
			}
			installTarget, err := activePythonVenv(target)
			if err != nil {
				return err
			}
			installContext, err := domain.NewPythonVenvInstallContext(installTarget)
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
		}
		return nil
	}
	return errors.New("pip install command is not registered")
}

func activePythonVenv(target string) (domain.InstallTarget, error) {
	if target == "" {
		target = os.Getenv("VIRTUAL_ENV")
	}
	if target == "" {
		return domain.InstallTarget{}, errors.New("pip install requires an active virtual environment")
	}
	value, err := domain.NewInstallTarget(target)
	if err != nil {
		return domain.InstallTarget{}, errors.New("active virtual environment is unsupported")
	}
	return value, nil
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
	if inspectCommand := findLeaf(root, "pypi", "inspect"); inspectCommand != nil {
		inspectCommand.RunE = func(command *cobra.Command, args []string) error {
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
		}
		return nil
	}
	return errors.New("pypi inspect command is not registered")
}

// AddNPMInspect adds the injected npm Inspect use case to a root command.
func AddNPMInspect(root *cobra.Command, inspector Inspector) error {
	if root == nil || inspector == nil {
		return errors.New("npm inspect command requires root and use case")
	}
	if inspectCommand := findLeaf(root, "npm", "inspect"); inspectCommand != nil {
		inspectCommand.RunE = func(command *cobra.Command, args []string) error {
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
		}
		return nil
	}
	return errors.New("npm inspect command is not registered")
}

// AddNPMInstall adds the injected npm Install/Promotion use case.
func AddNPMInstall(root *cobra.Command, installer Installer) error {
	if root == nil || installer == nil {
		return errors.New("npm install command requires root and use case")
	}
	if installCommand := findLeaf(root, "npm", "install"); installCommand != nil {
		installCommand.RunE = func(command *cobra.Command, args []string) error {
			reference, err := artifactnpm.ParseReference(args[0])
			if err != nil {
				return err
			}
			target, err := command.Flags().GetString("target")
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
		}
		return nil
	}
	return errors.New("npm install command is not registered")
}

func initializeCommandTree(root *cobra.Command) {
	npm := ensureNPMCommand(root)
	npm.AddCommand(newStaticLeaf("inspect <package-reference>", "Inspect an npm package", false))
	npm.AddCommand(newStaticLeaf("install <package-reference>", "Install an npm package into the current project", true))
	pypi := ensurePyPICommand(root)
	pypi.AddCommand(newStaticLeaf("inspect <project>[@<version>]", "Inspect a PyPI distribution", false))
	pip := ensurePipCommand(root)
	pip.AddCommand(newStaticLeaf("install <project>[@<version>]", "Install a PyPI distribution into the active virtual environment", true))
	github := ensureGitHubCommand(root)
	github.AddCommand(newStaticLeaf("inspect <owner>/<repo>@<tag>#<asset>", "Inspect a GitHub Release asset", false))
	githubInstall := newStaticLeaf("install <owner>/<repo>@<tag>#<asset>", "Install a GitHub Release asset", true)
	_ = githubInstall.MarkFlagRequired("target")
	github.AddCommand(githubInstall)
}

func newStaticLeaf(use, short string, withTarget bool) *cobra.Command {
	command := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(*cobra.Command, []string) error {
			return errors.New("command is not configured")
		},
	}
	if withTarget {
		command.Flags().String("target", "", "advanced absolute destination (optional unless required by this command)")
	}
	return command
}

func findLeaf(root *cobra.Command, parent, leaf string) *cobra.Command {
	for _, command := range root.Commands() {
		if command.Name() != parent {
			continue
		}
		for _, child := range command.Commands() {
			if child.Name() == leaf {
				return child
			}
		}
	}
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

func ensurePipCommand(root *cobra.Command) *cobra.Command {
	for _, command := range root.Commands() {
		if command.Name() == "pip" {
			return command
		}
	}
	command := &cobra.Command{Use: "pip", Short: "Inspect and install PyPI distributions"}
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
