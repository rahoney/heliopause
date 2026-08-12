package bootstrap_test

import (
	"bytes"
	"context"
	"errors"
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
