package promotion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence"
)

func (r *uvGuardResolverFixture) ResolveDependencies(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error) {
	r.t.Fatal("installed-aware request fell back to fresh selection")
	return domain.DependencyResolution{}, errors.New("unexpected fresh selection")
}

func TestUVInstalledOrdinaryInstallPreservesRequestGuardAndPublication(t *testing.T) {
	for _, mode := range []string{"exact", "bare", "noop", "new-root", "request-drift", "original-drift"} {
		t.Run(mode, func(t *testing.T) {
			f := newUVUpdateGuardFixture(t, artifactpypi.UVInstallSelection, mode == "noop")
			if mode == "new-root" {
				f.root, f.plan = makePypiVenvFixture(t)
				runVenvFixture(t, f.plan, map[string]string{"site/other-1.0.dist-info/METADATA": "Name: other\nVersion: 1.0\n\n", "site/other.py": "retained unrelated"}, "other")
				target, _ := domain.NewInstallTarget(f.root)
				f.install, _ = domain.NewPythonVenvInstallContext(target)
			}
			preparation, err := NewUVInstallPreparation(f.preparation.shared.promoter, f.resolver)
			if err != nil {
				t.Fatal(err)
			}
			reference, _ := artifactpypi.ParseReference("haa-promotion-fixture@1.0.0")
			if mode == "bare" {
				reference, _ = artifactpypi.ParseReference("haa-promotion-fixture")
			}
			s, err := preparation.BeginInstall(context.Background(), reference, f.install)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := s.Close(); err != nil {
					t.Error(err)
				}
			})
			if _, update := s.(ports.InstallUpdateSession); update {
				t.Fatal("ordinary install acquired UPDATE marker")
			}
			if f.resolver.calls != 0 {
				t.Fatal("selection ran before guard")
			}
			if other, err := preparation.BeginInstall(context.Background(), reference, f.install); err == nil || other != nil {
				t.Fatal("same target guard was released")
			}
			selected := reference
			if mode == "request-drift" {
				selected, _ = artifactpypi.ParseReference("other@1.0")
			}
			if mode == "original-drift" {
				if err := os.WriteFile(filepath.Join(f.plan.site, "other-1.0.dist-info/METADATA"), []byte("changed"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			resolution, err := s.ResolveDependencies(context.Background(), selected, f.install)
			if mode == "request-drift" || mode == "original-drift" {
				if err == nil || f.resolver.calls != 0 || f.runner.calls != 0 {
					t.Fatal("invalid ordinary request selected", err)
				}
				return
			}
			if err != nil || len(resolution.ProjectControls()) != 4 || len(resolution.Graph().Nodes()) != 1 {
				t.Fatal("installed selection failed", resolution, err)
			}
			op, _ := domain.NewOperationID()
			bundle, err := evidence.BuildVerifiedBundle(evidence.ManifestContext{OperationID: op, InstallContext: f.install, ResolverRuntime: resolution.RuntimeIdentity(), LockfileDigest: resolution.LockfileDigest(), ProjectControls: resolution.ProjectControls()}, f.seed.Set())
			if err != nil {
				t.Fatal(err)
			}
			staging, _ := NewLocalStaging(filepath.Join(f.records, "intake"), filepath.Join(f.records, "evidence"), filepath.Join(f.records, "staging"))
			staged, err := staging.Stage(context.Background(), bundle)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Promote(context.Background(), staged, bundle, f.install); err != nil {
				t.Fatal(err)
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			if err := s.Close(); err != nil {
				t.Fatal("close not idempotent", err)
			}
			body, err := os.ReadFile(filepath.Join(f.plan.site, "haa_promotion_fixture/__init__.py"))
			if err != nil || string(body) != "value = 42\n" {
				t.Fatal(string(body), err)
			}
			body, err = os.ReadFile(filepath.Join(f.plan.site, "other.py"))
			if err != nil || string(body) != "retained unrelated" {
				t.Fatal("unrelated files changed", err)
			}
		})
	}
}

func TestUVInstalledOrdinaryInstallTransfersFailureCleanupOnlyOnce(t *testing.T) {
	f := newUVUpdateGuardFixture(t, artifactpypi.UVInstallSelection, false)
	if err := os.Remove(filepath.Join(f.root, ".heliopause/pypi-transaction.json")); err != nil {
		t.Fatal(err)
	}
	preparation, err := NewUVInstallPreparation(f.preparation.shared.promoter, f.resolver)
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := artifactpypi.ParseReference("haa-promotion-fixture@1.0.0")
	s, err := preparation.BeginInstall(context.Background(), reference, f.install)
	if s != nil || ports.InstallUpdateFailureCode(err) != "UPDATE_ADOPTION_REQUIRED" || errors.Is(err, os.ErrNotExist) || f.resolver.calls != 0 {
		t.Fatal("ownership failure or cleanup cause changed", err)
	}
	guard, err := acquirePyPIVenvGuard(f.root)
	if err != nil {
		t.Fatal("failed transfer retained guard", err)
	}
	if err := guard.release(); err != nil {
		t.Fatal(err)
	}
}
