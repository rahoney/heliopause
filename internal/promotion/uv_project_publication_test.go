package promotion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type uvProjectEmptyResolver struct {
	t      *testing.T
	calls  int
	change func(domain.ProjectDependencyUpdate) (domain.ProjectDependencyUpdate, error)
}

func (r *uvProjectEmptyResolver) ResolveProjectLockUpdate(_ context.Context, install domain.InstallContext, original []domain.ProjectControlFile, selector string) (domain.ProjectDependencyUpdate, error) {
	r.calls++
	if selector != "" {
		r.t.Fatal("outer selector reached frozen native intent")
	}
	inputControls := []domain.ProjectControlFile{}
	for _, c := range original {
		if c.Name() == artifactpypi.UVProjectInputControl || c.Name() == artifactpypi.UVNativeLockControl || c.Name() == artifactpypi.UVProjectIntentControl {
			inputControls = append(inputControls, c)
		}
	}
	input, err := artifactpypi.ParseUVProjectControls(inputControls)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	body, err := os.ReadFile("../artifact/pypi/testdata/uv-0.13.4-empty-project.uv.lock")
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	lock, err := artifactpypi.ParseUVNativeProjectLock(input, body)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	export, err := lock.ParseTargetExport([]byte("lock-version='1.0'\ncreated-by='uv'\nrequires-python='==3.14.7'\npackages=[]\n"))
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	provenance, err := export.Controls()
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	selected := append([]domain.ProjectControlFile(nil), original...)
	for i, c := range selected {
		for _, p := range provenance {
			if c.Name() == p.Name() {
				selected[i] = p
			}
		}
	}
	runtime := sandbox.PinnedPythonRuntime()
	snapshot, err := export.BindProject(install, selected, nil, nil, artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}, artifactpypi.PublicPyPIProfile().ResourcePolicy().WheelLimits())
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	update, err := domain.NewProjectDependencyUpdateFromSnapshot(original, selected, snapshot)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	if r.change != nil {
		return r.change(update)
	}
	return update, nil
}

type uvProjectPublicationFixture struct {
	preparation *UVProjectPreparation
	publication *UVProjectPromotion
	resolver    *uvProjectEmptyResolver
	install     domain.InstallContext
	root        string
}

