//go:build realcorpus

package npm

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// Independent HTTPS metadata only. The input bytes are the frozen native
// samples; no native tool, archive, lifecycle, Policy or Host install runs here.
func TestBunPublicLockSourceBinding(t *testing.T) {
	if os.Getenv("HELOX_BUN_PUBLIC_SOURCE") != "1" {
		t.Fatal("declare HELOX_BUN_PUBLIC_SOURCE=1 for public metadata qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	r, err := NewPublicResolver(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(r.client.CloseIdleConnections)
	for _, name := range []string{"empty", "single", "transitive", "peer"} {
		if !t.Run(name, func(t *testing.T) {
			manifest, body, present := bunActualControls(t, name)
			lock, err := parseBunTestLock(t, manifest, body, present)
			if err != nil {
				t.Fatal(err)
			}
			selected, err := r.BindBunLock(ctx, lock)
			if err != nil {
				t.Fatal("independent public source binding failed", err)
			}
			controls := []domain.ProjectControlFile{bunTestControl(t, "package.json", manifest, true), bunTestControl(t, "bun.lock", body, present)}
			snapshot, err := selected.ProjectSnapshot(pnpmSnapshotContext(t, "/tmp/haa-bun-public-snapshot", true), controls, "fixture-runtime:frozen-bun-1.4.4")
			if err != nil || !snapshot.Valid() || snapshot.DependencyFree() != (name == "empty") {
				t.Fatal("source/snapshot binding failed", err)
			}
			t.Logf("bun_public_source=%s unique_archives=%d instances=%d roots=%d exact_controls=true; native_runtime/archive/Policy/Host_install=NOT_EXERCISED", name, len(selected.Artifacts()), len(selected.Lock().Packages()), len(selected.Lock().Roots()))
		}) {
			return
		}
	}
}
