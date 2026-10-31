package npm

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"testing"
)

type bunLayoutTestFixture struct {
	input       BunInspectionInput
	inventories []ArchiveInventory
	entries     []layoutTarTestEntry
}

func bunLayoutFixture(t *testing.T, repeated bool) bunLayoutTestFixture {
	t.Helper()
	manifest, lock, present := bunActualControls(t, "peer")
	roots := map[string]string{"js-tokens": "node_modules/js-tokens", "loose-envify": "node_modules/loose-envify", "react": "node_modules/react", "use-sync-external-store": "node_modules/use-sync-external-store"}
	if repeated {
		constraints, rows := bunScopedRows(true)
		manifest, lock = bunSyntheticLock(t, constraints, rows)
		present = true
		roots = map[string]string{"@one/root": "node_modules/@one/root", "@two/lib": "node_modules/@two/lib", "@one/root/@two/lib": "node_modules/@one/root/node_modules/@two/lib", "@one/root/consumer": "node_modules/@one/root/node_modules/consumer"}
	}
	selected, controls := bunTestSelection(t, manifest, lock, present)
	frozen, err := selected.InspectionContext(pnpmSnapshotContext(t, "/tmp/haa-bun-layout", true), controls, "fixture-runtime:bun")
	if err != nil {
		t.Fatal(err)
	}
	input, err := frozen.BindAcquired(frozen.Snapshot(), pnpmAcquiredInspectionFixture(t, frozen))
	if err != nil {
		t.Fatal(err)
	}
	f := bunLayoutTestFixture{input: input}
	packages := map[string]BunLockedPackage{}
	for _, pkg := range selected.Lock().Packages() {
		packages[pkg.Name+"@"+pkg.Version] = pkg
	}
	bodies := map[string]map[string]string{}
	modes := map[string]map[string]int64{}
	for _, subject := range input.Artifacts() {
		key := subject.Identity().Name() + "@" + subject.Identity().Version()
		pkg := packages[key]
		body, err := json.Marshal(bunMetadata(t, pkg))
		if err != nil {
			t.Fatal(err)
		}
		files := map[string]string{"package.json": string(body), "LICENSE": "same original license bytes\n", "index.js": "module.exports = {};\n"}
		fileModes := map[string]int64{}
		for _, file := range pkg.Bins {
			files[file] = "#!/usr/bin/env node\nconsole.log('bounded fixture');\n"
			fileModes[file] = 0o755
		}
		var entries []ArchiveEntry
		for name, content := range files {
			mode := fileModes[name]
			if mode == 0 {
				mode = 0o644
				fileModes[name] = mode
			}
			entries = append(entries, ArchiveEntry{Path: name, Size: int64(len(content)), Mode: mode, Digest: layoutDigestFixture(t, []byte(content))})
		}
		inventory, err := NewArchiveInventory(subject, entries, body)
		if err != nil {
			t.Fatal(err)
		}
		f.inventories = append(f.inventories, inventory)
		bodies[key] = files
		modes[key] = fileModes
	}
	wanted := map[string]layoutTarTestEntry{}
	add := func(e layoutTarTestEntry) {
		wanted[e.header.Name] = e
		for parent := path.Dir(e.header.Name); parent != "."; parent = path.Dir(parent) {
			wanted[parent] = layoutTestEntry(parent, tar.TypeDir, "", "")
		}
	}
	for _, pkg := range selected.Lock().Packages() {
		root := roots[pkg.Key]
		if root == "" {
			t.Fatal("fixture root missing")
		}
		key := pkg.Name + "@" + pkg.Version
		for name, content := range bodies[key] {
			e := layoutTestEntry(root+"/"+name, tar.TypeReg, content, "")
			e.header.Mode = modes[key][name]
			add(e)
		}
		for command, file := range pkg.Bins {
			add(layoutTestEntry("node_modules/.bin/"+command, tar.TypeSymlink, "", "../"+pkg.Name+"/"+file))
		}
	}
	var names []string
	for name := range wanted {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		f.entries = append(f.entries, wanted[name])
	}
	return f
}

