package promotion

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence"
	evidencelocal "github.com/rahoney/heliopause/internal/evidence/local"
)

type uvSyncPublicationFixture struct {
	preparation *UVSyncPreparation
	publication *UVSyncProjectPromotion
	cache       *PythonProjectCache
	guard       ports.ProjectMutationGuard
	update      domain.ProjectDependencyUpdate
	set         domain.ProjectVerifiedSet
	seed        uvUpdateGuardFixture
}

func uvSyncApprovedFixture(t *testing.T, empty bool) uvSyncPublicationFixture {
	t.Helper()
	f := newUVUpdateGuardFixture(t, artifactpypi.UVUpgradeSelection, false)
	intake, evidenceRoot, staging := filepath.Join(f.records, "intake"), filepath.Join(f.records, "evidence"), filepath.Join(f.records, "staging")
	store, err := evidencelocal.NewStore(evidenceRoot)
	if err != nil {
		t.Fatal(err)
	}
	operation, _ := domain.NewOperationID()
	cache, err := NewPythonProjectCache(intake, evidenceRoot, staging, store, evidence.Generator{}, operation)
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(realPromotionRoot(t), "requirements.txt")
	body := "haa-promotion-fixture==1.0.0\n"
	if empty {
		body = "# explicit empty\n"
	}
	if err := os.WriteFile(source, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	r := &uvSyncResolverFixture{t: t, sizes: map[string]int64{}}
	if !empty {
		a := f.seed.Set().Inspected().Graph().Nodes()[0].Artifact()
		r.artifacts = []domain.ResolvedArtifact{a}
		r.sizes[a.Identity().Name()] = int64(f.seed.Set().Inspected().Inspections()[0].Artifact().SizeBytes())
	}
	prep, err := NewUVSyncPreparation(f.preparation.shared.promoter, r, source, artifactpypi.UVSyncRequirements, empty, false, []string{intake, evidenceRoot, staging})
	if err != nil {
		t.Fatal(err)
	}
	p, err := NewUVSyncProjectPromotion(prep, cache)
	if err != nil {
		t.Fatal(err)
	}
	guard, err := p.Begin(context.Background(), f.install)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := guard.Close(); err != nil {
			t.Error(err)
		}
	})
	update, err := prep.ResolveProjectLockUpdate(context.Background(), f.install, guard.Controls(), "")
	if err != nil {
		t.Fatal(err)
	}
	inspections := []domain.DependencyInspection{}
	decision, err := domain.NewPolicyDecision(domain.DecisionAllow, "synthetic-project-policy", 1, []string{"SYNTHETIC_ALLOW"})
	if err != nil {
		t.Fatal(err)
	}
	if !empty {
		old := f.seed.Set().Inspected().Inspections()[0]
		a := old.Artifact()
		node, _ := domain.ProjectDependencyNodeID(a.Identity())
		var checks []domain.CheckExecution
		var items []domain.Evidence
		for _, kind := range []domain.CheckKind{domain.CheckVerification, domain.CheckInspection} {
			id, _ := domain.NewCheckID("synthetic-" + strings.ToLower(string(kind)))
			check, _ := domain.NewCheckExecution(id, kind, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
			checks = append(checks, check)
			eID, _ := domain.NewEvidenceID(id.String())
			item, err := domain.NewEvidence(eID, id, a.Identity(), a.Digest(), "synthetic", "Synthetic checker facts for transaction regression only.")
			if err != nil {
				t.Fatal(err)
			}
			items = append(items, item)
		}
		refs, err := store.Record(context.Background(), old.RunID(), items)
		if err != nil {
			t.Fatal(err)
		}
		i, err := domain.NewDependencyInspection(node, old.RunID(), a, checks, refs, decision)
		if err != nil {
			t.Fatal(err)
		}
		inspections = append(inspections, i)
		if err := os.WriteFile(filepath.Join(intake, old.RunID().String(), "filename"), []byte(f.seed.Set().Inspected().Graph().Nodes()[0].RecordPath()), 0o400); err != nil {
			t.Fatal(err)
		}
	}
	inspected, err := domain.NewInspectedProjectSet(update.Snapshot(), inspections)
	if err != nil {
		t.Fatal(err)
	}
	set, err := domain.NewProjectVerifiedSet(inspected, decision)
	if err != nil {
		t.Fatal(err)
	}
	return uvSyncPublicationFixture{prep, p, cache, guard, update, set, f}
}

