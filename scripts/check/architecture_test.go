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
		{ImportPath: modulePath + "/internal/core/domain", Imports: []string{"crypto/rand"}},
		{ImportPath: modulePath + "/internal/core/ports", Imports: []string{"context", modulePath + "/internal/core/domain"}},
		{ImportPath: modulePath + "/internal/testutil/fakeworkflow", Imports: []string{modulePath + "/internal/core/domain", modulePath + "/internal/core/ports"}},
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
		{ImportPath: modulePath + "/internal/core/domain", Imports: []string{modulePath + "/internal/cli", "example.com/domain", "os"}},
		{ImportPath: modulePath + "/internal/core/ports", Imports: []string{modulePath + "/internal/cli", "example.com/ports"}},
		{ImportPath: modulePath + "/internal/testutil/fakeworkflow", Imports: []string{modulePath + "/internal/cli", "example.com/fake", "net/http"}},
		{ImportPath: modulePath + "/scripts/check", Imports: []string{modulePath + "/internal/cli"}},
	}
	findings := strings.Join(validateCurrentImports(modulePath, packages), "\n")
	for _, expected := range []string{"cmd/helox", "example.com/sdk", "internal/cli", "internal/core/domain", "example.com/domain", "forbidden concrete package os", "internal/core/ports", "example.com/ports", "internal/testutil/fakeworkflow", "example.com/fake", "forbidden concrete package net/http", "scripts/check"} {
		if !strings.Contains(findings, expected) {
			t.Errorf("findings missing %q: %s", expected, findings)
		}
	}
}
