package npm

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"go.yaml.in/yaml/v3"
)

type inspectionLayoutFixture struct {
	input       PNPMInspectionInput
	inventories []ArchiveInventory
	entries     []layoutTarTestEntry
}

func layoutDigestFixture(t *testing.T, body []byte) domain.ContentDigest {
	t.Helper()
	hash := sha256.Sum256(body)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(hash[:]))
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func fullInspectionLayoutFixture(t *testing.T, multiple bool) inspectionLayoutFixture {
	t.Helper()
	manifest := []byte(pnpmPeerManifest)
	lockfile := pnpmFixture(t, "peer")
	if multiple {
		var wire pnpmLockWire
		if yaml.Unmarshal(lockfile, &wire) != nil {
			t.Fatal("invalid base graph")
		}
		wire.Importers["."].Dependencies["bridge"] = pnpmRootWire{"1.0.0", "1.0.0"}
		wire.Packages["react@17.0.2"] = wire.Packages["react@18.3.1"]
		wire.Packages["bridge@1.0.0"] = wire.Packages["react@18.3.1"]
		wire.Snapshots["react@17.0.2"] = pnpmSnapshotWire{}
		wire.Snapshots["bridge@1.0.0"] = pnpmSnapshotWire{Dependencies: map[string]string{"use-sync-external-store": "1.2.0(react@17.0.2)"}}
		wire.Snapshots["use-sync-external-store@1.2.0(react@17.0.2)"] = pnpmSnapshotWire{Dependencies: map[string]string{"react": "17.0.2"}}
		var err error
		lockfile, err = yaml.Marshal(wire)
		if err != nil {
			t.Fatal(err)
		}
		manifest = []byte(`{"scripts":{"install":"exit 99"},"config":{"private-data":"fixture"},"author":"fixture","dependencies":{"react":"18.3.1","use-sync-external-store":"1.2.0","bridge":"1.0.0"}}`)
	}
	bound, controls := pnpmSnapshotFixture(t, manifest, lockfile)
	frozen, err := bound.InspectionContext(pnpmSnapshotContext(t, "/tmp/haa-layout-project", true), controls, "fixture-runtime:pnpm")
	if err != nil {
		t.Fatal(err)
	}
	acquired := pnpmAcquiredInspectionFixture(t, frozen)
	input, err := frozen.BindAcquired(frozen.Snapshot(), acquired)
	if err != nil {
		t.Fatal(err)
	}
	f := inspectionLayoutFixture{input: input}
	bodies := map[string]map[string]string{}
	for _, subject := range input.Artifacts() {
		name := subject.Identity().Name()
		metadata := map[string]any{"name": name, "version": subject.Identity().Version()}
		files := map[string]string{"LICENSE": "same license bytes\n"}
		if name == "loose-envify" {
			metadata["bin"] = map[string]string{"loose-envify": "cli.js"}
			files["cli.js"] = "#!/usr/bin/env node\nconsole.log('fixture');\n"
		}
		body, err := json.Marshal(metadata)
		if err != nil {
			t.Fatal(err)
		}
		files["package.json"] = string(body)
		var entries []ArchiveEntry
		for file, content := range files {
			entries = append(entries, ArchiveEntry{Path: file, Size: int64(len(content)), Mode: 0o644, Digest: layoutDigestFixture(t, []byte(content))})
		}
		inventory, err := NewArchiveInventory(subject, entries, body)
		if err != nil {
			t.Fatal(err)
		}
		f.inventories = append(f.inventories, inventory)
		bodies[name+"@"+subject.Identity().Version()] = files
	}
	roots := map[string]string{
		"js-tokens@4.0.0":    "js-tokens@4.0.0/node_modules/js-tokens",
		"loose-envify@1.4.0": "loose-envify@1.4.0/node_modules/loose-envify",
		"react@18.3.1":       "react@18.3.1/node_modules/react",
		"use-sync-external-store@1.2.0(react@18.3.1)": "use-sync-external-store@1.2.0_react@18.3.1/node_modules/use-sync-external-store",
		"react@17.0.2": "react@17.0.2/node_modules/react",
		"bridge@1.0.0": "bridge@1.0.0/node_modules/bridge",
		"use-sync-external-store@1.2.0(react@17.0.2)": "use-sync-external-store@1.2.0_react@17.0.2/node_modules/use-sync-external-store",
	}
	for _, instance := range bound.Lock().Snapshots() {
		for name, body := range bodies[instance.Package] {
			item := layoutTestEntry("node_modules/.pnpm/"+roots[instance.Key]+"/"+name, tar.TypeReg, body, "")
			if name == "cli.js" {
				item.header.Mode = 0o755
			}
			f.entries = append(f.entries, item)
		}
	}
	links := map[string]string{
		"node_modules/react":                                                               ".pnpm/react@18.3.1/node_modules/react",
		"node_modules/use-sync-external-store":                                             ".pnpm/use-sync-external-store@1.2.0_react@18.3.1/node_modules/use-sync-external-store",
		"node_modules/.pnpm/loose-envify@1.4.0/node_modules/js-tokens":                     "../../js-tokens@4.0.0/node_modules/js-tokens",
		"node_modules/.pnpm/react@18.3.1/node_modules/loose-envify":                        "../../loose-envify@1.4.0/node_modules/loose-envify",
		"node_modules/.pnpm/use-sync-external-store@1.2.0_react@18.3.1/node_modules/react": "../../react@18.3.1/node_modules/react",
		"node_modules/.pnpm/node_modules/js-tokens":                                        "../js-tokens@4.0.0/node_modules/js-tokens",
		"node_modules/.pnpm/node_modules/loose-envify":                                     "../loose-envify@1.4.0/node_modules/loose-envify",
	}
	if multiple {
		links["node_modules/bridge"] = ".pnpm/bridge@1.0.0/node_modules/bridge"
		links["node_modules/.pnpm/bridge@1.0.0/node_modules/use-sync-external-store"] = "../../use-sync-external-store@1.2.0_react@17.0.2/node_modules/use-sync-external-store"
		links["node_modules/.pnpm/use-sync-external-store@1.2.0_react@17.0.2/node_modules/react"] = "../../react@17.0.2/node_modules/react"
	}
	for name, target := range links {
		f.entries = append(f.entries, layoutTestEntry(name, tar.TypeSymlink, "", target))
	}
	shims := map[string]string{
		"node_modules/.pnpm/loose-envify@1.4.0/node_modules/loose-envify/node_modules/.bin/loose-envify": "../../cli.js",
		"node_modules/.pnpm/react@18.3.1/node_modules/react/node_modules/.bin/loose-envify":              "../../../../../loose-envify@1.4.0/node_modules/loose-envify/cli.js",
		"node_modules/.pnpm/node_modules/.bin/loose-envify":                                              "../../loose-envify@1.4.0/node_modules/loose-envify/cli.js",
	}
	for name, target := range shims {
		item := layoutTestEntry(name, tar.TypeReg, nodeShimVector(t, target), "")
		item.header.Mode = 0o755
		f.entries = append(f.entries, item)
	}
	metadata, err := json.Marshal(map[string]any{
		"hoistedDependencies": map[string]any{"js-tokens@4.0.0": map[string]string{"js-tokens": "private"}, "loose-envify@1.4.0": map[string]string{"loose-envify": "private"}},
		"hoistPattern":        []string{"*"}, "included": map[string]bool{"dependencies": true, "devDependencies": true, "optionalDependencies": true},
		"layoutVersion": 5, "nodeLinker": "isolated", "packageManager": "pnpm@" + bound.Lock().version,
		"pendingBuilds": []string{}, "prunedAt": time.Now().UTC().Format(time.RFC1123), "publicHoistPattern": []string{},
		"skipped": []string{}, "storeDir": "/tmp/haa-store/v11", "virtualStoreDir": ".pnpm", "virtualStoreDirMaxLength": 120, "allowBuilds": map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	f.entries = append(f.entries, layoutTestEntry("node_modules/.modules.yaml", tar.TypeReg, string(metadata), ""), layoutTestEntry("node_modules/.pnpm/lock.yaml", tar.TypeReg, string(lockfile), ""))
	f.entries = append(f.entries, layoutTestEntry(pnpmWorkspaceCachePath, tar.TypeReg, `{"lastValidatedTimestamp":1,"validation":{"fixture":"untrusted freshness data"}}`, ""))
	directories := map[string]bool{}
	for _, entry := range f.entries {
		for parent := path.Dir(entry.header.Name); parent != "."; parent = path.Dir(parent) {
			directories[parent] = true
		}
	}
	for name := range directories {
		f.entries = append(f.entries, layoutTestEntry(name, tar.TypeDir, "", ""))
	}
	slices.SortFunc(f.entries, func(a, b layoutTarTestEntry) int { return strings.Compare(a.header.Name, b.header.Name) })
	return f
}

func (f inspectionLayoutFixture) parse(t *testing.T) PNPMLayoutArchive {
	t.Helper()
	layout, err := ParsePNPMLayoutArchive(context.Background(), layoutTarBody(t, f.entries))
	if err != nil {
		t.Fatal("negative failed at envelope instead of binding", err)
	}
	return layout
}

func TestPNPMInspectionLayoutBindsAllArchivesAndPeerInstances(t *testing.T) {
	for _, multiple := range []bool{false, true} {
		f := fullInspectionLayoutFixture(t, multiple)
		bound, err := f.input.BindLayout(context.Background(), f.inventories, f.parse(t), "/tmp/haa-store")
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, subject := range f.input.Artifacts() {
			instances, err := f.input.InstancesFor(subject)
			if err != nil {
				t.Fatal(err)
			}
			for _, instance := range instances {
				target, err := bound.Target(subject, instance.Key)
				if err != nil || !strings.HasSuffix(target, "/"+subject.Identity().Name()) {
					t.Fatal("instance lost", err)
				}
				count++
			}
		}
		want := 4
		if multiple {
			want = 7
		}
		if count != want || bound.Digest().String() == "" {
			t.Fatal("coverage lost", count)
		}
		manifest := bound.InspectionManifest()
		if strings.Contains(string(manifest), "scripts") || strings.Contains(string(manifest), "config") || strings.Contains(string(manifest), "author") || !strings.Contains(string(manifest), `"private":true`) {
			t.Fatal("root private metadata introduced")
		}
		manifest[0] = '!'
		if bound.InspectionManifest()[0] != '{' {
			t.Fatal("mutable manifest")
		}
		entries := bound.Entries()
		entries[0].Name = "changed"
		if bound.Entries()[0].Name == "changed" {
			t.Fatal("mutable entries")
		}
		for _, entry := range bound.Entries() {
			if entry.Name == pnpmWorkspaceCachePath {
				t.Fatal("native freshness cache in inspection payload")
			}
		}
		if _, err := bound.ReadFile(pnpmWorkspaceCachePath); err == nil {
			t.Fatal("native freshness cache exposed")
		}
		body, err := bound.ReadFile("node_modules/.modules.yaml")
		if err != nil {
			t.Fatal(err)
		}
		body[0] = '!'
		again, _ := bound.ReadFile("node_modules/.modules.yaml")
		if again[0] != '{' {
			t.Fatal("mutable file")
		}
		if _, err := bound.Target(domain.AcquiredArtifact{}, "react@18.3.1"); err == nil {
			t.Fatal("foreign target")
		}
	}
}

func TestPNPMInspectionLayoutRejectsChangedLayout(t *testing.T) {
	for _, change := range []string{"file-bytes", "missing-file", "extra-file", "edge", "bin-code", "bin-target", "exec", "write", "metadata-unknown", "metadata-duplicate", "metadata-hoist", "metadata-build", "metadata-store", "virtual-lock"} {
		t.Run(change, func(t *testing.T) {
			f := fullInspectionLayoutFixture(t, false)
			for index := range f.entries {
				e := &f.entries[index]
				switch {
				case change == "missing-file" && strings.HasSuffix(e.header.Name, "/react/LICENSE"):
					f.entries = slices.Delete(f.entries, index, index+1)
				case change == "file-bytes" && strings.HasSuffix(e.header.Name, "/react/LICENSE"):
					e.body = "changed"
					e.header.Size = int64(len(e.body))
				case change == "edge" && e.header.Name == "node_modules/react":
					e.header.Linkname = ".pnpm/js-tokens@4.0.0/node_modules/js-tokens"
				case (change == "bin-code" || change == "bin-target") && strings.Contains(e.header.Name, "/.bin/"):
					if change == "bin-code" {
						e.body += "\nprintf injected\n"
					} else {
						e.body = strings.ReplaceAll(e.body, "cli.js", "other.js")
					}
					e.header.Size = int64(len(e.body))
				case (change == "exec" || change == "write") && strings.HasSuffix(e.header.Name, "/react/LICENSE"):
					if change == "exec" {
						e.header.Mode = 0o755
					} else {
						e.header.Mode = 0o666
					}
				case strings.HasPrefix(change, "metadata-") && e.header.Name == "node_modules/.modules.yaml":
					switch change {
					case "metadata-unknown":
						e.body = strings.TrimSuffix(e.body, "}") + `,"unknown":true}`
					case "metadata-duplicate":
						e.body = strings.TrimSuffix(e.body, "}") + `,"layoutVersion":5}`
					case "metadata-hoist":
						e.body = strings.ReplaceAll(e.body, `"private"`, `"public"`)
					case "metadata-build":
						e.body = strings.Replace(e.body, `"allowBuilds":{}`, `"allowBuilds":{"foreign":true}`, 1)
					case "metadata-store":
						e.body = strings.Replace(e.body, "/tmp/haa-store/v11", "/tmp/other/v11", 1)
					}
					e.header.Size = int64(len(e.body))
				case change == "virtual-lock" && e.header.Name == "node_modules/.pnpm/lock.yaml":
					e.body = strings.Replace(e.body, "sha512-RdJU", "sha512-AdJU", 1)
					e.header.Size = int64(len(e.body))
				}
				if change == "missing-file" && len(f.entries) <= index+1 {
					break
				}
			}
			if change == "extra-file" {
				f.entries = append(f.entries, layoutTestEntry("node_modules/extra", tar.TypeReg, "extra", ""))
			}
			if bound, err := f.input.BindLayout(context.Background(), f.inventories, f.parse(t), "/tmp/haa-store"); err == nil || bound.Digest().String() != "" {
				t.Fatal("changed layout admitted", change)
			}
		})
	}
}

func TestPNPMInspectionLayoutRejectsWrongCoverageAndCancellation(t *testing.T) {
	f := fullInspectionLayoutFixture(t, false)
	layout := f.parse(t)
	for _, inventories := range [][]ArchiveInventory{nil, f.inventories[:3], {f.inventories[0], f.inventories[0], f.inventories[2], f.inventories[3]}, {{}, f.inventories[1], f.inventories[2], f.inventories[3]}} {
		if _, err := f.input.BindLayout(context.Background(), inventories, layout, "/tmp/haa-store"); err == nil {
			t.Fatal("bad coverage admitted")
		}
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("fixture cancellation")
	cancel(cause)
	if _, err := f.input.BindLayout(ctx, f.inventories, layout, "/tmp/haa-store"); !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation cause lost", err)
	}
	if _, err := (PNPMInspectionInput{}).BindLayout(context.Background(), f.inventories, layout, "/tmp/haa-store"); err == nil {
		t.Fatal("zero input admitted")
	}
	if _, err := (PNPMInspectionLayout{}).ReadFile("node_modules/.modules.yaml"); err == nil {
		t.Fatal("zero layout exposed file")
	}
}

func TestPNPMInspectionLayoutInternalHardlinkAndVolatileMetadata(t *testing.T) {
	f := fullInspectionLayoutFixture(t, false)
	original, err := f.input.BindLayout(context.Background(), f.inventories, f.parse(t), "/tmp/haa-store")
	if err != nil {
		t.Fatal(err)
	}
	for index := range f.entries {
		e := &f.entries[index]
		if e.header.Name == pnpmWorkspaceCachePath {
			e.body = `{"lastValidatedTimestamp":999999999999,"validation":{"foreign":"discarded data"}}`
			e.header.Size = int64(len(e.body))
		}
		if strings.Contains(e.header.Name, "use-sync-external-store/LICENSE") {
			e.header.Typeflag = tar.TypeLink
			e.header.Linkname = "node_modules/.pnpm/react@18.3.1/node_modules/react/LICENSE"
			e.header.Size = 0
			e.body = ""
		}
		if e.header.Name == "node_modules/.modules.yaml" {
			var metadata map[string]any
			if json.Unmarshal([]byte(e.body), &metadata) != nil {
				t.Fatal("invalid metadata")
			}
			metadata["prunedAt"] = "Mon, 05 Oct 2026 00:00:00 UTC"
			body, err := json.MarshalIndent(metadata, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			e.body = string(body)
			e.header.Size = int64(len(body))
		}
	}
	bound, err := f.input.BindLayout(context.Background(), f.inventories, f.parse(t), "/tmp/haa-store")
	if err != nil {
		t.Fatal("same source hardlink rejected", err)
	}
	if bound.Digest() == original.Digest() {
		t.Fatal("alias relationship lost from context")
	}
	for index := range f.entries {
		e := &f.entries[index]
		if strings.Contains(e.header.Name, "use-sync-external-store/LICENSE") {
			e.header.Typeflag = tar.TypeReg
			e.header.Linkname = ""
			e.body = "same license bytes\n"
			e.header.Size = int64(len(e.body))
		}
	}
	canonical, err := f.input.BindLayout(context.Background(), f.inventories, f.parse(t), "/tmp/haa-store")
	if err != nil || canonical.Digest() != original.Digest() {
		t.Fatal("volatile formatting/time changed semantic context", err)
	}
}