func bindBunFixture(t *testing.T, f bunLayoutTestFixture) BunInspectionLayout {
	t.Helper()
	layout, err := ParseBunLayoutArchive(context.Background(), layoutTarBody(t, f.entries))
	if err != nil {
		t.Fatal(err)
	}
	bound, err := f.input.BindLayout(context.Background(), f.inventories, layout)
	if err != nil {
		t.Fatal(err)
	}
	return bound
}

func TestBunLayoutBindsWholeHoistedPeerAndRepeatedScopedTree(t *testing.T) {
	for _, repeated := range []bool{false, true} {
		t.Run(fmt.Sprint(repeated), func(t *testing.T) {
			f := bunLayoutFixture(t, repeated)
			bound := bindBunFixture(t, f)
			if bound.Digest().String() == "" || !bound.MatchesInput(f.input) || bound.RuntimeIdentity() != f.input.Context().RuntimeIdentity() {
				t.Fatal("whole binding lost")
			}
			total := 0
			for _, subject := range f.input.Artifacts() {
				instances, err := f.input.InstancesFor(subject)
				if err != nil {
					t.Fatal(err)
				}
				for _, instance := range instances {
					target, err := bound.Target(subject, instance.Key)
					if err != nil {
						t.Fatal(err)
					}
					if instance.Key == "@one/root/@two/lib" && target != "node_modules/@one/root/node_modules/@two/lib" {
						t.Fatal("scoped ancestry collapsed")
					}
					body, err := bound.ReadFile(target + "/package.json")
					if err != nil || !bytes.Contains(body, []byte(subject.Identity().Name())) {
						t.Fatal("wrong exact subject target", err)
					}
					total++
					body[0] ^= 1
					again, err := bound.ReadFile(target + "/package.json")
					if err != nil || bytes.Equal(body, again) {
						t.Fatal("mutable layout body")
					}
				}
			}
			if total != 4 {
				t.Fatal("instance coverage collapsed")
			}
			manifest := bound.InspectionManifest()
			var fields map[string]json.RawMessage
			if json.Unmarshal(manifest, &fields) != nil || fields["scripts"] != nil || fields["dist"] != nil || fields["name"] != nil || string(fields["private"]) != "true" {
				t.Fatal("root metadata entered inspection manifest")
			}
			manifest[0] ^= 1
			if bytes.Equal(manifest, bound.InspectionManifest()) {
				t.Fatal("mutable manifest")
			}
			entries := bound.Entries()
			entries[0].Name = "foreign"
			if bound.Entries()[0].Name == "foreign" {
				t.Fatal("mutable entries")
			}
			reordered := f
			reordered.entries = append([]layoutTarTestEntry(nil), f.entries...)
			slices.Reverse(reordered.entries)
			if bindBunFixture(t, reordered).Digest() != bound.Digest() {
				t.Fatal("tar ordering changed semantic fingerprint")
			}
		})
	}
}

