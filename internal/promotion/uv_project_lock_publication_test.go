//go:build linux && amd64

package promotion

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func uvLockPublicationFixture(t *testing.T) (*UVProjectLockPromotion, domain.InstallContext, []byte, os.FileInfo) {
	t.Helper()
	f := newUVProjectPublicationFixture(t)
	body, err := os.ReadFile("../artifact/pypi/testdata/uv-0.13.4-empty-project.uv.lock")
	if err != nil {
		t.Fatal(err)
	}
	body = append([]byte("# original lock\n"), body...)
	path := filepath.Join(f.root, "uv.lock")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(f.root, ".venv"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, ".venv", "foreign-user-data"), []byte("not an installed venv"), 0o600); err != nil {
		t.Fatal(err)
	}
	target, err := domain.NewInstallTarget(f.root)
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewPythonProjectControlContext(target)
	if err != nil {
		t.Fatal(err)
	}
	p, err := NewUVProjectLockPromotion(f.root, f.resolver, f.publication.cache)
	if err != nil {
		t.Fatal(err)
	}
	return p, install, body, before
}

func TestUVProjectLockPublicationOnlyChangesApprovedNativeLock(t *testing.T) {
	p, install, original, _ := uvLockPublicationFixture(t)
	manifest, err := snapshotBoundedPromotionFile(filepath.Join(p.root, "pyproject.toml"), 4<<20)
	if err != nil {
		t.Fatal(err)
	}
	guard, err := p.Begin(context.Background(), install)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := guard.Close(); err != nil {
			t.Error(err)
		}
	})
	if _, err := acquireProjectTransactionGuard(p.root, uvProjectTransactionLock); err == nil {
		t.Fatal("installed project guard did not exclude lock-only mutation")
	}
	update, err := p.ResolveProjectLockUpdate(context.Background(), install, guard.Controls(), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.ResolveProjectLockUpdate(context.Background(), install, guard.Controls(), ""); err == nil {
		t.Fatal("native lock selected twice")
	}
	if _, _, _, ok := p.PublishedRecords(); ok {
		t.Fatal("records exposed before publication")
	}
	if err := guard.CommitSnapshot(context.Background(), update.Snapshot(), domain.StagedProjectSet{}); err == nil {
		t.Fatal("snapshot replaced exact native update")
	}
	staged, err := p.StageProject(context.Background(), uvProjectSyntheticApproval(t, update.Snapshot()))
	if err != nil {
		t.Fatal(err)
	}
	if err := guard.Commit(context.Background(), update, staged); err != nil {
		t.Fatal(err)
	}
	if _, _, _, ok := p.PublishedRecords(); !ok {
		t.Fatal("actual lock publication records missing")
	}
	current, err := os.ReadFile(filepath.Join(p.root, "uv.lock"))
	if err != nil || bytes.Equal(current, original) {
		t.Fatal("selected native lock not published", err)
	}
	if !manifest.matches(filepath.Join(p.root, "pyproject.toml")) {
		t.Fatal("control-only publication changed manifest identity or bytes")
	}
	if info, err := os.Lstat(filepath.Join(p.root, "uv.lock")); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatal("native lock mode changed", err)
	}
	if body, err := os.ReadFile(filepath.Join(p.root, ".venv", "foreign-user-data")); err != nil || string(body) != "not an installed venv" {
		t.Fatal("control-only update inspected or changed arbitrary venv", err)
	}
	if err := guard.Commit(context.Background(), update, staged); err == nil {
		t.Fatal("native lock published twice")
	}
}

func TestUVProjectLockPreflightRejectsUnheldOrUnsafeOriginals(t *testing.T) {
	for _, mode := range []string{"venv context", "canceled", "missing lock", "symlink", "hardlink"} {
		t.Run(mode, func(t *testing.T) {
			p, install, _, _ := uvLockPublicationFixture(t)
			ctx := context.Background()
			path := filepath.Join(p.root, "uv.lock")
			switch mode {
			case "venv context":
				var err error
				install, err = domain.NewPythonVenvInstallContext(install.Target())
				if err != nil {
					t.Fatal(err)
				}
			case "canceled":
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
			case "missing lock":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("pyproject.toml", path); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(path, filepath.Join(p.root, "shared-lock")); err != nil {
					t.Fatal(err)
				}
			}
			if guard, err := p.Begin(ctx, install); err == nil || guard != nil {
				t.Fatal("unsafe original accepted", mode, err)
			}
			if _, err := os.Lstat(filepath.Join(p.root, uvProjectTransactionLock)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("preflight leaked mutation guard", err)
			}
			if _, _, _, ok := p.PublishedRecords(); ok {
				t.Fatal("preflight failure published records")
			}
		})
	}
}

