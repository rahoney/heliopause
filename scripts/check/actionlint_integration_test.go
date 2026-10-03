//go:build actionlint

package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These fixtures exercise the same pinned tool invocation used by canonical
// workflow/quick; default unit tests need neither the tool nor a private cache.
func TestActionlintContextAvailability(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	c, err := newChecker(root, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(root, workflowRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	valid := "on: push\njobs:\n  test:\n    runs-on: ubuntu-24.04\n    steps:\n      - run: echo ok\n        env:\n          ANY_NAME: ${{ runner.temp }}/wheel-corpus\n"
	fixtures := []struct {
		name, body string
		fail       bool
	}{
		{"corrected full workflow", string(body), false},
		{"valid step context", valid, false},
		{"original corpus violation", strings.Replace(string(body), "      GOTOOLCHAIN: local", "      GOTOOLCHAIN: local\n      HELOX_CORPUS_ROOT: ${{ runner.temp }}/wheel-corpus", 1), true},
		{"renamed reordered job env", "on: push\njobs:\n  other:\n    env:\n      RENAMED: ${{ runner.temp }}\n      Z: value\n    runs-on: ubuntu-24.04\n    steps:\n      - run: echo ok\n", true},
		{"different job and key order", "on: push\njobs:\n  another:\n    runs-on: ubuntu-24.04\n    env:\n      Z: value\n      ARBITRARY: ${{ runner.temp }}\n    steps:\n      - run: echo ok\n", true},
		{"disallowed step context at job if", "on: push\njobs:\n  test:\n    if: ${{ steps.build.outcome == 'success' }}\n    runs-on: ubuntu-24.04\n    steps:\n      - run: echo ok\n", true},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "ci.yml")
			if err := os.WriteFile(path, []byte(fixture.body), 0o600); err != nil {
				t.Fatal(err)
			}
			err := c.lintActionsFiles(path)
			if fixture.fail {
				if err == nil || !strings.Contains(err.Error(), "context") || !strings.Contains(err.Error(), "not allowed here") {
					t.Fatalf("want context rejection, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
