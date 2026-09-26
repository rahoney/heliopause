package runtimeidentity

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path"
	"regexp"
)

const (
	LocalGVisorBundleRoot         = "/usr/libexec/heliopause/gvisor"
	LocalRunscPath                = LocalGVisorBundleRoot + "/runsc"
	LocalGVisorBundleManifestPath = LocalGVisorBundleRoot + "/gvisor-bundle.manifest.json"
	LocalGVisorBundleManifestSize = 8192
	LocalGVisorBundleSchema       = 2
)

var lowerHexSHA512 = regexp.MustCompile(`^[a-f0-9]{128}$`)

// LocalGVisorBundleManifest binds the complete locally built gVisor release
// bundle to the canonical source, patch, Bazel, and per-member identities.
// It is local build custody, not a distributed release identity.
type LocalGVisorBundleManifest struct {
	SchemaVersion          int                  `json:"schema_version"`
	Architecture           string               `json:"architecture"`
	GVisorCommit           string               `json:"gvisor_commit"`
	GVisorPatchSHA256      string               `json:"gvisor_patch_sha256"`
	BazelModuleLockSHA256  string               `json:"bazel_module_lock_sha256"`
	BuilderImageRepository string               `json:"builder_image_repository"`
	BuilderImageTag        string               `json:"builder_image_tag"`
	BuilderImageDigest     string               `json:"builder_image_digest"`
	BuilderArchitecture    string               `json:"builder_architecture"`
	BazelVersion           string               `json:"bazel_version"`
	BazelBinarySHA512      string               `json:"bazel_binary_sha512"`
	Members                []GVisorBundleMember `json:"members"`
}

// GVisorBundleMember is a single regular runtime file in the exact bundle.
type GVisorBundleMember struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA512 string `json:"sha512"`
}

// ParseLocalGVisorBundleManifest strictly validates a bounded manifest against
// the canonical runtime lock and the current Go architecture.
func ParseLocalGVisorBundleManifest(body []byte, goarch string) (LocalGVisorBundleManifest, error) {
	if len(body) == 0 || len(body) > LocalGVisorBundleManifestSize {
		return LocalGVisorBundleManifest{}, errors.New("local gVisor bundle manifest size is invalid")
	}
	if goarch != "amd64" {
		return LocalGVisorBundleManifest{}, errors.New("local gVisor bundle architecture is unsupported")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var manifest LocalGVisorBundleManifest
	if err := decoder.Decode(&manifest); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return LocalGVisorBundleManifest{}, errors.New("local gVisor bundle manifest is malformed")
	}
	if manifest.SchemaVersion != LocalGVisorBundleSchema ||
		manifest.Architecture != goarch ||
		manifest.GVisorCommit != GVisorCommit ||
		manifest.GVisorPatchSHA256 != GVisorPatchSHA256 ||
		manifest.BazelModuleLockSHA256 != GVisorBazelModuleLockSHA256 ||
		manifest.BuilderImageRepository != GVisorBuilderImageRepository ||
		manifest.BuilderImageTag != GVisorBuilderImageTag ||
		manifest.BuilderImageDigest != GVisorBuilderImageDigest ||
		manifest.BuilderArchitecture != GVisorBuilderArchitecture ||
		manifest.BazelVersion != BazelVersion ||
		manifest.BazelBinarySHA512 != BazelLinuxX8664SHA512 ||
		len(manifest.Members) != len(GVisorRuntimeBundleMembers) {
		return LocalGVisorBundleManifest{}, errors.New("local gVisor bundle manifest identity mismatch")
	}
	for index, member := range manifest.Members {
		expected := GVisorRuntimeBundleMembers[index]
		if member.Path != expected.Path || path.Clean(member.Path) != member.Path || path.IsAbs(member.Path) ||
			member.Size != expected.Size || !lowerHexSHA512.MatchString(member.SHA512) || member.SHA512 != expected.SHA512 {
			return LocalGVisorBundleManifest{}, errors.New("local gVisor bundle member identity mismatch")
		}
	}
	return manifest, nil
}
