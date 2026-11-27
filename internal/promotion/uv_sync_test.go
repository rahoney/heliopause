package promotion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type uvSyncResolverFixture struct {
	t         *testing.T
	artifacts []domain.ResolvedArtifact
	sizes     map[string]int64
	calls     int
	before    func()
	mutate    func(domain.ProjectDependencyUpdate) domain.ProjectDependencyUpdate
	err       error
}

func (r *uvSyncResolverFixture) ResolveProjectLockUpdate(_ context.Context, install domain.InstallContext, original []domain.ProjectControlFile, selector string) (domain.ProjectDependencyUpdate, error) {
	r.t.Helper()
	r.calls++
	if len(original) != 4 || selector != "" {
		r.t.Fatal("original proof leaked into native control protocol")
	}
	if r.before != nil {
		r.before()
	}
	if r.err != nil {
		return domain.ProjectDependencyUpdate{}, r.err
	}
	body := "lock-version='1.0'\ncreated-by='uv'\nrequires-python='>=3.14.7'\n"
	if len(r.artifacts) == 0 {
		body += "packages=[]\n"
	}
	for _, a := range r.artifacts {
		size := int64(1)
		if r.sizes[a.Identity().Name()] > 0 {
			size = r.sizes[a.Identity().Name()]
		}
		body += fmt.Sprintf("[[packages]]\nname=%q\nversion=%q\nwheels=[{url=%q,size=%d,hashes={sha256=%q}}]\n", a.Identity().Name(), a.Identity().Version(), a.AcquisitionLocator(), size, strings.TrimPrefix(a.DeclaredIntegrity(), "sha256:"))
	}
	lock, err := domain.NewProjectControlFile(artifactpypi.UVSelectionLockControlName, []byte(body), true)
	if err != nil {
		r.t.Fatal(err)
	}
	selected := append([]domain.ProjectControlFile(nil), original...)
	digests := []domain.ProjectControlDigest{}
	for i := range selected {
		if selected[i].Name() == lock.Name() {
			selected[i] = lock
		}
		digest, err := domain.NewProjectControlDigest(selected[i].Name(), selected[i].Digest())
		if err != nil {
			r.t.Fatal(err)
		}
		digests = append(digests, digest)
	}
	sum := sha256.Sum256([]byte(body))
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	var snapshot domain.ProjectDependencySnapshot
	if len(r.artifacts) == 0 {
		snapshot, err = domain.NewDependencyFreeProjectSnapshot(install, artifactpypi.PublicPyPIProfile().Source(), digests, digest)
	} else {
		snapshot, err = domain.NewProjectDependencySnapshot(install, artifactpypi.PublicPyPIProfile().Source(), digests, r.artifacts, digest)
	}
	if err != nil {
		r.t.Fatal(err)
	}
	update, err := domain.NewProjectDependencyUpdateFromSnapshot(original, selected, snapshot)
	if err != nil {
		r.t.Fatal(err)
	}
	if r.mutate != nil {
		update = r.mutate(update)
	}
	return update, nil
}

