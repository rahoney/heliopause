package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const workflowRelativePath = ".github/workflows/heliopause-ci.yml"
const securityWorkflowRelativePath = ".github/workflows/heliopause-security.yml"
const releaseWorkflowRelativePath = ".github/workflows/heliopause-release-build.yml"
const releasePublishWorkflowRelativePath = ".github/workflows/heliopause-release-publish.yml"

var actionReference = regexp.MustCompile(`(?m)^\s*uses:\s*(\S+)\s*(?:#.*)?$`)

func checkCIWorkflow(root string) error {
	if err := checkWorkflow(root, workflowRelativePath, validateCIWorkflow); err != nil {
		return err
	}
	if err := checkWorkflow(root, securityWorkflowRelativePath, validateSecurityWorkflow); err != nil {
		return err
	}
	if err := checkWorkflow(root, releaseWorkflowRelativePath, validateReleaseWorkflow); err != nil {
		return err
	}
	return checkWorkflow(root, releasePublishWorkflowRelativePath, validateReleasePublishWorkflow)
}

func checkWorkflow(root, relativePath string, validate func(string) []string) error {
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return &checkFailure{class: unavailable, step: "CI configuration", detail: relativePath + " is not a regular file"}
	}
	if info.Size() > outputLimit {
		return &checkFailure{class: executionFailure, step: "CI configuration", detail: "workflow exceeds size limit"}
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return &checkFailure{class: executionFailure, step: "CI configuration", cause: err}
	}
	findings := validate(string(contents))
	if relativePath == workflowRelativePath || relativePath == releaseWorkflowRelativePath {
		findings = append(findings, validateRuntimeLockWorkflow(root, string(contents))...)
	}
	if len(findings) != 0 {
		return &checkFailure{class: findingFailure, step: "CI configuration", detail: strings.Join(findings, "\n")}
	}
	return nil
}

// validateRuntimeLockWorkflow prevents CI from becoming a second owner of an
// exact runtime identity. The workflow may name lock keys, never their values.
func validateRuntimeLockWorkflow(root, contents string) []string {
	body, err := os.ReadFile(filepath.Join(root, "scripts", "runtimes.lock.json"))
	if err != nil {
		return []string{"cannot read canonical runtime lock for workflow validation"}
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return []string{"cannot parse canonical runtime lock for workflow validation"}
	}
	var findings []string
	for _, identity := range runtimeLockStrings(value) {
		if len(identity) >= 12 && strings.Contains(contents, identity) {
			findings = append(findings, fmt.Sprintf("workflow hand-copies runtime lock identity %q", identity))
		}
	}
	sort.Strings(findings)
	return findings
}

func runtimeLockStrings(value any) []string {
	var values []string
	var visit func(any)
	visit = func(current any) {
		switch typed := current.(type) {
		case map[string]any:
			for _, child := range typed {
				visit(child)
			}
		case []any:
			for _, child := range typed {
				visit(child)
			}
		case string:
			values = append(values, typed)
		}
	}
	visit(value)
	return values
}

