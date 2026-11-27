package promotion

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestUVProjectOriginalOwnershipAuthenticatesWholeInventoryBeforeSelection(t *testing.T) {
	for _, mode := range []string{"whole proof", "adoption required", "unselected tamper", "forged installer", "shared file"} {
		t.Run(mode, func(t *testing.T) {
			plan, _, _, original := originalUVInventoryFixture(t)
			if err := original.closeOriginal(); err != nil {
				t.Fatal(err)
			}
			f := newUVProjectPublicationFixture(t)
			f.preparation.adopt = mode != "adoption required"
			f.preparation.promoter.runner = unavailableDockerRunner{}
			f.preparation.trustedRoots = append(f.preparation.trustedRoots, original.root)
			prep, err := f.preparation.WithNativeEnrollment(original, original, original.root)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "unselected tamper" {
				original.corrupt = "proof-beta"
			}
			if mode == "forged installer" {
				if err := os.WriteFile(filepath.Join(plan.site, "proof_beta-1.0.0.dist-info/INSTALLER"), []byte("pip"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "shared file" {
				if err := os.Link(filepath.Join(plan.site, "proof_beta/__init__.py"), filepath.Join(plan.root, "foreign-copy")); err != nil {
					t.Fatal(err)
				}
			}
			target, err := domain.NewInstallTarget(plan.root)
			if err != nil {
				t.Fatal(err)
			}
			install, err := domain.NewPythonVenvInstallContext(target)
			if err != nil {
				t.Fatal(err)
			}
			guard, err := prep.Begin(context.Background(), install)
			if mode != "whole proof" {
				if err == nil || guard != nil || f.resolver.calls != 0 {
					t.Fatal("unproved original reached native project selection", err)
				}
				for _, lock := range []string{filepath.Join(plan.root, ".heliopause-pypi-transaction.lock"), filepath.Join(f.root, ".heliopause-uv-project-transaction.lock")} {
					if _, err := os.Lstat(lock); !errors.Is(err, os.ErrNotExist) {
						t.Fatal("refused original leaked lock", lock, err)
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := guard.Close(); err != nil {
					t.Error(err)
				}
			})
			if !reflect.DeepEqual(original.resolves, []string{"proof-alpha", "proof-beta"}) {
				t.Fatal("unselected original skipped", original.resolves)
			}
			var proof struct{ Proofs []pypiNativeProof }
			if json.Unmarshal(guard.proof.Body(), &proof) != nil || len(proof.Proofs) != 2 || len(guard.Controls()) != 7 {
				t.Fatal("complete original proof missing")
			}
			update, err := prep.ResolveProjectLockUpdate(context.Background(), install, guard.Controls(), "")
			if err != nil {
				t.Fatal(err)
			}
			if !update.Snapshot().DependencyFree() || len(update.SelectedControls()) != 7 || f.resolver.calls != 1 {
				t.Fatal("native selection repeated or proof dropped")
			}
			if _, ok := guard.transaction.current.Files["site/proof_beta/__init__.py"]; !ok {
				t.Fatal("unselected original omitted from held prestate")
			}
			if _, err := os.Lstat(filepath.Join(plan.root, pypiVenvMetadata)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("unapproved ownership persisted", err)
			}
			if _, err := os.Lstat(filepath.Join(f.root, "uv.lock")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("selection published project lock", err)
			}
		})
	}
}

func TestUVProjectCacheBindsOnlyHeldOriginalProofToExactNativeSelection(t *testing.T) {
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
	base, err := f.preparation.ResolveProjectLockUpdate(context.Background(), f.install, guard.Controls(), "")
	if err != nil {
		t.Fatal(err)
	}
	proof, err := domain.NewProjectControlFile("pypi-update-native-ownership.json", []byte(`{"held-original":"fixture"}`), true)
	if err != nil {
		t.Fatal(err)
	}
	controls := append(base.SelectedControls(), proof)
	snapshot, err := bindPythonOriginalProof(base.Snapshot(), controls, "uv-project-original-proof/v1")
	if err != nil {
		t.Fatal(err)
	}
	bound, err := domain.NewProjectDependencyUpdateFromSnapshot(append(base.OriginalControls(), proof), controls, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewUVProjectCache(f.publication.cache, bound); err == nil {
		t.Fatal("public cache accepted unheld seventh control")
	}
	cache, err := newUVProjectCacheWithOriginalProof(f.publication.cache, base, bound, proof)
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.validateUpdate(); err != nil {
		t.Fatal(err)
	}
	changed, err := domain.NewProjectControlFile(proof.Name(), []byte(`{"different-original":true}`), true)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name         string
		base, update domain.ProjectDependencyUpdate
		proof        domain.ProjectControlFile
	}{
		{"missing proof", base, bound, domain.ProjectControlFile{}},
		{"changed proof", base, bound, changed},
		{"unbound graph", base, base, proof},
		{"invalid base", domain.ProjectDependencyUpdate{}, bound, proof},
		{"recursive base", bound, bound, proof},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := &UVProjectCache{cache: f.publication.cache, base: tc.base, update: tc.update, proof: tc.proof}
			if err := bad.validateUpdate(); err == nil {
				t.Fatal("substituted proof accepted")
			}
			if _, err := bad.StageProject(context.Background(), uvProjectSyntheticApproval(t, bound.Snapshot())); err == nil {
				t.Fatal("substituted proof staged")
			}
			if _, err := bad.OpenProjectCache(context.Background(), domain.StagedProjectSet{}); err == nil {
				t.Fatal("substituted proof opened")
			}
		})
	}
}
