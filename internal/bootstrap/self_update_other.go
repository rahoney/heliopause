//go:build !linux

package bootstrap

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func selfUpdatePlatform() error {
	return ports.NewInstallUpdateFailure("SELF_UPDATE_PLATFORM_UNSUPPORTED", errors.New("native quarantine update requires Linux amd64"))
}
func selfUpdateFailure(err error) error {
	return ports.NewInstallUpdateFailure("SELF_UPDATE_PLATFORM_UNSUPPORTED", err)
}
func selfUpdateRuntime(context.Context) error { return selfUpdatePlatform() }
func newSelfUpdater(ctx context.Context, _ *hosttool.Executor, _ *sandbox.ObserverSupervisor, _ sandbox.TraceObserver) (cli.Installer, error) {
	return nil, selfUpdateRuntime(ctx)
}
