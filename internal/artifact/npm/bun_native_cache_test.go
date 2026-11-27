package npm

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"testing"
)

func bunCacheTarFixture(t *testing.T, f bunLayoutTestFixture, inventory ArchiveInventory) []layoutTarTestEntry {
	t.Helper()
	root, err := f.input.NativeCacheFolder(inventory.Artifact())
	if err != nil {
		t.Fatal(err)
	}
	wanted := map[string]layoutTarTestEntry{root: layoutTestEntry(root, tar.TypeDir, "", "")}
	for _, original := range inventory.Entries() {
		name := root + "/" + original.Path
		if original.Directory {
			wanted[name] = layoutTestEntry(name, tar.TypeDir, "", "")
		} else {
			found := false
			for _, entry := range f.entries {
				if entry.header.Name == "node_modules/"+inventory.Artifact().Identity().Name()+"/"+original.Path {
					copy := layoutTestEntry(name, tar.TypeReg, entry.body, "")
					copy.header.Mode = original.Mode &^ 0o022
					wanted[name] = copy
					found = true
					break
				}
			}
			if !found {
				t.Fatal("fixture source body missing")
			}
		}
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			wanted[parent] = layoutTestEntry(parent, tar.TypeDir, "", "")
		}
	}
	sri, _ := inventory.Artifact().DeclaredIntegrity()
	wanted[root+"/.bun-tag"] = layoutTestEntry(root+"/.bun-tag", tar.TypeReg, sri, "")
	var names []string
	for name := range wanted {
		names = append(names, name)
	}
	slices.Sort(names)
	var entries []layoutTarTestEntry
	for _, name := range names {
		entries = append(entries, wanted[name])
	}
	return entries
}

func bunBoundCachesFixture(t *testing.T, f bunLayoutTestFixture) []BunNativeCacheEntry {
	t.Helper()
	var caches []BunNativeCacheEntry
	for _, inventory := range f.inventories {
		entry, err := f.input.BindNativeCache(context.Background(), inventory, layoutTarBody(t, bunCacheTarFixture(t, f, inventory)))
		if err != nil {
			t.Fatal(err)
		}
		caches = append(caches, entry)
	}
	return caches
}

func TestBunNativeCacheMeasuresEverySourceFileAndDropsOnlyNativeMarker(t *testing.T) {
	f := bunLayoutFixture(t, false)
	if err := f.input.CheckNativeCacheInputs(f.inventories); err != nil {
		t.Fatal(err)
	}
	caches := bunBoundCachesFixture(t, f)
	body, err := f.input.RenderRegistryCache(context.Background(), caches)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parsePNPMTreeArchive(context.Background(), body, "cache")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range parsed.Entries() {
		if strings.HasSuffix(entry.Name, "/.bun-tag") || entry.Kind == tar.TypeSymlink || entry.Kind == tar.TypeLink {
			t.Fatal("native bookkeeping or alias entered registry envelope")
		}
	}
	for _, inventory := range f.inventories {
		folder, err := bunRegistryCacheFolder(inventory.Artifact())
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range inventory.Entries() {
			if file.Directory {
				continue
			}
			content, err := parsed.ReadFile("cache/" + folder + "/" + file.Path)
			if err != nil || layoutDigestFixture(t, content) != file.Digest || int64(len(content)) != file.Size {
				t.Fatal("source bytes changed", err)
			}
		}
	}
	for n := range body {
		body[n] = 0
	}
	again, err := f.input.RenderRegistryCache(context.Background(), caches)
	if err != nil || bytes.Equal(body, again) {
		t.Fatal("caller changed bound native data", err)
	}
}

