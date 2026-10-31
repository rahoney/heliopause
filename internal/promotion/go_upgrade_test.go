package promotion

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestGoUpgradeRetainsIdenticalControlsAndChangedPermissions(t *testing.T) {
	for _, changed := range []bool{false, true} {
		t.Run(map[bool]string{false: "noop", true: "changed"}[changed], func(t *testing.T) {
			root := writeManagedGoProject(t)
			before := map[string]os.FileInfo{}
			for _, name := range []string{"go.mod", "go.sum"} {
				if err := os.Chmod(filepath.Join(root, name), 0o640); err != nil {
					t.Fatal(err)
				}
				before[name], _ = os.Stat(filepath.Join(root, name))
			}
			plan, err := freezeGoProject(root)
			if err != nil {
				t.Fatal(err)
			}
			plan.authorized, plan.upgrade = true, true
			workspace, err := plan.privateWorkspace()
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(workspace)
			if changed {
				if err := os.WriteFile(filepath.Join(workspace, "go.sum"), []byte("changed sum\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			transaction, err := beginGoProjectTransaction(plan, workspace)
			if err != nil {
				t.Fatal(err)
			}
			if err := transaction.commit(); err != nil {
				t.Fatal(err)
			}
			for name, original := range before {
				after, err := os.Stat(filepath.Join(root, name))
				if err != nil || after.Mode() != original.Mode() {
					t.Fatalf("%s mode changed: %v", name, err)
				}
				if (!changed || name == "go.mod") && (!os.SameFile(original, after) || original.ModTime() != after.ModTime()) {
					t.Fatalf("identical %s was replaced", name)
				}
			}
		})
	}
}

func TestGoUpgradeNativeEnrollmentIsExplicitAndRejectsForeignState(t *testing.T) {
	for _, test := range []struct {
		name         string
		adopt, allow bool
	}{
		{"implicit", false, false}, {"explicit", true, true}, {"foreign-marker", true, false}, {"recovery", true, false}, {"aliased-control", true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := canonicalGoTestRoot(t)
			project := filepath.Join(root, "project")
			if err := os.Mkdir(project, 0o700); err != nil {
				t.Fatal(err)
			}
			mod := []byte("module example.com/app\ngo 1.26.0\nrequire example.com/module v1.0.0\n")
			if err := os.WriteFile(filepath.Join(project, "go.mod"), mod, 0o640); err != nil {
				t.Fatal(err)
			}
			cache, err := newGoCacheForTest(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache"))
			if err != nil {
				t.Fatal(err)
			}
			p, err := NewGoProjectPromotion(cache, filepath.Join(root, "state"))
			if err != nil {
				t.Fatal(err)
			}
			switch test.name {
			case "foreign-marker":
				if err := os.Mkdir(filepath.Join(project, ".heliopause"), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(project, goTransactionMetadata), []byte("{}"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "recovery":
				if err := os.Mkdir(filepath.Join(project, ".heliopause-go-commit-foreign"), 0o700); err != nil {
					t.Fatal(err)
				}
			case "aliased-control":
				if err := os.Link(filepath.Join(project, "go.mod"), filepath.Join(project, "alias")); err != nil {
					t.Fatal(err)
				}
			}
			target, _ := domain.NewInstallTarget(project)
			install, _ := domain.NewInstallContext(target)
			g, err := p.BeginUpgrade(context.Background(), install, test.adopt)
			if (err == nil) != test.allow {
				t.Fatalf("enrollment=%v", err)
			}
			if g != nil {
				if err := g.Close(); err != nil {
					t.Fatal(err)
				}
			}
			got, err := os.ReadFile(filepath.Join(project, "go.mod"))
			if err != nil || !bytes.Equal(got, mod) {
				t.Fatal("enrollment changed original")
			}
			if _, err := os.Lstat(filepath.Join(project, ".heliopause-go-transaction.lock")); !os.IsNotExist(err) {
				t.Fatal("guard leaked")
			}
		})
	}
}

func TestGoUpgradeChangedInputsStillRequireIndependentOldApproval(t *testing.T) {
	for _, kind := range []string{"matching", "changed-implicit", "changed-explicit", "cache", "evidence", "corrupt-state", "aliased-state"} {
		t.Run(kind, func(t *testing.T) {
			p, seed, update, staged := approvedGoFixture(t)
			ctx := context.Background()
			if err := seed.Commit(ctx, update, staged); err != nil {
				t.Fatal(err)
			}
			if err := seed.Close(); err != nil {
				t.Fatal(err)
			}
			project := update.Snapshot().Context().Target().String()
			if kind != "matching" {
				path := filepath.Join(project, "go.mod")
				body, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, append(body, []byte("// user edit\n")...), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			switch kind {
			case "cache":
				dir, err := p.cache.OpenProjectCache(ctx, staged)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "foreign"), []byte("bad"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "evidence":
				if err := os.RemoveAll(p.cache.evidenceRoot); err != nil {
					t.Fatal(err)
				}
			case "corrupt-state":
				if err := os.WriteFile(filepath.Join(p.stateRoot, seed.stateName), []byte("{}"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "aliased-state":
				if err := os.Link(filepath.Join(p.stateRoot, seed.stateName), filepath.Join(p.stateRoot, "alias")); err != nil {
					t.Fatal(err)
				}
			}
			g, err := p.BeginUpgrade(ctx, update.Snapshot().Context(), kind != "changed-implicit")
			allow := kind == "matching" || kind == "changed-explicit"
			if (err == nil) != allow {
				t.Fatalf("retained input=%v", err)
			}
			if g != nil {
				if err := g.Close(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestGoUpgradeFaultsRollbackExactOwnedChanges(t *testing.T) {
	for _, phase := range []string{"before-backup", "before-publication", "before-approval", "after-approval"} {
		t.Run(phase, func(t *testing.T) {
			root := writeManagedGoProject(t)
			plan, err := freezeGoProject(root)
			if err != nil {
				t.Fatal(err)
			}
			plan.authorized, plan.upgrade = true, true
			workspace, err := plan.privateWorkspace()
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(workspace)
			before := map[string][]byte{}
			for _, name := range []string{"go.mod", "go.sum", goTransactionMetadata} {
				before[name], err = os.ReadFile(filepath.Join(root, name))
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(workspace, "go.sum"), []byte("new sum\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			tx, err := beginGoProjectTransaction(plan, workspace)
			if err != nil {
				t.Fatal(err)
			}
			fault := errors.New("owned phase failure")
			tx.checkpoint = func(at string) error {
				if at == phase {
					return fault
				}
				return nil
			}
			if err := tx.commit(); !errors.Is(err, fault) {
				t.Fatalf("primary cause lost: %v", err)
			}
			for name, want := range before {
				got, err := os.ReadFile(filepath.Join(root, name))
				if err != nil || !bytes.Equal(got, want) {
					t.Fatalf("rollback changed %s: %v", name, err)
				}
			}
			if _, err := os.Lstat(tx.backup); !os.IsNotExist(err) {
				t.Fatal("successful rollback left journal")
			}
		})
	}
}

func TestGoUpgradeRetainedDriftAndRollbackUncertaintyStayFailed(t *testing.T) {
	for _, kind := range []string{"retained-inode", "retained-mode", "selected-content", "published-foreign"} {
		t.Run(kind, func(t *testing.T) {
			root := writeManagedGoProject(t)
			plan, err := freezeGoProject(root)
			if err != nil {
				t.Fatal(err)
			}
			plan.authorized, plan.upgrade = true, true
			workspace, err := plan.privateWorkspace()
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(workspace)
			if kind == "published-foreign" {
				if err := os.WriteFile(filepath.Join(workspace, "go.sum"), []byte("new sum\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			tx, err := beginGoProjectTransaction(plan, workspace)
			if err != nil {
				t.Fatal(err)
			}
			fault := errors.New("foreign publication fault")
			tx.checkpoint = func(phase string) error {
				if phase == "before-publication" {
					switch kind {
					case "retained-inode":
						body, err := os.ReadFile(filepath.Join(root, "go.mod"))
						if err != nil {
							return err
						}
						if err := os.WriteFile(filepath.Join(root, "foreign"), body, 0o600); err != nil {
							return err
						}
						return os.Rename(filepath.Join(root, "foreign"), filepath.Join(root, "go.mod"))
					case "retained-mode":
						return os.Chmod(filepath.Join(root, "go.mod"), 0o644)
					case "selected-content":
						return os.WriteFile(filepath.Join(tx.backup, "selected/go.sum"), []byte("poison\n"), 0o600)
					}
				}
				if phase == "before-approval" && kind == "published-foreign" {
					if err := os.WriteFile(filepath.Join(root, "foreign"), []byte("foreign retained bytes"), 0o600); err != nil {
						return err
					}
					if err := os.Rename(filepath.Join(root, "foreign"), filepath.Join(root, "go.sum")); err != nil {
						return err
					}
					return fault
				}
				return nil
			}
			err = tx.commit()
			if err == nil {
				t.Fatal("foreign control drift committed")
			}
			if kind == "published-foreign" {
				if !errors.Is(err, fault) {
					t.Fatal("primary cause lost", err)
				}
				got, readErr := os.ReadFile(filepath.Join(root, "go.sum"))
				if readErr != nil || string(got) != "foreign retained bytes" {
					t.Fatal("foreign output overwritten")
				}
				if _, err := os.Stat(tx.backup); err != nil {
					t.Fatal("uncertain recovery backup deleted")
				}
			}
		})
	}
}
