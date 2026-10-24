package npm

import (
	"archive/tar"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestPNPMInventoryBinsUseOnlyOriginalRegularTargets(t *testing.T) {
	f := fullInspectionLayoutFixture(t, false)
	var original ArchiveInventory
	for _, inventory := range f.inventories {
		if inventory.Artifact().Identity().Name() == "loose-envify" {
			original = inventory
		}
	}
	for _, test := range []struct {
		name, extra string
		accepted    bool
	}{
		{"string", `"bin":"./cli.js"`, true},
		{"object", `"bin":{"tool":"cli.js"}`, true},
		{"empty-bundles", `"bin":"cli.js","bundledDependencies":[]`, true},
		{"bundles", `"bin":"cli.js","bundledDependencies":["foreign"]`, false},
		{"bundle-all", `"bin":"cli.js","bundleDependencies":true`, false},
		{"directory-discovery", `"directories":{"bin":"bin"}`, false},
		{"missing-target", `"bin":"missing.js"`, false},
		{"escape", `"bin":"../cli.js"`, false},
		{"command-path", `"bin":{"dir/tool":"cli.js"}`, false},
		{"command-injection", `"bin":{"$(tool)":"cli.js"}`, false},
		{"array", `"bin":["cli.js"]`, false},
		{"duplicate", `"bin":"cli.js","bin":"other.js"`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			manifest := []byte(`{"name":"loose-envify","version":"1.4.0",` + test.extra + `}`)
			entries := original.Entries()
			for index := range entries {
				if entries[index].Path == "package.json" {
					entries[index].Size = int64(len(manifest))
					entries[index].Digest = layoutDigestFixture(t, manifest)
				}
			}
			inventory, err := NewArchiveInventory(original.Artifact(), entries, manifest)
			if err != nil {
				t.Fatal("invalid structural fixture", err)
			}
			bins, err := pnpmInventoryBins(inventory)
			if (err == nil) != test.accepted {
				t.Fatal("unexpected bin interpretation", bins, err)
			}
		})
	}
}

