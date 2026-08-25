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

func TestRuntimeLockWorkflowRejectsCopiedIdentity(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(workflowRelativePath)))
	if err != nil {
		t.Fatal(err)
	}
	fixture := string(contents) + "\n# node:22.23.1-slim@sha256:6c74791e557ce11fc957704f6d4fe134a7bc8d6f5ca4403205b2966bd488f6b3\n"
	if findings := validateRuntimeLockWorkflow(root, fixture); len(findings) == 0 {
		t.Fatal("hand-copied runtime identity was accepted")
	}
}

func TestValidateReleaseWorkflowRejectsSecurityRegressions(t *testing.T) {
	t.Parallel()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(releaseWorkflowRelativePath)))
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}

	tests := map[string]string{
		"floating attestation action": strings.Replace(string(contents), "actions/attest@a1948c3f048ba23858d222213b7c278aabede763", "actions/attest@v4", 1),
		"write content permission":    strings.Replace(string(contents), "  contents: read", "  contents: write", 1),
		"PR trigger":                  strings.Replace(string(contents), "  push:\n", "  pull_request:\n  push:\n", 1),
		"non-tag trigger":             strings.Replace(string(contents), "      - 'v*'", "      - '*'", 1),
		"release publishing":          string(contents) + "\n      - run: gh release create $GITHUB_REF_NAME\n",
	}
	for name, fixture := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if findings := validateReleaseWorkflow(fixture); len(findings) == 0 {
				t.Fatal("validateReleaseWorkflow findings = none, want non-empty")
			}
		})
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
