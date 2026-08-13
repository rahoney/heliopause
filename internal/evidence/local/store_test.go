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