func TestPNPMInspectionLayoutRejectsUnrelatedNodeSearchAndUnequalAlias(t *testing.T) {
	for _, change := range []string{"node-search", "unequal-alias", "foreign-inventory", "bundled-root"} {
		t.Run(change, func(t *testing.T) {
			f := fullInspectionLayoutFixture(t, false)
			switch change {
			case "node-search":
				for index := range f.entries {
					e := &f.entries[index]
					if e.header.Name != "node_modules/.pnpm/loose-envify@1.4.0/node_modules/loose-envify/node_modules/.bin/loose-envify" {
						continue
					}
					value := "$basedir_abs/../../../../../react@18.3.1/node_modules/react/node_modules"
					block := "if [ -z \"$NODE_PATH\" ]; then\n  export NODE_PATH=\"" + value + "\"\nelse\n  export NODE_PATH=\"" + value + ":$NODE_PATH\"\nfi\n"
					e.body = strings.Replace(e.body, "if [ -n \"$exe\" ]", block+"if [ -n \"$exe\" ]", 1)
					e.header.Size = int64(len(e.body))
					if _, err := ParsePNPMNodeBinShim([]byte(e.body), e.header.Name, "node_modules/.pnpm/loose-envify@1.4.0/node_modules/loose-envify/cli.js"); err != nil {
						t.Fatal("node-search fixture failed before context check", err)
					}
				}
			case "unequal-alias":
				for index := range f.entries {
					e := &f.entries[index]
					if !strings.HasSuffix(e.header.Name, "/react/LICENSE") {
						continue
					}
					e.header.Typeflag = tar.TypeLink
					e.header.Linkname = "node_modules/.pnpm/loose-envify@1.4.0/node_modules/loose-envify/cli.js"
					e.header.Size = 0
					e.body = ""
				}
			case "foreign-inventory":
				other := fullInspectionLayoutFixture(t, false)
				// Same identities, sizes and digests, but fresh exact acquired runs.
				f.inventories = other.inventories
			case "bundled-root":
				inventory := f.inventories[0]
				entries := append(inventory.Entries(), ArchiveEntry{Path: "node_modules", Directory: true, Mode: 0o755})
				var err error
				f.inventories[0], err = NewArchiveInventory(inventory.Artifact(), entries, inventory.Manifest())
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, err := f.input.BindLayout(context.Background(), f.inventories, f.parse(t), "/tmp/haa-store"); err == nil {
				t.Fatal("foreign layout context admitted")
			}
		})
	}
}

func TestPNPMPrivateManifestKeepsExactDevGroupWithoutUserMetadata(t *testing.T) {
	f := fullInspectionLayoutFixture(t, true)
	lock := f.input.Context().Lock()
	lock.roots[0].Group = "devDependencies"
	manifest, err := pnpmInspectionManifest(lock)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Private         bool              `json:"private"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if json.Unmarshal(manifest, &decoded) != nil || !decoded.Private || len(decoded.DevDependencies) != 1 || decoded.DevDependencies[lock.roots[0].Name] == "" || len(decoded.Dependencies) != 2 {
		t.Fatal("exact dependency grouping lost")
	}
}

func TestPNPMInspectionLayoutCacheIsAccountedForButNeverIntroduced(t *testing.T) {
	for _, change := range []string{"missing", "json", "duplicate", "scalar", "alias", "oversize"} {
		t.Run(change, func(t *testing.T) {
			f := fullInspectionLayoutFixture(t, false)
			for index := 0; index < len(f.entries); index++ {
				e := &f.entries[index]
				if e.header.Name != pnpmWorkspaceCachePath {
					continue
				}
				switch change {
				case "missing":
					f.entries = append(f.entries[:index], f.entries[index+1:]...)
					continue
				case "json":
					e.body = "{invalid}"
				case "duplicate":
					e.body = `{"validation":{},"validation":{}}`
				case "scalar":
					e.body = `"untrusted"`
				case "oversize":
					e.body = `{"padding":"` + strings.Repeat("x", ProjectControlLimit) + `"}`
				case "alias":
					e.body = ""
					e.header.Typeflag = tar.TypeLink
					e.header.Linkname = "node_modules/.modules.yaml"
				}
				e.header.Size = int64(len(e.body))
			}
			if _, err := f.input.BindLayout(context.Background(), f.inventories, f.parse(t), "/tmp/haa-store"); err == nil {
				t.Fatal("malformed native cache admitted")
			}
		})
	}
}

func TestPNPMInspectionDirectoryRestrictionRemovesOnlyWriteBits(t *testing.T) {
	f := fullInspectionLayoutFixture(t, false)
	for index := range f.entries {
		if f.entries[index].header.Typeflag == tar.TypeDir {
			f.entries[index].header.Mode = 0o777
		}
	}
	raw := f.parse(t)
	if _, err := f.input.BindLayout(context.Background(), f.inventories, raw, "/tmp/haa-store"); err == nil {
		t.Fatal("raw writable directories admitted")
	}
	restricted := raw.RestrictDirectoryWrites()
	if _, err := f.input.BindLayout(context.Background(), f.inventories, restricted, "/tmp/haa-store"); err != nil {
		t.Fatal("restricted inspection view rejected", err)
	}
	before, after := raw.Entries(), restricted.Entries()
	for index, original := range before {
		expected := original
		if expected.Kind == tar.TypeDir {
			expected.Mode &^= 0o022
		}
		if after[index] != expected || raw.Entries()[index] != original {
			t.Fatal("restriction changed bytes, links, identity or original headers")
		}
		if original.Kind == tar.TypeReg {
			a, err := raw.ReadFile(original.Name)
			if err != nil {
				t.Fatal(err)
			}
			b, err := restricted.ReadFile(original.Name)
			if err != nil || string(a) != string(b) {
				t.Fatal("restriction changed regular bytes", err)
			}
		}
	}
	for _, mode := range []int64{0o666, 0o555} {
		changed := fullInspectionLayoutFixture(t, false)
		for index := range changed.entries {
			e := &changed.entries[index]
			if mode == 0o666 && e.header.Typeflag == tar.TypeReg {
				e.header.Mode = mode
			}
			if mode == 0o555 && e.header.Typeflag == tar.TypeDir {
				e.header.Mode = mode
			}
		}
		if _, err := changed.input.BindLayout(context.Background(), changed.inventories, changed.parse(t).RestrictDirectoryWrites(), "/tmp/haa-store"); err == nil {
			t.Fatal("restriction repaired unsafe regular or missing owner permissions")
		}
	}
	alias := fullInspectionLayoutFixture(t, false)
	for index := range alias.entries {
		e := &alias.entries[index]
		if strings.HasSuffix(e.header.Name, "/react/LICENSE") {
			e.header.Typeflag = tar.TypeLink
			e.header.Linkname = "node_modules/.pnpm/use-sync-external-store@1.2.0_react@18.3.1/node_modules/use-sync-external-store/LICENSE"
			e.header.Mode = 0o666
			e.header.Size = 0
			e.body = ""
		}
	}
	if _, err := alias.input.BindLayout(context.Background(), alias.inventories, alias.parse(t).RestrictDirectoryWrites(), "/tmp/haa-store"); err == nil {
		t.Fatal("restriction changed an unsafe hardlink mode")
	}
}
