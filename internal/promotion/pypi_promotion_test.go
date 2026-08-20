package promotion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreparePyPIProjectBindsExactWheelAndHashRequirement(t *testing.T) {
	t.Parallel()
	root := realPromotionRoot(t)
	bundle, _ := validStagedPyPIFixture(t, root)
	project := filepath.Join(root, "project")
	if err := os.Mkdir(project, 0o700); err != nil {
		t.Fatal(err)
	}
	requirements, expected, err := preparePyPIProject(project, filepath.Join(root, "staging", bundle.ManifestID().String()), bundle)
	if err != nil {
		t.Fatal(err)
	}
	if len(expected) != 1 || !strings.Contains(string(requirements), "haa-promotion-fixture==1.0.0 --hash=sha256:") {
		t.Fatalf("requirements=%q expected=%v", requirements, expected)
	}
	if _, err := os.Stat(filepath.Join(project, "wheels", "haa_promotion_fixture-1.0.0-py3-none-any.whl")); err != nil {
		t.Fatalf("exact wheel not copied: %v", err)
	}
}

func TestValidatePyPIOutputRejectsUnrecordedOrMismatchedOutput(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	site := filepath.Join(root, "site")
	dist := filepath.Join(site, "haa_promotion_fixture-1.0.0.dist-info")
	if err := os.MkdirAll(dist, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "METADATA"), []byte("Name: haa-promotion-fixture\nVersion: 1.0.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "RECORD"), []byte("../escape,sha256=AAAA,1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	expected := map[string]pypiExpected{"haa-promotion-fixture": {name: "haa-promotion-fixture", version: "1.0.0"}}
	if err := validatePyPIOutput(site, expected, []byte("haa-promotion-fixture==1.0.0 --hash=sha256:abc\n")); err == nil {
		t.Fatal("validatePyPIOutput accepted unsafe RECORD")
	}
}
