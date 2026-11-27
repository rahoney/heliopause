package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path"
	"slices"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type pnpmPayloadTestFile struct {
	header tar.Header
	body   []byte
}

func pnpmPayloadTestTar(t *testing.T, files []pnpmPayloadTestFile) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := tar.NewWriter(&output)
	for _, file := range files {
		if err := writer.WriteHeader(&file.header); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(file.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func pnpmBoundPayloadFixture(t *testing.T, shared bool) artifactnpm.PNPMInspectionLayout {
	t.Helper()
	layout, _ := pnpmBoundPayloadInputs(t, shared)
	return layout
}

func pnpmBoundPayloadInputs(t *testing.T, shared bool) (artifactnpm.PNPMInspectionLayout, artifactnpm.PNPMInspectionInput) {
	t.Helper()
	return pnpmBoundPayloadScriptInputs(t, shared, nil)
}

// Controlled structural input only: supplied scripts are not public package bytes.
func pnpmBoundPayloadScriptInputs(t *testing.T, shared bool, scripts map[string]string) (artifactnpm.PNPMInspectionLayout, artifactnpm.PNPMInspectionInput) {
	t.Helper()
	return pnpmBoundPayloadNamedScriptInputs(t, shared, "is-number", scripts)
}

func pnpmBoundPayloadNamedScriptInputs(t *testing.T, shared bool, name string, scripts map[string]string) (artifactnpm.PNPMInspectionLayout, artifactnpm.PNPMInspectionInput) {
	t.Helper()
	_, input, inventories := pnpmPreparationNamedInputFixture(t, PNPMResolverRuntimeIdentity(), name)
	inventory := inventories[0]
	if scripts != nil {
		manifest, err := json.Marshal(map[string]any{"name": inventory.Artifact().Identity().Name(), "version": inventory.Artifact().Identity().Version(), "scripts": scripts})
		if err != nil {
			t.Fatal(err)
		}
		digest, err := pnpmPayloadSHA256(manifest)
		if err != nil {
			t.Fatal(err)
		}
		files := inventory.Entries()
		for index := range files {
			if files[index].Path == "package.json" {
				files[index].Size = int64(len(manifest))
				files[index].Digest = digest
			}
		}
		inventory, err = artifactnpm.NewArchiveInventory(inventory.Artifact(), files, manifest)
		if err != nil {
			t.Fatal(err)
		}
	}
	entries := inventory.Entries()
	license := []byte("shared fixture license\n")
	digest, err := pnpmPayloadSHA256(license)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"COPY", "LICENSE"} {
		entries = append(entries, artifactnpm.ArchiveEntry{Path: name, Size: int64(len(license)), Mode: 0o644, Digest: digest})
	}
	inventories[0], err = artifactnpm.NewArchiveInventory(inventory.Artifact(), entries, inventory.Manifest())
	if err != nil {
		t.Fatal(err)
	}
	instance := input.Context().Lock().Snapshots()[0]
	directory, err := artifactnpm.PNPMInstanceDirectory(instance.Key)
	if err != nil {
		t.Fatal(err)
	}
	root := "node_modules/.pnpm/" + directory + "/node_modules/" + inventory.Artifact().Identity().Name()
	files := []pnpmPayloadTestFile{}
	add := func(name string, kind byte, body []byte, link string) {
		mode := int64(0o644)
		if kind == tar.TypeDir {
			mode = 0o755
		}
		if kind == tar.TypeSymlink {
			mode = 0o777
		}
		files = append(files, pnpmPayloadTestFile{tar.Header{Name: name, Typeflag: kind, Mode: mode, Uid: 1000, Gid: 1000, Size: int64(len(body)), Linkname: link}, body})
	}
	add(root+"/package.json", tar.TypeReg, inventory.Manifest(), "")
	add(root+"/LICENSE", tar.TypeReg, license, "")
	if shared {
		add(root+"/COPY", tar.TypeLink, nil, root+"/LICENSE")
	} else {
		add(root+"/COPY", tar.TypeReg, license, "")
	}
	add("node_modules/"+inventory.Artifact().Identity().Name(), tar.TypeSymlink, nil, ".pnpm/"+directory+"/node_modules/"+inventory.Artifact().Identity().Name())
	modules, err := json.Marshal(map[string]any{"hoistedDependencies": map[string]any{}, "hoistPattern": []string{"*"}, "included": map[string]bool{"dependencies": true, "devDependencies": true, "optionalDependencies": true}, "layoutVersion": 5, "nodeLinker": "isolated", "packageManager": "pnpm@" + runtimeidentity.PNPMVersion, "pendingBuilds": []string{}, "prunedAt": "Sat, 24 Oct 2026 00:00:00 GMT", "publicHoistPattern": []string{}, "skipped": []string{}, "storeDir": pnpmProjectRoot + "/store/v11", "virtualStoreDir": ".pnpm", "virtualStoreDirMaxLength": artifactnpm.PNPMVirtualStoreDirMaxLength, "allowBuilds": map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	add("node_modules/.modules.yaml", tar.TypeReg, modules, "")
	for _, control := range input.Context().Controls() {
		if control.Name() == "pnpm-lock.yaml" {
			add("node_modules/.pnpm/lock.yaml", tar.TypeReg, control.Body(), "")
		}
	}
	add("node_modules/.pnpm-workspace-state-v1.json", tar.TypeReg, []byte(`{"fixture":true}`), "")
	directories := map[string]bool{}
	for _, file := range files {
		for parent := path.Dir(file.header.Name); parent != "."; parent = path.Dir(parent) {
			directories[parent] = true
		}
	}
	for name := range directories {
		add(name, tar.TypeDir, nil, "")
	}
	slices.SortFunc(files, func(a, b pnpmPayloadTestFile) int { return strings.Compare(a.header.Name, b.header.Name) })
	parsed, err := artifactnpm.ParsePNPMLayoutArchive(context.Background(), pnpmPayloadTestTar(t, files))
	if err != nil {
		t.Fatal(err)
	}
	bound, err := input.BindLayout(context.Background(), inventories, parsed, pnpmProjectRoot+"/store")
	if err != nil {
		t.Fatal(err)
	}
	return bound, input
}

func pnpmPayloadFiles(t *testing.T, body []byte) []pnpmPayloadTestFile {
	t.Helper()
	parsed, err := artifactnpm.ParsePNPMInspectionPayloadArchive(context.Background(), body)
	if err != nil {
		t.Fatal(err)
	}
	var files []pnpmPayloadTestFile
	for _, entry := range parsed.Entries() {
		var body []byte
		if entry.Kind == tar.TypeReg {
			body, err = parsed.ReadFile(entry.Name)
			if err != nil {
				t.Fatal(err)
			}
		}
		files = append(files, pnpmPayloadTestFile{tar.Header{Name: entry.Name, Typeflag: entry.Kind, Mode: entry.Mode, Uid: entry.UID, Gid: entry.GID, Size: entry.Size, Linkname: entry.Link}, body})
	}
	return files
}

func TestPNPMInspectionPayloadIsMinimalDeterministicAndKeepsSharing(t *testing.T) {
	for _, shared := range []bool{false, true} {
		bound := pnpmBoundPayloadFixture(t, shared)
		payload, err := preparePNPMInspectionPayload(context.Background(), bound)
		if err != nil {
			t.Fatal(err)
		}
		again, err := preparePNPMInspectionPayload(context.Background(), bound)
		if err != nil || !bytes.Equal(payload.body, again.body) || payload.digest != again.digest {
			t.Fatal("non-deterministic payload", err)
		}
		parsed, err := artifactnpm.ParsePNPMInspectionPayloadArchive(context.Background(), payload.body)
		if err != nil {
			t.Fatal(err)
		}
		manifest, err := parsed.ReadFile("project/package.json")
		if err != nil || !bytes.Equal(manifest, bound.InspectionManifest()) || bytes.Contains(manifest, []byte("scripts")) || bytes.Contains(manifest, []byte("config")) || bytes.Contains(manifest, []byte("author")) {
			t.Fatal("user root metadata introduced", err)
		}
		for _, name := range []string{"project/node_modules/.modules.yaml", "project/node_modules/.pnpm/lock.yaml", "project/node_modules/.pnpm-workspace-state-v1.json"} {
			if _, err := parsed.ReadFile(name); err == nil {
				t.Fatal("unnecessary generated metadata introduced", name)
			}
		}
		aliases := 0
		for _, entry := range parsed.Entries() {
			if entry.Kind == tar.TypeLink {
				aliases++
			}
		}
		if shared && aliases != 1 || !shared && aliases != 0 {
			t.Fatal("hardlink partition changed", aliases)
		}
		if err := comparePNPMInspectionPayload(context.Background(), payload, parsed); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPNPMInspectionPayloadReadbackRejectsChangesAndForeignSettings(t *testing.T) {
	payload, err := preparePNPMInspectionPayload(context.Background(), pnpmBoundPayloadFixture(t, true))
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"bytes", "missing", "extra-root-settings", "mode", "owner", "link", "split-sharing"} {
		t.Run(change, func(t *testing.T) {
			files := pnpmPayloadFiles(t, payload.body)
			changed := false
			for index := 0; index < len(files); index++ {
				f := &files[index]
				switch {
				case change == "bytes" && f.header.Name == "project/package.json":
					f.body = append(f.body, ' ')
					f.header.Size = int64(len(f.body))
					changed = true
				case change == "missing" && strings.HasSuffix(f.header.Name, "/package.json") && f.header.Name != "project/package.json":
					files = slices.Delete(files, index, index+1)
					changed = true
					index--
				case change == "mode" && f.header.Name == "project/package.json":
					f.header.Mode = 0o644
					changed = true
				case change == "owner" && f.header.Name == "project/package.json":
					f.header.Uid = 0
					changed = true
				case change == "link" && f.header.Typeflag == tar.TypeSymlink:
					f.header.Linkname += "-foreign"
					changed = true
				case change == "split-sharing" && f.header.Typeflag == tar.TypeLink:
					f.body = []byte("shared fixture license\n")
					f.header.Typeflag = tar.TypeReg
					f.header.Linkname = ""
					f.header.Size = int64(len(f.body))
					changed = true
				}
			}
			if change == "extra-root-settings" {
				body := []byte("ignore-scripts=false\n")
				files = append(files, pnpmPayloadTestFile{tar.Header{Name: "project/.npmrc", Typeflag: tar.TypeReg, Mode: 0o600, Uid: 1000, Gid: 1000, Size: int64(len(body))}, body})
				changed = true
			}
			if !changed {
				t.Fatal("negative did not change its target")
			}
			parsed, parseErr := artifactnpm.ParsePNPMInspectionPayloadArchive(context.Background(), pnpmPayloadTestTar(t, files))
			if change == "owner" {
				if parseErr == nil {
					t.Fatal("envelope admitted Root owner")
				}
				return
			}
			if parseErr != nil {
				t.Fatal("negative failed before intended semantic comparison", parseErr)
			}
			if err := comparePNPMInspectionPayload(context.Background(), payload, parsed); err == nil {
				t.Fatal("changed execution tree admitted")
			}
		})
	}
}

func TestPNPMInspectionPayloadAllowsOnlyEquivalentAliasRepresentation(t *testing.T) {
	payload, err := preparePNPMInspectionPayload(context.Background(), pnpmBoundPayloadFixture(t, true))
	if err != nil {
		t.Fatal(err)
	}
	files := pnpmPayloadFiles(t, payload.body)
	var first, alias string
	for _, file := range files {
		if file.header.Typeflag == tar.TypeLink {
			alias = file.header.Name
			first = file.header.Linkname
		}
	}
	if alias == "" || first == "" {
		t.Fatal("shared fixture missing")
	}
	for index := range files {
		f := &files[index]
		if f.header.Name == alias {
			f.header.Typeflag = tar.TypeReg
			f.header.Linkname = ""
			f.body = []byte("shared fixture license\n")
			f.header.Size = int64(len(f.body))
		}
		if f.header.Name == first {
			f.header.Typeflag = tar.TypeLink
			f.header.Linkname = alias
			f.body = nil
			f.header.Size = 0
		}
	}
	parsed, err := artifactnpm.ParsePNPMInspectionPayloadArchive(context.Background(), pnpmPayloadTestTar(t, files))
	if err != nil {
		t.Fatal(err)
	}
	if err := comparePNPMInspectionPayload(context.Background(), payload, parsed); err != nil {
		t.Fatal("same inode partition rejected because of tar representative", err)
	}
	independent, err := preparePNPMInspectionPayload(context.Background(), pnpmBoundPayloadFixture(t, false))
	if err != nil {
		t.Fatal(err)
	}
	if err := comparePNPMInspectionPayload(context.Background(), independent, parsed); err == nil {
		t.Fatal("independent same-byte files silently merged into one inode")
	}
}

func TestPNPMInspectionPayloadCopyReadbackAndCancellation(t *testing.T) {
	payload, err := preparePNPMInspectionPayload(context.Background(), pnpmBoundPayloadFixture(t, true))
	if err != nil {
		t.Fatal(err)
	}
	runner := &pnpmControlTransport{body: payload.body}
	container := strings.Repeat("a", 64)
	if err := copyPNPMInspectionPayload(context.Background(), runner, container, payload); err != nil || runner.calls != 2 || !bytes.Equal(runner.input, payload.body) {
		t.Fatal("complete copy/readback failed", err)
	}
	cause := errors.New("injected transport failure")
	runner = &pnpmControlTransport{err: cause}
	if err := copyPNPMInspectionPayload(context.Background(), runner, container, payload); !errors.Is(err, cause) || runner.calls != 1 {
		t.Fatal("transport first cause lost or readback ran after failure", err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	runner = &pnpmControlTransport{}
	if err := copyPNPMInspectionPayload(ctx, runner, container, payload); !errors.Is(err, cause) || runner.calls != 0 {
		t.Fatal("canceled copy performed work", err)
	}
	if _, err := preparePNPMInspectionPayload(ctx, pnpmBoundPayloadFixture(t, true)); !errors.Is(err, cause) {
		t.Fatal("canceled renderer lost cause", err)
	}
	if _, err := preparePNPMInspectionPayload(context.Background(), artifactnpm.PNPMInspectionLayout{}); err == nil {
		t.Fatal("zero bound layout admitted")
	}
}
