//go:build !linux

package bootstrap

import (
	"context"

	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/releaseinstall"
)

func commandRuntime(ctx context.Context) (context.Context, func() error, error) { return ctx, nil, nil }

func installationAndHostChecks(ctx context.Context, root string) []cli.DoctorCheck {
	installation := releaseinstall.CheckInstallation(root)
	return append([]cli.DoctorCheck{{Name: installation.Name, Healthy: installation.Healthy, Detail: installation.Detail}}, doctorHostChecks(ctx)...)
}

func TryProtectedHealth(context.Context, []string) (bool, error) { return false, nil }
