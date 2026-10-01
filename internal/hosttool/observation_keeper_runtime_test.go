//go:build linux && haa_local_runtime

package hosttool

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// Run explicitly as root against the pinned local Docker/systemd installation.
// It is intentionally excluded from ordinary unit tests; absence of the local
// runtime must never turn an integration check into a skipped success.
func TestObservationKeeperPinnedLocalRuntime(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Fatal("pinned observation keeper integration requires root")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	config, err := loadSystemConfig()
	if err != nil {
		t.Fatal(err)
	}
	executor, err := newExecutor(ctx, config, true)
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close()
	id, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	parent, err := observationParent(id.String())
	if err != nil {
		t.Fatal(err)
	}
	k := systemdObservationKeeper{executor: executor}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_ = executor.RunDiscard(cleanupCtx, "systemctl", "stop", observationKeeperService(parent), parent)
		_ = executor.RunDiscard(cleanupCtx, "systemctl", "revert", parent)
	})
	path, scope, baseline, err := k.Start(ctx, parent)
	if err != nil {
		t.Fatal(err)
	}
	if err := k.Verify(ctx, parent, scope); err != nil {
		t.Fatal(err)
	}
	usage, err := k.Close(ctx, parent, scope)
	if err != nil || usage < baseline {
		t.Fatalf("final CPU accounting = %d, baseline = %d, error = %v", usage, baseline, err)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("transaction parent remains after close: %v", err)
	}
}