func TestBunLayoutRejectsMissingChangedForeignAndAliasEntries(t *testing.T) {
	for _, fault := range []string{"missing", "extra", "bytes", "mode", "owner", "bin", "hardlink", "package-link", "metadata"} {
		t.Run(fault, func(t *testing.T) {
			f := bunLayoutFixture(t, false)
			for n := range f.entries {
				e := &f.entries[n]
				if e.header.Name == "node_modules/react/LICENSE" {
					switch fault {
					case "missing":
						f.entries = append(f.entries[:n], f.entries[n+1:]...)
					case "bytes":
						e.body = "X" + e.body[1:]
					case "mode":
						e.header.Mode = 0o755
					case "owner":
						e.header.Uid = 0
					case "hardlink":
						e.header.Typeflag = tar.TypeLink
						e.header.Linkname = "node_modules/js-tokens/LICENSE"
						e.header.Size = 0
						e.body = ""
					}
					break
				}
			}
			if fault == "bin" {
				for n := range f.entries {
					if f.entries[n].header.Name == "node_modules/.bin/loose-envify" {
						f.entries[n].header.Linkname = "../react/package.json"
					}
				}
			}
			if fault == "package-link" {
				for n := range f.entries {
					if f.entries[n].header.Name == "node_modules/react" {
						f.entries[n].header.Typeflag = tar.TypeSymlink
						f.entries[n].header.Mode = 0o777
						f.entries[n].header.Linkname = "js-tokens"
					}
				}
			}
			if fault == "extra" || fault == "metadata" {
				name := "node_modules/react/extra.js"
				if fault == "metadata" {
					name = "node_modules/.modules.yaml"
				}
				f.entries = append(f.entries, layoutTestEntry(name, tar.TypeReg, "unknown bytes", ""))
			}
			layout, err := ParseBunLayoutArchive(context.Background(), layoutTarBody(t, f.entries))
			if err != nil {
				if fault != "owner" && fault != "package-link" {
					t.Fatal("negative failed before binder", err)
				}
				return
			}
			if got, err := f.input.BindLayout(context.Background(), f.inventories, layout); err == nil || got.Digest().String() != "" {
				t.Fatal("unbound native entry admitted")
			}
		})
	}
}

func TestBunLayoutChecksArchiveDependencyPeerAndBinMetadata(t *testing.T) {
	for _, field := range []string{"dependencies", "peerDependencies", "bin", "optionalDependencies"} {
		t.Run(field, func(t *testing.T) {
			f := bunLayoutFixture(t, false)
			for n, inventory := range f.inventories {
				if inventory.Artifact().Identity().Name() != "react" {
					continue
				}
				var fields map[string]any
				if json.Unmarshal(inventory.Manifest(), &fields) != nil {
					t.Fatal("fixture manifest")
				}
				fields[field] = map[string]string{"foreign": "1.0.0"}
				if field == "bin" {
					fields[field] = map[string]string{"foreign": "index.js"}
				}
				body, err := json.Marshal(fields)
				if err != nil {
					t.Fatal(err)
				}
				entries := inventory.Entries()
				for k := range entries {
					if entries[k].Path == "package.json" {
						entries[k].Size = int64(len(body))
						entries[k].Digest = layoutDigestFixture(t, body)
					}
				}
				f.inventories[n], err = NewArchiveInventory(inventory.Artifact(), entries, body)
				if err != nil {
					t.Fatal(err)
				}
				for k := range f.entries {
					if f.entries[k].header.Name == "node_modules/react/package.json" {
						f.entries[k].body = string(body)
						f.entries[k].header.Size = int64(len(body))
					}
				}
			}
			layout, err := ParseBunLayoutArchive(context.Background(), layoutTarBody(t, f.entries))
			if err != nil {
				t.Fatal(err)
			}
			if bound, err := f.input.BindLayout(context.Background(), f.inventories, layout); err == nil || bound.Digest().String() != "" {
				t.Fatal("archive metadata substituted the frozen graph")
			}
		})
	}
}

