//go:build linux

package main

import (
	"context"

	"github.com/rahoney/heliopause/internal/releaseactivation"
)

func runOperation(ctx context.Context, operation string) error {
	if operation == "bootstrap-files" {
		return releaseactivation.BootstrapFiles(ctx)
	}
	if operation == "register" {
		return releaseactivation.Register(ctx)
	}
	if operation == "recover" {
		return releaseactivation.Recover(ctx)
	}
	return releaseactivation.Serve(ctx)
}
