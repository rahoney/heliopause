package bootstrap

import (
	"bytes"
	"context"
	"os"
	"testing"
)

func TestPNPMHelpAndInvalidArgumentsDoNotPrepareRuntimeOrCache(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("PATH", t.TempDir())
	for _, operation := range []string{"add", "install", "update"} {
		var out, diagnostic bytes.Buffer
		if err := Run(context.Background(), []string{"--lang", "en", "pnpm", operation, "--help"}, &out, &diagnostic); err != nil {
			t.Fatal("help required external tools", operation, err)
		}
		if out.Len() == 0 {
			t.Fatal("missing static help")
		}
	}
	for _, args := range [][]string{{"pnpm", "add"}, {"pnpm", "install", "extra"}, {"pnpm", "install", "--target="}, {"pnpm", "update", "--registry=https://example.invalid"}} {
		var out, diagnostic bytes.Buffer
		if err := Run(context.Background(), append([]string{"--lang", "en"}, args...), &out, &diagnostic); err == nil {
			t.Fatal("invalid PNPM request accepted", args)
		}
	}
	entries, err := os.ReadDir(cache)
	if err != nil || len(entries) != 0 {
		t.Fatal("inert requests created cache state", err)
	}
}
