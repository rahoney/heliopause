package promotion

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
	evidencelocal "github.com/rahoney/heliopause/internal/evidence/local"
)

func TestGoCacheStagesApprovedSubjectAndRechecksWholeTree(t *testing.T) {
	for _, test := range []string{"normal", "intake-tamper", "extracted-tamper", "zip-tamper", "receipt-tamper", "extra-file", "extra-directory", "symlink", "foreign-approval"} {
		t.Run(test, func(t *testing.T) {
			root := canonicalGoTestRoot(t)
			cache, err := newGoCacheForTest(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "verified"))
			if err != nil {
				t.Fatal(err)
			}
			set, intake := goCacheFixture(t, cache.intakeRoot, filepath.Join(root, "project"), "example.com/module", map[string]string{"go.mod": "module example.com/module\ngo 1.26.0\n", "module.go": "package module\nfunc Value() int {return 7}\n"})
			if test == "intake-tamper" {
				if err := os.WriteFile(intake, []byte("tampered"), 0o600); err != nil {
					t.Fatal(err)
				}
				staged, err := cache.StageProject(context.Background(), set)
				if err == nil || staged.Valid() {
					t.Fatal("tampered intake staged")
				}
				entries, err := os.ReadDir(cache.cacheRoot)
				if err != nil || len(entries) != 0 {
					t.Fatalf("partial cache retained: %v", err)
				}
				return
			}
			staged, err := cache.StageProject(context.Background(), set)
			if err != nil {
				t.Fatal(err)
			}
			dir, err := cache.OpenProjectCache(context.Background(), staged)
			if err != nil {
				t.Fatal(err)
			}
			extracted, download, err := artifactgo.ModuleCachePaths(set.Inspected().Snapshot().Dependencies()[0].Identity())
			if err != nil {
				t.Fatal(err)
			}
			if test == "normal" {
				body, err := os.ReadFile(filepath.Join(dir, extracted, "module.go"))
				if err != nil || !bytes.Contains(body, []byte("return 7")) {
					t.Fatal("wrong module materialized")
				}
				info, err := os.Stat(filepath.Join(dir, extracted, "module.go"))
				if err != nil || info.Mode().Perm()&0o222 != 0 {
					t.Fatal("cache source is writable")
				}
				return
			}
			name := filepath.Join(dir, extracted, "module.go")
			switch test {
			case "extracted-tamper":
				if err := os.Chmod(name, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(name, []byte("package module\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "zip-tamper":
				name = filepath.Join(dir, download+".zip")
				if err := os.Chmod(name, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(name, []byte("tampered"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "receipt-tamper":
				if err := os.WriteFile(filepath.Join(filepath.Dir(dir), goCacheReceipt), []byte("{}"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "extra-file":
				if err := os.WriteFile(filepath.Join(dir, "ambient.go"), []byte("poison"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "extra-directory":
				if err := os.Mkdir(filepath.Join(dir, "ambient"), 0o700); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Remove(name); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(intake, name); err != nil {
					t.Fatal(err)
				}
			case "foreign-approval":
				other, _ := goCacheFixture(t, filepath.Join(root, "other-intake"), filepath.Join(root, "other-project"), "example.com/module", map[string]string{"go.mod": "module example.com/module\ngo 1.26.0\n", "module.go": "package module\nfunc Value() int {return 7}\n"})
				staged, err = domain.NewStagedProjectSet(other, staged.ContentHandle(), staged.Digest())
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, err := cache.OpenProjectCache(context.Background(), staged); err == nil {
				t.Fatal("changed/foreign cache reopened")
			}
		})
	}
}

func TestGoCacheRejectsUnapprovedAndCanonicalArchiveCollision(t *testing.T) {
	root := canonicalGoTestRoot(t)
	cache, err := newGoCacheForTest(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "verified"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cache.StageProject(context.Background(), domain.ProjectVerifiedSet{}); err == nil {
		t.Fatal("unapproved cache staged")
	}
	if _, err := newGoCacheForTest(cache.intakeRoot, cache.evidenceRoot, filepath.Join(cache.intakeRoot, "cache")); err == nil {
		t.Fatal("overlapping cache accepted")
	}
	set, _ := goCacheFixture(t, cache.intakeRoot, filepath.Join(root, "project"), "example.com/module", map[string]string{"go.mod": "module example.com/module\n", "a": "source", "a/b.go": "package a\n"})
	staged, err := cache.StageProject(context.Background(), set)
	if err == nil || staged.Valid() {
		t.Fatal("file/ancestor archive collision staged")
	}
	entries, err := os.ReadDir(cache.cacheRoot)
	if err != nil || len(entries) != 0 {
		t.Fatal("partial collision cache retained")
	}
}

func TestGoCacheContentBudgetFailureDoesNotPublishPartialCache(t *testing.T) {
	root := canonicalGoTestRoot(t)
	cache, err := newGoCacheForTest(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "verified"))
	if err != nil {
		t.Fatal(err)
	}
	// An individually bounded, approved archive can still exceed the complete
	// cache budget once its required proxy metadata is included.
	files := map[string]string{"go.mod": "module example.com/module\ngo 1.26.0\n"}
	for n := 1; n < artifactgo.MaxModuleFiles; n++ {
		files[fmt.Sprintf("testdata/asset-%05d", n)] = ""
	}
	set, _ := goCacheFixture(t, cache.intakeRoot, filepath.Join(root, "project"), "example.com/module", files)
	staged, err := cache.StageProject(context.Background(), set)
	if err == nil || staged.Valid() || !strings.Contains(err.Error(), "go project cache exceeds bounded content limits:") || !strings.Contains(err.Error(), "file_limit=10000") {
		t.Fatalf("cache budget failure lost: staged=%v error=%v", staged.Valid(), err)
	}
	entries, err := os.ReadDir(cache.cacheRoot)
	if err != nil || len(entries) != 0 {
		t.Fatal("budget failure retained partial or published cache")
	}
}

// A unit-test approval fixture is deliberately distinct from actual SumDB /
// Application qualification; the public integration uses the real producer.
func goCacheFixture(t *testing.T, intakeRoot, project, modulePath string, files map[string]string) (domain.ProjectVerifiedSet, string) {
	t.Helper()
	var archive bytes.Buffer
	w := zip.NewWriter(&archive)
	for name, body := range files {
		f, err := w.Create(modulePath + "@v1.0.0/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	mod := []byte(files["go.mod"])
	body := []byte("HAA-GO-MODULE-1\n")
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(mod)))
	body = append(body, length[:]...)
	body = append(body, mod...)
	body = append(body, archive.Bytes()...)
	run, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(intakeRoot, run.String())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "module.bundle")
	if err := os.WriteFile(file, body, 0o600); err != nil {
		t.Fatal(err)
	}
	identity, err := domain.NewResolvedArtifactIdentity(artifactgo.Source(), modulePath, "v1.0.0", "module")
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(body)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(hash[:]))
	if err != nil {
		t.Fatal(err)
	}
	a, err := domain.NewAcquiredArtifact(identity, digest, "intake:"+run.String()+":module", uint64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := artifactgo.ReadIntake(intakeRoot, a)
	if err != nil {
		t.Fatal(err)
	}
	zSum, mSum, err := artifactgo.HashBundle(bundle, identity)
	if err != nil {
		t.Fatal(err)
	}
	integrity := "h1=" + zSum + ";go.mod=" + mSum
	a, err = domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, a.ContentHandle(), a.SizeBytes(), integrity)
	if err != nil {
		t.Fatal(err)
	}
	url, err := artifactgo.ProxyURL(modulePath, "v1.0.0", ".zip")
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := domain.NewResolvedArtifact(identity, url, integrity)
	if err != nil {
		t.Fatal(err)
	}
	target, err := domain.NewInstallTarget(project)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := domain.NewInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := domain.NewSHA256Digest(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	control, err := domain.NewProjectControlDigest("go.mod", graph)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := domain.NewProjectDependencySnapshot(ctx, artifactgo.Source(), []domain.ProjectControlDigest{control}, []domain.ResolvedArtifact{resolved}, graph)
	if err != nil {
		t.Fatal(err)
	}
	node, err := domain.ProjectDependencyNodeID(identity)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := domain.NewPolicyDecision(domain.DecisionAllow, "fixture", 1, []string{"FIXTURE_ALLOW"})
	if err != nil {
		t.Fatal(err)
	}
	var checks []domain.CheckExecution
	for _, kind := range []domain.CheckKind{domain.CheckVerification, domain.CheckInspection} {
		id, err := domain.NewCheckID("fixture-" + strings.ToLower(string(kind)))
		if err != nil {
			t.Fatal(err)
		}
		check, err := domain.NewCheckExecution(id, kind, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		if err != nil {
			t.Fatal(err)
		}
		checks = append(checks, check)
	}
	store, err := evidencelocal.NewStore(filepath.Join(filepath.Dir(intakeRoot), "evidence"))
	if err != nil {
		t.Fatal(err)
	}
	var items []domain.Evidence
	for _, check := range checks {
		eID, err := domain.NewEvidenceID(check.ID().String())
		if err != nil {
			t.Fatal(err)
		}
		item, err := domain.NewEvidence(eID, check.ID(), identity, digest, "fixture", "Synthetic complete check evidence.")
		if err != nil {
			t.Fatal(err)
		}
		items = append(items, item)
	}
	refs, err := store.Record(context.Background(), run, items)
	if err != nil {
		t.Fatal(err)
	}
	i, err := domain.NewDependencyInspection(node, run, a, checks, refs, decision)
	if err != nil {
		t.Fatal(err)
	}
	set, err := domain.NewInspectedProjectSet(snapshot, []domain.DependencyInspection{i})
	if err != nil {
		t.Fatal(err)
	}
	approved, err := domain.NewProjectVerifiedSet(set, decision)
	if err != nil {
		t.Fatal(err)
	}
	return approved, file
}

func newGoCacheForTest(intake, evidence, cache string) (*GoVerifiedCache, error) {
	store, err := evidencelocal.NewStore(evidence)
	if err != nil {
		return nil, err
	}
	return NewGoVerifiedCache(intake, evidence, cache, store)
}

func canonicalGoTestRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root
}
