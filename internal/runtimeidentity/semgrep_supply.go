package runtimeidentity

import (
	"encoding/hex"
	"regexp"
	"strings"
)

// SemgrepSupplyOwner is immutable selected supply data shared by the generator,
// developer supply tools and release consumers. It selects no version itself.
type SemgrepSupplyAsset struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type SemgrepSupplyOwner struct {
	Version              string             `json:"semgrep_version"`
	Architecture         string             `json:"architecture"`
	RuntimeImage         string             `json:"runtime_image"`
	SourceCommit         string             `json:"source_commit"`
	SourceTagObject      string             `json:"source_tag_object"`
	SourceManifestSHA256 string             `json:"source_manifest_sha256"`
	SourceDateEpoch      int64              `json:"source_date_epoch"`
	Tool                 SemgrepSupplyAsset `json:"tool"`
	Sources              SemgrepSupplyAsset `json:"sources"`
	Notices              SemgrepSupplyAsset `json:"notices"`
	Rules                SemgrepSupplyAsset `json:"rules"`
	Patch                SemgrepSupplyAsset `json:"patch"`
	Worker               SemgrepSupplyAsset `json:"worker"`
}

var semgrepSupplyVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
var semgrepSupplySHA256 = regexp.MustCompile(`^[a-f0-9]{64}$`)

func ValidSemgrepSupplyOwner(pin SemgrepSupplyOwner) bool {
	if !semgrepSupplyVersion.MatchString(pin.Version) || pin.Architecture != "amd64" || pin.RuntimeImage != "python_image" || !semgrepSupplySHA256.MatchString(pin.SourceManifestSHA256) || pin.SourceDateEpoch <= 0 || pin.SourceDateEpoch >= 1<<33 {
		return false
	}
	for _, value := range []string{pin.SourceCommit, pin.SourceTagObject} {
		decoded, err := hex.DecodeString(value)
		if err != nil || len(decoded) != 20 || strings.ToLower(value) != value {
			return false
		}
	}
	for _, selected := range []struct {
		asset SemgrepSupplyAsset
		path  string
		limit int64
	}{
		{pin.Tool, "semgrep-core-linux-amd64", 256 << 20},
		{pin.Sources, "semgrep-SOURCES.tar", 512 << 20},
		{pin.Notices, "semgrep-NOTICES.tar", 4 << 20},
		{pin.Rules, "tools/semgrep/rules.json", 1 << 20},
		{pin.Patch, "tools/semgrep/quarantine.patch", 1 << 20},
		{pin.Worker, "scripts/update-runtime/semgrep_build_worker.sh", 1 << 20},
	} {
		if selected.asset.Path != selected.path || !semgrepSupplySHA256.MatchString(selected.asset.SHA256) || selected.asset.Size <= 0 || selected.asset.Size > selected.limit {
			return false
		}
	}
	return true
}
