package bootstrap_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/application"
	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectioncargo "github.com/rahoney/heliopause/internal/inspection/cargo"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationcargo "github.com/rahoney/heliopause/internal/verification/cargo"
)

// This gate starts with an empty intake and authenticates public bytes twice.
// It qualifies cache materialization; SDK resolution/add/build are separate.
func TestCargoPublicVerifiedCacheIntegration(t *testing.T) {
	if os.Getenv("HELOX_CARGO_CACHE_INTEGRATION") != "1" {
		t.Skip("requires canonical public crates.io HTTPS")
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	intake, evidenceRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence")
	client, err := artifactcargo.NewPublicClient(intake)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := verificationcargo.NewIntegrityVerifier(intake)
	if err != nil {
		t.Fatal(err)
	}
	inspector, err := inspectioncargo.NewStaticInspector(intake)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := local.NewStore(evidenceRoot)
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewProjectInspectService(client, verifier, inspector, evidence, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := promotion.NewCargoVerifiedCache(intake, evidenceRoot, filepath.Join(root, "verified-cache"), evidence)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := os.ReadFile("../artifact/cargo/testdata/metadata-v1-rust-1.99.0.json")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := os.ReadFile("../artifact/cargo/testdata/metadata-v1-rust-1.99.0.lock")
	if err != nil {
		t.Fatal(err)
	}
	manifest := []byte("[package]\nname='haa_schema_fixture'\nversion='0.1.0'\nedition='2021'\n[dependencies]\nitoa='=1.0.17'\n")
	target, _ := domain.NewInstallTarget(filepath.Join(root, "project"))
	install, _ := domain.NewInstallContext(target)
	snapshot, err := artifactcargo.BuildProjectSnapshot(install, metadata, manifest, lock, "/fixture", sandbox.PinnedCargoRuntime().ImageReference, map[string][]byte{"Cargo.toml": manifest, "Cargo.lock": lock, "src/main.rs": []byte("fn main() {}\n")})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	approved, err := service.InspectProject(ctx, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !approved.Valid() || len(approved.Inspected().Inspections()) != 1 {
		t.Fatal("public project coverage incomplete")
	}
	entry := approved.Inspected().Inspections()[0]
	if entry.Artifact().Digest().String() != "92ecc6618181def0457392ccd0ee51198e065e016d1d527a7ac1b6dc7c1f09d2" || len(entry.Checks()) != 2 || len(entry.Evidence()) != 2 || entry.PolicyDecision().Decision() != domain.DecisionAllow {
		t.Fatal("public source/integrity/Evidence/Policy differs")
	}
	staged, err := cache.StageProject(ctx, approved)
	if err != nil {
		t.Fatal(err)
	}
	vendor, err := cache.OpenProjectCache(ctx, staged)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(vendor, "itoa-1.0.17", ".cargo-checksum.json"))
	if err != nil {
		t.Fatal(err)
	}
	var checksum struct {
		Files   map[string]string `json:"files"`
		Package string            `json:"package"`
	}
	if json.Unmarshal(body, &checksum) != nil || checksum.Package != entry.Artifact().Digest().String() || len(checksum.Files) == 0 {
		t.Fatal("verified vendor checksum incomplete")
	}
	for _, ref := range entry.Evidence() {
		if _, _, err := evidence.ReadReference(ctx, entry.RunID(), ref, entry.Artifact().Identity(), entry.Artifact().Digest()); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := cache.OpenProjectCache(ctx, staged); err != nil {
		t.Fatal(err)
	}
	t.Logf("public_crates=%d recorded_evidence=%d vendor_files=%d entry_policy=%s receipt_sha256=%s", len(snapshot.Dependencies()), len(entry.Evidence()), len(checksum.Files), entry.PolicyDecision().Decision(), staged.Digest())
}
