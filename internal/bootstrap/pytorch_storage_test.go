package bootstrap_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPyTorchQualificationCacheIsFreshAndSeparate(t *testing.T) {
	project := t.TempDir()
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	first, err := pytorchIntegrationCache(project, "cu126", parent)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(first, "retained"), []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := pytorchIntegrationCache(project, "cu126", parent)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(second)
	if err != nil || len(entries) != 0 || first == second || filepath.Dir(first) != parent || filepath.Dir(second) != parent {
		t.Fatalf("qualification reused or escaped cache: %v", err)
	}
	info, err := os.Stat(second)
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Fatal("qualification cache is not private", err)
	}
	body, err := os.ReadFile(filepath.Join(first, "retained"))
	if err != nil || string(body) != "original" {
		t.Fatal("earlier cache changed", err)
	}
	if _, err := os.Stat(filepath.Join(project, "cache")); !os.IsNotExist(err) {
		t.Fatal("separate cache mutated project root", err)
	}
}

func TestPyTorchQualificationCacheRejectsUntrustedParent(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(parent, alias); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"relative", parent + "/../parent", alias, alias + "/missing", filepath.Join(root, "missing")} {
		if _, err := pytorchIntegrationCache(root, "cu126", path); err == nil {
			t.Errorf("untrusted qualification cache parent accepted: %s", path)
		}
	}
	if err := os.Chmod(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := pytorchIntegrationCache(root, "cu126", parent); err == nil {
		t.Fatal("non-private qualification cache parent accepted")
	}
	cache, err := pytorchIntegrationCache(root, "cu126", "")
	if err != nil || cache != filepath.Join(root, "cache") {
		t.Fatal("default qualification cache changed", err)
	}
}
