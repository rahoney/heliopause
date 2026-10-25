// Command runtime-image-manifest emits the lock-owned runtime image contract.
// It does not build or republish upstream official images.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const manifestSchema = "helox.runtime-image-manifest/v2"

var (
	digestPattern        = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	versionPattern       = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	nodeReferencePattern = regexp.MustCompile(`^node:([0-9]+\.[0-9]+\.[0-9]+)-slim@sha256:[0-9a-f]{64}$`)
)

type runtimeLock struct {
	SchemaVersion int `json:"schema_version"`
	GoImage       struct {
		Reference    string `json:"reference"`
		GoVersion    string `json:"go_version"`
		Architecture string `json:"architecture"`
	} `json:"go_image"`
	NodeImage struct {
		Reference  string `json:"reference"`
		NPMVersion string `json:"npm_version"`
	} `json:"node_image"`
	PythonImage struct {
		Reference     string `json:"reference"`
		PythonVersion string `json:"python_version"`
		PipVersion    string `json:"pip_version"`
	} `json:"python_image"`
	UVImage struct {
		Reference    string `json:"reference"`
		Version      string `json:"uv_version"`
		Architecture string `json:"architecture"`
	} `json:"uv_image"`
	RustImage struct {
		Reference    string `json:"reference"`
		RustVersion  string `json:"rust_version"`
		Architecture string `json:"architecture"`
		Target       string `json:"target"`
	} `json:"rust_image"`
	PNPMTool struct {
		Version      string `json:"pnpm_version"`
		Architecture string `json:"architecture"`
		Tool         struct {
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
		} `json:"tool"`
		Notices struct {
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
		} `json:"notices"`
	} `json:"pnpm_tool"`
}

type runtimeImageManifest struct {
	Schema              string              `json:"schema"`
	CustomGHCRImage     bool                `json:"custom_ghcr_image"`
	CustomImageDecision string              `json:"custom_image_decision"`
	Provenance          provenanceContract  `json:"provenance"`
	Images              []runtimeImageEntry `json:"images"`
	Tools               []runtimeToolEntry  `json:"tools"`
}

