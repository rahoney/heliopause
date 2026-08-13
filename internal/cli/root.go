// Package cli owns Heliopause command parsing and presentation.
package cli

import (
	"context"
	"errors"
	"io"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
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
	npmCommand := &cobra.Command{Use: "npm", Short: "Inspect npm registry packages"}
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
	root.AddCommand(npmCommand)
	return nil
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