func newUVSyncGuardFixture(t *testing.T, empty bool) (*UVSyncPreparation, domain.InstallContext, *uvSyncResolverFixture) {
	t.Helper()
	f := newUVUpdateGuardFixture(t, artifactpypi.UVUpgradeSelection, false)
	body := "haa-promotion-fixture==1.0.0\n"
	resolver := &uvSyncResolverFixture{t: t, artifacts: []domain.ResolvedArtifact{f.seed.Set().Inspected().Graph().Nodes()[0].Artifact()}}
	if empty {
		body = "# explicit empty\n"
		resolver.artifacts = nil
	}
	inputRoot := realPromotionRoot(t)
	source := filepath.Join(inputRoot, "requirements.txt")
	if err := os.WriteFile(source, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	prep, err := NewUVSyncPreparation(f.preparation.shared.promoter, resolver, source, artifactpypi.UVSyncRequirements, empty, false, []string{f.preparation.shared.promoter.stagingRoot})
	if err != nil {
		t.Fatal(err)
	}
	return prep, f.install, resolver
}

func TestUVSyncHeldSelectionPreservesSourceAndEveryManagedOwner(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(fmt.Sprint(empty), func(t *testing.T) {
			prep, install, resolver := newUVSyncGuardFixture(t, empty)
			if _, err := prep.SelectedUpdate(context.Background()); err == nil {
				t.Fatal("selection existed before guard")
			}
			guard, err := prep.Begin(context.Background(), install)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := guard.Close(); err != nil {
					t.Error(err)
				}
			})
			before := len(guard.transaction.current.Files)
			if before == 0 || len(guard.original.metadataFiles()) != 2 {
				t.Fatal("whole managed original was not retained")
			}
			if _, err := beginPyPIVenvTransactionWithDiscovery(guard.transaction.plan, discoverUVPythonVenv); err == nil {
				t.Fatal("sync did not hold the shared target lock")
			}
			update, err := prep.ResolveProjectLockUpdate(context.Background(), install, guard.Controls(), "")
			if err != nil || update.Snapshot().DependencyFree() != empty || update.HasRequestedResolution() || resolver.calls != 1 {
				t.Fatal("complete selection failed", err)
			}
			selected, err := prep.SelectedUpdate(context.Background())
			if err != nil || !reflect.DeepEqual(selected, update) {
				t.Fatal("one actual update was not retained", err)
			}
			if before != len(guard.transaction.current.Files) {
				t.Fatal("selection removed original ownership before approval")
			}
			if _, err := prep.ResolveProjectLockUpdate(context.Background(), install, guard.Controls(), ""); err == nil || resolver.calls != 1 {
				t.Fatal("selection repeated")
			}
			if _, err := prep.Begin(context.Background(), install); err == nil {
				t.Fatal("request acquired another guard")
			}
			if err := guard.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := prep.SelectedUpdate(context.Background()); err == nil {
				t.Fatal("closed guard returned selection")
			}
		})
	}
}