func validateCIWorkflow(contents string) []string {
	var findings []string
	requiredSnippets := []string{
		"name: Heliopause CI",
		"  pull_request:",
		"  push:\n    branches:\n      - main",
		"  merge_group:",
		"  workflow_dispatch:",
		"permissions:\n  contents: read",
		"runs-on: ubuntu-24.04",
		"runs-on: macos-26-intel",
		"persist-credentials: false",
		"go-version: '1.26.7'",
		"go-version: '1.25.13'",
		".docker.ci_ubuntu_24_04_amd64.docker_ce_package",
		".docker.ci_ubuntu_24_04_amd64.containerd_package",
		"test \"$(docker version --format '{{.Server.Version}}')\" = \"$docker_engine_version\"",
		"runtime_lock=scripts/runtimes.lock.json",
		"jq -er",
		"HELOX_PROMOTION_INTEGRATION=1 go test -v -timeout=5m ./internal/promotion -run TestLinuxNPMPromotionIntegration",
		"check-latest: false",
		"cache: false",
		"    if: ${{ always() }}",
		"    needs:\n      - quick\n      - docs\n      - security\n      - vulnerability\n      - minimum-go\n      - macos\n      - gvisor-observer\n      - gvisor-integration",
		"run: go run ./scripts/check bootstrap-modules",
		"run: go run ./scripts/check platform",
		"run: go run ./scripts/check security",
		"run: go run ./scripts/check vulnerability",
		`run: go run ./scripts/check required "$QUICK_RESULT" "$DOCS_RESULT" "$SECURITY_RESULT" "$VULNERABILITY_RESULT" "$MINIMUM_GO_RESULT" "$MACOS_RESULT" "$GVISOR_OBSERVER_RESULT" "$GVISOR_INTEGRATION_RESULT"`,
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(contents, snippet) {
			findings = append(findings, fmt.Sprintf("missing required workflow contract %q", snippet))
		}
	}

	forbidden := []string{
		"pull_request_target:",
		"paths:",
		"paths-ignore:",
		"secrets:",
		"permissions: write",
		"contents: write",
		"@main",
		"@master",
		"@latest",
		"ubuntu-latest",
		"    env:\n      HELOX_TOOL_CACHE: ${{ runner.temp }}",
		"5ceb9a5fd5750d6c73dd166441f28306039300d0",
		"4463ce276e207f5a516a08ec627a768a19cf7bed0094d522b0810bee3424585caa8d344e093204012b974f5c508ab2362dcb0d7236f0c1992fccc426beeb7ffc",
		"c876a1619c885f44f3bdc87998eca59c79581954631c9d7fab4eb53cc0409b68e4be74c08ef3fe599c51b75d56262070f0c314f9908336221e7764fdf981b7f5",
		"node:22.23.1-slim@sha256:",
		"python:3.14.7-slim-bookworm@sha256:",
	}
	for _, token := range forbidden {
		if strings.Contains(contents, token) {
			findings = append(findings, fmt.Sprintf("forbidden workflow token %q", token))
		}
	}

	allowedActions := map[string]int{
		"actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1": 9,
		"actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e": 8,
	}
	actualActions := make(map[string]int)
	for _, match := range actionReference.FindAllStringSubmatch(contents, -1) {
		actualActions[match[1]]++
	}
	for action, count := range actualActions {
		if _, allowed := allowedActions[action]; !allowed {
			findings = append(findings, fmt.Sprintf("unapproved action reference %q", action))
		}
		if strings.Count(action, "@") != 1 || len(strings.SplitN(action, "@", 2)[1]) != 40 {
			findings = append(findings, fmt.Sprintf("action reference is not pinned to a full SHA: %q", action))
		}
		_ = count
	}
	for action, expectedCount := range allowedActions {
		if actualActions[action] != expectedCount {
			findings = append(findings, fmt.Sprintf("action %q appears %d times, require %d", action, actualActions[action], expectedCount))
		}
	}

	jobs := workflowJobIDs(contents)
	wantJobs := []string{"docs", "gvisor-integration", "gvisor-observer", "macos", "minimum-go", "quick", "required", "security", "vulnerability"}
	if strings.Join(jobs, ",") != strings.Join(wantJobs, ",") {
		findings = append(findings, fmt.Sprintf("workflow jobs are %q, require %q", jobs, wantJobs))
	}
	sort.Strings(findings)
	return findings
}

