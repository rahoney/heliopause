package releaseinstall

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func decodeReleaseMetadata(body []byte, limit int, value any, keys map[string]bool) error {
	if len(body) == 0 || len(body) > limit || !utf8.Valid(body) || !uniqueMetadataJSON(body, keys) {
		return errors.New("release metadata JSON is invalid or excessive")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(value) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("release metadata schema is unsupported")
	}
	return nil
}

type sbomComponent struct {
	Type       string `json:"type"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	Properties []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"properties,omitempty"`
}

// ValidateReleaseSBOM accepts only the passive format emitted by the current
// release producer, bound to the supplied manifest and same compiled runtime.
func ValidateReleaseSBOM(body []byte, manifest ReleaseManifest) error {
	var value struct {
		BOMFormat   string `json:"bomFormat"`
		SpecVersion string `json:"specVersion"`
		Version     int    `json:"version"`
		Metadata    struct {
			Component sbomComponent `json:"component"`
		} `json:"metadata"`
		Components []sbomComponent `json:"components"`
	}
	keys := map[string]bool{"bomFormat": true, "specVersion": true, "version": true, "metadata": true, "component": true, "components": true, "type": true, "name": true, "properties": true, "value": true}
	if err := decodeReleaseMetadata(body, 8<<20, &value, keys); err != nil {
		return err
	}
	root := value.Metadata.Component
	if manifest.SourceCommit() == "" || value.BOMFormat != "CycloneDX" || value.SpecVersion != "1.7" || value.Version != 1 || root.Type != "application" || root.Name != "helox" || root.Version != manifest.Version() || len(root.Properties) != 2 || len(value.Components) == 0 || len(value.Components) > 4096 {
		return errors.New("release SBOM identity is invalid")
	}
	expected := map[string]string{"heliopause:source-commit": manifest.SourceCommit(), "heliopause:go-version": "go" + runtimeidentity.GoVersion}
	for _, property := range root.Properties {
		want, found := expected[property.Name]
		if !found || property.Value != want {
			return errors.New("release SBOM source binding is invalid")
		}
		delete(expected, property.Name)
	}
	seen := map[string]bool{}
	applications := map[string]string{"pnpm": runtimeidentity.PNPMVersion, "bun": runtimeidentity.BunVersion}
	for _, item := range value.Components {
		if version, found := applications[item.Name]; found {
			if item.Type != "application" || item.Version != version || len(item.Properties) != 0 || seen[item.Name] {
				return errors.New("release SBOM native component is invalid or duplicated")
			}
			seen[item.Name] = true
			delete(applications, item.Name)
			continue
		}
		if item.Type != "library" || !metadataText(item.Name, 256) || !metadataText(item.Version, 128) || len(item.Properties) != 0 || seen[item.Name] {
			return errors.New("release SBOM component is invalid or duplicated")
		}
		seen[item.Name] = true
	}
	if len(applications) != 0 {
		return errors.New("release SBOM lacks current native components")
	}
	return nil
}