func newUVProjectPublicationFixture(t *testing.T) uvProjectPublicationFixture {
	t.Helper()
	_, plan := makePypiVenvFixture(t)
	target, err := domain.NewInstallTarget(plan.root)
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewPythonVenvInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	root, storage := realPromotionRoot(t), realPromotionRoot(t)
	manifest, err := os.ReadFile("../artifact/pypi/testdata/uv-0.13.4-empty-project.pyproject.toml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	intake, evidenceRoot, cacheRoot := filepath.Join(storage, "intake"), filepath.Join(storage, "evidence"), filepath.Join(storage, "cache")
	store, err := local.NewStore(evidenceRoot)
	if err != nil {
		t.Fatal(err)
	}
	operation, err := domain.NewOperationID()
	if err != nil {
		t.Fatal(err)
	}
	cache, err := NewPythonProjectCache(intake, evidenceRoot, cacheRoot, store, evidence.Generator{}, operation)
	if err != nil {
		t.Fatal(err)
	}
	r := &uvProjectEmptyResolver{t: t}
	preparation, err := NewUVProjectPreparation(&PyPIPromotion{stagingRoot: cacheRoot, goos: "linux", goarch: "amd64"}, r, root, artifactpypi.UVProjectSync, "", false, []string{intake, evidenceRoot, cacheRoot})
	if err != nil {
		t.Fatal(err)
	}
	publication, err := NewUVProjectPromotion(preparation, cache)
	if err != nil {
		t.Fatal(err)
	}
	return uvProjectPublicationFixture{preparation, publication, r, install, root}
}

func uvProjectSyntheticApproval(t *testing.T, snapshot domain.ProjectDependencySnapshot) domain.ProjectVerifiedSet {
	t.Helper()
	inspected, err := domain.NewInspectedProjectSet(snapshot, nil)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := domain.NewPolicyDecision(domain.DecisionAllow, "synthetic-project-policy", 1, []string{"SYNTHETIC_ALLOW"})
	if err != nil {
		t.Fatal(err)
	}
	set, err := domain.NewProjectVerifiedSet(inspected, decision)
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func TestUVProjectPublicationUsesOneHeldCompleteSelection(t *testing.T) {
	f := newUVProjectPublicationFixture(t)
	guard, err := f.publication.Begin(context.Background(), f.install)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := guard.Close(); err != nil {
			t.Error(err)
		}
	})
	if _, _, _, ok := f.publication.PublishedRecords(); ok {
		t.Fatal("records exposed before commit")
	}
	update, err := f.preparation.ResolveProjectLockUpdate(context.Background(), f.install, guard.Controls(), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.preparation.ResolveProjectLockUpdate(context.Background(), f.install, guard.Controls(), ""); err == nil || f.resolver.calls != 1 {
		t.Fatal("native selection repeated", err)
	}
	if err := guard.CommitSnapshot(context.Background(), update.Snapshot(), domain.StagedProjectSet{}); err == nil {
		t.Fatal("snapshot substituted native update")
	}
	if err := guard.Commit(context.Background(), update, domain.StagedProjectSet{}); err == nil {
		t.Fatal("unapproved selection committed")
	}
	staged, err := f.publication.StageProject(context.Background(), uvProjectSyntheticApproval(t, update.Snapshot()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.publication.StageProject(context.Background(), staged.Set()); err == nil {
		t.Fatal("repeated staging accepted")
	}
	if err := guard.Commit(context.Background(), update, staged); err != nil {
		t.Fatal(err)
	}
	if _, _, _, ok := f.publication.PublishedRecords(); !ok {
		t.Fatal("committed records unavailable")
	}
	if err := guard.Commit(context.Background(), update, staged); err == nil {
		t.Fatal("second commit accepted")
	}
	if _, err := os.Lstat(filepath.Join(f.root, "uv.lock")); err != nil {
		t.Fatal("native selected lock missing", err)
	}
}

func TestUVProjectPublicationRejectsResolverSubstitutionAndGuardDrift(t *testing.T) {
	for _, change := range []string{"original controls", "target", "source drift", "primary failure"} {
		t.Run(change, func(t *testing.T) {
			f := newUVProjectPublicationFixture(t)
			guard, err := f.publication.Begin(context.Background(), f.install)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := guard.Close(); err != nil {
					t.Error(err)
				}
			})
			cause := errors.New("synthetic native failure")
			f.resolver.change = func(update domain.ProjectDependencyUpdate) (domain.ProjectDependencyUpdate, error) {
				switch change {
				case "original controls":
					controls := update.OriginalControls()
					c := controls[0]
					controls[0], err = domain.NewProjectControlFile(c.Name(), append(c.Body(), ' '), c.Present())
					if err != nil {
						t.Fatal(err)
					}
					return domain.NewProjectDependencyUpdateFromSnapshot(controls, update.SelectedControls(), update.Snapshot())
				case "target":
					target, err := domain.NewInstallTarget(realPromotionRoot(t))
					if err != nil {
						t.Fatal(err)
					}
					install, err := domain.NewPythonVenvInstallContext(target)
					if err != nil {
						t.Fatal(err)
					}
					s, err := domain.NewDependencyFreeProjectSnapshot(install, update.Snapshot().Source(), update.Snapshot().ControlDigests(), update.Snapshot().GraphDigest())
					if err != nil {
						t.Fatal(err)
					}
					return domain.NewProjectDependencyUpdateFromSnapshot(update.OriginalControls(), update.SelectedControls(), s)
				case "source drift":
					if err := os.WriteFile(filepath.Join(f.root, "pyproject.toml"), []byte("changed"), 0o600); err != nil {
						t.Fatal(err)
					}
				case "primary failure":
					return domain.ProjectDependencyUpdate{}, cause
				}
				return update, nil
			}
			update, err := f.preparation.ResolveProjectLockUpdate(context.Background(), f.install, guard.Controls(), "")
			if err == nil || update.Valid() {
				t.Fatal("unbound native output accepted")
			}
			if change == "primary failure" && !errors.Is(err, cause) {
				t.Fatal("primary native cause lost", err)
			}
			if _, _, _, ok := f.publication.PublishedRecords(); ok {
				t.Fatal("failed selection exposed completed records")
			}
			if _, err := os.Lstat(filepath.Join(f.root, "uv.lock")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("rejected selection changed Host lock", err)
			}
		})
	}
}

func TestUVProjectPublicationRefusesForeignCompleteApproval(t *testing.T) {
	f := newUVProjectPublicationFixture(t)
	guard, err := f.publication.Begin(context.Background(), f.install)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := guard.Close(); err != nil {
			t.Error(err)
		}
	})
	update, err := f.preparation.ResolveProjectLockUpdate(context.Background(), f.install, guard.Controls(), "")
	if err != nil {
		t.Fatal(err)
	}
	digest, err := domain.NewSHA256Digest(strings.Repeat("f", 64))
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := domain.NewDependencyFreeProjectSnapshot(f.install, update.Snapshot().Source(), update.Snapshot().ControlDigests(), digest)
	if err != nil {
		t.Fatal(err)
	}
	if s, err := f.publication.StageProject(context.Background(), uvProjectSyntheticApproval(t, foreign)); err == nil || s.Valid() {
		t.Fatal("foreign complete approval accepted")
	}
	if _, err := os.Lstat(filepath.Join(f.root, "uv.lock")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("foreign approval changed Host lock", err)
	}
}

func TestUVProjectPublicationRejectsUnmanagedOriginalBeforeSelection(t *testing.T) {
	f := newUVProjectPublicationFixture(t)
	directory := filepath.Join(f.install.Target().String(), "lib/python3.14/site-packages/old-1.0.dist-info")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	metadata := filepath.Join(directory, "METADATA")
	body := []byte("Metadata-Version: 2.4\nName: old\nVersion: 1.0\n\n")
	if err := os.WriteFile(metadata, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if guard, err := f.publication.Begin(context.Background(), f.install); guard != nil || ports.InstallUpdateFailureCode(err) != "UPDATE_ADOPTION_REQUIRED" {
		t.Fatal("unmanaged original accepted or wrong refusal", err)
	}
	if f.resolver.calls != 0 {
		t.Fatal("unmanaged original reached native selection")
	}
	for _, path := range []string{filepath.Join(f.root, uvProjectTransactionLock), filepath.Join(f.install.Target().String(), ".heliopause-pypi-transaction.lock")} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("failed begin retained normal guard", err)
		}
	}
	if got, err := os.ReadFile(metadata); err != nil || string(got) != string(body) {
		t.Fatal("unmanaged original changed", err)
	}
}
