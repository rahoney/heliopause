package releaseinstall

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func publishedManifest(t *testing.T) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "docs", "planning", "evidence", "m13-003-release-build", "v0.1.0", manifestName))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestStrictReaderPreservesPublishedReleaseAndRequiresInstallerForUpdate(t *testing.T) {
	body := publishedManifest(t)
	parsed, err := ParseReleaseManifest(body)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Version() != "v0.1.0" || parsed.SourceCommit() != "ef5ef0e0e0437a9083133674ef8b9e6d693b83a9" || parsed.WorkflowRun() != "39036210335" || len(parsed.Files()) != 10 || parsed.RuntimeLockSHA256() == "" || parsed.GVisorRelease() == "" || parsed.GVisorCommit() == "" {
		t.Fatal("published identity or files differ")
	}
	if _, err := parsed.NativeUpdateFiles(); err == nil {
		t.Fatal("historical release without separate installer became update-capable")
	}
	files := parsed.Files()
	files[0].Name = "mutated"
	if parsed.Files()[0].Name == "mutated" {
		t.Fatal("caller mutated manifest view")
	}
	var value manifest
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatal(err)
	}
	value.Assets = append(value.Assets, asset{"haa-release-installer-linux-amd64", "linux/amd64", "release-installer", strings.Repeat("a", 64), 123})
	updated, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err = ParseReleaseManifest(updated)
	if err != nil {
		t.Fatal(err)
	}
	native, err := parsed.NativeUpdateFiles()
	if err != nil || len(native) != 4 {
		t.Fatalf("native=%v error=%v", native, err)
	}
	if len((ReleaseManifest{}).Files()) != 0 || (ReleaseManifest{}).WorkflowRun() != "" {
		t.Fatal("zero view has files or run")
	}
	if _, err := (ReleaseManifest{}).NativeUpdateFiles(); err == nil {
		t.Fatal("zero view accepted")
	}
}

func TestStrictReaderRejectsAmbiguousMalformedAndUnboundMetadata(t *testing.T) {
	body := publishedManifest(t)
	change := func(edit func(*manifest)) []byte {
		var value manifest
		if err := json.Unmarshal(body, &value); err != nil {
			t.Fatal(err)
		}
		edit(&value)
		result, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	cases := map[string][]byte{
		"empty":                  nil,
		"excessive":              bytes.Repeat([]byte(" "), maximumManifestBytes+1),
		"invalid-utf8":           append([]byte{0xff}, body...),
		"trailing":               append(append([]byte(nil), body...), []byte(" {}")...),
		"duplicate-field":        bytes.Replace(body, []byte(`"schema":`), []byte(`"schema":"different","schema":`), 1),
		"case-alias":             bytes.Replace(body, []byte(`"schema":`), []byte(`"SCHEMA":`), 1),
		"unknown-field":          bytes.Replace(body, []byte(`"schema":`), []byte(`"unsupported":true,"schema":`), 1),
		"nested-duplicate":       bytes.Replace(body, []byte(`"gvisor_commit":`), []byte(`"gvisor_commit":"bad","gvisor_commit":`), 1),
		"missing-native":         change(func(m *manifest) { m.Assets = m.Assets[1:] }),
		"duplicate-asset":        change(func(m *manifest) { m.Assets[1] = m.Assets[0] }),
		"unknown-asset":          change(func(m *manifest) { m.Assets[0].Name = "unexpected" }),
		"path-asset":             change(func(m *manifest) { m.Assets[0].Name = "../helox" }),
		"role":                   change(func(m *manifest) { m.Assets[0].Role = "release-installer" }),
		"platform":               change(func(m *manifest) { m.Assets[0].Platform = "linux/arm64" }),
		"zero-size":              change(func(m *manifest) { m.Assets[0].Size = 0 }),
		"large-size":             change(func(m *manifest) { m.Assets[0].Size = maxBundleFileSize + 1 }),
		"digest":                 change(func(m *manifest) { m.Assets[0].SHA256 = strings.Repeat("A", 64) }),
		"source":                 change(func(m *manifest) { m.SourceCommit = strings.Repeat("A", 40) }),
		"tag":                    change(func(m *manifest) { m.Version = "v01.0.0" }),
		"prerelease":             change(func(m *manifest) { m.Version = "v1.2.3-.." }),
		"wrong-repository":       change(func(m *manifest) { m.WorkflowRun = "https://github.com/other/heliopause/actions/runs/1" }),
		"zero-run":               change(func(m *manifest) { m.WorkflowRun = "https://github.com/rahoney/heliopause/actions/runs/0" }),
		"sbom-name":              change(func(m *manifest) { m.SBOM.Name = manifestName }),
		"sbom-size":              change(func(m *manifest) { m.SBOM.Size = 8<<20 + 1 }),
		"runtime-images-size":    change(func(m *manifest) { m.RuntimeImages.Size = 1<<20 + 1 }),
		"lock-digest":            change(func(m *manifest) { m.RuntimeLock.SHA256 = "bad" }),
		"gvisor-commit":          change(func(m *manifest) { m.RuntimeLock.GVisorCommit = "bad" }),
		"gvisor-release-control": change(func(m *manifest) { m.RuntimeLock.GVisorRelease = "release\tinvalid" }),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseReleaseManifest(input); err == nil {
				t.Fatal("invalid manifest accepted")
			}
		})
	}
}
