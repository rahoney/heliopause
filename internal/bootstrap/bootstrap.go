// Package bootstrap owns the Heliopause composition root.
package bootstrap

import (
	"context"
	"io"

	"github.com/rahoney/heliopause/internal/cli"
)

// Run constructs and executes the CLI with process-owned dependencies.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	command, err := cli.New(stdout, stderr)
	if err != nil {
		return err
	}
	command.SetArgs(args)

	return command.ExecuteContext(ctx)
}
