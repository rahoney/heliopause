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
	fixture := string(contents) + "\n# node:24.21.0-slim@sha256:713cfbf4a0ac19f40e1bb9919893e126b74a5c8cf5d0623c9f89515c8f74c6fa\n"
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
		"missing release gate":        strings.Replace(string(contents), "go run ./scripts/check release-gate", "echo skipped", 1),
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

func TestValidateReleasePublishWorkflowRejectsSecurityRegressions(t *testing.T) {
	t.Parallel()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(releasePublishWorkflowRelativePath)))
	if err != nil {
		t.Fatalf("read release publication workflow: %v", err)
	}

	tests := map[string]string{
		"floating action":          strings.Replace(string(contents), "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1", "actions/checkout@main", 1),
		"push trigger":             strings.Replace(string(contents), "  workflow_dispatch:\n", "  push:\n    tags:\n      - 'v*'\n  workflow_dispatch:\n", 1),
		"clobber":                  string(contents) + "\n      --clobber\n",
		"missing attestation":      strings.Replace(string(contents), "          gh attestation verify", "          echo skipped", 1),
		"missing main provenance":  strings.Replace(string(contents), "repos/$GH_REPO/compare/$tag_sha...$main_sha", "repos/$GH_REPO/compare/skipped", 1),
		"missing Required success": strings.Replace(string(contents), ".name == \"Required\" and .status == \"completed\" and .conclusion == \"success\" and .app.slug == \"github-actions\"", ".name == \"Skipped\"", 1),
		"missing draft binding":    strings.Replace(string(contents), "Verify draft release asset bindings before publication", "Verify skipped release assets", 1),
		"missing quarantine":       strings.Replace(string(contents), "PUBLISHED_BUT_QUARANTINED", "PUBLISHED", 1),
		"missing exact candidate asset": strings.Replace(
			string(contents),
			"            helox-release-sbom.cdx.json\n",
			"",
			1,
		),
		"draft verification after publication": strings.Replace(
			strings.Replace(string(contents), "Verify draft release asset bindings before publication", "Draft binding skipped", 1),
			"          gh release edit \"$RELEASE_TAG\" -R \"$GH_REPO\" --draft=false",
			"          gh release edit \"$RELEASE_TAG\" -R \"$GH_REPO\" --draft=false\n          # Verify draft release asset bindings before publication",
			1,
		),
	}
	for name, fixture := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if findings := validateReleasePublishWorkflow(fixture); len(findings) == 0 {
				t.Fatal("validateReleasePublishWorkflow findings = none, want non-empty")
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
		"floating action":                             strings.Replace(string(contents), "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1", "actions/checkout@main", 1),
		"missing always":                              strings.Replace(string(contents), "    if: ${{ always() }}\n", "", 1),
		"write token":                                 strings.Replace(string(contents), "  contents: read", "  contents: write", 1),
		"moving macOS runner":                         strings.Replace(string(contents), "runs-on: macos-26-intel", "runs-on: macos-latest", 1),
		"missing minimum Go":                          strings.ReplaceAll(string(contents), "go-version: '1.26.8'", "go-version: '1.26.7'"),
		"missing pinned Go image":                     strings.Replace(string(contents), `docker pull "$go_image"`, `echo skipped`, 1),
		"Go build integration silently skipped":       strings.Replace(string(contents), "HELOX_GO_BUILD_CLI_INTEGRATION=1", "HELOX_GO_BUILD_CLI_INTEGRATION=0", 1),
		"Go source integration silently skipped":      strings.ReplaceAll(string(contents), "HELOX_GO_RESOLVER_INTEGRATION=1", "HELOX_GO_RESOLVER_INTEGRATION=0"),
		"missing pinned Rust image":                   strings.Replace(string(contents), `docker pull "$rust_image"`, `echo skipped`, 1),
		"Cargo resolver integration silently skipped": strings.ReplaceAll(string(contents), "HELOX_CARGO_RESOLVER_INTEGRATION=1", "HELOX_CARGO_RESOLVER_INTEGRATION=0"),
		"Cargo add integration silently skipped":      strings.Replace(string(contents), "HELOX_CARGO_ADD_INTEGRATION=1", "HELOX_CARGO_ADD_INTEGRATION=0", 1),
		"Cargo build integration silently skipped":    strings.Replace(string(contents), "HELOX_CARGO_BUILD_CLI_INTEGRATION=1", "HELOX_CARGO_BUILD_CLI_INTEGRATION=0", 1),
		"missing platform check":                      strings.ReplaceAll(string(contents), "run: go run ./scripts/check platform", "run: go test ./..."),
		"missing CUDA single-selection guard":         strings.Replace(string(contents), "select at most one CUDA PyTorch qualification profile", "CUDA profiles unchecked", 1),
		"missing CUDA strict freshness gate":          strings.Replace(string(contents), "go run ./scripts/check qualification-freshness", "echo skipped", 1),
		"legacy CUDA profile":                         string(contents) + "\n# HELOX_PYTORCH_PROFILE=cu128\n",
		"sidecar fallback":                            strings.Replace(string(contents), "--sidecar-usage-policy=STRICT", "--sidecar-usage-policy=LEGACY_DEPRECATED_SLOW_EMBEDDED_FALLBACK", 1),
		"permissive sidecar download":                 strings.Replace(string(contents), "--download-sidecars=NEVER", "--download-sidecars=ALWAYS", 1),
		"extra job":                                   string(contents) + "\n  security:\n    runs-on: ubuntu-24.04\n",
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

func TestCargoRequiredGatesMustBeActiveInTheirLifecycleOwner(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", workflowRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(contents), "\n") {
		command := strings.TrimSpace(line)
		if !strings.HasPrefix(command, "HELOX_CARGO_") && command != `docker pull "$rust_image"` {
			continue
		}
		t.Run(command, func(t *testing.T) {
			// A shell comment inside a valid YAML run block must not satisfy a gate.
			fixture := strings.Replace(string(contents), line, "          # "+command, 1)
			if len(validateCIWorkflow(fixture)) == 0 {
				t.Fatal("commented Cargo gate satisfied Required contract")
			}
		})
	}
}

func TestCargoIntegrationRejectsSkippedOrSubstitutedOwner(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", workflowRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	for name, fixture := range map[string]string{
		"wrong step":        strings.Replace(source, "- name: Install pinned runtime and run lifecycle integration", "- name: Cargo commands outside lifecycle owner", 1),
		"conditional step":  strings.Replace(source, "- name: Install pinned runtime and run lifecycle integration", "- name: Install pinned runtime and run lifecycle integration\n        if: false", 1),
		"ignored failure":   strings.Replace(source, "- name: Install pinned runtime and run lifecycle integration", "- name: Install pinned runtime and run lifecycle integration\n        continue-on-error: true", 1),
		"duplicate command": strings.Replace(source, `          docker pull "$rust_image"`, `          docker pull "$rust_image"`+"\n"+`          docker pull "$rust_image"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if len(validateCIWorkflow(fixture)) == 0 {
				t.Fatal("inactive or duplicate Cargo gate accepted")
			}
		})
	}
}
