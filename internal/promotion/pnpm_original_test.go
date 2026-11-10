package promotion

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// A real prepared copy is only the mechanical original fixture. This callback
// is explicitly synthetic; independent public source proof has other tests.
func pnpmOriginalTransactionFixture(t *testing.T) (pnpmCacheFixture, *PNPMProjectPromotion, domain.StagedProjectSet, domain.ProjectDependencyUpdate) {
	t.Helper()
	f, owner, staged, _ := pnpmTransactionFixture(t)
	prepared, err := f.cache.PrepareProject(context.Background(), staged)
	if err != nil {
		t.Fatal(err)
	}
	target := f.cache.frozen.Snapshot().Context().Target().String()
	if err := os.Rename(filepath.Join(prepared.Directory(), "node_modules"), filepath.Join(target, "node_modules")); err != nil {
		t.Fatal(err)
	}
	(&pnpmProjectProfile{prepared: prepared}).releaseWorkspaceMember("node_modules")
	if err := prepared.Close(); err != nil {
		t.Fatal(err)
	}
	for _, control := range f.cache.frozen.Controls() {
		if err := os.WriteFile(filepath.Join(target, control.Name()), control.Body(), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	owner.verifyOriginal = func(context.Context, domain.InstallContext, []domain.ProjectControlFile, artifactnpm.PNPMOriginalLayout) (domain.ContentDigest, error) {
		return f.cache.frozen.Snapshot().GraphDigest(), nil
	}
	update, err := domain.NewProjectDependencyUpdateFromSnapshot(f.cache.frozen.Controls(), f.cache.frozen.Controls(), f.cache.frozen.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	return f, owner, staged, update
}

func TestPNPMOriginalAdoptionRequiresProofAndGuardsCallback(t *testing.T) {
	for _, change := range []string{"implicit", "verifier-error", "zero-proof", "drift", "cancel", "foreign-marker", "bad-state", "hardlink", "unsafe-mode", "escape"} {
		t.Run(change, func(t *testing.T) {
			f, owner, _, _ := pnpmOriginalTransactionFixture(t)
			install := f.cache.frozen.Snapshot().Context()
			target := install.Target().String()
			code := filepath.Join(target, "node_modules/.pnpm/is-number@6.0.0/node_modules/is-number/package.json")
			before, err := os.Lstat(code)
			if err != nil {
				t.Fatal(err)
			}
			original, err := os.ReadFile(code)
			if err != nil {
				t.Fatal(err)
			}
			cause := errors.New("controlled original proof failure")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			calls := 0
			owner.verifyOriginal = func(context.Context, domain.InstallContext, []domain.ProjectControlFile, artifactnpm.PNPMOriginalLayout) (domain.ContentDigest, error) {
				calls++
				switch change {
				case "verifier-error":
					return domain.ContentDigest{}, cause
				case "zero-proof":
					return domain.ContentDigest{}, nil
				case "drift":
					if err := os.WriteFile(code, []byte("foreign edit"), 0o600); err != nil {
						t.Fatal(err)
					}
				case "cancel":
					cancel(cause)
				}
				return f.cache.frozen.Snapshot().GraphDigest(), nil
			}
			beforeCallback := false
			switch change {
			case "implicit":
				beforeCallback = true
			case "foreign-marker":
				if err := os.Mkdir(filepath.Join(target, ".heliopause"), 0o700); err != nil {
					t.Fatal(err)
				}
				beforeCallback = true
			case "bad-state":
				if err := ensureTrustedRoot(owner.stateRoot); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(owner.stateRoot, pnpmCacheHash([]byte(target))+".json"), []byte(`{}`), 0o600); err != nil {
					t.Fatal(err)
				}
				beforeCallback = true
			case "hardlink":
				if err := os.Link(code, filepath.Join(target, "shared-copy")); err != nil {
					t.Fatal(err)
				}
				beforeCallback = true
			case "unsafe-mode":
				if err := os.Chmod(filepath.Dir(code), 0o777); err != nil {
					t.Fatal(err)
				}
				beforeCallback = true
			case "escape":
				if err := os.Symlink("../../outside", filepath.Join(target, "node_modules/foreign")); err != nil {
					t.Fatal(err)
				}
				beforeCallback = true
			}
			g, err := owner.BeginUpgrade(ctx, install, change != "implicit")
			if err == nil {
				_ = g.Close()
				t.Fatal("unproven original admitted")
			}
			if beforeCallback && calls != 0 || !beforeCallback && calls != 1 {
				t.Fatal("unexpected original callback coverage", calls, err)
			}
			if (change == "cancel" || change == "verifier-error") && !errors.Is(err, cause) {
				t.Fatal("first proof cause lost", err)
			}
			after, statErr := os.Lstat(code)
			if statErr != nil || !os.SameFile(before, after) {
				t.Fatal("original identity changed on refusal", statErr)
			}
			body, readErr := os.ReadFile(code)
			want := string(original)
			if change == "drift" {
				want = "foreign edit"
			}
			if readErr != nil || string(body) != want {
				t.Fatal("refused proof changed original/foreign edit", readErr)
			}
		})
	}
}

func TestPNPMOriginalAdoptionCommitsProofAndRollsBackNativeCopy(t *testing.T) {
	for _, phase := range []string{"success", "after-publish:node_modules", "after-metadata", "after-state"} {
		t.Run(phase, func(t *testing.T) {
			f, owner, staged, update := pnpmOriginalTransactionFixture(t)
			install := f.cache.frozen.Snapshot().Context()
			original, err := capturePNPMOwnedMember(context.Background(), filepath.Join(install.Target().String(), "node_modules"))
			if err != nil {
				t.Fatal(err)
			}
			cause := errors.New("controlled native adoption commit failure")
			owner.checkpoint = func(actual string) error {
				if actual == phase {
					return cause
				}
				return nil
			}
			g, err := owner.BeginUpgrade(context.Background(), install, true)
			if err != nil || g.previous != nil || g.originalProof == "" {
				t.Fatal("fresh native proof not bound", err)
			}
			err = g.Commit(context.Background(), update, staged)
			if phase == "success" {
				if err != nil {
					t.Fatal(err)
				}
				var state pnpmProjectApproval
				body, err := os.ReadFile(filepath.Join(owner.stateRoot, g.stateName))
				if err != nil || json.Unmarshal(body, &state) != nil || state.OriginalProof != f.cache.frozen.Snapshot().GraphDigest().String() {
					t.Fatal("original provenance absent from independent state", err)
				}
			} else {
				if !errors.Is(err, cause) {
					t.Fatal("native commit first cause lost", err)
				}
				if err := original.verify(context.Background(), filepath.Join(install.Target().String(), "node_modules")); err != nil {
					t.Fatal("native original was not restored exactly", err)
				}
				if _, err := os.Lstat(filepath.Join(owner.stateRoot, g.stateName)); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("failed native adoption published state", err)
				}
			}
			if err := g.Close(); err != nil {
				t.Fatal(err)
			}
			if phase == "success" {
				owner.verifyOriginal = func(context.Context, domain.InstallContext, []domain.ProjectControlFile, artifactnpm.PNPMOriginalLayout) (domain.ContentDigest, error) {
					t.Fatal("managed state fell back to native proof")
					return domain.ContentDigest{}, nil
				}
				next, err := owner.BeginUpgrade(context.Background(), install, true)
				if err != nil {
					t.Fatal(err)
				}
				if err := next.Close(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestPNPMNativeOriginalRollbackConflictPreservesBackupAndForeignOutput(t *testing.T) {
	f, owner, staged, update := pnpmOriginalTransactionFixture(t)
	ctx := context.Background()
	install := f.cache.frozen.Snapshot().Context()
	cause := errors.New("controlled conflict after native publication")
	foreign := filepath.Join(install.Target().String(), "node_modules/foreign")
	owner.checkpoint = func(phase string) error {
		if phase == "after-publish:node_modules" {
			if err := os.WriteFile(foreign, []byte("preserve me"), 0o600); err != nil {
				return err
			}
			return cause
		}
		return nil
	}
	original, err := capturePNPMOwnedMember(ctx, filepath.Join(install.Target().String(), "node_modules"))
	if err != nil {
		t.Fatal(err)
	}
	g, err := owner.BeginUpgrade(ctx, install, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Commit(ctx, update, staged); !errors.Is(err, cause) || !strings.Contains(err.Error(), "rollback is incomplete") {
		t.Fatal("conflict lost cause or claimed clean rollback", err)
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(foreign); err != nil || string(body) != "preserve me" {
		t.Fatal("foreign output deleted", err)
	}
	entries, err := os.ReadDir(install.Target().String())
	if err != nil {
		t.Fatal(err)
	}
	journals := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".heliopause-npm-commit-") {
			journals++
			if err := original.verify(ctx, filepath.Join(install.Target().String(), entry.Name(), "node_modules")); err != nil {
				t.Fatal("uncertain rollback lost original backup", err)
			}
		}
	}
	if journals != 1 {
		t.Fatal("uncertain recovery journal missing", journals)
	}
	if g, err := owner.BeginUpgrade(ctx, install, true); err == nil {
		_ = g.Close()
		t.Fatal("new mutation entered an unresolved journal")
	}
}
