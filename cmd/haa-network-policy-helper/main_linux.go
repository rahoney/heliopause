//go:build linux

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/rahoney/heliopause/internal/hosttool"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "serve" {
		os.Clearenv()
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || len(args) == 1 && (args[0] == "--help" || args[0] == "help") {
		if _, err := fmt.Fprintln(stdout, "Heliopause network-policy helper\nusage: haa-network-policy-helper serve"); err != nil {
			return 1
		}
		return 0
	}
	if len(args) == 1 && args[0] == "--version" {
		if _, err := fmt.Fprintf(stdout, "haa-network-policy-helper go=%s\n", runtime.Version()); err != nil {
			return 1
		}
		return 0
	}
	if len(args) != 1 || args[0] != "serve" || ctx == nil {
		_, _ = fmt.Fprintln(stderr, "haa-network-policy-helper: unsupported operation")
		return 1
	}
	if err := ctx.Err(); err != nil {
		return 1
	}
	if err := hosttool.ServeNetworkPolicy(ctx); err != nil {
		_, _ = fmt.Fprintf(stderr, "haa-network-policy-helper: %v\n", err)
		return 1
	}
	return 0
}
