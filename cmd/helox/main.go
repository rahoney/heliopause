package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/rahoney/heliopause/internal/bootstrap"
)

const (
	exitSuccess = 0
	exitFailure = 1
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if err := bootstrap.Run(ctx, args, stdout, stderr); err != nil {
		var exitError interface{ ExitCode() int }
		if errors.As(err, &exitError) {
			return exitError.ExitCode()
		}
		if _, writeErr := fmt.Fprintf(stderr, "helox: %v\n", err); writeErr != nil {
			return exitFailure
		}
		return exitFailure
	}

	return exitSuccess
}
