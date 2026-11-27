package bootstrap_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/bootstrap"
)

func TestUVSyncBlankAndInvalidSourceDoNotPrepareRuntimeOrCache(t *testing.T) {
	for _, mode := range []string{"blank", "invalid", "help"} {
		t.Run(mode, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			cache := filepath.Join(root, "unused-cache")
			t.Setenv("XDG_CACHE_HOME", cache)
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
			source := filepath.Join(root, "requirements.txt")
			body := "# empty input\n"
			if mode == "invalid" {
				body = "-r other.txt\n"
			}
			if err := os.WriteFile(source, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			args := []string{"--lang", "en", "--output", "json", "uv", "pip", "sync", source, "--target", filepath.Join(root, "untouched-target")}
			if mode == "help" {
				args = []string{"--lang", "en", "uv", "pip", "sync", "--help"}
			}
			var out, diagnostic bytes.Buffer
			err = bootstrap.Run(context.Background(), args, &out, &diagnostic)
			if (err != nil) != (mode == "invalid") {
				t.Fatal("wrong preflight result", mode, err, out.String(), diagnostic.String())
			}
			if mode == "blank" {
				var doc map[string]json.RawMessage
				if json.Unmarshal(out.Bytes(), &doc) != nil || string(doc["operation_status"]) != `"SKIPPED"` {
					t.Fatal(out.String())
				}
				if _, exists := doc["policy"]; exists {
					t.Fatal("blank input generated Policy")
				}
			}
			if _, err := os.Lstat(cache); !os.IsNotExist(err) {
				t.Fatal("inert input prepared cache", err)
			}
			if _, err := os.Lstat(filepath.Join(root, "untouched-target")); !os.IsNotExist(err) {
				t.Fatal("inert input touched target", err)
			}
		})
	}
}
