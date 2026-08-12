package cli_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/cli"
)

func TestRootHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command, err := cli.New(&stdout, &stderr)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	command.SetArgs([]string{"--help"})

	if err = command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Usage:\n  helox") {
		t.Fatalf("help output missing usage: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestNewRejectsNilWriters(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stdout io.Writer
		stderr io.Writer
	}{
		"stdout": {stdout: nil, stderr: io.Discard},
		"stderr": {stdout: io.Discard, stderr: nil},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := cli.New(test.stdout, test.stderr); err == nil {
				t.Fatal("New error = nil, want non-nil")
			}
		})
	}
}
