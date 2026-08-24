//go:build linux

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rahoney/heliopause/internal/hosttool"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := hosttool.ServeNetworkPolicy(ctx); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "haa-network-policy-helper: %v\n", err)
		os.Exit(1)
	}
}
