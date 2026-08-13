package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReadToolLock(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestToolLock(t, root)
	lock, err := readToolLock(filepath.Join(root, "scripts", "tools.lock.json"))
	if err != nil {
		t.Fatalf("readToolLock error: %v", err)
	}
	if len(lock.Tools) != 1 || lock.Tools[0].ExpectedVersion != "staticcheck 2026.1 (v0.7.0)" {
		t.Fatalf("readToolLock = %#v", lock)
	}
}

func TestReadToolLockRejectsUnknownAndFloatingValues(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"unknown field": strings.Replace(validTestToolLock, `"schemaVersion": 1`, `"schemaVersion": 1, "unexpected": true`, 1),
		"floating":      strings.Replace(validTestToolLock, `"version": "2026.1"`, `"version": "latest"`, 1),
		"trailing":      validTestToolLock + `{}`,
	}
	for name, contents := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "tools.lock.json")
			writeTestFile(t, path, contents)
			if _, err := readToolLock(path); err == nil {
				t.Fatal("readToolLock error = nil, want non-nil")
			}
		})
	}
}

func TestResolveToolCache(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "quality-cache")
	resolved, err := resolveToolCache(root, outside)
	if err != nil {
		t.Fatalf("resolveToolCache outside error: %v", err)
	}
	if resolved == "" || !filepath.IsAbs(resolved) {
		t.Fatalf("resolveToolCache = %q", resolved)
	}
	if _, err := resolveToolCache(root, filepath.Join(root, "cache")); err == nil {
		t.Fatal("resolveToolCache inside source error = nil, want non-nil")
	}
	if _, err := resolveToolCache(root, "relative/cache"); err == nil {
		t.Fatal("resolveToolCache relative error = nil, want non-nil")
	}
}

func TestResolveToolCacheRejectsSymlinkIntoSource(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic link permissions vary on Windows")
	}
	t.Parallel()

	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(outside, "cache-link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, err := resolveToolCache(root, filepath.Join(link, "tools")); err == nil {
		t.Fatal("resolveToolCache symlink into source error = nil, want non-nil")
	}
}

func TestValidateToolVersion(t *testing.T) {
	t.Parallel()

	tool := toolSpec{Command: "staticcheck", ExpectedVersion: "staticcheck 2026.1 (v0.7.0)"}
	if err := validateToolVersion(tool, tool.ExpectedVersion); err != nil {
		t.Fatalf("validateToolVersion matching error: %v", err)
	}
	var failure *checkFailure
	err := validateToolVersion(tool, "staticcheck 2025.1.1 (v0.6.1)")
	if !errors.As(err, &failure) || failure.class != unavailable {
		t.Fatalf("validateToolVersion mismatch error = %v, want unavailable", err)
	}
}

func TestBootstrapRejectsSetupGoMismatchBeforeMutation(t *testing.T) {
	t.Parallel()

	cache := filepath.Join(t.TempDir(), "missing-cache")
	checker := checker{
		stdout:    io.Discard,
		toolCache: cache,
		toolLock: toolLock{Tools: []toolSpec{{
			SetupGo: "0.0.0",
		}}},
	}
	var failure *checkFailure
	err := checker.bootstrap()
	if !errors.As(err, &failure) || failure.class != unavailable {
		t.Fatalf("bootstrap error = %v, want unavailable", err)
	}
	if _, err := os.Stat(cache); !os.IsNotExist(err) {
		t.Fatalf("bootstrap created cache before rejecting setup Go: %v", err)
	}
}

func TestPrepareModuleCacheExcludesQualityTools(t *testing.T) {
	t.Parallel()

	cache := filepath.Join(t.TempDir(), "cache")
	checker := checker{toolCache: cache}
	if err := checker.prepareModuleCache(); err != nil {
		t.Fatalf("prepareModuleCache error: %v", err)
	}
	for _, name := range []string{"go-build", "go-mod"} {
		if info, err := os.Stat(filepath.Join(cache, name)); err != nil || !info.IsDir() {
			t.Fatalf("module cache directory %q unavailable: %v", name, err)
		}
	}
	for _, name := range []string{"bin", "staticcheck"} {
		if _, err := os.Stat(filepath.Join(cache, name)); !os.IsNotExist(err) {
			t.Fatalf("quality tool directory %q created by module bootstrap: %v", name, err)
		}
	}
}

func TestVerifyToolRejectsMissingPinnedExecutable(t *testing.T) {
	t.Parallel()

	checker := checker{toolCache: t.TempDir()}
	var failure *checkFailure
	err := checker.verifyTool(toolSpec{Command: "staticcheck"})
	if !errors.As(err, &failure) || failure.class != unavailable {
		t.Fatalf("verifyTool error = %v, want unavailable", err)
	}
}

func TestBootstrapEnvironmentKeepsVerificationEnabled(t *testing.T) {
	t.Setenv("GOINSECURE", "*")
	t.Setenv("GONOSUMDB", "*")
	t.Setenv("GOPRIVATE", "*")

	checker := checker{toolCache: filepath.Join(t.TempDir(), "cache")}
	environment := strings.Join(checker.bootstrapEnvironment(), "\n")
	for _, expected := range []string{
		"GOENV=off",
		"GOINSECURE=",
		"GONOSUMDB=",
		"GOPRIVATE=",
		"GOPROXY=https://proxy.golang.org",
		"GOSUMDB=sum.golang.org",
		"GOTOOLCHAIN=local",
	} {
		if !strings.Contains(environment, expected) {
			t.Errorf("bootstrap environment missing %q", expected)
		}
	}
	for _, forbidden := range []string{"GOINSECURE=*", "GONOSUMDB=*", "GOPRIVATE=*"} {
		if strings.Contains(environment, forbidden) {
			t.Errorf("bootstrap environment retained %q", forbidden)
		}
	}
}

const validTestToolLock = `{
  "schemaVersion": 1,
  "tools": [{
    "command": "staticcheck",
    "package": "honnef.co/go/tools/cmd/staticcheck",
    "version": "2026.1",
    "expectedVersion": "staticcheck 2026.1 (v0.7.0)",
    "setupGo": "1.26.5"
  }]
}`
