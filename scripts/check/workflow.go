package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const workflowRelativePath = ".github/workflows/heliopause-ci.yml"

var actionReference = regexp.MustCompile(`(?m)^\s*uses:\s*(\S+)\s*(?:#.*)?$`)

func checkCIWorkflow(root string) error {
	path := filepath.Join(root, filepath.FromSlash(workflowRelativePath))
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return &checkFailure{class: unavailable, step: "CI configuration", detail: workflowRelativePath + " is not a regular file"}
	}
	if info.Size() > outputLimit {
		return &checkFailure{class: executionFailure, step: "CI configuration", detail: "workflow exceeds size limit"}
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return &checkFailure{class: executionFailure, step: "CI configuration", cause: err}
	}
	findings := validateCIWorkflow(string(contents))
	if len(findings) != 0 {
		return &checkFailure{class: findingFailure, step: "CI configuration", detail: strings.Join(findings, "\n")}
	}
	return nil
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
		"go-version: '1.26.5'",
		"go-version: '1.25.12'",
		"docker_package_version=5:29.6.2-1~ubuntu.24.04~noble",
		"containerd_package_version=2.1.5-1~ubuntu.24.04~noble",
		"test \"$(docker version --format '{{.Server.Version}}')\" = 29.6.2",
		"check-latest: false",
		"cache: false",
		"    if: ${{ always() }}",
		"    needs:\n      - quick\n      - docs\n      - minimum-go\n      - macos\n      - gvisor-observer\n      - gvisor-integration",
		"run: go run ./scripts/check bootstrap-modules",
		"run: go run ./scripts/check platform",
		`run: go run ./scripts/check required "$QUICK_RESULT" "$DOCS_RESULT" "$MINIMUM_GO_RESULT" "$MACOS_RESULT" "$GVISOR_OBSERVER_RESULT" "$GVISOR_INTEGRATION_RESULT"`,
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
	}
	for _, token := range forbidden {
		if strings.Contains(contents, token) {
			findings = append(findings, fmt.Sprintf("forbidden workflow token %q", token))
		}
	}

	allowedActions := map[string]int{
		"actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1": 7,
		"actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e": 6,
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
	wantJobs := []string{"docs", "gvisor-integration", "gvisor-observer", "macos", "minimum-go", "quick", "required"}
	if strings.Join(jobs, ",") != strings.Join(wantJobs, ",") {
		findings = append(findings, fmt.Sprintf("workflow jobs are %q, require %q", jobs, wantJobs))
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
