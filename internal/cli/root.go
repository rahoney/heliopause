// Package cli owns Heliopause command parsing and presentation.
package cli

import (
	"context"
	"errors"
	"io"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
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

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
