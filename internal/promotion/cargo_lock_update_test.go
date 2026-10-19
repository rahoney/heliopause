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

func approvedCargoLockFixture(t *testing.T, noop bool) (*CargoProjectPromotion, *approvedCargoProjectGuard, domain.ProjectDependencyUpdate, domain.StagedProjectSet) {
	t.Helper()
	p, old, update, staged := approvedCargoFixture(t)
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}
	root := update.Snapshot().Context().Target().String()
	for _, control := range update.SelectedControls() {
		body := control.Body()
		if control.Name() == "Cargo.lock" && !noop {
			body = bytes.ReplaceAll(body, []byte("1.0.17"), []byte("1.0.16"))
		}
		name := filepath.Join(root, control.Name())
		if err := os.WriteFile(name, body, 0o600); err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0o644)
		if control.Name() == "Cargo.lock" {
			mode = 0o640
		}
		if err := os.Chmod(name, mode); err != nil {
			t.Fatal(err)
		}
	}
	g, err := p.Begin(context.Background(), update.Snapshot().Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if !g.closed {
			if err := g.Close(); err != nil {
				t.Error(err)
			}
		}
	})
	complete, err := domain.NewProjectDependencyUpdateFromSnapshot(g.Controls(), update.SelectedControls(), update.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	return p, g, complete, staged
}

func TestCargoLockUpdatePreservesRetainedControlsAndLockPermissions(t *testing.T) {
	for _, noop := range []bool{false, true} {
		t.Run(map[bool]string{false: "changed-lock", true: "no-op"}[noop], func(t *testing.T) {
			_, g, update, staged := approvedCargoLockFixture(t, noop)
			manifest := filepath.Join(g.plan.root, "Cargo.toml")
			lock := filepath.Join(g.plan.root, "Cargo.lock")
			beforeManifest, err := os.Lstat(manifest)
			if err != nil {
				t.Fatal(err)
			}
			beforeLock, err := os.Lstat(lock)
			if err != nil {
				t.Fatal(err)
			}
			if err := g.Commit(context.Background(), update, staged); err != nil {
				t.Fatal(err)
			}
			afterManifest, err := os.Lstat(manifest)
			if err != nil || !os.SameFile(beforeManifest, afterManifest) || beforeManifest.Mode() != afterManifest.Mode() || beforeManifest.ModTime() != afterManifest.ModTime() {
				t.Error("lock update replaced or changed the retained manifest identity/mode/time")
			}
			afterLock, err := os.Lstat(lock)
			if err != nil || beforeLock.Mode() != afterLock.Mode() {
				t.Error("lock update changed original lock permissions")
			}
			if noop && (!os.SameFile(beforeLock, afterLock) || beforeLock.ModTime() != afterLock.ModTime()) {
				t.Error("no-op replaced or modified identical original lock")
			}
			for _, control := range update.SelectedControls() {
				body, err := os.ReadFile(filepath.Join(g.plan.root, control.Name()))
				if err != nil || !bytes.Equal(body, control.Body()) {
					t.Error("published controls differ from the inspected selection")
				}
			}
		})
	}
}

func TestExistingProjectGuardsRejectCompleteSelectionVariant(t *testing.T) {
	t.Run("Go", func(t *testing.T) {
		_, g, update, staged := approvedGoFixture(t)
		t.Cleanup(func() {
			if !g.closed {
				if err := g.Close(); err != nil {
					t.Error(err)
				}
			}
		})
		complete, err := domain.NewProjectDependencyUpdateFromSnapshot(update.OriginalControls(), update.SelectedControls(), update.Snapshot())
		if err != nil {
			t.Fatal(err)
		}
		if g.Commit(context.Background(), complete, staged) == nil {
			t.Fatal("existing Go get accepted a selection without its requested primary")
		}
	})
	t.Run("Terraform", func(t *testing.T) {
		p, install := newTerraformPromotionFixture(t)
		g, err := p.Begin(context.Background(), install)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if !g.closed {
				if err := g.Close(); err != nil {
					t.Error(err)
				}
			}
		})
		update, set, _ := terraformApprovedFixture(t, p, g)
		staged, err := p.cache.StageProject(context.Background(), set)
		if err != nil {
			t.Fatal(err)
		}
		complete, err := domain.NewProjectDependencyUpdateFromSnapshot(update.OriginalControls(), update.SelectedControls(), update.Snapshot())
		if err != nil {
			t.Fatal(err)
		}
		if g.Commit(context.Background(), complete, staged) == nil {
			t.Fatal("existing Terraform init accepted a selection without its requested primary")
		}
	})
}

