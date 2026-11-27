package bootstrap

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
)

func TestUVProjectHelpAndInvalidArgumentsDoNotPrepareRuntimeOrCache(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("PATH", t.TempDir())
	for _, language := range []string{"en", "ko"} {
		for _, operation := range []string{"add", "sync"} {
			var out, diagnostic bytes.Buffer
			if err := Run(context.Background(), []string{"--lang", language, "uv", operation, "--help"}, &out, &diagnostic); err != nil {
				t.Fatal("static UV help required runtime", err)
			}
			if !strings.Contains(out.String(), "--project") || !strings.Contains(out.String(), ".venv") {
				t.Fatal("UV scope missing from help", out.String())
			}
		}
	}
	for _, item := range []struct {
		args    []string
		message string
	}{
		{[]string{"uv", "add"}, "accepts 1 arg"},
		{[]string{"uv", "sync", "extra"}, "unknown command"},
		{[]string{"uv", "sync", "--project="}, "UV --project must not be empty"},
		{[]string{"uv", "sync", "--index-url=https://example.invalid"}, "unknown flag"},
	} {
		var out, diagnostic bytes.Buffer
		err := Run(context.Background(), append([]string{"--lang", "en"}, item.args...), &out, &diagnostic)
		if err == nil || !strings.Contains(err.Error(), item.message) {
			t.Fatal("input did not fail at syntax preflight", item.args, err)
		}
	}
	if entries, err := os.ReadDir(cache); err != nil || len(entries) != 0 {
		t.Fatal("inert UV request created cache state", err)
	}
}