func TestBunLayoutRejectsWrongRunContextZeroAndCancellation(t *testing.T) {
	f := bunLayoutFixture(t, false)
	bound := bindBunFixture(t, f)
	foreign := bunLayoutFixture(t, false)
	if bound.MatchesInput(foreign.input) {
		t.Fatal("different acquired runs reused binding")
	}
	if bound.Digest() == bindBunFixture(t, foreign).Digest() {
		t.Fatal("different acquired runs share fingerprint")
	}
	if _, err := bound.Target(foreign.input.Artifacts()[0], "js-tokens"); err == nil {
		t.Fatal("foreign run target")
	}
	if _, err := bound.Target(f.input.Artifacts()[0], "foreign"); err == nil {
		t.Fatal("foreign instance target")
	}
	if (BunInspectionLayout{}).MatchesInput(f.input) {
		t.Fatal("zero bound input")
	}
	if _, err := (BunInspectionLayout{}).ReadFile("node_modules/react/package.json"); err == nil {
		t.Fatal("zero layout returned file")
	}
	layout, err := ParseBunLayoutArchive(context.Background(), layoutTarBody(t, f.entries))
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("bun layout canceled")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if got, err := f.input.BindLayout(ctx, f.inventories, layout); !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || got.Digest().String() != "" {
		t.Fatal("cancel cause/result lost", err)
	}

	if _, err := (BunInspectionInput{}).BindLayout(context.Background(), f.inventories, layout); err == nil {
		t.Fatal("zero input accepted")
	}
	for _, inventories := range [][]ArchiveInventory{nil, f.inventories[:3], {f.inventories[0], f.inventories[0], f.inventories[2], f.inventories[3]}, foreign.inventories} {
		if got, err := f.input.BindLayout(context.Background(), inventories, layout); err == nil || got.Digest().String() != "" {
			t.Fatal("partial/foreign inventory accepted")
		}
	}
}

func TestBunLayoutRestrictsOnlyAnUnverifiedDirectoryView(t *testing.T) {
	f := bunLayoutFixture(t, false)
	for n := range f.entries {
		if f.entries[n].header.Name == "node_modules/react" {
			f.entries[n].header.Mode = 0o777
		}
	}
	layout, err := ParseBunLayoutArchive(context.Background(), layoutTarBody(t, f.entries))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := f.input.BindLayout(context.Background(), f.inventories, layout); err == nil || got.Digest().String() != "" {
		t.Fatal("unsafe raw directory admitted")
	}
	view := layout.RestrictDirectoryWrites()
	if _, err := f.input.BindLayout(context.Background(), f.inventories, view); err != nil {
		t.Fatal("restricted view failed full byte binding", err)
	}
	for _, entry := range layout.Entries() {
		if entry.Name == "node_modules/react" && entry.Mode != 0o777 {
			t.Fatal("native raw entry changed")
		}
	}
	for n := range f.entries {
		if f.entries[n].header.Name == "node_modules/react/LICENSE" {
			f.entries[n].header.Mode = 0o666
		}
	}
	unsafe, err := ParseBunLayoutArchive(context.Background(), layoutTarBody(t, f.entries))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.input.BindLayout(context.Background(), f.inventories, unsafe.RestrictDirectoryWrites()); err == nil {
		t.Fatal("directory restriction hid unsafe regular file")
	}
}

func TestBunLayoutBindsExactRestrictedCacheModeAndPreservesOriginalInventory(t *testing.T) {
	f := bunLayoutFixture(t, false)
	for n, inventory := range f.inventories {
		if inventory.Artifact().Identity().Name() != "js-tokens" {
			continue
		}
		entries := inventory.Entries()
		for k := range entries {
			if entries[k].Path == "LICENSE" {
				entries[k].Mode = 0o664
			}
		}
		var err error
		f.inventories[n], err = NewArchiveInventory(inventory.Artifact(), entries, inventory.Manifest())
		if err != nil {
			t.Fatal(err)
		}
	}
	if bindBunFixture(t, f).Digest().String() == "" {
		t.Fatal("source packing write bit blocked private restricted view")
	}
	for _, inventory := range f.inventories {
		if inventory.Artifact().Identity().Name() == "js-tokens" {
			for _, entry := range inventory.Entries() {
				if entry.Path == "LICENSE" && entry.Mode != 0o664 {
					t.Fatal("source facts rewritten")
				}
			}
		}
	}
	for n := range f.entries {
		if f.entries[n].header.Name == "node_modules/js-tokens/LICENSE" {
			f.entries[n].header.Mode = 0o664
		}
	}
	layout, err := ParseBunLayoutArchive(context.Background(), layoutTarBody(t, f.entries))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.input.BindLayout(context.Background(), f.inventories, layout.RestrictDirectoryWrites()); err == nil {
		t.Fatal("unsafe native regular mode admitted")
	}
}
