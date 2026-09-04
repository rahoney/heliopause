package runtimeidentity

import (
	"encoding/json"
	"strings"
	"testing"
)

func validLocalManifest() LocalRunscManifest {
	return LocalRunscManifest{
		SchemaVersion: LocalRunscSchema, Architecture: "amd64",
		GVisorCommit: GVisorCommit, GVisorPatchSHA256: GVisorPatchSHA256,
		BazelVersion: BazelVersion, BazelBinarySHA512: BazelLinuxX8664SHA512,
		RunscBinarySHA512: strings.Repeat("a", 128),
	}
}

func TestParseLocalRunscManifest(t *testing.T) {
	valid := validLocalManifest()
	body, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseLocalRunscManifest(body, "amd64"); err != nil {
		t.Fatalf("valid local manifest rejected: %v", err)
	}
	for _, test := range []struct {
		name string
		edit func(*LocalRunscManifest)
	}{
		{"wrong commit", func(m *LocalRunscManifest) { m.GVisorCommit = strings.Repeat("0", 40) }},
		{"wrong patch", func(m *LocalRunscManifest) { m.GVisorPatchSHA256 = strings.Repeat("0", 64) }},
		{"wrong architecture", func(m *LocalRunscManifest) { m.Architecture = "arm64" }},
		{"wrong Bazel version", func(m *LocalRunscManifest) { m.BazelVersion = "0.0.0" }},
		{"wrong Bazel digest", func(m *LocalRunscManifest) { m.BazelBinarySHA512 = strings.Repeat("0", 128) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			manifest := valid

			test.edit(&manifest)
			body, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ParseLocalRunscManifest(body, "amd64"); err == nil {
				t.Fatal("mismatched manifest accepted")
			}
		})
	}
}

func TestParseLocalRunscManifestRejectsUnknownTrailingAndOversizedInput(t *testing.T) {
	for _, body := range [][]byte{
		[]byte(`{"schema_version":1,"unknown":true}`),
		[]byte(`{} {}`),
		make([]byte, LocalRunscManifestSize+1),
	} {
		if _, err := ParseLocalRunscManifest(body, "amd64"); err == nil {
			t.Fatal("invalid manifest accepted")
		}
	}
}
