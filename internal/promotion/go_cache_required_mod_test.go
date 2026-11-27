package promotion

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
)

func TestGoCacheIncludesOnlyApprovedRequiredModuleControls(t *testing.T) {
	for _, fault := range []string{"missing", "tampered"} {
		t.Run(fault, func(t *testing.T) {
			root := canonicalGoTestRoot(t)
			cache, err := newGoCacheForTest(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache"))
			if err != nil {
				t.Fatal(err)
			}
			set, _ := goCacheFixture(t, cache.intakeRoot, filepath.Join(root, "project"), "example.com/module", map[string]string{"go.mod": "module example.com/module\ngo 1.26.0\n", "module.go": "package module\n"}, artifactgo.RequiredModControl{Version: "v0.9.0", Body: []byte("module example.com/module\ngo 1.25\n")})
			staged, err := cache.StageProject(context.Background(), set)
			if err != nil {
				t.Fatal(err)
			}
			directory, err := cache.OpenProjectCache(context.Background(), staged)
			if err != nil {
				t.Fatal(err)
			}
			old := filepath.Join(directory, "cache/download/example.com/module/@v/v0.9.0.mod")
			info, err := os.Stat(old)
			if err != nil || info.Mode().Perm() != 0o444 {
				t.Fatal("required control is not sealed", err)
			}
			if len(staged.Set().Inspected().Snapshot().Dependencies()) != 1 {
				t.Fatal("metadata invented another selected module")
			}
			for _, path := range []string{"cache/download/example.com/module/@v/v0.9.0.zip", "example.com/module@v0.9.0"} {
				if _, err := os.Lstat(filepath.Join(directory, path)); !os.IsNotExist(err) {
					t.Fatal("old archive/source was installed")
				}
			}
			doc, err := cache.goCacheApproval(context.Background(), set)
			if err != nil || len(doc.Entries) != 1 || !strings.Contains(doc.Entries[0].Integrity, ";required-mods=v0.9.0:") {
				t.Fatal("approved context lost", err)
			}
			if fault == "missing" {
				if err := os.Remove(old); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.Chmod(old, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(old, []byte("unapproved metadata\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := cache.OpenProjectCache(context.Background(), staged); err == nil {
				t.Fatal("required control drift reused as approved cache")
			}
		})
	}
}
