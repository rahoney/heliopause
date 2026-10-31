package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
)

func TestUVRunHelpAndInvalidInputAreInertBeforeRuntime(t *testing.T) {
	cache, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("PATH", cache)
	for _, path := range [][]string{{"uvx"}, {"uv", "tool", "run"}} {
		for _, language := range []string{"en", "ko"} {
			args := append([]string{"--lang", language}, path...)
			args = append(args, "--help")
			var out, diagnostic bytes.Buffer
			if err := Run(context.Background(), args, &out, &diagnostic); err != nil {
				t.Fatal("static native help", err)
			}
			if !strings.Contains(out.String(), "--from") || !strings.Contains(out.String(), "--show-output") {
				t.Fatal("native help absent")
			}
		}
		for _, tail := range [][]string{{}, {"--python", "3.14", "cowsay"}, {"cowsay[extra]"}, {"--from=", "cowsay"}, {"python3.14"}, {"--from", "cowsay", "--from", "other", "command"}} {
			args := append(append([]string{}, path...), tail...)
			if err := Run(context.Background(), args, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
				t.Fatal("invalid native input accepted", args)
			}
		}
	}
	entries, err := os.ReadDir(cache)
	if err != nil || len(entries) != 0 {
		t.Fatal("help/invalid input created cache or runtime", err)
	}
}
func TestUVRunCLIPlatformAndCancellation(t *testing.T) {
	for _, platform := range [][2]string{{"darwin", "amd64"}, {"linux", "arm64"}, {"windows", "amd64"}} {
		if err := uvRunExecutionPlatform(platform[0], platform[1]); err == nil {
			t.Fatal("unsupported platform admitted")
		}
	}
	if err := uvRunExecutionPlatform("linux", "amd64"); err != nil {
		t.Fatal(err)
	}
	cause := errors.New("caller stopped native CLI before factory")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if _, err := (pythonUVRunCLI{}).Run(ctx, application.ExecutionPreparationRequest{}); !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
		t.Fatal("original CLI cancellation lost", err)
	}
	for _, input := range []context.Context{nil, context.Background()} {
		if _, err := (pythonUVRunCLI{}).Run(input, application.ExecutionPreparationRequest{}); err == nil {
			t.Fatal("absent context or current runtime bindings accepted")
		}
	}
}
