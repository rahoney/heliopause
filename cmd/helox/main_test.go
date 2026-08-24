package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const processHelperEnv = "HELOX_PROCESS_SMOKE_HELPER"

func TestProcessHelpSmoke(t *testing.T) {
	t.Parallel()

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve test executable: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, executable, "-test.run=^TestProcessHelper$", "--", "--help")
	command.Dir = t.TempDir()
	command.Env = []string{processHelperEnv + "=1"}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run process smoke test: %v\n%s", err, output)
	}
	if ctx.Err() != nil {
		t.Fatalf("process smoke test context: %v", ctx.Err())
	}
	if !strings.Contains(string(output), "Usage:\n  helox") {
		t.Fatalf("help output missing usage: %q", output)
	}
}

func TestNativeBinaryDefaultPaths(t *testing.T) {
	packageDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve package directory: %v", err)
	}
	moduleRoot := filepath.Clean(filepath.Join(packageDirectory, "..", ".."))
	if _, err := os.Stat(filepath.Join(moduleRoot, "go.mod")); err != nil {
		t.Fatalf("validate module root: %v", err)
	}

	goExecutable, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("find Go executable: %v", err)
	}
	binaryPath := filepath.Join(t.TempDir(), "helox")

	// Hosted macOS runners can take longer than 30 seconds for this deliberately
	// uncached, nested production build. Keep a bound while avoiding a scheduler-
	// load-dependent failure in the canonical platform profile.
	buildContext, cancelBuild := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelBuild()
	build := exec.CommandContext(buildContext, goExecutable, "build", "-trimpath", "-o", binaryPath, "./cmd/helox")
	build.Dir = moduleRoot
	buildOutput, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build native helox executable: %v\n%s", err, buildOutput)
	}
	if buildContext.Err() != nil {
		t.Fatalf("native build context: %v", buildContext.Err())
	}

	for _, arguments := range [][]string{nil, {"--help"}} {
		runContext, cancelRun := context.WithTimeout(context.Background(), 10*time.Second)
		command := exec.CommandContext(runContext, binaryPath, arguments...)
		command.Dir = t.TempDir()
		output, err := command.CombinedOutput()
		contextErr := runContext.Err()
		cancelRun()
		if err != nil {
			t.Fatalf("run native helox %q: %v\n%s", arguments, err, output)
		}
		if contextErr != nil {
			t.Fatalf("native helox %q context: %v", arguments, contextErr)
		}
		if !strings.Contains(string(output), "Usage:\n  helox") {
			t.Fatalf("native helox %q output missing usage: %q", arguments, output)
		}
	}
}

func TestProcessHelper(t *testing.T) {
	if os.Getenv(processHelperEnv) != "1" {
		return
	}

	separator := -1
	for index, argument := range os.Args {
		if argument == "--" {
			separator = index
			break
		}
	}
	if separator == -1 {
		return
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run(context.Background(), os.Args[separator+1:], &stdout, &stderr); code != exitSuccess {
		t.Fatalf("run returned %d: %s", code, stderr.String())
	}
	if _, err := os.Stdout.Write(stdout.Bytes()); err != nil {
		t.Fatalf("write helper output: %v", err)
	}
}

func TestRunReportsFailure(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(context.Background(), []string{"unexpected"}, &stdout, &stderr)

	if code != exitFailure {
		t.Fatalf("run returned %d, want %d", code, exitFailure)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.HasPrefix(stderr.String(), "helox: ") {
		t.Fatalf("stderr = %q, want helox prefix", stderr.String())
	}
}