func validateSecurityWorkflow(contents string) []string {
	var findings []string
	for _, snippet := range []string{
		"name: Heliopause Scheduled Security",
		"  schedule:\n    - cron: '17 3 * * 1'",
		"  workflow_dispatch:",
		"permissions:\n  contents: read",
		"fetch-depth: 0",
		"persist-credentials: false",
		"runs-on: ubuntu-24.04",
		"go-version: '1.26.7'",
		"run: go run ./scripts/check security-history",
		"run: go run ./scripts/check vulnerability",
		"run: go run ./scripts/check fuzz",
	} {
		if !strings.Contains(contents, snippet) {
			findings = append(findings, fmt.Sprintf("missing required scheduled security workflow contract %q", snippet))
		}
	}
	for _, token := range []string{"pull_request:", "pull_request_target:", "push:", "secrets:", "contents: write", "@main", "@master", "@latest", "ubuntu-latest"} {
		if strings.Contains(contents, token) {
			findings = append(findings, fmt.Sprintf("forbidden scheduled security workflow token %q", token))
		}
	}
	actions := actionReference.FindAllStringSubmatch(contents, -1)
	if len(actions) != 2 || actions[0][1] != "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1" || actions[1][1] != "actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e" {
		findings = append(findings, "scheduled security workflow action allowlist or pin is unexpected")
	}
	if jobs := workflowJobIDs(contents); strings.Join(jobs, ",") != "scheduled-security" {
		findings = append(findings, fmt.Sprintf("scheduled security workflow jobs are %q, require %q", jobs, "scheduled-security"))
	}
	sort.Strings(findings)
	return findings
}

func validateReleaseWorkflow(contents string) []string {
	var findings []string
	for _, snippet := range []string{
		"name: Heliopause Release Build",
		"  push:\n    tags:\n      - 'v*'",
		"permissions:\n  contents: read\n  attestations: write\n  id-token: write",
		"runs-on: ubuntu-24.04",
		"GOTOOLCHAIN: local",
		"persist-credentials: false",
		"go-version: '1.26.7'",
		"check-latest: false",
		"cache: false",
		"go build -trimpath -buildvcs=true",
		"scripts/build-gvisor-observer-release.sh",
		"go run ./scripts/runtime-image-manifest",
		"docker buildx imagetools inspect \"$node_image\"",
		"docker buildx imagetools inspect \"$python_image\"",
		"go run ./scripts/release-manifest",
		"--workflow-run \"$GITHUB_SERVER_URL/$GITHUB_REPOSITORY/actions/runs/$GITHUB_RUN_ID\"",
		"--runtime-images \"$dist/helox-runtime-images.json\"",
		"actions/attest@a1948c3f048ba23858d222213b7c278aabede763",
		"actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02",
		"subject-path: '${{ runner.temp }}/helox-release/*'",
		"if-no-files-found: error",
	} {
		if !strings.Contains(contents, snippet) {
			findings = append(findings, fmt.Sprintf("missing required release workflow contract %q", snippet))
		}
	}
	for _, token := range []string{
		"pull_request:", "pull_request_target:", "workflow_dispatch:", "schedule:", "contents: write", "packages: write", "secrets:",
		"@main", "@master", "@latest", "ubuntu-latest", "actions/create-release", "gh release", "softprops/action-gh-release",
	} {
		if strings.Contains(contents, token) {
			findings = append(findings, fmt.Sprintf("forbidden release workflow token %q", token))
		}
	}
	allowedActions := map[string]int{
		"actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1":        1,
		"actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e":        1,
		"actions/attest@a1948c3f048ba23858d222213b7c278aabede763":          1,
		"actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02": 1,
	}
	actualActions := make(map[string]int)
	for _, match := range actionReference.FindAllStringSubmatch(contents, -1) {
		actualActions[match[1]]++
	}
	for action := range actualActions {
		if _, allowed := allowedActions[action]; !allowed {
			findings = append(findings, fmt.Sprintf("unapproved release action reference %q", action))
		}
		if strings.Count(action, "@") != 1 || len(strings.SplitN(action, "@", 2)[1]) != 40 {
			findings = append(findings, fmt.Sprintf("release action reference is not pinned to a full SHA: %q", action))
		}
	}
	for action, expectedCount := range allowedActions {
		if actualActions[action] != expectedCount {
			findings = append(findings, fmt.Sprintf("release action %q appears %d times, require %d", action, actualActions[action], expectedCount))
		}
	}
	if jobs := workflowJobIDs(contents); strings.Join(jobs, ",") != "build-and-attest" {
		findings = append(findings, fmt.Sprintf("release workflow jobs are %q, require %q", jobs, "build-and-attest"))
	}
	sort.Strings(findings)
	return findings
}

