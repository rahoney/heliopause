package runtimeidentity

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGVisorBundleLockInventory(t *testing.T) {
	if GVisorRelease != "release-20260907.0" || BazelVersion != "8.3.1" {
		t.Fatalf("unexpected gVisor/Bazel baseline: %q / %q", GVisorRelease, BazelVersion)
	}
	want := []string{
		"containerd-shim-runsc-v1",
		"gvisor-bin/checkpointgofer",
		"gvisor-bin/gvisor-sentry-prewarmer",
		"gvisor-bin/gvisor_sentry",
		"gvisor-bin/runsc-metric-server",
		"runsc",
	}
	if len(GVisorRuntimeBundleMembers) != len(want) {
		t.Fatalf("bundle member count=%d, want %d", len(GVisorRuntimeBundleMembers), len(want))
	}
	for i, member := range GVisorRuntimeBundleMembers {
		if member.Path != want[i] || member.Size <= 0 || !lowerHexSHA512.MatchString(member.SHA512) {
			t.Fatalf("invalid bundle member[%d]=%+v", i, member)
		}
	}
}

func validLocalGVisorBundleManifest() LocalGVisorBundleManifest {
	members := make([]GVisorBundleMember, len(GVisorRuntimeBundleMembers))
	for i, member := range GVisorRuntimeBundleMembers {
		members[i] = GVisorBundleMember(member)
	}
	return LocalGVisorBundleManifest{
		SchemaVersion: LocalGVisorBundleSchema, Architecture: "amd64",
		GVisorCommit: GVisorCommit, GVisorPatchSHA256: GVisorPatchSHA256,
		BazelModuleLockSHA256:  GVisorBazelModuleLockSHA256,
		BuilderImageRepository: GVisorBuilderImageRepository,
		BuilderImageTag:        GVisorBuilderImageTag,
		BuilderImageDigest:     GVisorBuilderImageDigest,
		BuilderArchitecture:    GVisorBuilderArchitecture,
		BazelVersion:           BazelVersion, BazelBinarySHA512: BazelLinuxX8664SHA512,
		Members: members,
	}
}

func TestParseLocalGVisorBundleManifest(t *testing.T) {
	valid := validLocalGVisorBundleManifest()
	body, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseLocalGVisorBundleManifest(body, "amd64"); err != nil {
		t.Fatalf("valid local bundle manifest rejected: %v", err)
	}
	for _, test := range []struct {
		name string
		edit func(*LocalGVisorBundleManifest)
	}{
		{"wrong commit", func(m *LocalGVisorBundleManifest) { m.GVisorCommit = strings.Repeat("0", 40) }},
		{"wrong patch", func(m *LocalGVisorBundleManifest) { m.GVisorPatchSHA256 = strings.Repeat("0", 64) }},
		{"wrong Bazel module lock", func(m *LocalGVisorBundleManifest) { m.BazelModuleLockSHA256 = strings.Repeat("0", 64) }},
		{"wrong builder digest", func(m *LocalGVisorBundleManifest) { m.BuilderImageDigest = "sha256:" + strings.Repeat("0", 64) }},
		{"wrong architecture", func(m *LocalGVisorBundleManifest) { m.Architecture = "arm64" }},
		{"wrong Bazel version", func(m *LocalGVisorBundleManifest) { m.BazelVersion = "0.0.0" }},
		{"wrong member hash", func(m *LocalGVisorBundleManifest) { m.Members[0].SHA512 = strings.Repeat("0", 128) }},
		{"missing member", func(m *LocalGVisorBundleManifest) { m.Members = m.Members[:len(m.Members)-1] }},
		{"path traversal", func(m *LocalGVisorBundleManifest) { m.Members[0].Path = "../runsc" }},
		{"absolute member path", func(m *LocalGVisorBundleManifest) { m.Members[0].Path = "/runsc" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			manifest := validLocalGVisorBundleManifest()
			test.edit(&manifest)
			body, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ParseLocalGVisorBundleManifest(body, "amd64"); err == nil {
				t.Fatal("mismatched bundle manifest accepted")
			}
		})
	}
}

func TestParseLocalGVisorBundleManifestRejectsUnknownTrailingAndOversizedInput(t *testing.T) {
	for _, body := range [][]byte{
		[]byte(`{"schema_version":1,"unknown":true}`),
		[]byte(`{} {}`),
		make([]byte, LocalGVisorBundleManifestSize+1),
	} {
		if _, err := ParseLocalGVisorBundleManifest(body, "amd64"); err == nil {
			t.Fatal("invalid bundle manifest accepted")
		}
	}
}
