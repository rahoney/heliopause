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
}
