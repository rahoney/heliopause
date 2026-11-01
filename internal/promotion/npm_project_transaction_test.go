package promotion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type npmEmptyLayoutFixture struct{}

func (npmEmptyLayoutFixture) PreparedLayoutFor(domain.ProjectDependencySnapshot, map[domain.DependencyNodeID]domain.AcquiredArtifact) (artifactnpm.NPMProjectInspectionLayout, error) {
	return artifactnpm.NPMProjectInspectionLayout{}, errors.New("empty npm transaction must not request a dependency layout")
}

// Mechanical empty-project transaction only; native source/quarantine is
// covered separately. Use real npm controls, cache records and transaction.
func npmEmptyTransactionFixture(t *testing.T) (*NPMProjectVerifiedCache, *PNPMProjectPromotion, domain.StagedProjectSet, domain.ProjectDependencyUpdate) {
	t.Helper()
	ctx := context.Background()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "project")
	if err = os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(target, "package.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(target, "notes.txt"), []byte("user-owned\n"), 0600); err != nil {
		t.Fatal(err)
	}
	manifest, err := domain.NewProjectControlFile("package.json", []byte(`{}`), true)
	if err != nil {
		t.Fatal(err)
	}
	absent, err := domain.NewProjectControlFile("package-lock.json", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := domain.NewProjectControlFile("package-lock.json", []byte(`{"lockfileVersion":3,"packages":{"":{}}}`), true)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := artifactnpm.ParseNPMProjectLock(manifest.Body(), lock.Body(), runtimeidentity.NodeNPMVersion)
	if err != nil {
		t.Fatal(err)
	}
	intake, evidenceRoot, cacheRoot, state := filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache"), filepath.Join(root, "state")
	source, err := artifactnpm.NewPublicResolver(intake)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := source.BindNPMProjectLock(ctx, parsed)
	if err != nil {
		t.Fatal(err)
	}
	destination, err := domain.NewInstallTarget(target)
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewNPMProjectInstallContext(destination)
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := selected.InspectionContext(install, []domain.ProjectControlFile{manifest, lock}, "controlled-npm-runtime")
	if err != nil {
		t.Fatal(err)
	}
	inspected, err := domain.NewInspectedProjectSet(frozen.Snapshot(), nil)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := (policy.M4{}).EvaluateProjectSet(inspected)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := domain.NewProjectVerifiedSet(inspected, decision)
	if err != nil {
		t.Fatal(err)
	}
	store, err := local.NewStore(evidenceRoot)
	if err != nil {
		t.Fatal(err)
	}
	operation, err := domain.NewOperationID()
	if err != nil {
		t.Fatal(err)
	}
	cache, err := NewNPMProjectVerifiedCache(intake, evidenceRoot, cacheRoot, frozen, store, npmEmptyLayoutFixture{}, evidence.Generator{}, operation)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := NewNPMProjectPromotion(intake, evidenceRoot, cacheRoot, state, store, operation, func(domain.ProjectDependencySnapshot) (*NPMProjectVerifiedCache, error) { return cache, nil })
	if err != nil {
		t.Fatal(err)
	}
	staged, err := cache.StageProject(ctx, verified)
	if err != nil {
		t.Fatal(err)
	}
	update, err := domain.NewProjectDependencyUpdateFromSnapshotWithControlPresence([]domain.ProjectControlFile{manifest, absent}, frozen.Controls(), frozen.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	return cache, owner, staged, update
}

func TestNPMProjectTransactionRestoresControlsPayloadRecordsAndState(t *testing.T) {
	for _, phase := range []string{"before-backup:package.json", "before-publish:package.json", "after-publish:package-lock.json", "after-publish:node_modules", "after-publish:haa", "before-metadata", "after-metadata", "before-state", "after-state", "cancelled"} {
		t.Run(phase, func(t *testing.T) {
			cache, owner, staged, update := npmEmptyTransactionFixture(t)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("controlled npm publication stop")
			guard, err := owner.Begin(ctx, cache.frozen.Snapshot().Context())
			if err != nil {
				t.Fatal(err)
			}
			if phase == "cancelled" {
				cancel(cause)
			} else {
				owner.checkpoint = func(actual string) error {
					if actual == phase {
						return cause
					}
					return nil
				}
			}
			if err = guard.Commit(ctx, update, staged); !errors.Is(err, cause) {
				t.Fatal("npm publication lost original failure", err)
			}
			if _, _, _, ok := owner.PublishedRecords(); ok {
				t.Fatal("failed npm publication claimed records")
			}
			if err = guard.Close(); err != nil {
				t.Fatal(err)
			}
			target := cache.frozen.Snapshot().Context().Target().String()
			for name, expected := range map[string]string{"package.json": `{}`, "notes.txt": "user-owned\n"} {
				body, err := os.ReadFile(filepath.Join(target, name))
				if err != nil || string(body) != expected {
					t.Fatal("npm rollback changed original file", name, err)
				}
			}
			for _, name := range []string{"package-lock.json", "node_modules", ".heliopause"} {
				if _, err = os.Lstat(filepath.Join(target, name)); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("npm rollback retained uncommitted member", name, err)
				}
			}
			entries, err := os.ReadDir(target)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".heliopause-npm-commit-") {
					t.Fatal("npm rollback retained completed journal")
				}
			}
			state, err := os.ReadDir(owner.stateRoot)
			if err != nil || len(state) != 0 {
				t.Fatal("npm rollback retained independent approval", err)
			}
		})
	}
}

func TestNPMProjectRootLifecycleGuardBindsImplicitGypPresence(t *testing.T) {
	for _, mode := range []string{"absent then appears", "present", "symlink", "hardlink"} {
		t.Run(mode, func(t *testing.T) {
			cache, owner, _, _ := npmEmptyTransactionFixture(t)
			target := cache.frozen.Snapshot().Context().Target().String()
			gyp := filepath.Join(target, "binding.gyp")
			switch mode {
			case "present":
				if err := os.WriteFile(gyp, []byte(`{}`), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink("package.json", gyp); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(filepath.Join(target, "notes.txt"), gyp); err != nil {
					t.Fatal(err)
				}
			}
			ctx := context.Background()
			guard, err := owner.Begin(ctx, cache.frozen.Snapshot().Context())
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := guard.Close(); err != nil {
					t.Error(err)
				}
			}()
			required, err := guard.NPMProjectRootLifecycleRequired(ctx)
			if mode == "symlink" || mode == "hardlink" {
				if err == nil {
					t.Fatal("unsafe gyp source accepted")
				}
				return
			}
			if err != nil || required != (mode == "present") {
				t.Fatal("implicit native build action was omitted", required, err)
			}
			if mode == "absent then appears" {
				if err = os.WriteFile(gyp, []byte(`{}`), 0600); err != nil {
					t.Fatal(err)
				}
				if err = guard.VerifyUnchanged(ctx); err == nil {
					t.Fatal("gyp appeared after no-action decision without invalidating guard")
				}
			}
		})
	}
}
