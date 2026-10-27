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

func retiredRuntimeImagePolicyToken(value string) bool {
	return value == "node:22.23.1-slim@sha256:" || value == "python:3.14.7-slim-bookworm@sha256:"
}

func checkCIWorkflow(root string) error {
	tools, err := readToolLock(filepath.Join(root, "scripts", "tools.lock.json"))
	if err != nil {
		return err
	}
	owners, err := runtimeVersionOwners(root, tools)
	if err != nil {
		return err
	}
	selected := owners["Go"]
	entries, err := os.ReadDir(filepath.Join(root, ".github", "workflows"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
			continue
		}
		path := ".github/workflows/" + name
		if path != workflowRelativePath && path != securityWorkflowRelativePath && path != releaseWorkflowRelativePath && path != releasePublishWorkflowRelativePath {
			return fmt.Errorf("unowned workflow version consumer %s", path)
		}
	}
	if err := checkWorkflow(root, workflowRelativePath, func(body string) []string { return validateCIWorkflow(body, selected) }); err != nil {
		return err
	}
	if err := checkWorkflow(root, securityWorkflowRelativePath, func(body string) []string { return validateSecurityWorkflow(body, selected) }); err != nil {
		return err
	}
	if err := checkWorkflow(root, releaseWorkflowRelativePath, func(body string) []string { return validateReleaseWorkflow(body, selected) }); err != nil {
		return err
	}
	return checkWorkflow(root, releasePublishWorkflowRelativePath, func(body string) []string { return validateReleasePublishWorkflow(body, selected) })
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
	if relativePath != workflowRelativePath {
		findings = append(findings, validateWorkflowStructure(string(contents), false)...)
	}
	if relativePath == workflowRelativePath || relativePath == releaseWorkflowRelativePath {
		findings = append(findings, validateRuntimeLockWorkflow(root, string(contents))...)
	}
	if relativePath == workflowRelativePath {
		script, err := os.ReadFile(filepath.Join(root, "scripts", "ci-pnpm-runtime.sh"))
		if err != nil {
			findings = append(findings, "PNPM runtime qualification script is unavailable")
		} else {
			findings = append(findings, validatePNPMRuntimeScript(string(script))...)
		}
		script, err = os.ReadFile(filepath.Join(root, "scripts", "ci-bun-runtime.sh"))
		if err != nil {
			findings = append(findings, "Bun runtime qualification script is unavailable")
		} else {
			findings = append(findings, validateBunRuntimeScript(string(script))...)
		}
		script, err = os.ReadFile(filepath.Join(root, "scripts", "ci-uv-tools.sh"))
		if err != nil {
			findings = append(findings, "UV tools qualification script is unavailable")
		} else {
			findings = append(findings, validateUVToolsScript(string(script))...)
		}
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
	var visit func(any, string)
	visit = func(current any, location string) {
		switch typed := current.(type) {
		case map[string]any:
			for key, child := range typed {
				next := location + "/" + key
				// These exact validated enum fields reference owner keys; they are not
				// copied runtime identities. Other fields retain the literal check.
				if next == "/pipx_tool/runtime_image" && child == "python_image" || next == "/bun_tool/runtime_image" && child == "node_image" {
					continue
				}
				visit(child, next)
			}
		case []any:
			for _, child := range typed {
				visit(child, location+"/[]")
			}
		case string:
			values = append(values, typed)
		}
	}
	visit(value, "")
	return values
}

func validateCIWorkflow(contents, selectedGo string) []string {
	findings := validateWorkflowStructure(contents, true)
	findings = append(findings, validatePlatformBuildCache(contents)...)
	findings = append(findings, validateGoWorkflowVersions(contents, selectedGo)...)
	findings = append(findings, validateCargoIntegrationGates(contents)...)
	findings = append(findings, validateLifecycleIntegrationCommands(contents, "UV", uvIntegrationCommands)...)
	findings = append(findings, validateLifecycleIntegrationCommands(contents, "PNPM", []string{"bash scripts/ci-pnpm-runtime.sh", "bash scripts/ci-pnpm-runtime.sh project"})...)
	findings = append(findings, validatePNPMInspectionClientOrder(contents)...)
	findings = append(findings, validatePNPMProjectClientOrder(contents)...)
	findings = append(findings, validateLifecycleIntegrationCommands(contents, "Bun", []string{"bash scripts/ci-bun-runtime.sh", "bash scripts/ci-bun-runtime.sh project"})...)
	findings = append(findings, validateBunInspectionClientOrder(contents)...)
	findings = append(findings, validateBunProjectClientOrder(contents)...)
	findings = append(findings, validateLifecycleIntegrationCommands(contents, "UV tools", []string{"bash scripts/ci-uv-tools.sh", "bash scripts/ci-uv-tools.sh bootstrap"})...)
	findings = append(findings, validateUVToolsClientOrder(contents)...)
	findings = append(findings, validatePackageFixtureWorkingDirectories(contents)...)
	findings = append(findings, validateTerraformIntegrationGates(contents)...)
	findings = append(findings, validateActiveJobCommands(contents, "gvisor-observer", "Build exact gVisor observer helper with Bazel", "gVisor observer SDK", []string{
		`scripts/build-gvisor-observer-release.sh "$RUNNER_TEMP/haa_gvisor_observer"`,
	})...)
	findings = append(findings, validateLifecycleIntegrationCommands(contents, "npm inspect", []string{`HELOX_NPM_INSPECT_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestLinuxNPMInspectCLIIntegration$'`})...)
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
		".docker.ci_ubuntu_24_04_amd64.docker_ce_package",
		".docker.ci_ubuntu_24_04_amd64.containerd_package",
		"test \"$(docker version --format '{{.Server.Version}}')\" = \"$docker_engine_version\"",
		"runtime_lock=scripts/runtimes.lock.json",
		"jq -er",
		"--download-sidecars=NEVER --require-sidecars=ALWAYS --runtime=runsc-trace -- --sidecar-usage-policy=STRICT",
		"pytorch_cu126_qualification:",
		"pytorch_cu130_qualification:",
		"pytorch_cu132_qualification:",
		"select at most one CUDA PyTorch qualification profile",
		"if [ \"$selected_cuda_profiles\" -gt 0 ]; then\n            go run ./scripts/check qualification-freshness\n          fi",
		"HELOX_PYTORCH_PROFILE=cu126",
		"HELOX_PYTORCH_PROFILE=cu130",
		"HELOX_PYTORCH_PROFILE=cu132",
		`docker pull "$go_image"`,

		`HELOX_GO_RESOLVER_INTEGRATION=1 HELOX_GO_RESOLVER_PROJECT_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestLinuxGo(IsolatedResolver|IsolatedProjectResolver|SourceProjectSnapshot|DependencyFreeProjectSnapshot)Integration$'`,
		`HELOX_GO_RESOLVER_INTEGRATION=1 HELOX_GO_RESOLVER_PROJECT_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=10m -test.run '^TestLinuxGo(GetDownload|TransitiveGetDownload|DependencyFreeDownload)Integration$'`,
		`HELOX_GO_BUILD_CLI_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=15m -test.run '^TestLinuxGo(DependencyFreeBuildCLI|PublicBuildCLI|TransitiveBuildCLI|InvalidTestdataBuildCLI|CgoBuildCLI|LibraryBuildCLI|MixedPackagesBuildCLI|MixedInvalidPackageBuildCLI|BuildSecurityCLI)Integration$'`,
		"HELOX_PROMOTION_INTEGRATION=1 go test -v -timeout=5m ./internal/promotion -run TestLinuxNPMPromotionIntegration",
		"check-latest: false",
		"cache: false",
		"    if: ${{ always() }}",
		"    needs:\n      - quick\n      - docs\n      - security\n      - vulnerability\n      - minimum-go\n      - macos\n      - gvisor-observer\n      - gvisor-integration\n      - wheel-corpus",
		"run: go run ./scripts/check bootstrap-modules",
		"run: go run -trimpath ./scripts/check platform",
		"run: go run ./scripts/check corpus",
		`run: python3 scripts/prepare-wheel-corpus.py --root "$HELOX_CORPUS_ROOT"`,
		"run: go run ./scripts/check security",
		"run: go run ./scripts/check vulnerability",
		`run: go run ./scripts/check required "$QUICK_RESULT" "$DOCS_RESULT" "$SECURITY_RESULT" "$VULNERABILITY_RESULT" "$MINIMUM_GO_RESULT" "$MACOS_RESULT" "$GVISOR_OBSERVER_RESULT" "$GVISOR_INTEGRATION_RESULT" "$CORPUS_RESULT"`,
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
		"5ceb9a5fd5750d6c73dd166441f28306039300d0",
		"4463ce276e207f5a516a08ec627a768a19cf7bed0094d522b0810bee3424585caa8d344e093204012b974f5c508ab2362dcb0d7236f0c1992fccc426beeb7ffc",
		"c876a1619c885f44f3bdc87998eca59c79581954631c9d7fab4eb53cc0409b68e4be74c08ef3fe599c51b75d56262070f0c314f9908336221e7764fdf981b7f5",
		"node:22.23.1-slim@sha256:",
		"python:3.14.7-slim-bookworm@sha256:",
		"pytorch_cuda_qualification",
		"HELOX_PYTORCH_PROFILE=cu128",
		"--download-sidecars=ALWAYS",
		"--sidecar-usage-policy=LEGACY_DEPRECATED_SLOW_EMBEDDED_FALLBACK",
	}
	for _, token := range forbidden {
		if strings.Contains(contents, token) {
			findings = append(findings, fmt.Sprintf("forbidden workflow token %q", token))
		}
	}

	allowedActions := map[string]int{
		"actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1": 10,
		"actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e": 9,
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
	wantJobs := []string{"docs", "gvisor-integration", "gvisor-observer", "macos", "minimum-go", "quick", "required", "security", "vulnerability", "wheel-corpus"}
	if strings.Join(jobs, ",") != strings.Join(wantJobs, ",") {
		findings = append(findings, fmt.Sprintf("workflow jobs are %q, require %q", jobs, wantJobs))
	}
	sort.Strings(findings)
	return findings
}

func validateSecurityWorkflow(contents, selectedGo string) []string {
	findings := validateGoWorkflowVersions(contents, selectedGo)
	for _, snippet := range []string{
		"name: Heliopause Scheduled Security",
		"  schedule:\n    - cron: '17 3 * * 1'",
		"  workflow_dispatch:",
		"permissions:\n  contents: read",
		"fetch-depth: 0",
		"persist-credentials: false",
		"runs-on: ubuntu-24.04",
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

func validateReleaseWorkflow(contents, selectedGo string) []string {
	findings := validateGoWorkflowVersions(contents, selectedGo)
	for _, snippet := range []string{
		"name: Heliopause Release Build",
		"  push:\n    tags:\n      - 'v*'",
		"permissions:\n  contents: read\n  attestations: write\n  id-token: write",
		"runs-on: ubuntu-24.04",
		"GOTOOLCHAIN: local",
		"persist-credentials: false",
		"check-latest: false",
		"cache: false",
		"go build -trimpath -buildvcs=true",
		"go run ./scripts/check release-gate",
		"scripts/build-gvisor-observer-release.sh",
		"go run ./scripts/runtime-image-manifest",
		"docker buildx imagetools inspect \"$node_image\"",
		"docker buildx imagetools inspect \"$python_image\"",
		"docker buildx imagetools inspect \"$go_image\"",
		"docker buildx imagetools inspect \"$rust_image\"",
		"go run ./scripts/release-packages",
		"--source-date-epoch \"$(git show -s --format=%ct \"$GITHUB_SHA\")\"",
		"--checksums-output \"$dist/helox-release-checksums.txt\"",
		"--asset \"$dist/helox-linux-amd64.deb\"",
		"--asset \"$dist/helox-linux-amd64.rpm\"",
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
	findings = append(findings, validateActiveJobCommands(contents, "build-and-attest", "Build deterministic native release assets", "OS package", []string{
		`go run ./scripts/release-packages \`,
		`--source-date-epoch "$(git show -s --format=%ct "$GITHUB_SHA")" \`,
		`--checksums-output "$dist/helox-release-checksums.txt" \`,
		`--asset "$dist/helox-linux-amd64.deb" \`,
		`--asset "$dist/helox-linux-amd64.rpm"`,
	})...)
	findings = append(findings, validateActiveJobCommands(contents, "build-and-attest", "Build deterministic native release assets", "release installer", []string{
		`CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -buildvcs=true -o "$dist/haa-release-installer-linux-amd64" ./cmd/haa-release-installer`,
		`--asset "$dist/haa-release-installer-linux-amd64" \`,
	})...)
	findings = append(findings, validatePNPMReleaseSupply(contents)...)
	findings = append(findings, validateBunReleaseSupply(contents)...)
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

func validateReleasePublishWorkflow(contents, selectedGo string) []string {
	findings := validateGoWorkflowVersions(contents, selectedGo)
	for _, snippet := range []string{
		"name: Heliopause Verified Release Publish",
		"  workflow_dispatch:",
		"release_tag:",
		"source_run_id:",
		"permissions:\n  contents: write\n  attestations: read",
		"runs-on: ubuntu-24.04",
		"environment: release",
		"persist-credentials: false",
		"gh run view \"$SOURCE_RUN_ID\" -R \"$GH_REPO\"",
		"test \"$(jq -er '.workflowName' <<<\"$run_json\")\" = 'Heliopause Release Build'",
		"repos/$GH_REPO/branches/main",
		"repos/$GH_REPO/compare/$tag_sha...$main_sha",
		"repos/$GH_REPO/commits/$tag_sha/check-runs?per_page=100",
		".name == \"Required\" and .status == \"completed\" and .conclusion == \"success\" and .app.slug == \"github-actions\"",
		"gh run download \"$SOURCE_RUN_ID\" -R \"$GH_REPO\" -n \"$artifact_name\"",
		"sha256sum -c helox-release-checksums.txt",
		"helox-release-checksums.txt",
		"test -z \"$(comm -3",
		"gh attestation verify",
		"--signer-workflow \"$signer\"",
		"--source-digest",
		"go run ./scripts/check release-gate",
		"gh api --include \"repos/$GH_REPO/releases/tags/$RELEASE_TAG\"",
		"release already exists; refusing overwrite",
		"gh release create \"$RELEASE_TAG\"",
		"--verify-tag",
		"--draft",
		"Verify draft release asset bindings before publication",
		".draft",
		".digest",
		"gh release edit \"$RELEASE_TAG\" -R \"$GH_REPO\" --draft=false",
		"gh release verify \"$RELEASE_TAG\" -R \"$GH_REPO\"",
		"gh release verify-asset \"$RELEASE_TAG\"",
		"PUBLISHED_BUT_QUARANTINED",
	} {
		if !strings.Contains(contents, snippet) {
			findings = append(findings, fmt.Sprintf("missing required release publication contract %q", snippet))
		}
	}
	findings = append(findings, validateReleaseBundleConsumers(contents)...)
	draftVerification := strings.Index(contents, "Verify draft release asset bindings before publication")
	publish := strings.Index(contents, "gh release edit \"$RELEASE_TAG\" -R \"$GH_REPO\" --draft=false")
	if draftVerification < 0 || publish < 0 || draftVerification > publish {
		findings = append(findings, "draft asset verification does not precede public release transition")
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

// The manifest verifier owns package names and exact set validation. The workflow
// must authenticate that manifest before consuming its output, then pass the same
// verified control file to every publication/provenance consumer.
func validateReleaseBundleConsumers(contents string) []string {
	doc, err := parseWorkflow(contents)
	if err != nil {
		return nil
	}
	job := workflowMap(workflowMap(doc["jobs"])["publish"])
	steps, _ := job["steps"].([]any)
	commands := map[string][]string{
		"Verify candidate identity and download immutable artifact": {
			`gh attestation verify "$candidate/helox-release-manifest.json" \`,
			`--source-digest "$source_sha" \`,
			`expected_names="$(go run ./scripts/release-manifest \`,
			`--verify-bundle "$candidate" \`,
			`--version "$RELEASE_TAG" \`,
			`--commit "$source_sha" \`,
			`--workflow-run "$GITHUB_SERVER_URL/$GH_REPO/actions/runs/$SOURCE_RUN_ID")"`,
			`mapfile -t expected_files <<<"$expected_names"`,
			`(umask 077; set -o noclobber; printf '%s\n' "${expected_files[@]}" > "$RUNNER_TEMP/helox-release-files.txt")`,
		},
		"Verify GitHub artifact provenance": {
			`mapfile -t expected_files < "$RUNNER_TEMP/helox-release-files.txt"`,
			`for name in "${expected_files[@]}"; do`,
			`gh attestation verify "$candidate/$name" \`,
		},
		"Refuse an existing release and create a draft": {
			`mapfile -t expected_files < "$RUNNER_TEMP/helox-release-files.txt"`,
			`release_files+=("$candidate/$name")`,
			`"${release_files[@]}"`,
		},
		"Verify draft release asset bindings before publication": {
			`mapfile -t expected_files < "$RUNNER_TEMP/helox-release-files.txt"`,
			`test "$(jq -er '.assets | length' <<<"$release_json")" -eq "${#expected_files[@]}"`,
			`for name in "${expected_files[@]}"; do`,
		},
		"Publish and verify immutable release": {
			`mapfile -t expected_files < "$RUNNER_TEMP/helox-release-files.txt"`,
			`for name in "${expected_files[@]}"; do`,
			`gh release verify-asset "$RELEASE_TAG" "$candidate/$name" -R "$GH_REPO" || status=1`,
		},
	}
	var findings []string
	for name, required := range commands {
		bodies := []string{}
		for _, raw := range steps {
			step := workflowMap(raw)
			if step["name"] == name && step["if"] == nil && job["if"] == nil && step["continue-on-error"] == nil && job["continue-on-error"] == nil {
				run, _ := step["run"].(string)
				bodies = append(bodies, run)
			}
		}
		if len(bodies) != 1 {
			findings = append(findings, "release bundle requires one active step: "+name)
			continue
		}
		lines := strings.Split(bodies[0], "\n")
		prior := -1
		for _, command := range required {
			count, position := 0, -1
			for i, line := range lines {
				if strings.TrimSpace(line) == command {
					count++
					position = i
				}
			}
			if count != 1 || position <= prior {
				findings = append(findings, fmt.Sprintf("release bundle %s requires ordered active command %q", name, command))
			}
			prior = position
		}
	}
	return findings
}

func workflowJobIDs(contents string) []string {
	doc, err := parseWorkflow(contents)
	if err != nil {
		return nil
	}
	var jobs []string
	for id := range workflowMap(doc["jobs"]) {
		jobs = append(jobs, id)
	}
	sort.Strings(jobs)
	return jobs
}

// These fixed commands belong to the actual unconditional lifecycle step.
// Their presence in comments, another job or a skipped step supplies no gate.
const uvPublicResolverIntegrationCommand = `HELOX_UV_PUBLIC_RESOLVER_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=10m -test.run '^TestUVPublicNativeResolverIntegration$'`
const cargoSourceIntegrationCommand = `HELOX_CARGO_RESOLVER_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestLinuxCargoSourceProjectSnapshotIntegration$'`
const uvSyncCLIIntegrationCommand = `HELOX_UV_SYNC_CLI_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=12m -test.failfast -test.run '^TestLinuxUVSync(Complete|Closure)CLIIntegration$'`

var uvIntegrationCommands = []string{
	uvSyncCLIIntegrationCommand,
	`HELOX_UV_PUBLIC_UPDATE_RESOLVER_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=10m -test.run '^TestUVPublicNativeUpdateResolverIntegration$'`,
	`HELOX_UV_INSTALL_CLI_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=6m -test.run '^TestLinuxUVInstallCLIIntegration$'`,
	`uv_image="$(jq -er '.uv_image.reference' "$runtime_lock")"`,
	`docker pull "$uv_image"`,
	`HELOX_UV_NATIVE_RUNTIME=1 go test -tags=realcorpus -v -timeout=3m ./internal/sandbox -run '^TestUVActualIsolated(NativeStartup|EmptyTargetQuery|NativeLockCollection)$'`,
	uvPublicResolverIntegrationCommand,
}

// The same ephemeral job cache and compiler flags must reach both the outer
// checker bootstrap and its offline build/test children. No remote cache is used.
func validatePlatformBuildCache(contents string) []string {
	doc, err := parseWorkflow(contents)
	if err != nil {
		return nil
	}
	const root = "${{ runner.temp }}/heliopause-platform-tools"
	var findings []string
	for _, jobID := range []string{"minimum-go", "macos"} {
		job := workflowMap(workflowMap(doc["jobs"])[jobID])
		steps, _ := job["steps"].([]any)
		for _, stage := range []struct{ name, command string }{
			{"Bootstrap product modules", "go run -trimpath ./scripts/check bootstrap-modules"},
			{"Run canonical platform profile offline", "go run -trimpath ./scripts/check platform"},
		} {
			findings = append(findings, validateActiveJobCommands(contents, jobID, stage.name, jobID+" platform", []string{stage.command})...)
			for _, raw := range steps {
				step := workflowMap(raw)
				if step["name"] != stage.name {
					continue
				}
				env := workflowMap(step["env"])
				if env["HELOX_TOOL_CACHE"] != root || env["GOCACHE"] != root+"/go-build" || env["GOFLAGS"] != "" {
					findings = append(findings, fmt.Sprintf("%s %s requires one fixed job-local build cache and empty GOFLAGS", jobID, stage.name))
				}
			}
		}
	}
	return findings
}

var cargoIntegrationCommands = []string{
	`rust_image="$(jq -er '.rust_image.reference' "$runtime_lock")"`,
	`docker pull "$rust_image"`,
	cargoSourceIntegrationCommand,
	`HELOX_CARGO_RESOLVER_INTEGRATION=1 HELOX_CARGO_ADD_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=10m -test.run '^TestLinuxCargo(Add|TransitiveAdd)Integration$'`,
	`HELOX_CARGO_RESOLVER_INTEGRATION=1 HELOX_CARGO_BUILD_CLI_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=15m -test.run '^TestLinuxCargo.*Build.*CLIIntegration$|^TestLinuxCargoProcMacroSecurityCLIIntegration$'`,
}

var terraformIntegrationCommands = []string{
	`HELOX_RUN_TERRAFORM_SOURCE_INTEGRATION=1 go test -v -timeout=3m ./internal/verification/terraformprovider -run '^TestPublicTerraformProviderSourceIntegration$'`,
	`HELOX_TERRAFORM_PROVIDER_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestLinuxTerraformProvider(Probe|Security)Integration$'`,
	`HELOX_TERRAFORM_INIT_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=10m -test.run '^TestLinuxTerraformInit(Integration|NegativeIntegration)$'`,
}

func validateTerraformIntegrationGates(contents string) []string {
	return validateLifecycleIntegrationCommands(contents, "Terraform", terraformIntegrationCommands)
}
func validateCargoIntegrationGates(contents string) []string {
	return validateLifecycleIntegrationCommands(contents, "Cargo", cargoIntegrationCommands)
}
func validateLifecycleIntegrationCommands(contents, label string, commands []string) []string {
	return validateActiveJobCommands(contents, "gvisor-integration", "Install pinned runtime and run lifecycle integration", label, commands)
}

func validateActiveJobCommands(contents, jobID, stepName, label string, commands []string) []string {
	blocks, err := activeJobRunLines(contents, jobID, stepName)
	if err != nil {
		return nil
	}
	counts := map[string]int{}
	for _, lines := range blocks {
		for _, line := range lines {
			counts[line]++
		}
	}
	var findings []string
	for _, command := range commands {
		if counts[command] != 1 {
			findings = append(findings, fmt.Sprintf("%s integration requires one active lifecycle command %q", label, command))
		}
	}
	return findings
}

func activeJobRunLines(contents, jobID, stepName string) ([][]string, error) {
	doc, err := parseWorkflow(contents)
	if err != nil {
		return nil, err
	} // The structural validator owns parse failures.
	job := workflowMap(workflowMap(doc["jobs"])[jobID])
	steps, _ := job["steps"].([]any)
	var blocks [][]string
	for _, raw := range steps {
		step := workflowMap(raw)
		if step["name"] != stepName {
			continue
		}
		if step["if"] != nil || job["if"] != nil || step["continue-on-error"] != nil || job["continue-on-error"] != nil {
			continue
		}
		run, _ := step["run"].(string)
		var lines []string
		for _, line := range strings.Split(run, "\n") {
			lines = append(lines, strings.TrimSpace(line))
		}
		blocks = append(blocks, lines)
	}
	return blocks, nil
}

// These real consumers read tracked package fixtures. Require the same
// narrow subshell shape as go test's package cwd, without changing later steps.
// This validates this repository convention, not arbitrary Bash semantics.
func validatePackageFixtureWorkingDirectories(contents string) []string {
	blocks, err := activeJobRunLines(contents, "gvisor-integration", "Install pinned runtime and run lifecycle integration")
	if err != nil {
		return nil
	}
	scopes := map[string]string{uvPublicResolverIntegrationCommand: "internal/sandbox", cargoSourceIntegrationCommand: "internal/sandbox", uvSyncCLIIntegrationCommand: "internal/bootstrap"}
	counts := map[string]int{}
	for _, block := range blocks {
		var lines []string
		for _, line := range block {
			if line != "" && !strings.HasPrefix(line, "#") {
				lines = append(lines, line)
			}
		}
		for i := 2; i+1 < len(lines); i++ {
			if directory := scopes[lines[i]]; directory != "" && lines[i-2] == "(" && lines[i-1] == "cd "+directory && lines[i+1] == ")" {
				counts[lines[i]]++
			}
		}
	}
	var findings []string
	for _, command := range []string{uvPublicResolverIntegrationCommand, cargoSourceIntegrationCommand, uvSyncCLIIntegrationCommand} {
		if counts[command] != 1 {
			findings = append(findings, fmt.Sprintf("fixture integration requires one matching package subshell for %q", command))
		}
	}
	return findings
}

// Independent expected producer policy: no command presence or source-derived
// expected list can stand in for the acquired->verified->packaged->bound order.
func validatePNPMReleaseSupply(contents string) []string {
	return validateOrderedNativeReleaseSupply(contents, "pnpm", []string{
		`go run ./scripts/generate-runtime-lock.go -check`,
		`pnpm_archive="$RUNNER_TEMP/helox-pnpm-linux-x64.tar.gz"`,
		`pnpm_url="$(jq -er '.pnpm_tool.archive.url' scripts/runtimes.lock.json)"`,
		`pnpm_size="$(jq -er '.pnpm_tool.archive.size' scripts/runtimes.lock.json)"`,
		`curl --fail --show-error --silent --location --proto '=https' --proto-redir '=https' --connect-timeout 15 --max-time 120 --max-filesize "$pnpm_size" --output "$pnpm_archive" "$pnpm_url"`,
		`go run ./scripts/update-runtime materialize-pnpm \`,
		`--root "$GITHUB_WORKSPACE" \`,
		`--archive "$pnpm_archive" \`,
		`--output "$dist/pnpm-linux-amd64" \`,
		`--notices-output "$dist/pnpm-NOTICE.txt"`,
		`rm -- "$pnpm_archive"`,
		`go run ./scripts/release-packages \`,
		`go run ./scripts/release-manifest \`,
		`--asset "$dist/pnpm-linux-amd64" \`,
		`--asset "$dist/pnpm-NOTICE.txt" \`,
	})
}

func validateBunReleaseSupply(contents string) []string {
	return validateOrderedNativeReleaseSupply(contents, "bun", []string{
		`go run ./scripts/generate-runtime-lock.go -check`,
		`bun_archive="$RUNNER_TEMP/helox-bun-linux-x64.zip"`,
		`bun_url="$(jq -er '.bun_tool.archive.url' scripts/runtimes.lock.json)"`,
		`bun_size="$(jq -er '.bun_tool.archive.size' scripts/runtimes.lock.json)"`,
		`curl --fail --show-error --silent --location --proto '=https' --proto-redir '=https' --connect-timeout 15 --max-time 120 --max-filesize "$bun_size" --output "$bun_archive" "$bun_url"`,
		`go run ./scripts/update-runtime materialize-bun --root "$GITHUB_WORKSPACE" --archive "$bun_archive" --output "$dist/bun-linux-amd64" --notices-output "$dist/bun-NOTICE.txt"`,
		`rm -- "$bun_archive"`,
		`go run ./scripts/release-packages \`,
		`go run ./scripts/release-manifest \`,
		`--asset "$dist/bun-linux-amd64" \`,
		`--asset "$dist/bun-NOTICE.txt" \`,
	})
}

func validateOrderedNativeReleaseSupply(contents, client string, expected []string) []string {
	blocks, err := activeJobRunLines(contents, "build-and-attest", "Build deterministic native release assets")
	if err != nil {
		return nil
	}
	if len(blocks) != 1 {
		return []string{client + " release supply requires one unconditional native build step"}
	}
	var findings []string
	previous := -1
	for _, command := range expected {
		found, count := -1, 0
		for i, line := range blocks[0] {
			if line == command {
				found = i
				count++
			}
		}
		if count != 1 || found <= previous {
			findings = append(findings, client+" release supply requires ordered active command: "+command)
		}
		previous = found
	}
	return findings
}
