package npm

import (
	"archive/tar"
	"context"
	"errors"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func pnpmRunCacheFixture(t *testing.T) []layoutTarTestEntry {
	return []layoutTarTestEntry{
		layoutTestEntry("dlx", tar.TypeDir, "", ""),
		layoutTestEntry("dlx/key", tar.TypeDir, "", ""),
		layoutTestEntry("dlx/key/prepared", tar.TypeDir, "", ""),
		layoutTestEntry("dlx/key/pkg", tar.TypeSymlink, "", "prepared"),
		layoutTestEntry("dlx/key/prepared/package.json", tar.TypeReg, pnpmSingleManifest, ""),
		layoutTestEntry("dlx/key/prepared/pnpm-lock.yaml", tar.TypeReg, string(pnpmFixture(t, "single")), ""),
	}
}

func TestPNPMRunCacheBindsSingleNativeRootAndRetainsUnapprovedData(t *testing.T) {
	for _, locator := range []string{"is-number", "is-number@6.0.0"} {
		reference, _ := ParseReference(locator)
		cache, err := ParsePNPMRunCache(context.Background(), layoutTarBody(t, pnpmRunCacheFixture(t)), reference, runtimeidentity.PNPMVersion)
		if err != nil || cache.Root() != "dlx/key/prepared" || len(cache.Controls()) != 2 || len(cache.Selection().Packages()) != 1 || len(cache.Layout().Entries()) != 6 {
			t.Fatal("complete native DATA missing", err)
		}
		controls := cache.Controls()
		controls[0] = domain.ProjectControlFile{}
		if !cache.Controls()[0].Present() {
			t.Fatal("caller changed retained native controls")
		}
	}
	if (PNPMRunCache{}).Root() != "" || len((PNPMRunCache{}).Controls()) != 0 {
		t.Fatal("zero cache contains a native selection")
	}
}

func TestPNPMRunCacheRejectsForeignAliasIncompleteControlsAndSubstitution(t *testing.T) {
	for _, mutation := range []string{"foreign-link", "traversal-link", "missing-link", "extra-cache", "missing-lock", "control-symlink", "control-owner", "wrong-version", "wrong-package"} {
		t.Run(mutation, func(t *testing.T) {
			entries := pnpmRunCacheFixture(t)
			locator := "is-number@6.0.0"
			switch mutation {
			case "foreign-link":
				entries[3].header.Linkname = "/foreign/key/prepared"
			case "traversal-link":
				entries[3].header.Linkname = "/fixed/cache/dlx/key/../key/prepared"
			case "missing-link":
				entries = append(entries[:3], entries[4:]...)
			case "extra-cache":
				entries = append(entries, layoutTestEntry("dlx/other", tar.TypeDir, "", ""))
			case "missing-lock":
				entries = entries[:5]
			case "control-symlink":
				entries[4] = layoutTestEntry(entries[4].header.Name, tar.TypeSymlink, "", "elsewhere")
			case "control-owner":
				entries[4].header.Uid = 0
			case "wrong-version":
				locator = "is-number@7.0.0"
			case "wrong-package":
				locator = "other@6.0.0"
			}
			reference, _ := ParseReference(locator)
			cache, err := ParsePNPMRunCache(context.Background(), layoutTarBody(t, entries), reference, runtimeidentity.PNPMVersion)
			if err == nil || cache.Root() != "" || len(cache.Controls()) != 0 {
				t.Fatal("incomplete or substituted native DATA survived", err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	reference, _ := ParseReference("is-number@6.0.0")
	if cache, err := ParsePNPMRunCache(ctx, layoutTarBody(t, pnpmRunCacheFixture(t)), reference, runtimeidentity.PNPMVersion); !errors.Is(err, context.Canceled) || cache.Root() != "" {
		t.Fatal("canceled cache survived", err)
	}
	for _, locator := range []string{"node", "npm@11.0.0", "pnpm", "yarn", "bun", "deno", "rust", "is-number@latest", "is-number@next", "is-number@^6", "https://example.invalid/pkg.tgz"} {
		source, _ := domain.NewSourceID("npm")
		reference, _ := domain.NewArtifactReference(source, locator)
		if err := ValidatePNPMRunReference(reference); err == nil {
			t.Fatal("provisioning or unbounded reference accepted", locator)
		}
	}
	foreign, _ := domain.NewSourceID("pypi")
	reference, _ = domain.NewArtifactReference(foreign, "is-number")
	if ValidatePNPMRunReference(reference) == nil {
		t.Fatal("foreign source became npm selection")
	}
}
