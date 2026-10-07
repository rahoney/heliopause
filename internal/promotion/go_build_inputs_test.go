package promotion

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGoBuildInputsRequireLiveGuardAndRecheckedRetainedApproval(t *testing.T) {
	for _, kind := range []string{"normal", "unmanaged", "closed", "controls", "cache", "evidence", "state"} {
		t.Run(kind, func(t *testing.T) {
			p, guard, update, staged := approvedGoFixture(t)
			if kind == "unmanaged" {
				defer guard.Close()
				if snapshot, cache, err := guard.OpenBuildInputs(context.Background()); err == nil || snapshot.Valid() || cache != "" {
					t.Fatal("unmanaged input restored build approval")
				}
				return
			}
			if err := guard.Commit(context.Background(), update, staged); err != nil {
				t.Fatal(err)
			}
			if err := guard.Close(); err != nil {
				t.Fatal(err)
			}
			guard, err := p.Begin(context.Background(), update.Snapshot().Context())
			if err != nil {
				t.Fatal(err)
			}
			defer guard.Close()
			switch kind {
			case "closed":
				if err := guard.Close(); err != nil {
					t.Fatal(err)
				}
			case "controls":
				if err := os.WriteFile(filepath.Join(guard.plan.root, "go.mod"), []byte("module example.com/changed\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "cache":
				root, err := p.cache.OpenProjectCache(context.Background(), staged)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "undeclared"), []byte("poison"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "evidence":
				if err := os.RemoveAll(p.cache.evidenceRoot); err != nil {
					t.Fatal(err)
				}
			case "state":
				if err := os.WriteFile(filepath.Join(p.stateRoot, guard.stateName), []byte("{}"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			snapshot, root, err := guard.OpenBuildInputs(context.Background())
			if kind == "normal" {
				if err != nil || !snapshot.Valid() || root == "" || snapshot.GraphDigest() != update.Snapshot().GraphDigest() || snapshot.Context() != update.Snapshot().Context() {
					t.Fatalf("approved build inputs: %v", err)
				}
				want, got := update.Snapshot().Dependencies(), snapshot.Dependencies()
				if len(got) != len(want) {
					t.Fatal("build dependency coverage changed")
				}
				for i := range want {
					if got[i] != want[i] {
						t.Fatal("build dependency identity or integrity changed")
					}
				}
			} else if err == nil || snapshot.Valid() || root != "" {
				t.Fatalf("changed %s restored build approval", kind)
			}
		})
	}
}
