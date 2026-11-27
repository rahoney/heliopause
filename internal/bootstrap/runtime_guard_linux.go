//go:build linux

package bootstrap

import (
	"context"
	"io"

	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/releaseactivation"
	"github.com/rahoney/heliopause/internal/releaseinstall"
)

type commandRuntimeKey struct{}

func commandRuntime(ctx context.Context) (context.Context, func() error, error) {
	lease, err := releaseactivation.AcquireRuntimeLease(ctx)
	if err != nil {
		return ctx, nil, err
	}
	if lease == nil {
		return ctx, nil, nil
	}
	return context.WithValue(ctx, commandRuntimeKey{}, lease), lease.Close, nil
}

func installationAndHostChecks(ctx context.Context, root string) (checks []cli.DoctorCheck) {
	lease, err := releaseactivation.AcquireRuntimeLease(ctx)
	if err != nil {
		return []cli.DoctorCheck{{Name: "release-installation", Detail: "SELF_UPDATE_PROTECTED_RUNTIME_UNAVAILABLE"}}
	}
	if lease != nil {
		root = releaseinstall.SystemInstallRoot
		defer func() {
			if err := lease.Close(); err != nil {
				checks = append(checks, cli.DoctorCheck{Name: "release-installation", Detail: "SELF_UPDATE_RUNTIME_LEASE_CLOSE_FAILED"})
			}
		}()
	}
	installation := releaseinstall.CheckInstallation(root)
	return append([]cli.DoctorCheck{{Name: installation.Name, Healthy: installation.Healthy, Detail: installation.Detail}}, doctorHostChecks(ctx)...)
}

// TryProtectedHealth handles only the fixed inherited-channel operation.
// No user-selected command, target or privilege bypass is accepted.
func TryProtectedHealth(ctx context.Context, args []string) (bool, error) {
	if len(args) != 1 || args[0] != releaseactivation.ProtectedHealthArgument {
		return false, nil
	}
	return true, releaseactivation.RunProtectedHealth(ctx, func(healthContext context.Context, full bool) error {
		return checkProtectedCompatibilityHealth(healthContext, full, (doctorService{}).Diagnose, Run)
	})
}

func checkProtectedCompatibilityHealth(ctx context.Context, full bool, diagnose func(context.Context) cli.DoctorReport, run func(context.Context, []string, io.Writer, io.Writer) error) error {
	if !diagnose(ctx).Healthy {
		return releaseactivation.ErrHealth
	}
	if !full {
		return nil
	}
	return run(ctx, []string{"npm", "inspect", "is-number@7.0.0", "--output", "json"}, io.Discard, io.Discard)
}