func TestCargoLockUpdateRollbackPreservesOriginalsAndCause(t *testing.T) {
	for _, noop := range []bool{false, true} {
		for _, phase := range []string{"before-backup", "before-publication", "before-approval", "after-approval"} {
			t.Run(phase+map[bool]string{false: "/changed", true: "/noop"}[noop], func(t *testing.T) {
				p, g, update, staged := approvedCargoLockFixture(t, noop)
				primary := errors.New("injected lock-update transaction fault")
				p.checkpoint = func(current string) error {
					if current == phase {
						return primary
					}
					return nil
				}
				if err := g.Commit(context.Background(), update, staged); !errors.Is(err, primary) {
					t.Fatalf("primary cause lost: %v", err)
				}
				for _, control := range g.Controls() {
					body, info, err := readGoTransactionControl(g.guard.root, control.Name())
					original := g.plan.members[control.Name()]
					if err != nil || !bytes.Equal(body, control.Body()) || !os.SameFile(original, info) || original.Mode() != info.Mode() || !original.ModTime().Equal(info.ModTime()) {
						t.Fatal("rollback failed to retain exact original controls")
					}
				}
				if _, err := os.Lstat(filepath.Join(p.stateRoot, g.stateName)); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("failed update retained new approval")
				}
				if err := g.checkRecoveryState(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestCargoLockUpdateRejectsRetainedControlReplacement(t *testing.T) {
	for _, phase := range []string{"before-publication", "before-approval", "after-approval"} {
		t.Run(phase, func(t *testing.T) {
			p, g, update, staged := approvedCargoLockFixture(t, false)
			var foreign os.FileInfo
			p.checkpoint = func(current string) error {
				if current != phase {
					return nil
				}
				name := filepath.Join(g.plan.root, "Cargo.toml")
				body, err := os.ReadFile(name)
				if err != nil {
					return err
				}
				temp := filepath.Join(g.plan.root, ".replacement")
				if err := os.WriteFile(temp, body, 0o644); err != nil {
					return err
				}
				if err := os.Rename(temp, name); err != nil {
					return err
				}
				foreign, err = os.Lstat(name)
				return err
			}
			if err := g.Commit(context.Background(), update, staged); err == nil {
				t.Fatal("same-byte foreign identity authorized update")
			}
			current, err := os.Lstat(filepath.Join(g.plan.root, "Cargo.toml"))
			if err != nil || foreign == nil || !os.SameFile(foreign, current) {
				t.Fatal("rollback clobbered foreign retained control")
			}
		})
	}
}

func TestCargoLockUpdateEnrollmentRequiresExplicitIntentAndIndependentProof(t *testing.T) {
	for _, kind := range []string{"native", "managed", "changed-control", "foreign-marker", "corrupt-state", "corrupt-cache", "missing-evidence", "missing-lock", "recovery"} {
		for _, adopt := range []bool{false, true} {
			t.Run(kind+map[bool]string{false: "/implicit", true: "/adopt"}[adopt], func(t *testing.T) {
				p, g, update, staged := approvedCargoLockFixture(t, true)
				managed := kind == "managed" || kind == "changed-control" || kind == "corrupt-state" || kind == "corrupt-cache" || kind == "missing-evidence"
				if managed {
					if err := g.Commit(context.Background(), update, staged); err != nil {
						t.Fatal(err)
					}
				}
				if err := g.Close(); err != nil {
					t.Fatal(err)
				}
				root := g.plan.root
				write := func(name string, body []byte) {
					t.Helper()
					if err := os.WriteFile(name, body, 0o600); err != nil {
						t.Fatal(err)
					}
				}
				switch kind {
				case "changed-control":
					name := filepath.Join(root, "Cargo.toml")
					body, err := os.ReadFile(name)
					if err != nil {
						t.Fatal(err)
					}
					write(name, append(body, []byte("\n# user edit\n")...))
				case "foreign-marker":
					if err := os.MkdirAll(filepath.Join(root, ".heliopause"), 0o700); err != nil {
						t.Fatal(err)
					}
					write(filepath.Join(root, cargoTransactionMetadata), []byte("{}"))
				case "corrupt-state":
					write(filepath.Join(p.stateRoot, g.stateName), []byte("{}"))
				case "corrupt-cache":
					vendor, err := p.cache.OpenProjectCache(context.Background(), staged)
					if err != nil {
						t.Fatal(err)
					}
					write(filepath.Join(vendor, "foreign"), []byte("tampered"))
				case "missing-evidence":
					if err := os.RemoveAll(p.cache.evidenceRoot); err != nil {
						t.Fatal(err)
					}
				case "missing-lock":
					if err := os.Remove(filepath.Join(root, "Cargo.lock")); err != nil {
						t.Fatal(err)
					}
				case "recovery":
					if err := os.Mkdir(filepath.Join(root, ".heliopause-cargo-commit-incomplete"), 0o700); err != nil {
						t.Fatal(err)
					}
				}
				next, err := p.BeginLockUpdate(context.Background(), g.context, adopt)
				want := kind == "managed" || (adopt && (kind == "native" || kind == "changed-control"))
				if (err == nil) != want || (next != nil) != want {
					if next != nil {
						_ = next.Close()
					}
					t.Fatalf("enrollment=%v want success=%v", err, want)
				}
				if next != nil {
					if err := next.Close(); err != nil {
						t.Fatal(err)
					}
				}
			})
		}
	}
}
