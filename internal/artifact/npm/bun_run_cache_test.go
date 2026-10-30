package npm

import (
	"archive/tar"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func bunRunCacheFixture(t *testing.T) []layoutTarTestEntry {
	t.Helper()
	read := func(name string) string {
		body, err := os.ReadFile("testdata/bun-run-cache/exact." + name)
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	}
	entries := []layoutTarTestEntry{
		layoutTestEntry(".bunx-1000", tar.TypeDir, "", ""),
		layoutTestEntry(".bunx-1000/opaque", tar.TypeDir, "", ""),
		layoutTestEntry(".bunx-1000/opaque.lock", tar.TypeReg, "", ""),
		layoutTestEntry(".bunx-1000/opaque/package.json", tar.TypeReg, read("package.json"), ""),
		layoutTestEntry(".bunx-1000/opaque/bun.lock", tar.TypeReg, read("bun.lock"), ""),
	}
	entries[0].header.Mode = 0o700
	entries[2].header.Mode = 0o600
	return entries
}

func TestBunRunCacheRetainsActualControlsAndSingleRequestedRoot(t *testing.T) {
	for _, locator := range []string{"typescript", "typescript@5.9.3"} {
		reference, _ := ParseReference(locator)
		cache, err := ParseBunRunCache(context.Background(), layoutTarBody(t, bunRunCacheFixture(t)), reference, runtimeidentity.BunVersion)
		if err != nil || cache.Root() != ".bunx-1000/opaque" || len(cache.Controls()) != 2 || len(cache.Selection().Roots()) != 1 || len(cache.Selection().Packages()) != 1 || len(cache.Selection().Packages()[0].Bins) != 2 {
			t.Fatal("native DATA/control/bin selection differs", err)
		}
		controls := cache.Controls()
		controls[0] = domain.ProjectControlFile{}
		if !cache.Controls()[0].Present() || len(cache.Layout().Entries()) != 5 {
			t.Fatal("caller altered retained native DATA")
		}
	}
}

func TestBunRunCacheRejectsIncompleteForeignAndSubstitutedData(t *testing.T) {
	for _, mutation := range []string{"missing-lock", "missing-flock", "flock-payload", "flock-link", "public-root", "foreign-owner", "extra-cache", "control-link", "wrong-version", "wrong-package", "unsupported-shape"} {
		t.Run(mutation, func(t *testing.T) {
			entries := bunRunCacheFixture(t)
			locator := "typescript@5.9.3"
			switch mutation {
			case "missing-lock":
				entries = entries[:4]
			case "missing-flock":
				entries = append(entries[:2], entries[3:]...)
			case "flock-payload":
				entries[2] = layoutTestEntry(entries[2].header.Name, tar.TypeReg, "foreign", "")
			case "flock-link":
				entries[2] = layoutTestEntry(entries[2].header.Name, tar.TypeSymlink, "", "opaque/package.json")
			case "public-root":
				entries[0].header.Mode = 0o755
			case "foreign-owner":
				entries[3].header.Uid = 0
			case "extra-cache":
				entries = append(entries, layoutTestEntry(".bunx-1000/foreign", tar.TypeDir, "", ""))
			case "control-link":
				entries[3] = layoutTestEntry(entries[3].header.Name, tar.TypeSymlink, "", "bun.lock")
			case "wrong-version":
				locator = "typescript@5.9.2"
			case "wrong-package":
				locator = "foreign@5.9.3"
			case "unsupported-shape":
				entries[4] = layoutTestEntry(entries[4].header.Name, tar.TypeReg, `{"lockfileVersion":99}`, "")
			}
			reference, _ := ParseReference(locator)
			cache, err := ParseBunRunCache(context.Background(), layoutTarBody(t, entries), reference, runtimeidentity.BunVersion)
			if err == nil || cache.Root() != "" || len(cache.Controls()) != 0 || len(cache.Selection().Roots()) != 0 {
				t.Fatal("incomplete DATA exposed a native selection", err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	reference, _ := ParseReference("typescript")
	if cache, err := ParseBunRunCache(ctx, layoutTarBody(t, bunRunCacheFixture(t)), reference, runtimeidentity.BunVersion); !errors.Is(err, context.Canceled) || cache.Root() != "" {
		t.Fatal("canceled native DATA survived", err)
	}
	for _, locator := range []string{"node", "bun", "pnpm", "npm", "deno", "rust", "yarn", "typescript@latest", "typescript@^5", "file:typescript"} {
		source, _ := domain.NewSourceID("npm")
		reference, _ := domain.NewArtifactReference(source, locator)
		if ValidateBunRunReference(reference) == nil {
			t.Fatal("provisioning or non-package native input accepted", locator)
		}
	}
}
