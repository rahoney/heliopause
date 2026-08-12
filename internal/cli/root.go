// Package cli owns Heliopause command parsing and presentation.
package cli

import (
	"errors"
	"io"

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
