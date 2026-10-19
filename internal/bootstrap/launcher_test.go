package bootstrap_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/bootstrap"
)

func TestBareLauncherFallbackDoesNotResolveReleaseRoot(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HELOX_LANG", "en")
	var output, diagnostic bytes.Buffer
	if err := bootstrap.Run(context.Background(), []string{}, &output, &diagnostic); err != nil {
		t.Fatal("bare help eagerly resolved/composed a runtime:", err)
	}
	if !strings.Contains(output.String(), "helox [") || diagnostic.Len() != 0 {
		t.Fatal("bare fallback failed without a user config root", output.String(), diagnostic.String())
	}
}
