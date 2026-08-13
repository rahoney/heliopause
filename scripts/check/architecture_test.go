package main

import (
	"strings"
	"testing"
)

func TestValidateCurrentImports(t *testing.T) {
	t.Parallel()

	const modulePath = "example.test/project"
	packages := []packageMetadata{
		{ImportPath: modulePath + "/cmd/helox", Imports: []string{"context", modulePath + "/internal/bootstrap"}},
		{ImportPath: modulePath + "/internal/bootstrap", Imports: []string{modulePath + "/internal/cli"}},
		{ImportPath: modulePath + "/internal/cli", Imports: []string{"github.com/spf13/cobra"}},
		{ImportPath: modulePath + "/scripts/check", Imports: []string{"os"}},
	}
	if findings := validateCurrentImports(modulePath, packages); len(findings) != 0 {
		t.Fatalf("validateCurrentImports findings = %v, want none", findings)
	}
}

func TestValidateCurrentImportsRejectsBoundaryShortcut(t *testing.T) {
	t.Parallel()

	const modulePath = "example.test/project"
	packages := []packageMetadata{
		{ImportPath: modulePath + "/cmd/helox", Imports: []string{modulePath + "/internal/cli", "example.com/sdk"}},
		{ImportPath: modulePath + "/internal/cli", Imports: []string{modulePath + "/internal/bootstrap"}},
		{ImportPath: modulePath + "/scripts/check", Imports: []string{modulePath + "/internal/cli"}},
	}
	findings := strings.Join(validateCurrentImports(modulePath, packages), "\n")
	for _, expected := range []string{"cmd/helox", "example.com/sdk", "internal/cli", "scripts/check"} {
		if !strings.Contains(findings, expected) {
			t.Errorf("findings missing %q: %s", expected, findings)
		}
	}
}
