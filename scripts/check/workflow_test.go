package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryCIWorkflow(t *testing.T) {
	t.Parallel()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	if err := checkCIWorkflow(root); err != nil {
		t.Fatalf("checkCIWorkflow error: %v", err)
	}
}

func TestValidateCIWorkflowRejectsSecurityRegressions(t *testing.T) {
	t.Parallel()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(workflowRelativePath)))
	if err != nil {
		t.Fatalf("read workflow: %v", err)
	}

	tests := map[string]string{
		"floating action":        strings.Replace(string(contents), "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1", "actions/checkout@main", 1),
		"missing always":         strings.Replace(string(contents), "    if: ${{ always() }}\n", "", 1),
		"write token":            strings.Replace(string(contents), "  contents: read", "  contents: write", 1),
		"moving macOS runner":    strings.Replace(string(contents), "runs-on: macos-26-intel", "runs-on: macos-latest", 1),
		"missing minimum Go":     strings.Replace(string(contents), "go-version: '1.25.13'", "go-version: '1.26.7'", 1),
		"missing platform check": strings.ReplaceAll(string(contents), "run: go run ./scripts/check platform", "run: go test ./..."),
		"runner context at job env": strings.Replace(
			string(contents),
			"    env:\n      GOTOOLCHAIN: local",
			"    env:\n      HELOX_TOOL_CACHE: ${{ runner.temp }}/heliopause-quality-tools\n      GOTOOLCHAIN: local",
			1,
		),
		"extra job": string(contents) + "\n  security:\n    runs-on: ubuntu-24.04\n",
	}
	for name, fixture := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if findings := validateCIWorkflow(fixture); len(findings) == 0 {
				t.Fatal("validateCIWorkflow findings = none, want non-empty")
			}
		})
	}
}
