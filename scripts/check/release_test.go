package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseGateFailsClosedWithoutLicense(t *testing.T) {
	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o700); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "heliopause-release-build.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "heliopause-release-build.yml"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	err = checkReleaseGate(root)
	if err == nil || !strings.Contains(err.Error(), "LICENSE is missing") {
		t.Fatalf("checkReleaseGate error = %v, want missing LICENSE finding", err)
	}
	if !strings.Contains(err.Error(), "public release publication workflow is not configured") {
		t.Fatalf("checkReleaseGate error = %v, want publication workflow finding", err)
	}
}

func TestReleaseGateRunsStrictFreshness(t *testing.T) {
	root := t.TempDir()
	body, err := os.ReadFile(filepath.Join("..", "version-support.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o700); err != nil {
		t.Fatal(err)
	}
	stale := strings.ReplaceAll(string(body), "2026-09-17", "2026-06-18")
	if err := os.WriteFile(filepath.Join(root, "scripts", "version-support.lock.json"), []byte(stale), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkReleaseGate(root); err == nil || !strings.Contains(err.Error(), "version support review is stale") {
		t.Fatalf("checkReleaseGate error = %v, want strict freshness failure", err)
	}
}

func TestReleaseGateSummaryIsBounded(t *testing.T) {
	root := t.TempDir()
	summary := releaseGateSummary(root)
	if !strings.Contains(summary, "release gate") {
		t.Fatalf("releaseGateSummary = %q, want release gate context", summary)
	}
}
