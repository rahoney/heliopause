//go:build realcorpus

package npm

import (
	"context"
	"os"
	"testing"
	"time"
)

// This exercises only independent HTTPS registry metadata binding. The native
// lock fixtures are synthetic shape observations, not public-runtime evidence.
func TestPNPMPublicLockSourceBinding(t *testing.T) {
	if os.Getenv("HELOX_PNPM_PUBLIC_SOURCE") != "1" {
		t.Fatal("declare HELOX_PNPM_PUBLIC_SOURCE=1 for public metadata qualification")
	}
	for _, item := range []struct{ fixture, manifest string }{
		{"single", pnpmSingleManifest}, {"peer", pnpmPeerManifest},
	} {
		t.Run(item.fixture, func(t *testing.T) {
			lock, err := parsePNPMTestLock([]byte(item.manifest), pnpmFixture(t, item.fixture))
			if err != nil {
				t.Fatal(err)
			}
			resolver, err := NewPublicResolver(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			selection, err := resolver.BindPNPMLock(ctx, lock)
			if err != nil || !selection.SourceBound() || len(selection.Artifacts()) != len(lock.Packages()) {
				t.Fatal("independent public PNPM metadata binding failed", err)
			}
			t.Logf("public_metadata_binding artifacts=%d instances=%d roots=%d; native runtime/archives/Policy not exercised", len(selection.Artifacts()), len(selection.Lock().Snapshots()), len(selection.Lock().Roots()))
		})
	}
}
