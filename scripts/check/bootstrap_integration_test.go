//go:build bootstrapintegration

package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

// Explicit network preparation into an empty inner cache, followed by the real
// offline consumer. The outer Go test compiler cache is deliberately separate.
func TestColdModulePreparationToOfflineTidy(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	c, err := newChecker(root, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	c.toolCache = t.TempDir()
	t.Cleanup(func() {
		if err := makeWritableRemoveAll(c.toolCache); err != nil {
			t.Error(err)
		}
	})
	t.Logf("outer GOMODCACHE=%s; inner cache=%s", os.Getenv("GOMODCACHE"), c.toolCache)
	beforeMod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	beforeSum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.prepareModuleCache(); err != nil {
		t.Fatal(err)
	}
	if err := c.downloadProductModules(); err != nil {
		t.Fatal(err)
	}
	if err := c.checkModuleDrift(); err != nil {
		t.Fatal(err)
	}
	for name, before := range map[string]string{"go.mod": string(beforeMod), "go.sum": string(beforeSum)} {
		after, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || string(after) != before {
			t.Fatalf("preparation changed %s: %v", name, err)
		}
	}
}
