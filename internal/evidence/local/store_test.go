package local

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestStoreRecordsOpaqueReference(t *testing.T) {
	t.Parallel()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	runID, _ := domain.ParseRunID("run_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	source, _ := domain.NewSourceID("npm")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "tiny", "1.2.3", "tarball")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	checkID, _ := domain.NewCheckID("npm-static-archive")
	evidenceID, _ := domain.NewEvidenceID("npm-static-result")
	evidence, err := domain.NewEvidence(evidenceID, checkID, identity, digest, "npm-static-archive", "Static archive inspection completed.")
	if err != nil {
		t.Fatal(err)
	}
	references, err := store.Record(context.Background(), runID, []domain.Evidence{evidence})
	if err != nil {
		t.Fatal(err)
	}
	if len(references) != 1 || references[0].Handle() != "evidence:"+runID.String()+":npm-static-result" {
		t.Fatalf("references = %#v", references)
	}
	path := filepath.Join(root, runID.String(), "npm-static-result.json")
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("record info = %#v, %v", info, err)
	}
	var document record
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(contents, &document); err != nil || document.RecordSHA256 == "" || document.RunID != runID.String() || document.SHA256 != digest.String() {
		t.Fatalf("record = %#v, %v", document, err)
	}
}

func TestStoreRejectsDuplicateBatchWithoutPublishingPartialRun(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	root := filepath.Join(parent, "evidence")
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	runID, evidence := storeFixture(t)
	if _, err := store.Record(context.Background(), runID, []domain.Evidence{evidence, evidence}); err == nil {
		t.Fatal("Record() accepted a duplicate Evidence batch")
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("duplicate batch published Evidence root: %v", err)
	}
}

func TestStoreRetainsCommittedRunWhenRetryIsRejected(t *testing.T) {
	t.Parallel()

	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	runID, evidence := storeFixture(t)
	if _, err := store.Record(context.Background(), runID, []domain.Evidence{evidence}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, runID.String(), evidence.ID().String()+".json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Record(context.Background(), runID, []domain.Evidence{evidence}); err == nil {
		t.Fatal("Record() overwrote a retained Evidence Run")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(before) {
		t.Fatalf("retained Evidence changed after rejected retry: %q, %v", after, err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != runID.String() {
		t.Fatalf("Evidence root entries after rejected retry = %#v, %v", entries, err)
	}
}

func TestStoreDeletesOnlyTrustedEvidenceRun(t *testing.T) {
	t.Parallel()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	runID, evidence := storeFixture(t)
	if _, err := store.Record(context.Background(), runID, []domain.Evidence{evidence}); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteRun(context.Background(), runID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, runID.String())); !os.IsNotExist(err) {
		t.Fatalf("Evidence Run remains after deletion: %v", err)
	}

	outside := filepath.Join(filepath.Dir(root), "outside")
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outside)
	linkRun, err := domain.ParseRunID("run_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, linkRun.String())); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteRun(context.Background(), linkRun); err == nil {
		t.Fatal("DeleteRun accepted a symlink target")
	}
	if _, err := os.Lstat(filepath.Join(root, linkRun.String())); err != nil {
		t.Fatalf("symlink target was removed after rejected deletion: %v", err)
	}
}

func storeFixture(t *testing.T) (domain.RunID, domain.Evidence) {
	t.Helper()
	runID, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	source, _ := domain.NewSourceID("npm")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "fixture", "1.2.3", "tarball")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("b", 64))
	checkID, _ := domain.NewCheckID("fixture-check")
	evidenceID, _ := domain.NewEvidenceID("fixture-evidence")
	evidence, err := domain.NewEvidence(evidenceID, checkID, identity, digest, "fixture", "normalized evidence")
	if err != nil {
		t.Fatal(err)
	}
	return runID, evidence
}
