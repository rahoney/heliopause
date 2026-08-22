package promotion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestM7QualificationRetainsExactManifestSBOMAndArtifactsThroughPromotion(t *testing.T) {
	t.Parallel()

	root := realPromotionRoot(t)
	bundle, staged := stagedPromotionFixture(t, root)
	stagedRoot := filepath.Join(root, "staging", bundle.ManifestID().String())
	assertExactRecord(t, filepath.Join(stagedRoot, manifestFilename), bundle.ManifestDocument())
	assertExactRecord(t, filepath.Join(stagedRoot, sbomFilename), bundle.SBOMDocument())
	assertBundleArtifacts(t, stagedRoot, bundle)

	target := filepath.Join(root, "published")
	promoter, err := newNPMPromotion(filepath.Join(root, "staging"), &fixturePromotionRunner{bundle: bundle}, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := promoter.Promote(context.Background(), staged, bundle, installContextFor(t, target)); err != nil {
		t.Fatal(err)
	}
	targetRecords := filepath.Join(target, ".heliopause")
	assertExactRecord(t, filepath.Join(targetRecords, manifestFilename), bundle.ManifestDocument())
	assertExactRecord(t, filepath.Join(targetRecords, sbomFilename), bundle.SBOMDocument())
	assertBundleArtifacts(t, targetRecords, bundle)
}

func assertExactRecord(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(want) {
		t.Fatalf("record %q differs from sealed bundle: %q, %v", path, got, err)
	}
}

func assertBundleArtifacts(t *testing.T, root string, bundle domain.VerifiedBundle) {
	t.Helper()
	seen := map[string]bool{}
	for _, inspection := range bundle.Set().Inspected().Inspections() {
		digest := inspection.Artifact().Digest().String()
		if seen[digest] {
			continue
		}
		seen[digest] = true
		contents, err := os.ReadFile(filepath.Join(root, "artifacts", digest+".tgz"))
		if err != nil {
			t.Fatalf("read staged artifact %q: %v", digest, err)
		}
		sum := sha256.Sum256(contents)
		if hex.EncodeToString(sum[:]) != digest {
			t.Fatalf("artifact digest = %x, want %s", sum, digest)
		}
	}
}