func TestUVSyncOriginalProofIncludesExcludedNativeAndDoesNotPublishOwnership(t *testing.T) {
	for _, mode := range []string{"whole-proof", "adoption-required", "excluded-tamper"} {
		t.Run(mode, func(t *testing.T) {
			plan, _, _, original := originalUVInventoryFixture(t)
			if err := original.closeOriginal(); err != nil {
				t.Fatal(err)
			}
			root := realPromotionRoot(t)
			source := filepath.Join(root, "source.txt")
			if err := os.WriteFile(source, []byte("proof-alpha==1.0.0\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			promoter, err := newPyPIPromotion(filepath.Join(root, "staging"), unavailableDockerRunner{}, "linux", "amd64")
			if err != nil {
				t.Fatal(err)
			}
			resolver := &uvSyncResolverFixture{t: t, artifacts: []domain.ResolvedArtifact{original.entries["proof-alpha"].resolved}}
			prep, err := NewUVSyncPreparation(promoter, resolver, source, artifactpypi.UVSyncRequirements, false, mode != "adoption-required", []string{promoter.stagingRoot, original.root})
			if err != nil {
				t.Fatal(err)
			}
			prep, err = prep.WithNativeEnrollment(original, original, original.root)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "excluded-tamper" {
				original.corrupt = "proof-beta"
			}
			target, _ := domain.NewInstallTarget(plan.root)
			install, _ := domain.NewPythonVenvInstallContext(target)
			guard, err := prep.Begin(context.Background(), install)
			if mode != "whole-proof" {
				if err == nil || guard != nil || resolver.calls != 0 {
					t.Fatal("unproven old distribution reached selection", err)
				}
				if _, err := os.Lstat(filepath.Join(plan.root, ".heliopause-pypi-transaction.lock")); err == nil {
					t.Fatal("failed guard leaked lock")
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
				t.Fatal("excluded original not proved", original.resolves)
			}
			var proof struct{ Proofs []pypiNativeProof }
			if json.Unmarshal(guard.proof.Body(), &proof) != nil || len(proof.Proofs) != 2 || len(guard.Controls()) != 5 {
				t.Fatal("whole original proof is incomplete")
			}
			if _, err := os.Lstat(filepath.Join(plan.root, pypiVenvMetadata)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("proof published unapproved ownership", err)
			}
			update, err := prep.ResolveProjectLockUpdate(context.Background(), install, guard.Controls(), "")
			if err != nil {
				t.Fatal(err)
			}
			if len(update.SelectedControls()) != 5 || len(update.Snapshot().Dependencies()) != 1 || !sameUVSyncControls(update.OriginalControls(), guard.Controls()) {
				t.Fatal("original proof did not bind complete selection")
			}
			if _, ok := guard.transaction.current.Files["site/proof_beta/__init__.py"]; !ok {
				t.Fatal("excluded native owner was omitted from transaction")
			}
			if _, err := os.Lstat(filepath.Join(plan.root, pypiVenvMetadata)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("selection published ownership", err)
			}
		})
	}
}

func TestUVSyncSelectionRejectsDriftAndForeignControls(t *testing.T) {
	for _, mode := range []string{"source-before", "source-during", "metadata-during", "runtime-control", "lock-integrity", "foreign-target", "repeated-after-error", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			prep, install, resolver := newUVSyncGuardFixture(t, false)
			guard, err := prep.Begin(context.Background(), install)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := guard.Close(); err != nil {
					t.Error(err)
				}
			})
			ctx := context.Background()
			controls := guard.Controls()
			mutateSource := func() {
				if err := os.WriteFile(prep.source, []byte("haa-promotion-fixture==2.0.0\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			switch mode {
			case "source-before":
				mutateSource()
			case "source-during":
				resolver.before = mutateSource
			case "metadata-during":
				resolver.before = func() {
					for path := range guard.original.metadata {
						if err := os.WriteFile(path, []byte("changed"), 0o600); err != nil {
							t.Fatal(err)
						}
						break
					}
				}
			case "runtime-control":
				controls[0], _ = domain.NewProjectControlFile("uv-sync-runtime.json", []byte("forged"), true)
			case "lock-integrity":
				a := resolver.artifacts[0]
				id := a.Identity()
				resolver.artifacts[0], _ = domain.NewResolvedArtifact(id, a.AcquisitionLocator(), "sha256:"+strings.Repeat("b", 64))
				// The returned snapshot matches the forged lock, but the original exact
				// source still binds its version. Tamper the returned lock independently.
				resolver.mutate = func(u domain.ProjectDependencyUpdate) domain.ProjectDependencyUpdate {
					selected := u.SelectedControls()
					for i, c := range selected {
						if c.Name() == artifactpypi.UVSelectionLockControlName {
							selected[i], _ = domain.NewProjectControlFile(c.Name(), []byte(strings.ReplaceAll(string(c.Body()), strings.Repeat("b", 64), strings.Repeat("c", 64))), true)
						}
					}
					digests := []domain.ProjectControlDigest{}
					for _, c := range selected {
						d, _ := domain.NewProjectControlDigest(c.Name(), c.Digest())
						digests = append(digests, d)
					}
					snap, _ := domain.NewProjectDependencySnapshot(install, u.Snapshot().Source(), digests, u.Snapshot().Dependencies(), u.Snapshot().GraphDigest())
					next, e := domain.NewProjectDependencyUpdateFromSnapshot(u.OriginalControls(), selected, snap)
					if e != nil {
						t.Fatal(e)
					}
					return next
				}
			case "foreign-target":
				target, _ := domain.NewInstallTarget(realPromotionRoot(t))
				install, _ = domain.NewPythonVenvInstallContext(target)
			case "repeated-after-error":
				resolver.err = errors.New("native first failure")
			case "cancel":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if _, err := prep.ResolveProjectLockUpdate(ctx, install, controls, ""); err == nil {
				t.Fatal("unsafe selection accepted")
			}
			if _, err := prep.SelectedUpdate(context.Background()); err == nil {
				t.Fatal("failed selection was retained")
			}
			if guard.transaction.backup != "" {
				t.Fatal("selection failure mutated original")
			}
			if mode == "repeated-after-error" {
				resolver.err = nil
				if _, err := prep.ResolveProjectLockUpdate(context.Background(), install, controls, ""); err == nil || resolver.calls != 1 {
					t.Fatal("failed native selection was retried")
				}
			}
		})
	}
}
