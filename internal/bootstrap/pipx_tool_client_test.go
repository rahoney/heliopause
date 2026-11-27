package bootstrap

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestPipxHelpAndInvalidInputDoNotPrepareRuntime(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("PATH", cache)
	for _, lang := range []string{"en", "ko"} {
		for _, operation := range []string{"install", "upgrade"} {
			var out, diagnostic bytes.Buffer
			if err := Run(context.Background(), []string{"--lang", lang, "pipx", operation, "--help"}, &out, &diagnostic); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "--bin-dir") || !strings.Contains(out.String(), "--target") || !strings.Contains(out.String(), "cowsay") {
				t.Fatal("pipx static supported scope missing")
			}
		}
	}
	for _, args := range [][]string{{"pipx", "install", "cowsay"}, {"pipx", "upgrade", "cowsay"}, {"pipx", "upgrade", "cowsay", "--adopt"}, {"pipx", "run", "cowsay"}} {
		if err := Run(context.Background(), args, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
			t.Fatal("invalid pipx input accepted", args)
		}
	}
	entries, err := os.ReadDir(cache)
	if err != nil || len(entries) != 0 {
		t.Fatal("pipx preflight prepared runtime/cache", err)
	}
}
func TestPipxCompositionHasDistinctNativeAndExclusiveLifetimeOwner(t *testing.T) {
	client, err := pythonPackageClientFor(cli.PipxPackageClient)
	if err != nil || client.id != cli.PipxPackageClient || client.derivation || client.resolver == nil || client.bind == nil || client.close == nil {
		t.Fatal("distinct pipx composition missing", err)
	}
	for _, platform := range [][2]string{{"linux", "arm64"}, {"darwin", "amd64"}, {"windows", "amd64"}} {
		if client.checkPlatform(platform[0], platform[1]) == nil {
			t.Fatal("unsupported platform accepted")
		}
		if _, err := client.resolver(platform[0], platform[1], nil, nil); err == nil {
			t.Fatal("unsupported platform fell back")
		}
	}
	if client.checkPlatform("linux", "amd64") != nil {
		t.Fatal("supported platform rejected")
	}
	for _, id := range []cli.PackageClientID{"", cli.PipPackageClient, cli.UVPipPackageClient, cli.UVToolPackageClient} {
		other, err := pythonPackageClientFor(id)
		if err != nil || other.close != nil {
			t.Fatal("existing client lifetime changed", id, err)
		}
	}
	if err := client.bind(nil, pythonClientBindings{nativeLookup: new(sandbox.UVResolver)}); err == nil {
		t.Fatal("pipx silently bound UV tool selection")
	}
	if err := client.close(new(sandbox.UVResolver)); err == nil {
		t.Fatal("foreign resolver treated as absence")
	}
	if _, err := newPipxToolInstallService(pythonClientBindings{}, nil, ""); err == nil {
		t.Fatal("unprepared install accepted")
	}
	if _, err := newPipxToolUpdateService(pythonClientBindings{}, nil, ""); err == nil {
		t.Fatal("unprepared update accepted")
	}
}
