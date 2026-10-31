package bootstrap

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUVToolHelpAndInvalidInputDoNotPrepareRuntime(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("PATH", cache)
	for _, language := range []string{"en", "ko"} {
		for _, operation := range []string{"install", "upgrade"} {
			var out, diagnostic bytes.Buffer
			if err := Run(context.Background(), []string{"--lang", language, "uv", "tool", operation, "--help"}, &out, &diagnostic); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "--bin-dir") || !strings.Contains(out.String(), "--target") || !strings.Contains(out.String(), "cowsay") {
				t.Fatal("tool scope missing from static help")
			}
		}
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"uv", "tool", "install", "cowsay"}, {"uv", "tool", "install", "cowsay", "--target", root, "--bin-dir", root}, {"uv", "tool", "install", "cowsay", "--with", "other"}} {
		if err := Run(context.Background(), args, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
			t.Fatal("invalid tool input accepted")
		}
	}
	for _, args := range [][]string{{"uv", "tool", "upgrade", "cowsay"}, {"uv", "tool", "upgrade", "cowsay", "--all"}} {
		if err := Run(context.Background(), args, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
			t.Fatal("invalid upgrade accepted")
		}
	}
	entries, err := os.ReadDir(cache)
	if err != nil || len(entries) != 0 {
		t.Fatal("help/invalid input prepared runtime or cache")
	}
}

func TestUVToolPlatformAdmission(t *testing.T) {
	for _, platform := range [][2]string{{"linux", "arm64"}, {"darwin", "amd64"}, {"windows", "amd64"}} {
		if err := uvToolClientPlatform(platform[0], platform[1]); err == nil {
			t.Fatal("unsupported tool platform accepted")
		}
	}
	if err := uvToolClientPlatform("linux", "amd64"); err != nil {
		t.Fatal(err)
	}
}