func TestUVProjectLockPublicationRollsBackOriginalAtEveryMutationBoundary(t *testing.T) {
	for _, step := range []string{"before-backup", "before-publish", "after-publish", "before-cleanup"} {
		t.Run(step, func(t *testing.T) {
			p, install, original, identity := uvLockPublicationFixture(t)
			guard, err := p.Begin(context.Background(), install)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := guard.Close(); err != nil {
					t.Error(err)
				}
			})
			update, err := p.ResolveProjectLockUpdate(context.Background(), install, guard.Controls(), "")
			if err != nil {
				t.Fatal(err)
			}
			staged, err := p.StageProject(context.Background(), uvProjectSyntheticApproval(t, update.Snapshot()))
			if err != nil {
				t.Fatal(err)
			}
			p.guard.files.checkpoint = func(name string) error {
				if name == step {
					return syscall.ENOSPC
				}
				return nil
			}
			if err := guard.Commit(context.Background(), update, staged); !errors.Is(err, syscall.ENOSPC) {
				t.Fatal("primary storage failure lost", err)
			}
			current, err := os.ReadFile(filepath.Join(p.root, "uv.lock"))
			info, statErr := os.Lstat(filepath.Join(p.root, "uv.lock"))
			if err != nil || statErr != nil || !bytes.Equal(current, original) || !os.SameFile(identity, info) || info.Mode() != identity.Mode() {
				t.Fatal("exact original not restored", err, statErr)
			}
			if p.guard.files.recovery {
				t.Fatal("ordinary failure left uncertain recovery")
			}
			if _, _, _, ok := p.PublishedRecords(); ok {
				t.Fatal("failed lock publication reported success")
			}
		})
	}
}

func TestUVProjectLockPublicationPreservesForeignStateAndBlocksRecoveryReentry(t *testing.T) {
	for _, mode := range []string{"foreign backup", "foreign published lock", "manifest drift"} {
		t.Run(mode, func(t *testing.T) {
			p, install, original, identity := uvLockPublicationFixture(t)
			guard, err := p.Begin(context.Background(), install)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := guard.Close(); err != nil {
					t.Error(err)
				}
			})
			update, err := p.ResolveProjectLockUpdate(context.Background(), install, guard.Controls(), "")
			if err != nil {
				t.Fatal(err)
			}
			staged, err := p.StageProject(context.Background(), uvProjectSyntheticApproval(t, update.Snapshot()))
			if err != nil {
				t.Fatal(err)
			}
			foreign := ""
			p.guard.files.checkpoint = func(step string) error {
				if mode == "foreign backup" && step == "before-cleanup" {
					foreign = filepath.Join(p.root, p.guard.files.backup, "foreign-note")
				}
				if mode == "foreign published lock" && step == "after-publish" {
					foreign = filepath.Join(p.root, "uv.lock")
				}
				if mode == "manifest drift" && step == "before-publish" {
					foreign = filepath.Join(p.root, "pyproject.toml")
				}
				if foreign != "" {
					return os.WriteFile(foreign, []byte("foreign data"), 0o600)
				}
				return nil
			}
			if err := guard.Commit(context.Background(), update, staged); err == nil {
				t.Fatal("foreign state accepted")
			}
			if body, err := os.ReadFile(foreign); err != nil || string(body) != "foreign data" {
				t.Fatal("foreign bytes deleted or overwritten", err)
			}
			if mode != "foreign published lock" {
				body, err := os.ReadFile(filepath.Join(p.root, "uv.lock"))
				info, statErr := os.Lstat(filepath.Join(p.root, "uv.lock"))
				if err != nil || statErr != nil || !bytes.Equal(body, original) || !os.SameFile(identity, info) {
					t.Fatal("original lock not restored", err, statErr)
				}
			} else if !p.guard.files.snapshots["uv.lock"].matches(filepath.Join(p.root, p.guard.files.backup, "uv.lock")) {
				t.Fatal("authentic original backup lost")
			}
			if !p.guard.files.recovery {
				t.Fatal("uncertain cleanup released recovery guard")
			}
			if err := guard.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := freezeUVProjectLockFiles(context.Background(), p.root, []string{p.cache.storage.intakeRoot, p.cache.storage.evidenceRoot, p.cache.storage.stagingRoot}); err == nil {
				t.Fatal("recovery lock allowed another mutation")
			}
			if _, _, _, ok := p.PublishedRecords(); ok {
				t.Fatal("uncertain publication reported success")
			}
		})
	}
}
