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

func TestReleaseGateSummaryIsBounded(t *testing.T) {
	root := t.TempDir()
	summary := releaseGateSummary(root)
	if !strings.Contains(summary, "release gate") {
		t.Fatalf("releaseGateSummary = %q, want release gate context", summary)
	}
}
