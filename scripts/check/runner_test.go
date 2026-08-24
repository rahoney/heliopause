package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestValidateModuleRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.test/project\n\ngo 1.25\n")
	resolved, err := validateModuleRoot(root)
	if err != nil {
		t.Fatalf("validateModuleRoot error: %v", err)
	}
	wantRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	if resolved != wantRoot {
		t.Fatalf("validateModuleRoot = %q, want %q", resolved, wantRoot)
	}

	subdirectory := filepath.Join(root, "nested")
	if err := os.Mkdir(subdirectory, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if _, err := validateModuleRoot(subdirectory); err == nil {
		t.Fatal("validateModuleRoot from subdirectory error = nil, want non-nil")
	}
}

func TestBoundedBuffer(t *testing.T) {
	t.Parallel()

	buffer := boundedBuffer{limit: 4}
	written, err := buffer.Write([]byte("123456"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if written != 6 {
		t.Fatalf("Write = %d, want 6", written)
	}
	if got := buffer.String(); got != "1234\n[output truncated]" {
		t.Fatalf("String = %q", got)
	}
}

func TestDeterministicGoEnvironment(t *testing.T) {
	t.Parallel()

	environment := deterministicGoEnvironment([]string{"HOME=/tmp/home", "GOPROXY=https://proxy.invalid", "GOFLAGS=-mod=mod"})
	joined := strings.Join(environment, "\n")
	for _, expected := range []string{"HOME=/tmp/home", "GOENV=off", "GOPROXY=off", "GOFLAGS=", "GOTOOLCHAIN=local", "GOWORK=off"} {
		if !strings.Contains(joined, expected) {
			t.Errorf("environment missing %q: %q", expected, environment)
		}
	}
	if strings.Contains(joined, "proxy.invalid") || strings.Contains(joined, "-mod=mod") {
		t.Fatalf("environment retained overridden values: %q", environment)
	}
}

func TestRunCommandRejectsPATHExecutable(t *testing.T) {
	t.Parallel()

	checker := checker{root: t.TempDir(), stdout: io.Discard}
	_, err := checker.runCommand("test", "go", "version")
	var failure *checkFailure
	if !errors.As(err, &failure) || failure.class != unavailable {
		t.Fatalf("runCommand error = %v, want unavailable failure", err)
	}
}

func TestRunRejectsUnknownProfile(t *testing.T) {
	t.Parallel()

	checker := checker{stdout: io.Discard}
	err := checker.runProfile("unknown")
	var failure *checkFailure
	if !errors.As(err, &failure) || failure.class != unavailable {
		t.Fatalf("runProfile error = %v, want unavailable failure", err)
	}
}

func TestRunSequentialPreservesFirstFailure(t *testing.T) {
	t.Parallel()

	want := &checkFailure{class: findingFailure, step: "first"}
	secondRan := false
	checker := checker{stdout: io.Discard}
	err := checker.runSequential([]checkStep{
		{name: "first", run: func() error { return want }},
		{name: "second", run: func() error { secondRan = true; return nil }},
	})
	if !errors.Is(err, want) {
		t.Fatalf("runSequential error = %v, want first failure", err)
	}
	if secondRan {
		t.Fatal("runSequential ran a step after the first failure")
	}
}

func TestQuickStepComposition(t *testing.T) {
	t.Parallel()

	checker := checker{}
	steps := checker.quickSteps()
	var names []string
	for _, step := range steps {
		names = append(names, step.name)
	}
	want := []string{
		"format check",
		"runtime lock drift",
		"module drift",
		"module integrity",
		"production build",
		"test build validity",
		"architecture",
		"CI configuration",
		"go vet",
		"Staticcheck",
		"default test",
	}
	if !slices.Equal(names, want) {
		t.Fatalf("quick steps = %q, want %q", names, want)
	}
}

func TestPlatformStepComposition(t *testing.T) {
	t.Parallel()

	checker := checker{}
	steps := checker.platformSteps()
	var names []string
	for _, step := range steps {
		names = append(names, step.name)
	}
	want := []string{"production build", "default test"}
	if !slices.Equal(names, want) {
		t.Fatalf("platform steps = %q, want %q", names, want)
	}
}

func TestFormatCheckAndExplicitFormatProfile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestToolLock(t, root)
	path := filepath.Join(root, "main.go")
	writeTestFile(t, path, "package main\nfunc main(){println(\"hello\")}\n")
	checker, err := newChecker(root, io.Discard)
	if err != nil {
		t.Fatalf("newChecker error: %v", err)
	}
	var failure *checkFailure
	if err := checker.checkFormat(); !errors.As(err, &failure) || failure.class != findingFailure {
		t.Fatalf("checkFormat error = %v, want finding failure", err)
	}
	if err := checker.applyFormat(); err != nil {
		t.Fatalf("applyFormat error: %v", err)
	}
	if err := checker.checkFormat(); err != nil {
		t.Fatalf("checkFormat after applyFormat: %v", err)
	}
}

func TestModuleDriftIsFinding(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestToolLock(t, root)
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.test/project\n\ngo 1.25.13\n\nrequire ()\n")
	writeTestFile(t, filepath.Join(root, "main.go"), "package main\n\nfunc main() {}\n")
	checker, err := newChecker(root, io.Discard)
	if err != nil {
		t.Fatalf("newChecker error: %v", err)
	}
	var failure *checkFailure
	if err := checker.checkModuleDrift(); !errors.As(err, &failure) || failure.class != findingFailure {
		t.Fatalf("checkModuleDrift error = %v, want finding failure", err)
	}
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeTestToolLock(t *testing.T, root string) {
	t.Helper()
	writeTestFile(t, filepath.Join(root, "scripts", "tools.lock.json"), validTestToolLock)
}