func validateReleasePublishWorkflow(contents string) []string {
	var findings []string
	for _, snippet := range []string{
		"name: Heliopause Verified Release Publish",
		"  workflow_dispatch:",
		"release_tag:",
		"source_run_id:",
		"permissions:\n  contents: write\n  attestations: read",
		"runs-on: ubuntu-24.04",
		"environment: release",
		"persist-credentials: false",
		"go-version: '1.26.7'",
		"gh run view \"$SOURCE_RUN_ID\" -R \"$GH_REPO\"",
		"test \"$(jq -er '.workflowName' <<<\"$run_json\")\" = 'Heliopause Release Build'",
		"gh run download \"$SOURCE_RUN_ID\" -R \"$GH_REPO\" -n \"$artifact_name\"",
		"sha256sum -c helox-release-checksums.txt",
		"gh attestation verify",
		"--signer-workflow \"$signer\"",
		"--source-digest",
		"go run ./scripts/check release-gate",
		"gh api --include \"repos/$GH_REPO/releases/tags/$RELEASE_TAG\"",
		"release already exists; refusing overwrite",
		"gh release create \"$RELEASE_TAG\"",
		"--verify-tag",
		"--draft",
		"gh release edit \"$RELEASE_TAG\" -R \"$GH_REPO\" --draft=false",
		"gh release verify \"$RELEASE_TAG\" -R \"$GH_REPO\"",
		"gh release verify-asset \"$RELEASE_TAG\"",
	} {
		if !strings.Contains(contents, snippet) {
			findings = append(findings, fmt.Sprintf("missing required release publication contract %q", snippet))
		}
	}
	for _, token := range []string{
		"push:", "pull_request:", "pull_request_target:", "schedule:", "workflow_run:",
		"contents: read", "packages: write", "secrets:", "@main", "@master", "@latest",
		"ubuntu-latest", "--clobber", "--target", "--force",
	} {
		if strings.Contains(contents, token) {
			findings = append(findings, fmt.Sprintf("forbidden release publication workflow token %q", token))
		}
	}
	allowedActions := map[string]int{
		"actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1": 1,
		"actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e": 1,
	}
	actualActions := make(map[string]int)
	for _, match := range actionReference.FindAllStringSubmatch(contents, -1) {
		actualActions[match[1]]++
	}
	for action := range actualActions {
		if _, allowed := allowedActions[action]; !allowed {
			findings = append(findings, fmt.Sprintf("unapproved release publication action reference %q", action))
		}
		if strings.Count(action, "@") != 1 || len(strings.SplitN(action, "@", 2)[1]) != 40 {
			findings = append(findings, fmt.Sprintf("release publication action reference is not pinned to a full SHA: %q", action))
		}
	}
	for action, expectedCount := range allowedActions {
		if actualActions[action] != expectedCount {
			findings = append(findings, fmt.Sprintf("release publication action %q appears %d times, require %d", action, actualActions[action], expectedCount))
		}
	}
	if jobs := workflowJobIDs(contents); strings.Join(jobs, ",") != "publish" {
		findings = append(findings, fmt.Sprintf("release publication workflow jobs are %q, require %q", jobs, "publish"))
	}
	sort.Strings(findings)
	return findings
}

func workflowJobIDs(contents string) []string {
	lines := strings.Split(contents, "\n")
	inJobs := false
	var jobs []string
	for _, line := range lines {
		if line == "jobs:" {
			inJobs = true
			continue
		}
		if !inJobs || !strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "    ") || !strings.HasSuffix(line, ":") {
			continue
		}
		identifier := strings.TrimSuffix(strings.TrimSpace(line), ":")
		if identifier != "" {
			jobs = append(jobs, identifier)
		}
	}
	sort.Strings(jobs)
	return jobs
}
