//go:build linux || darwin

package main

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestCommandTimeoutRetainsOutputAndCause(t *testing.T) {
	t.Parallel()

	c := checker{root: t.TempDir(), stdout: io.Discard}
	const stdoutMarker = "CHECK_TIMEOUT_STDOUT"
	const stderrMarker = "CHECK_TIMEOUT_STDERR"
	// The fixed shell fixture prints before replacing itself with sleep.
	// CommandContext therefore owns/kills that same child, not an orphan.
	output, err := c.runCommandWithTimeout("timeout fixture", 2*time.Second,
		[]string{"PATH=/usr/bin:/bin"}, "/bin/sh", "-c",
		"printf '%s\\n' 'CHECK_TIMEOUT_STDOUT'; printf '%s\\n' 'CHECK_TIMEOUT_STDERR' >&2; exec /bin/sleep 30")
	for _, marker := range []string{stdoutMarker, stderrMarker} {
		if !strings.Contains(output, marker) {
			t.Fatalf("fixture did not produce partial output before timeout: %q, %v", output, err)
		}
	}
	var failure *checkFailure
	if !errors.Is(err, context.DeadlineExceeded) || !errors.As(err, &failure) ||
		failure.class != executionFailure || failure.step != "timeout fixture" {
		t.Fatalf("timeout classification or original cause was lost: %v", err)
	}
	for _, marker := range []string{stdoutMarker, stderrMarker} {
		if !strings.Contains(err.Error(), marker) {
			t.Fatalf("timeout error discarded bounded partial output %q: %v", marker, err)
		}
	}
	if !strings.Contains(err.Error(), "available=true exited=false success=false exit_code=-1") {
		t.Fatalf("actual killed child terminal state was lost: %v", err)
	}
}

func TestGoTestCaptureTimeoutPreservesActiveTestAndCause(t *testing.T) {
	t.Parallel()
	c := checker{root: t.TempDir(), stdout: io.Discard}
	output := newGoTestCapture(time.Now)
	text, err := c.runCapturedCommand("default test diagnostic fixture", 2*time.Second, []string{"PATH=/usr/bin:/bin"}, "/bin/sh", output, goTestStderr{output}, output.String, "-c",
		`printf '%s\n' '{"Action":"start","Package":"github.com/rahoney/heliopause/internal/promotion"}' '{"Action":"run","Package":"github.com/rahoney/heliopause/internal/promotion","Test":"TestSlow"}'; printf '%s\n' 'PRESERVED_STDERR_DETAIL' >&2; exec /bin/sleep 30`)
	var failure *checkFailure
	if !errors.Is(err, context.DeadlineExceeded) || !errors.As(err, &failure) || failure.class != executionFailure || failure.step != "default test diagnostic fixture" {
		t.Fatal("process deadline/cause lost", err)
	}
	for _, wanted := range []string{"PRESERVED_STDERR_DETAIL", "test=TestSlow state=running", "command timed out", "available=true exited=false success=false exit_code=-1"} {
		if !strings.Contains(err.Error(), wanted) {
			t.Fatal("first output/active test absent", wanted, text, err)
		}
	}
}