func TestUVSyncApprovedCompletePublicationAndExplicitEmpty(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(map[bool]string{false: "exact-nonempty", true: "explicit-empty"}[empty], func(t *testing.T) {
			f := uvSyncApprovedFixture(t, empty)
			if _, _, _, ok := f.publication.PublishedRecords(); ok {
				t.Fatal("unpublished records reported success")
			}
			staged, err := f.publication.StageProject(context.Background(), f.set)
			if err != nil {
				t.Fatal(err)
			}
			root, material, err := f.cache.openProject(context.Background(), staged, f.update)
			if err != nil {
				t.Fatal(err)
			}
			var manifest map[string]any
			if json.Unmarshal(material.document, &manifest) != nil || manifest["schema"] != "helox.verified-project-manifest/v1" || manifest["dependency_free"] != empty || manifest["operation_id"] != f.cache.operation.String() {
				t.Fatal("complete manifest lost actual context")
			}
			if _, ok := manifest["primary"]; ok {
				t.Fatal("complete records invented primary")
			}
			if err := f.guard.CommitSnapshot(context.Background(), f.update.Snapshot(), staged); err == nil {
				t.Fatal("snapshot bypass accepted")
			}
			if err := f.guard.Commit(context.Background(), f.update, staged); err != nil {
				t.Fatal(err)
			}
			if f.seed.runner.calls != map[bool]int{true: 0, false: 1}[empty] {
				t.Fatal("empty sync materialized a fake wheel", f.seed.runner.calls)
			}
			if _, err := os.Lstat(filepath.Join(f.seed.plan.site, "other.py")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("excluded managed owner remained", err)
			}
			if _, err := os.Lstat(filepath.Join(f.seed.plan.site, "obsolete.py")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("old obsolete owner remained", err)
			}
			if empty {
				if _, err := os.Lstat(filepath.Join(f.seed.root, pypiVenvMetadata)); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("empty selection wrote unreadable empty ownership", err)
				}
			} else if _, err := os.Stat(filepath.Join(f.seed.plan.site, "haa_promotion_fixture", "__init__.py")); err != nil {
				t.Fatal("approved selected wheel was not installed", err)
			}
			id, m, s, ok := f.publication.PublishedRecords()
			if !ok || id != material.manifest || m != staged.ContentHandle()+":manifest.json" || s != staged.ContentHandle()+":sbom.cdx.json" {
				t.Fatal("published report is not bound to verified cache")
			}
			if _, err := os.Stat(filepath.Join(root, manifestFilename)); err != nil {
				t.Fatal(err)
			}
			if err := f.guard.Commit(context.Background(), f.update, staged); err == nil {
				t.Fatal("publication repeated")
			}
			if err := f.guard.Close(); err != nil {
				t.Fatal(err)
			}
			next, err := beginPyPIVenvTransactionWithDiscovery(f.seed.plan, discoverUVPythonVenv)
			if err != nil {
				t.Fatal("next mutation could not read resulting ownership", err)
			}
			if err := next.close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUVSyncCacheTamperAndLostEvidenceRefusePublication(t *testing.T) {
	for _, mode := range []string{"receipt", "manifest", "sbom", "archive", "extra-file", "extra-root", "hardlink", "evidence", "source", "original", "foreign-staged", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			f := uvSyncApprovedFixture(t, false)
			staged, err := f.publication.StageProject(context.Background(), f.set)
			if err != nil {
				t.Fatal(err)
			}
			root, _, err := f.cache.openProject(context.Background(), staged, f.update)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			path := ""
			switch mode {
			case "receipt":
				path = filepath.Join(filepath.Dir(root), "receipt.json")
			case "manifest":
				path = filepath.Join(root, manifestFilename)
			case "sbom":
				path = filepath.Join(root, sbomFilename)
			case "archive":
				path = filepath.Join(root, "artifacts", f.set.Inspected().Inspections()[0].Artifact().Digest().String()+".whl")
			case "extra-file":
				path = filepath.Join(root, "injected.py")
			case "extra-root":
				path = filepath.Join(filepath.Dir(root), "undeclared")
			case "hardlink":
				path = filepath.Join(root, manifestFilename)
				if err := os.Link(path, filepath.Join(realPromotionRoot(t), "alias")); err != nil {
					t.Fatal(err)
				}
				path = ""
			case "evidence":
				if err := os.RemoveAll(f.cache.storage.evidenceRoot); err != nil {
					t.Fatal(err)
				}
			case "source":
				path = f.preparation.source
			case "original":
				path = filepath.Join(f.seed.plan.site, "other.py")
			case "foreign-staged":
				digest, _ := domain.NewSHA256Digest(strings.Repeat("c", 64))
				staged, err = domain.NewStagedProjectSet(f.set, staged.ContentHandle(), digest)
				if err != nil {
					t.Fatal(err)
				}
			case "cancel":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if path != "" {
				if _, err := os.Lstat(path); err == nil {
					if err := os.Chmod(path, 0o600); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.WriteFile(path, []byte("tampered"), 0o600); err != nil {
					t.Fatal(err)
				}
				if strings.HasPrefix(path, root) || mode == "receipt" {
					if err := os.Chmod(path, 0o444); err != nil {
						t.Fatal(err)
					}
				}
			}
			if err := f.guard.Commit(ctx, f.update, staged); err == nil {
				t.Fatal("unsafe cache/original reached publication")
			}
			if f.seed.runner.calls != 0 {
				t.Fatal("invalid approval launched materialization")
			}
			if f.publication.guard.transaction.backup != "" {
				t.Fatal("cache failure mutated original")
			}
			if _, _, _, ok := f.publication.PublishedRecords(); ok {
				t.Fatal("failed publication reported success")
			}
		})
	}
}

func TestUVSyncStagingRequiresRecordedEvidenceAndExactSelectedSet(t *testing.T) {
	for _, mode := range []string{"missing-evidence", "intake-filename", "intake-archive", "foreign-snapshot", "repeat"} {
		t.Run(mode, func(t *testing.T) {
			f := uvSyncApprovedFixture(t, false)
			i := f.set.Inspected().Inspections()[0]
			switch mode {
			case "missing-evidence":
				if err := os.RemoveAll(f.cache.storage.evidenceRoot); err != nil {
					t.Fatal(err)
				}
			case "intake-filename":
				if err := os.Chmod(filepath.Join(f.cache.storage.intakeRoot, i.RunID().String(), "filename"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(f.cache.storage.intakeRoot, i.RunID().String(), "filename"), []byte("substituted-1.0-py3-none-any.whl"), 0o400); err != nil {
					t.Fatal(err)
				}
			case "intake-archive":
				if err := os.WriteFile(filepath.Join(f.cache.storage.intakeRoot, i.RunID().String(), "wheel.whl"), []byte("substituted"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "foreign-snapshot":
				snapshot := f.update.Snapshot()
				digest, _ := domain.NewSHA256Digest(strings.Repeat("d", 64))
				different, err := domain.NewProjectDependencySnapshot(snapshot.Context(), snapshot.Source(), snapshot.ControlDigests(), snapshot.Dependencies(), digest)
				if err != nil {
					t.Fatal(err)
				}
				inspected, err := domain.NewInspectedProjectSet(different, f.set.Inspected().Inspections())
				if err != nil {
					t.Fatal(err)
				}
				f.set, err = domain.NewProjectVerifiedSet(inspected, f.set.Decision())
				if err != nil {
					t.Fatal(err)
				}
			case "repeat":
				if _, err := f.publication.StageProject(context.Background(), f.set); err != nil {
					t.Fatal(err)
				}
			}
			if staged, err := f.publication.StageProject(context.Background(), f.set); err == nil || staged.Valid() {
				t.Fatal("unsafe or repeated staging accepted")
			}
			if f.seed.runner.calls != 0 {
				t.Fatal("staging executed artifact")
			}
		})
	}
}

func TestUVSyncCompletePublicationRollsBackAndPreservesUserFiles(t *testing.T) {
	for _, phase := range []string{"after-publish:site", "metadata", "sync", "reconcile"} {
		t.Run(phase, func(t *testing.T) {
			f := uvSyncApprovedFixture(t, false)
			user := filepath.Join(f.seed.root, "user.txt")
			if err := os.WriteFile(user, []byte("user-owned"), 0o600); err != nil {
				t.Fatal(err)
			}
			metadata := filepath.Join(f.seed.root, pypiVenvMetadata)
			before, err := os.ReadFile(metadata)
			if err != nil {
				t.Fatal(err)
			}
			g := f.publication.guard
			hit := false
			g.transaction.fault = func(p string) error {
				if p == phase && !hit {
					hit = true
					return errors.New("injected publication boundary")
				}
				return nil
			}
			staged, err := f.publication.StageProject(context.Background(), f.set)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.guard.Commit(context.Background(), f.update, staged); err == nil || !hit {
				t.Fatal("failure boundary did not refuse commit", phase, err)
			}
			after, err := os.ReadFile(metadata)
			if err != nil || string(before) != string(after) {
				t.Fatal("rollback lost original ownership", err)
			}
			if b, err := os.ReadFile(user); err != nil || string(b) != "user-owned" {
				t.Fatal("user file changed", err)
			}
			if _, err := os.Stat(filepath.Join(f.seed.plan.site, "other.py")); err != nil {
				t.Fatal("rollback lost excluded original", err)
			}
		})
	}
}
