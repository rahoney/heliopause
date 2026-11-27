//go:build linux

package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/ports"
)

func TestBootstrapPreparationCompositionUsesOrdinaryServiceBeforeRuntime(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out, diagnostic bytes.Buffer
	err := Run(ctx, []string{"--lang", "en", "--output", "json", "bootstrap", "prepare", "--version", "v1.2.3"}, &out, &diagnostic)
	var failure *ports.BootstrapPreparationFailure
	if !errors.As(err, &failure) || failure.Code() != ports.BootstrapCancelled || !strings.Contains(out.String(), string(ports.BootstrapCancelled)) {
		t.Fatal("bootstrap constructed an unrelated runtime", out.String(), err)
	}
}
