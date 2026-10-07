package promotion

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGoCacheRequiresRecordedEvidenceAtBothBoundaries(t *testing.T) {
	for _, stage := range []string{"before-stage", "before-open"} {
		t.Run(stage, func(t *testing.T) {
			root := canonicalGoTestRoot(t)
			cache, err := newGoCacheForTest(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache"))
			if err != nil {
				t.Fatal(err)
			}
			set, _ := goCacheFixture(t, cache.intakeRoot, filepath.Join(root, "project"), "example.com/module", map[string]string{"go.mod": "module example.com/module\n", "module.go": "package module\n"})
			if stage == "before-stage" {
				if err := os.RemoveAll(cache.evidenceRoot); err != nil {
					t.Fatal(err)
				}
				staged, err := cache.StageProject(context.Background(), set)
				if err == nil || staged.Valid() {
					t.Fatal("cache accepted missing recorded Evidence")
				}
				return
			}
			staged, err := cache.StageProject(context.Background(), set)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(cache.evidenceRoot); err != nil {
				t.Fatal(err)
			}
			if _, err := cache.OpenProjectCache(context.Background(), staged); err == nil {
				t.Fatal("cache reopened after required Evidence disappeared")
			}
		})
	}
}
