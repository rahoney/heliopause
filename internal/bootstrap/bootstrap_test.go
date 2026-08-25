package bootstrap_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/bootstrap"
)

func TestRunPreservesCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := bootstrap.Run(ctx, nil, &stdout, &stderr)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context.Canceled", err)
	}
}

func TestRunHelpDoesNotRequireRuntimeComposition(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := bootstrap.Run(context.Background(), []string{"npm", "--help"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !strings.Contains(stdout.String(), "install") || !strings.Contains(stdout.String(), "inspect") {
		t.Fatalf("npm help output = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}
