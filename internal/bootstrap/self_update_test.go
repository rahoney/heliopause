package bootstrap_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/bootstrap"
)

func TestSelfUpdateHelpAndInvalidInputDoNotComposeRuntime(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	for _, args := range [][]string{{"self-update", "--help", "--lang", "ko"}, {"self-update", "--version", "latest"}, {"self-update", "--version", "v1.2"}, {"self-update", "--version", "v1.2.3", "--target", "/tmp/unsupported"}, {"self-update", "--version", "v1.2.3", "--version=v2.0.0"}} {
		var out, diagnostic bytes.Buffer
		err := bootstrap.Run(context.Background(), args, &out, &diagnostic)
		if args[1] == "--help" {
			if err != nil || !strings.Contains(out.String(), "--version") {
				t.Fatal(args, err, out.String())
			}
		} else if err == nil {
			t.Fatal("invalid input admitted", args)
		}
	}
	if _, err := os.Lstat(filepath.Join(cache, "heliopause")); !os.IsNotExist(err) {
		t.Fatal("inert request created runtime/acquisition state", err)
	}
}
