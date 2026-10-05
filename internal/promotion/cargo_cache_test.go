package promotion

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
	evidencelocal "github.com/rahoney/heliopause/internal/evidence/local"
)

// Synthetic ALLOW records exercise the staging boundary, not public registry
// authentication. Public acquisition/inspection has its own integration gate.
func cargoCacheFixture(t *testing.T, c *CargoVerifiedCache, project string, files map[string][]byte) (domain.ProjectVerifiedSet, string) {
	t.Helper()
	if files == nil {
		body, err := os.ReadFile("../artifact/cargo/testdata/itoa-1.0.17.crate")
		if err != nil {
			t.Fatal(err)
		}
		identity, _ := domain.NewResolvedArtifactIdentity(artifactcargo.Source(), "itoa", "1.0.17", "crate")
		files, err = artifactcargo.ArchiveFiles(body, identity)
		if err != nil {
			t.Fatal(err)
		}
	}
	var raw bytes.Buffer
	compressed := gzip.NewWriter(&raw)
	archive := tar.NewWriter(compressed)
	var names []string
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		body := files[name]
		if err := archive.WriteHeader(&tar.Header{Name: "itoa-1.0.17/" + name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := archive.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	run, _ := domain.NewRunID()
	dir := filepath.Join(c.intakeRoot, run.String())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "package.crate")
	if err := os.WriteFile(file, raw.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	identity, _ := domain.NewResolvedArtifactIdentity(artifactcargo.Source(), "itoa", "1.0.17", "crate")
	hash := sha256.Sum256(raw.Bytes())
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(hash[:]))
	artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+run.String()+":crate", uint64(raw.Len()), "sha256="+digest.String())
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := domain.NewResolvedArtifact(identity, "https://static.crates.io/crates/itoa/itoa-1.0.17.crate", "sha256="+digest.String())
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(project)
	install, _ := domain.NewInstallContext(target)
	graph, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	control, _ := domain.NewProjectControlDigest("Cargo.toml", graph)
	snapshot, err := domain.NewProjectDependencySnapshot(install, artifactcargo.Source(), []domain.ProjectControlDigest{control}, []domain.ResolvedArtifact{resolved}, graph)
	if err != nil {
		t.Fatal(err)
	}
	decision, _ := domain.NewPolicyDecision(domain.DecisionAllow, "fixture", 1, []string{"FIXTURE_ALLOW"})
	var checks []domain.CheckExecution
	var items []domain.Evidence
	for _, kind := range []domain.CheckKind{domain.CheckVerification, domain.CheckInspection} {
		id, _ := domain.NewCheckID("fixture-" + strings.ToLower(string(kind)))
		check, _ := domain.NewCheckExecution(id, kind, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		checks = append(checks, check)
		eID, _ := domain.NewEvidenceID(id.String())
		item, err := domain.NewEvidence(eID, id, identity, digest, "fixture", "Synthetic complete cache check evidence.")
		if err != nil {
			t.Fatal(err)
		}
		items = append(items, item)
	}
	store, err := evidencelocal.NewStore(c.evidenceRoot)
	if err != nil {
		t.Fatal(err)
	}
	refs, err := store.Record(context.Background(), run, items)
	if err != nil {
		t.Fatal(err)
	}
	node, _ := domain.ProjectDependencyNodeID(identity)
	inspection, err := domain.NewDependencyInspection(node, run, artifact, checks, refs, decision)
	if err != nil {
		t.Fatal(err)
	}
	inspected, err := domain.NewInspectedProjectSet(snapshot, []domain.DependencyInspection{inspection})
	if err != nil {
		t.Fatal(err)
	}
	set, err := domain.NewProjectVerifiedSet(inspected, decision)
	if err != nil {
		t.Fatal(err)
	}
	return set, file
}

func newCargoCacheForTest(t *testing.T) (*CargoVerifiedCache, string) {
	t.Helper()
	root := canonicalGoTestRoot(t)
	evidence := filepath.Join(root, "evidence")
	reader, err := evidencelocal.NewStore(evidence)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := NewCargoVerifiedCache(filepath.Join(root, "intake"), evidence, filepath.Join(root, "cache"), reader)
	if err != nil {
		t.Fatal(err)
	}
	return cache, root
}

func TestCargoCacheStagesApprovedCrateAndRejectsDrift(t *testing.T) {
	for _, scenario := range []string{"normal", "intake-tamper", "missing-evidence-stage", "missing-evidence-reuse", "file-tamper", "writable-file", "receipt-tamper", "extra-file", "extra-directory", "symlink", "hardlink", "foreign-project", "foreign-cache-handle"} {
		t.Run(scenario, func(t *testing.T) {
			cache, root := newCargoCacheForTest(t)
			set, intake := cargoCacheFixture(t, cache, filepath.Join(root, "project"), nil)
			if scenario == "intake-tamper" {
				if err := os.WriteFile(intake, []byte("tampered"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "missing-evidence-stage" {
				if err := os.RemoveAll(filepath.Join(cache.evidenceRoot, set.Inspected().Inspections()[0].RunID().String())); err != nil {
					t.Fatal(err)
				}
			}
			staged, err := cache.StageProject(context.Background(), set)
			if scenario == "intake-tamper" || scenario == "missing-evidence-stage" {
				if err == nil || staged.Valid() {
					t.Fatal("invalid approval/intake staged")
				}
				entries, readErr := os.ReadDir(cache.cacheRoot)
				if readErr != nil && !os.IsNotExist(readErr) || len(entries) != 0 {
					t.Fatal("partial cache retained")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			vendor, err := cache.OpenProjectCache(context.Background(), staged)
			if err != nil {
				t.Fatal(err)
			}
			crate := filepath.Join(vendor, "itoa-1.0.17")
			member := filepath.Join(crate, "Cargo.toml")
			if scenario == "normal" {
				body, err := os.ReadFile(filepath.Join(crate, ".cargo-checksum.json"))
				if err != nil {
					t.Fatal(err)
				}
				var checksum struct {
					Files   map[string]string `json:"files"`
					Package string            `json:"package"`
				}
				if json.Unmarshal(body, &checksum) != nil || checksum.Package != set.Inspected().Inspections()[0].Artifact().Digest().String() {
					t.Fatal("controller checksum package differs")
				}
				manifest, err := os.ReadFile(member)
				if err != nil {
					t.Fatal(err)
				}
				hash := sha256.Sum256(manifest)
				if checksum.Files["Cargo.toml"] != hex.EncodeToString(hash[:]) {
					t.Fatal("controller checksum file differs")
				}
				return
			}
			switch scenario {
			case "file-tamper", "writable-file":
				if err := os.Chmod(member, 0o600); err != nil {
					t.Fatal(err)
				}
				if scenario == "file-tamper" {
					if err := os.WriteFile(member, []byte("changed"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			case "receipt-tamper":
				if err := os.WriteFile(filepath.Join(filepath.Dir(vendor), cargoCacheReceipt), []byte("{}"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "extra-file":
				if err := os.WriteFile(filepath.Join(vendor, "unexpected"), []byte("poison"), 0o444); err != nil {
					t.Fatal(err)
				}
			case "extra-directory":
				if err := os.Mkdir(filepath.Join(vendor, "unexpected"), 0o755); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Remove(member); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(intake, member); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(member, filepath.Join(root, "outside-alias")); err != nil {
					t.Fatal(err)
				}
			case "missing-evidence-reuse":
				if err := os.RemoveAll(filepath.Join(cache.evidenceRoot, set.Inspected().Inspections()[0].RunID().String())); err != nil {
					t.Fatal(err)
				}
			case "foreign-project":
				other, _ := cargoCacheFixture(t, cache, filepath.Join(root, "other-project"), nil)
				staged, err = domain.NewStagedProjectSet(other, staged.ContentHandle(), staged.Digest())
				if err != nil {
					t.Fatal(err)
				}
			case "foreign-cache-handle":
				staged, err = domain.NewStagedProjectSet(set, "project-cache:"+set.Inspected().Inspections()[0].RunID().String(), staged.Digest())
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, err := cache.OpenProjectCache(context.Background(), staged); err == nil {
				t.Fatal("changed or foreign cache reopened")
			}
		})
	}
}

func TestCargoCacheControllerChecksumCollisionAndLimits(t *testing.T) {
	for _, name := range []string{".cargo-checksum.json", ".CARGO-CHECKSUM.JSON", ".cargo-checksum.json/poison"} {
		t.Run(name, func(t *testing.T) {
			cache, root := newCargoCacheForTest(t)
			files := map[string][]byte{"Cargo.toml": []byte("[package]\nname='itoa'\nversion='1.0.17'\n"), name: []byte("artifact authority")}
			set, _ := cargoCacheFixture(t, cache, filepath.Join(root, "project"), files)
			if staged, err := cache.StageProject(context.Background(), set); err == nil || staged.Valid() {
				t.Fatal("artifact checksum authority staged")
			}
			entries, err := os.ReadDir(cache.cacheRoot)
			if err != nil || len(entries) != 0 {
				t.Fatal("incomplete cache retained")
			}
		})
	}
	for _, count := range []int{maxCargoProjectCacheFiles - 1, maxCargoProjectCacheFiles} {
		t.Run(fmt.Sprintf("files-%d", count), func(t *testing.T) {
			cache, root := newCargoCacheForTest(t)
			files := map[string][]byte{"Cargo.toml": []byte("[package]\nname='itoa'\nversion='1.0.17'\n")}
			for i := 1; i < count; i++ {
				files[fmt.Sprintf("testdata/asset-%05d", i)] = nil
			}
			set, _ := cargoCacheFixture(t, cache, filepath.Join(root, "project"), files)
			staged, err := cache.StageProject(context.Background(), set)
			if count == maxCargoProjectCacheFiles {
				if err == nil || staged.Valid() {
					t.Fatal("aggregate file limit bypassed")
				}
				entries, err := os.ReadDir(cache.cacheRoot)
				if err != nil || len(entries) != 0 {
					t.Fatal("over-limit partial cache retained")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := cache.OpenProjectCache(context.Background(), staged); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCargoCacheRejectsMissingApprovalCancellationAndOverlappingRoots(t *testing.T) {
	cache, root := newCargoCacheForTest(t)
	if _, err := cache.StageProject(context.Background(), domain.ProjectVerifiedSet{}); err == nil {
		t.Fatal("missing approval staged")
	}
	if _, err := NewCargoVerifiedCache(cache.intakeRoot, cache.evidenceRoot, filepath.Join(cache.intakeRoot, "cache"), cache.evidence); err == nil {
		t.Fatal("overlapping roots accepted")
	}
	set, _ := cargoCacheFixture(t, cache, filepath.Join(root, "project"), nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := cache.StageProject(ctx, set); err == nil {
		t.Fatal("cancelled staging succeeded")
	}
}