func TestBunNativeCacheRejectsIncompleteChangedForeignAndAliasData(t *testing.T) {
	for _, fault := range []string{"missing", "extra", "marker", "missing-marker", "bytes", "exec-mode", "owner", "alias", "directory-mode"} {
		t.Run(fault, func(t *testing.T) {
			f := bunLayoutFixture(t, false)
			inventory := f.inventories[0]
			entries := bunCacheTarFixture(t, f, inventory)
			root, err := f.input.NativeCacheFolder(inventory.Artifact())
			if err != nil {
				t.Fatal(err)
			}
			for n := range entries {
				entry := &entries[n]
				if entry.header.Name == root+"/LICENSE" {
					switch fault {
					case "missing":
						entries = append(entries[:n], entries[n+1:]...)
					case "bytes":
						entry.body = "X" + entry.body[1:]
					case "exec-mode":
						entry.header.Mode = 0o755
					case "owner":
						entry.header.Uid = 0
					case "alias":
						entry.header.Typeflag = tar.TypeSymlink
						entry.header.Mode = 0o777
						entry.header.Linkname = "index.js"
						entry.body = ""
						entry.header.Size = 0
					}
					break
				}
			}
			if fault == "extra" {
				entries = append(entries, layoutTestEntry(root+"/unknown", tar.TypeReg, "extra", ""))
			}
			for n := range entries {
				if entries[n].header.Name == root && fault == "directory-mode" {
					entries[n].header.Mode = 0o644
				}
				if entries[n].header.Name == root+"/.bun-tag" {
					if fault == "marker" {
						entries[n].body = "X" + entries[n].body[1:]
					}
					if fault == "missing-marker" {
						entries = append(entries[:n], entries[n+1:]...)
						break
					}
				}
			}
			if entry, err := f.input.BindNativeCache(context.Background(), inventory, layoutTarBody(t, entries)); err == nil || entry.bound {
				t.Fatal("unmeasured cache accepted")
			}
		})
	}
}

func TestBunRegistryCacheRequiresCompleteSameContextAndPreservesCancellation(t *testing.T) {
	f := bunLayoutFixture(t, false)
	caches := bunBoundCachesFixture(t, f)
	foreign := bunLayoutFixture(t, false)
	other := bunBoundCachesFixture(t, foreign)
	duplicate := append([]BunNativeCacheEntry(nil), caches...)
	duplicate[0] = duplicate[1]
	for _, entries := range [][]BunNativeCacheEntry{nil, caches[:3], duplicate, other, {caches[0], caches[1], caches[2], {}}} {
		if body, err := f.input.RenderRegistryCache(context.Background(), entries); err == nil || len(body) != 0 {
			t.Fatal("partial/duplicate/foreign cache accepted")
		}
	}
	if _, err := (BunInspectionInput{}).NativeCacheFolder(f.input.Artifacts()[0]); err == nil {
		t.Fatal("zero cache input")
	}
	if _, err := f.input.NativeCacheFolder(foreign.input.Artifacts()[0]); err == nil {
		t.Fatal("foreign acquisition run locator")
	}
	cause := errors.New("cache collection canceled")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if body, err := f.input.RenderRegistryCache(ctx, caches); !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || len(body) != 0 {
		t.Fatal("cancel cause or output lost", err)
	}
	if entry, err := f.input.BindNativeCache(ctx, f.inventories[0], layoutTarBody(t, bunCacheTarFixture(t, f, f.inventories[0]))); !errors.Is(err, cause) || entry.bound {
		t.Fatal("cache bind cancel cause lost", err)
	}
}

func TestBunNativeCacheRejectsOriginalMarkerCollisionBeforeResourceWork(t *testing.T) {
	f := bunLayoutFixture(t, false)
	inventory := f.inventories[0]
	entries := inventory.Entries()
	entries = append(entries, ArchiveEntry{Path: ".bun-tag", Size: 4, Mode: 0o644, Digest: layoutDigestFixture(t, []byte("data"))})
	changed, err := NewArchiveInventory(inventory.Artifact(), entries, inventory.Manifest())
	if err != nil {
		t.Fatal(err)
	}
	f.inventories[0] = changed
	if err := f.input.CheckNativeCacheInputs(f.inventories); err == nil {
		t.Fatal("original source marker collision accepted")
	}
}

func TestBunNativeCacheBoundsImpliedDirectoriesBeforeNativeWork(t *testing.T) {
	f := bunLayoutFixture(t, false)
	inventory := f.inventories[0]
	entries := inventory.Entries()
	for n := 0; n < 1000; n++ {
		name := fmt.Sprintf("deep-%04d/", n) + strings.Repeat("sub/", 99) + "data"
		entries = append(entries, ArchiveEntry{Path: name, Size: 1, Mode: 0o644, Digest: layoutDigestFixture(t, []byte("x"))})
	}
	changed, err := NewArchiveInventory(inventory.Artifact(), entries, inventory.Manifest())
	if err != nil {
		t.Fatal(err)
	}
	f.inventories[0] = changed
	if err := f.input.CheckWorkspace(f.inventories); err != nil {
		t.Fatal("byte/logical budget fixture failed before directory bound", err)
	}
	if err := f.input.CheckNativeCacheInputs(f.inventories); err == nil {
		t.Fatal("unbounded implied cache directories admitted")
	}
}