type runtimeToolEntry struct {
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

type provenanceContract struct {
	Required     bool   `json:"required"`
	Verification string `json:"verification"`
}

type runtimeImageEntry struct {
	Name       string `json:"name"`
	Reference  string `json:"reference"`
	Repository string `json:"repository"`
	Digest     string `json:"digest"`
	Version    string `json:"runtime_version"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "runtime-image-manifest: %v\n", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("runtime-image-manifest", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	lockPath := flags.String("runtime-lock", "", "canonical runtime lock")
	outputPath := flags.String("output", "", "runtime image manifest output")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("positional arguments are not supported")
	}
	if *lockPath == "" || *outputPath == "" {
		return errors.New("runtime lock and output are required")
	}
	lockBody, err := os.ReadFile(*lockPath)
	if err != nil {
		return fmt.Errorf("read runtime lock: %w", err)
	}
	var lock runtimeLock
	if err := json.Unmarshal(lockBody, &lock); err != nil {
		return fmt.Errorf("decode runtime lock: %w", err)
	}
	manifest, err := buildManifest(lock)
	if err != nil {
		return err
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if _, err := os.Lstat(*outputPath); err == nil || !errors.Is(err, os.ErrNotExist) {
		return errors.New("output already exists or is unavailable")
	}
	file, err := os.OpenFile(*outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(body); err != nil {
		_ = file.Close()
		_ = os.Remove(*outputPath)
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(*outputPath)
		return err
	}
	return file.Close()
}

func buildManifest(lock runtimeLock) (runtimeImageManifest, error) {
	if lock.SchemaVersion != 4 {
		return runtimeImageManifest{}, errors.New("runtime lock schema is unsupported")
	}
	nodeMatch := nodeReferencePattern.FindStringSubmatch(lock.NodeImage.Reference)
	if len(nodeMatch) != 2 || !versionPattern.MatchString(lock.NodeImage.NPMVersion) {
		return runtimeImageManifest{}, errors.New("node runtime image identity is invalid")
	}
	node, err := imageEntry("node", lock.NodeImage.Reference, "docker.io/library/node", nodeMatch[1], "-slim")
	if err != nil {
		return runtimeImageManifest{}, err
	}
	python, err := imageEntry("python", lock.PythonImage.Reference, "docker.io/library/python", lock.PythonImage.PythonVersion, "-slim-bookworm")
	if err != nil || !versionPattern.MatchString(lock.PythonImage.PipVersion) {
		return runtimeImageManifest{}, errors.New("python runtime image identity is invalid")
	}
	goImage, err := imageEntry("golang", lock.GoImage.Reference, "docker.io/library/golang", lock.GoImage.GoVersion, "-bookworm")
	if err != nil || lock.GoImage.Architecture != "amd64" || !strings.HasPrefix(lock.GoImage.Reference, "golang:"+lock.GoImage.GoVersion+"-bookworm@") {
		return runtimeImageManifest{}, errors.New("go runtime image identity is invalid")
	}
	rustImage, err := imageEntry("rust", lock.RustImage.Reference, "docker.io/library/rust", lock.RustImage.RustVersion, "-bookworm")
	if err != nil || lock.RustImage.Architecture != "amd64" || lock.RustImage.Target != "x86_64-unknown-linux-gnu" {
		return runtimeImageManifest{}, errors.New("rust runtime image identity is invalid")
	}
	uv := lock.UVImage
	prefix := "ghcr.io/astral-sh/uv:" + uv.Version + "@"
	if !versionPattern.MatchString(uv.Version) || uv.Architecture != "amd64" || !strings.HasPrefix(uv.Reference, prefix) || !digestPattern.MatchString(strings.TrimPrefix(uv.Reference, prefix)) {
		return runtimeImageManifest{}, errors.New("uv upstream runtime image identity is invalid")
	}
	uvImage := runtimeImageEntry{Name: "uv", Reference: uv.Reference, Repository: "ghcr.io/astral-sh/uv", Digest: strings.TrimPrefix(uv.Reference, prefix), Version: uv.Version}
	pnpm := lock.PNPMTool
	if !versionPattern.MatchString(pnpm.Version) || pnpm.Architecture != "amd64" || !digestPattern.MatchString("sha256:"+pnpm.Tool.SHA256) || pnpm.Tool.Size <= 0 || pnpm.Tool.Size > 64<<20 || !digestPattern.MatchString("sha256:"+pnpm.Notices.SHA256) || pnpm.Notices.Size <= 0 || pnpm.Notices.Size > 64<<10 {
		return runtimeImageManifest{}, errors.New("pnpm native runtime tool identity is invalid")
	}
	return runtimeImageManifest{
		Schema:              manifestSchema,
		CustomGHCRImage:     false,
		CustomImageDecision: "upstream official immutable images, including signed Astral uv supply; no HAA-owned image is published",
		Provenance: provenanceContract{
			Required:     true,
			Verification: "release candidate attestation must bind this manifest and each exact image reference",
		},
		Images: []runtimeImageEntry{node, python, goImage, rustImage, uvImage},
		Tools:  []runtimeToolEntry{{Name: "pnpm", Version: pnpm.Version, Platform: "linux/amd64", Asset: "pnpm-linux-amd64", SHA256: pnpm.Tool.SHA256, Size: pnpm.Tool.Size, NoticesAsset: "pnpm-NOTICE.txt", NoticesSHA256: pnpm.Notices.SHA256, NoticesSize: pnpm.Notices.Size}},
	}, nil
}

func imageEntry(name, reference, repository, version, variant string) (runtimeImageEntry, error) {
	separator := strings.LastIndex(reference, "@")
	if separator <= len(name)+1 || reference == "" || version == "" || !versionPattern.MatchString(version) || !digestPattern.MatchString(reference[separator+1:]) {
		return runtimeImageEntry{}, errors.New("runtime image identity is invalid")
	}
	if reference[:separator] != name+":"+version+variant {
		return runtimeImageEntry{}, errors.New("runtime image repository is not an official image")
	}
	return runtimeImageEntry{Name: name, Reference: reference, Repository: repository, Digest: reference[strings.LastIndex(reference, "@")+1:], Version: version}, nil
}
