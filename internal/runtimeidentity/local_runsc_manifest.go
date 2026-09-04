package runtimeidentity

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
)

const (
	LocalRunscPath         = "/usr/libexec/heliopause/runsc"
	LocalRunscManifestPath = "/usr/libexec/heliopause/runsc.manifest.json"
	LocalRunscManifestSize = 4096
	LocalRunscSchema       = 1
)

var lowerHexSHA512 = regexp.MustCompile(`^[a-f0-9]{128}$`)

// LocalRunscManifest binds one locally built runsc to the canonical source,
// patch and Bazel inputs. It is local build custody, not a distributed release
// identity.
type LocalRunscManifest struct {
	SchemaVersion     int    `json:"schema_version"`
	Architecture      string `json:"architecture"`
	GVisorCommit      string `json:"gvisor_commit"`
	GVisorPatchSHA256 string `json:"gvisor_patch_sha256"`
	BazelVersion      string `json:"bazel_version"`
	BazelBinarySHA512 string `json:"bazel_binary_sha512"`
	RunscBinarySHA512 string `json:"runsc_binary_sha512"`
}

// ParseLocalRunscManifest strictly validates a bounded manifest against the
// canonical runtime lock and the current Go architecture.
func ParseLocalRunscManifest(body []byte, goarch string) (LocalRunscManifest, error) {
	if len(body) == 0 || len(body) > LocalRunscManifestSize {
		return LocalRunscManifest{}, errors.New("local runsc manifest size is invalid")
	}
	if goarch != "amd64" {
		return LocalRunscManifest{}, errors.New("local patched runsc architecture is unsupported")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var manifest LocalRunscManifest
	if err := decoder.Decode(&manifest); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return LocalRunscManifest{}, errors.New("local runsc manifest is malformed")
	}
	if manifest.SchemaVersion != LocalRunscSchema ||
		manifest.Architecture != goarch ||
		manifest.GVisorCommit != GVisorCommit ||
		manifest.GVisorPatchSHA256 != GVisorPatchSHA256 ||
		manifest.BazelVersion != BazelVersion ||
		manifest.BazelBinarySHA512 != BazelLinuxX8664SHA512 ||
		!lowerHexSHA512.MatchString(manifest.RunscBinarySHA512) {
		return LocalRunscManifest{}, errors.New("local runsc manifest identity mismatch")
	}
	return manifest, nil
}