// ValidateRuntimeImages compares the fixed passive runtime image records to
// the running controller's lock. Artifact metadata never selects host tools.
func ValidateRuntimeImages(body []byte) error {
	type image struct {
		Name       string `json:"name"`
		Reference  string `json:"reference"`
		Repository string `json:"repository"`
		Digest     string `json:"digest"`
		Version    string `json:"runtime_version"`
	}
	type nativeTool struct {
		Name          string `json:"name"`
		Version       string `json:"tool_version"`
		Platform      string `json:"platform"`
		Asset         string `json:"asset"`
		SHA256        string `json:"sha256"`
		Size          int64  `json:"size"`
		NoticesAsset  string `json:"notices_asset"`
		NoticesSHA256 string `json:"notices_sha256"`
		NoticesSize   int64  `json:"notices_size"`
	}
	var value struct {
		Schema              string `json:"schema"`
		CustomGHCRImage     *bool  `json:"custom_ghcr_image"`
		CustomImageDecision string `json:"custom_image_decision"`
		Provenance          struct {
			Required     bool   `json:"required"`
			Verification string `json:"verification"`
		} `json:"provenance"`
		Images []image      `json:"images"`
		Tools  []nativeTool `json:"tools"`
	}
	if err := decodeReleaseMetadata(body, 1<<20, &value, nil); err != nil {
		return err
	}
	if value.Schema != "helox.runtime-image-manifest/v2" || value.CustomGHCRImage == nil || *value.CustomGHCRImage || !metadataText(value.CustomImageDecision, 1024) || !value.Provenance.Required || !metadataText(value.Provenance.Verification, 1024) || len(value.Images) != 5 || len(value.Tools) != 2 {
		return errors.New("runtime image manifest contract is unsupported")
	}
	nodeVersion, _, _ := strings.Cut(strings.TrimPrefix(runtimeidentity.NodeImageReference, "node:"), "-slim@")
	expected := map[string]image{}
	for _, item := range []image{
		{Name: "node", Reference: runtimeidentity.NodeImageReference, Repository: "docker.io/library/node", Version: nodeVersion},
		{Name: "python", Reference: runtimeidentity.PythonImageReference, Repository: "docker.io/library/python", Version: runtimeidentity.PythonVersion},
		{Name: "golang", Reference: runtimeidentity.GoImageReference, Repository: "docker.io/library/golang", Version: runtimeidentity.GoVersion},
		{Name: "uv", Reference: runtimeidentity.UVImageReference, Repository: "ghcr.io/astral-sh/uv", Version: runtimeidentity.UVVersion},
		{Name: "rust", Reference: runtimeidentity.RustImageReference, Repository: "docker.io/library/rust", Version: runtimeidentity.RustVersion},
	} {
		_, item.Digest, _ = strings.Cut(item.Reference, "@")
		expected[item.Name] = item
	}
	for _, item := range value.Images {
		want, found := expected[item.Name]
		if !found || item != want {
			return errors.New("runtime image identity differs from the active lock")
		}
		delete(expected, item.Name)
	}
	expectedTools := map[string]nativeTool{
		"pnpm": {Name: "pnpm", Version: runtimeidentity.PNPMVersion, Platform: "linux/amd64", Asset: "pnpm-linux-amd64", SHA256: runtimeidentity.PNPMToolSHA256, Size: runtimeidentity.PNPMToolSize, NoticesAsset: "pnpm-NOTICE.txt", NoticesSHA256: runtimeidentity.PNPMNoticesSHA256, NoticesSize: runtimeidentity.PNPMNoticesSize},
		"bun":  {Name: "bun", Version: runtimeidentity.BunVersion, Platform: "linux/amd64", Asset: runtimeidentity.BunNativeAssetName, SHA256: runtimeidentity.BunToolSHA256, Size: runtimeidentity.BunToolSize, NoticesAsset: "bun-NOTICE.txt", NoticesSHA256: runtimeidentity.BunNoticesSHA256, NoticesSize: runtimeidentity.BunNoticesSize},
	}
	for _, tool := range value.Tools {
		want, found := expectedTools[tool.Name]
		if !found || tool != want {
			return errors.New("runtime native tool differs from the active lock")
		}
		delete(expectedTools, tool.Name)
	}
	return nil
}

func metadataText(value string, maximum int) bool {
	if value == "" || len(value) > maximum || strings.TrimSpace(value) != value {
		return false
	}
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return false
		}
	}
	return true
}

// ValidatePNPMNotices accepts only the passive reviewed bytes bound by the
// compiled current runtime owner. It grants no executable role.
func ValidatePNPMNotices(body []byte) error {
	sum := sha256.Sum256(body)
	if int64(len(body)) != runtimeidentity.PNPMNoticesSize || len(body) > 64<<10 || !utf8.Valid(body) || hex.EncodeToString(sum[:]) != runtimeidentity.PNPMNoticesSHA256 || !strings.HasPrefix(string(body), "PNPM upstream source: https://github.com/pnpm/pnpm/tree/"+runtimeidentity.PNPMSourceCommit+"\n\n") {
		return errors.New("pnpm passive notices differ from the current runtime owner")
	}
	return nil
}

// ValidateBunNotices binds passive source inventory to its compiled owner.
func ValidateBunNotices(body []byte) error {
	sum := sha256.Sum256(body)
	if int64(len(body)) != runtimeidentity.BunNoticesSize || len(body) > 64<<10 || !utf8.Valid(body) || hex.EncodeToString(sum[:]) != runtimeidentity.BunNoticesSHA256 || !strings.HasPrefix(string(body), "Bun upstream source: https://github.com/oven-sh/bun/tree/"+runtimeidentity.BunSourceCommitObserved+"\n\n") {
		return errors.New("bun passive notices differ from the current runtime owner")
	}
	return nil
}

// ValidatePipxNotices preserves original whole-wheel notices as bounded DATA.
func ValidatePipxNotices(body []byte) error {
	sum := sha256.Sum256(body)
	if int64(len(body)) != runtimeidentity.PipxNoticesSize || len(body) > 64<<10 || !utf8.Valid(body) || hex.EncodeToString(sum[:]) != runtimeidentity.PipxNoticesSHA256 {
		return errors.New("pipx passive notices differ from the selected runtime owner")
	}
	return nil
}
