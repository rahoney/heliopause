// Command release-manifest produces the deterministic M10 release manifest and
// CycloneDX SBOM that bind native release assets to one source commit and the
// canonical runtime lock.
package main

import (
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	manifestSchema = "helox.release-manifest/v1"
	sbomSchema     = "1.7"
	maxAssetBytes  = 1 << 30
)

var (
	releaseVersion = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
	commitID       = regexp.MustCompile(`^[0-9a-f]{40}$`)
	workflowRunURL = regexp.MustCompile(`^https://github\.com/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+/actions/runs/[1-9][0-9]*$`)
)

type stringList []string

func (values *stringList) String() string { return strings.Join(*values, ",") }
func (values *stringList) Set(value string) error {
	*values = append(*values, value)
	return nil
}

type runtimeLock struct {
	PipxTool struct {
		Version      string `json:"pipx_version"`
		Architecture string `json:"architecture"`
		RuntimeImage string `json:"runtime_image"`
		Inputs       []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"inputs"`
		Tool struct {
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
		} `json:"tool"`
		Notices struct {
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
		} `json:"notices"`
	} `json:"pipx_tool"`

	SchemaVersion int `json:"schema_version"`
	PNPMTool      struct {
		Version string `json:"pnpm_version"`
		Tool    struct {
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
		} `json:"tool"`
		Notices struct {
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
		} `json:"notices"`
	} `json:"pnpm_tool"`
	BunTool struct {
		Version string `json:"bun_version"`
		Tool    struct {
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
		} `json:"tool"`
		Notices struct {
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
		} `json:"notices"`
	} `json:"bun_tool"`
	GVisor struct {
		Release string `json:"release"`
		Commit  string `json:"commit"`
	} `json:"gvisor"`
}

type releaseAsset struct {
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Role     string `json:"role"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
}

type releaseManifest struct {
	Schema        string          `json:"schema"`
	Version       string          `json:"version"`
	SourceCommit  string          `json:"source_commit"`
	WorkflowRun   string          `json:"workflow_run"`
	RuntimeLock   runtimeIdentity `json:"runtime_lock"`
	SBOM          releaseRecord   `json:"sbom"`
	RuntimeImages releaseRecord   `json:"runtime_images"`
	Assets        []releaseAsset  `json:"assets"`
}

type runtimeIdentity struct {
	SHA256        string `json:"sha256"`
	GVisorRelease string `json:"gvisor_release"`
	GVisorCommit  string `json:"gvisor_commit"`
}

type releaseRecord struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type cyclonedxSBOM struct {
	BOMFormat   string               `json:"bomFormat"`
	SpecVersion string               `json:"specVersion"`
	Version     int                  `json:"version"`
	Metadata    cyclonedxMetadata    `json:"metadata"`
	Components  []cyclonedxComponent `json:"components"`
}

type cyclonedxMetadata struct {
	Component cyclonedxComponent `json:"component"`
}

type cyclonedxComponent struct {
	Type       string              `json:"type"`
	Name       string              `json:"name"`
	Version    string              `json:"version,omitempty"`
	Properties []cyclonedxProperty `json:"properties,omitempty"`
}

type cyclonedxProperty struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "release-manifest: %v\n", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("release-manifest", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	version := flags.String("version", "", "release version")
	commit := flags.String("commit", "", "source commit")
	workflowRun := flags.String("workflow-run", "", "GitHub Actions workflow run URL")
	runtimeLockPath := flags.String("runtime-lock", "", "canonical runtime lock")
	sbomPath := flags.String("sbom-output", "", "CycloneDX SBOM output")
	runtimeImagesPath := flags.String("runtime-images", "", "runtime image manifest")
	manifestPath := flags.String("output", "", "release manifest output")
	checksumsPath := flags.String("checksums-output", "", "canonical checksum output alongside the manifest")
	bundle := flags.String("verify-bundle", "", "verify a complete bundle and print its exact file names")
	var assets stringList
	flags.Var(&assets, "asset", "release asset path (repeatable)")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("positional arguments are not supported")
	}
	if *bundle != "" {
		if len(assets) != 0 || *checksumsPath != "" || *manifestPath != "" || *runtimeLockPath != "" || *sbomPath != "" || *runtimeImagesPath != "" {
			return errors.New("bundle verification cannot also build outputs")
		}
		names, err := verifyBundle(*bundle, *version, *commit, *workflowRun)
		if err != nil {
			return err
		}
		for _, name := range names {
			if _, err := fmt.Fprintln(os.Stdout, name); err != nil {
				return err
			}
		}
		return nil
	}
	if err := buildRelease(*version, *commit, *workflowRun, *runtimeLockPath, *sbomPath, *runtimeImagesPath, *manifestPath, assets); err != nil {
		return err
	}
	if *checksumsPath != "" {
		return writeChecksums(*manifestPath, *checksumsPath, *version, *commit, *workflowRun)
	}
	return nil
}

func buildRelease(version, commit, workflowRun, runtimeLockPath, sbomPath, runtimeImagesPath, manifestPath string, assetPaths []string) error {
	if !releaseVersion.MatchString(version) {
		return errors.New("release version is invalid")
	}
	if !commitID.MatchString(commit) {
		return errors.New("source commit is invalid")
	}
	if !workflowRunURL.MatchString(workflowRun) {
		return errors.New("workflow run URL is invalid")
	}
	lockBody, err := readRegularFile(runtimeLockPath)
	if err != nil {
		return fmt.Errorf("read runtime lock: %w", err)
	}
	var lock runtimeLock
	if err := json.Unmarshal(lockBody, &lock); err != nil || lock.SchemaVersion != 5 || lock.GVisor.Release == "" || !commitID.MatchString(lock.GVisor.Commit) || !releaseVersion.MatchString("v"+lock.PNPMTool.Version) || !releaseVersion.MatchString("v"+lock.BunTool.Version) || !releaseVersion.MatchString("v"+lock.PipxTool.Version) || lock.PipxTool.Architecture != "amd64" || lock.PipxTool.RuntimeImage != "python_image" {
		return errors.New("runtime lock is invalid")
	}
	assets, primary, err := collectAssets(assetPaths)
	if err != nil {
		return err
	}
	if err := checkNativeAssetBindings(lock, assets); err != nil {
		return err
	}
	distributions, err := pipxSourceDistributions(lock)
	if err != nil {
		return err
	}
	if err := writeSBOM(sbomPath, version, commit, primary, lock.PNPMTool.Version, lock.BunTool.Version, distributions); err != nil {
		return err
	}
	sbom, err := fileRecord(sbomPath)
	if err != nil {
		return fmt.Errorf("read release SBOM: %w", err)
	}
	runtimeImages, err := fileRecord(runtimeImagesPath)
	if err != nil {
		return fmt.Errorf("read runtime image manifest: %w", err)
	}
	lockDigest := sha256.Sum256(lockBody)
	manifest := releaseManifest{
		Schema:       manifestSchema,
		Version:      version,
		SourceCommit: commit,
		WorkflowRun:  workflowRun,
		RuntimeLock: runtimeIdentity{
			SHA256:        hex.EncodeToString(lockDigest[:]),
			GVisorRelease: lock.GVisor.Release,
			GVisorCommit:  lock.GVisor.Commit,
		},
		SBOM:          sbom,
		RuntimeImages: runtimeImages,
		Assets:        assets,
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return writeNewFile(manifestPath, append(body, '\n'))
}

func checkNativeAssetBindings(lock runtimeLock, assets []releaseAsset) error {
	for _, asset := range assets {
		if asset.Name == "pnpm-linux-amd64" && (asset.SHA256 != lock.PNPMTool.Tool.SHA256 || asset.Size != lock.PNPMTool.Tool.Size) || asset.Name == "pnpm-NOTICE.txt" && (asset.SHA256 != lock.PNPMTool.Notices.SHA256 || asset.Size != lock.PNPMTool.Notices.Size) {
			return errors.New("pnpm release asset differs from its selected runtime owner")
		}
		if asset.Name == "bun-linux-amd64" && (asset.SHA256 != lock.BunTool.Tool.SHA256 || asset.Size != lock.BunTool.Tool.Size) || asset.Name == "bun-NOTICE.txt" && (asset.SHA256 != lock.BunTool.Notices.SHA256 || asset.Size != lock.BunTool.Notices.Size) {
			return errors.New("bun release asset differs from its selected runtime owner")
		}
		if asset.Name == "pipx-runtime-linux-amd64.zip" && (asset.SHA256 != lock.PipxTool.Tool.SHA256 || asset.Size != lock.PipxTool.Tool.Size || asset.Size > 8<<20) || asset.Name == "pipx-NOTICE.txt" && (asset.SHA256 != lock.PipxTool.Notices.SHA256 || asset.Size != lock.PipxTool.Notices.Size || asset.Size > 64<<10) {
			return errors.New("pipx release DATA differs from its selected runtime owner")
		}

	}
	return nil
}

func assetSpecs() map[string]struct {
	platform string
	role     string
} {
	return map[string]struct {
		platform string
		role     string
	}{
		"helox-linux-amd64":                     {platform: "linux/amd64", role: "helox"},
		"helox-linux-arm64":                     {platform: "linux/arm64", role: "helox"},
		"helox-darwin-amd64":                    {platform: "darwin/amd64", role: "helox"},
		"helox-darwin-arm64":                    {platform: "darwin/arm64", role: "helox"},
		"haa_gvisor_observer-linux-amd64":       {platform: "linux/amd64", role: "gvisor-observer"},
		"haa-network-policy-helper-linux-amd64": {platform: "linux/amd64", role: "network-policy-helper"},
		"haa-release-installer-linux-amd64":     {platform: "linux/amd64", role: "release-installer"},
		"pnpm-linux-amd64":                      {platform: "linux/amd64", role: "pnpm"},
		"pnpm-NOTICE.txt":                       {platform: "linux/amd64", role: "release-metadata"},
		"bun-linux-amd64":                       {platform: "linux/amd64", role: "bun"},
		"bun-NOTICE.txt":                        {platform: "linux/amd64", role: "release-metadata"},
		"pipx-runtime-linux-amd64.zip":          {platform: "linux/amd64", role: "pipx-runtime"},
		"pipx-NOTICE.txt":                       {platform: "linux/amd64", role: "release-metadata"},
		"helox-linux-amd64.deb":                 {platform: "linux/amd64", role: "os-package-deb"},
		"helox-linux-amd64.rpm":                 {platform: "linux/amd64", role: "os-package-rpm"},
	}
}

func collectAssets(paths []string) ([]releaseAsset, string, error) {
	expected := assetSpecs()
	if len(paths) != len(expected) {
		return nil, "", errors.New("release asset set is incomplete")
	}
	seen := make(map[string]bool, len(expected))
	assets := make([]releaseAsset, 0, len(expected))
	primary := ""
	for _, path := range paths {
		name := filepath.Base(path)
		identity, known := expected[name]
		if !known || seen[name] {
			return nil, "", errors.New("release asset name is invalid or duplicate")
		}
		record, err := fileRecord(path)
		if err != nil {
			return nil, "", fmt.Errorf("read release asset %s: %w", name, err)
		}
		seen[name] = true
		assets = append(assets, releaseAsset{Name: record.Name, Platform: identity.platform, Role: identity.role, SHA256: record.SHA256, Size: record.Size})
		if name == "helox-linux-amd64" {
			primary = path
		}
	}
	if len(seen) != len(expected) || primary == "" {
		return nil, "", errors.New("release asset set is incomplete")
	}
	sort.Slice(assets, func(left, right int) bool { return assets[left].Name < assets[right].Name })
	return assets, primary, nil
}

// Filename parsing supplies source-owner inventory metadata only. Whole wheel
// semantics/proofs remain owned by the offline materializer and acquired inspector.
func pipxSourceDistributions(lock runtimeLock) ([]artifactpypi.PipxRuntimeDistribution, error) {
	if len(lock.PipxTool.Inputs) != 42 {
		return nil, errors.New("pipx source inventory is incomplete")
	}
	var distributions []artifactpypi.PipxRuntimeDistribution
	seen := map[string]bool{}
	main := false
	for _, input := range lock.PipxTool.Inputs {
		if !strings.HasSuffix(input.Path, ".whl") {
			continue
		}
		directory, filename, ok := strings.Cut(input.Path, "/")
		if !ok || strings.Contains(filename, "/") {
			return nil, errors.New("pipx source wheel path is invalid")
		}
		project, version, py, abi, platform, err := artifactpypi.ParseWheelFilename(filename)
		if err != nil || directory != project || seen[project] || strings.Join(py, ".") != "py3" || strings.Join(abi, ".") != "none" || strings.Join(platform, ".") != "any" {
			return nil, errors.New("pipx source wheel inventory is invalid")
		}
		seen[project] = true
		if project == "pipx" {
			if version != lock.PipxTool.Version {
				return nil, errors.New("pipx source version differs")
			}
			main = true
		}
		distributions = append(distributions, artifactpypi.PipxRuntimeDistribution{Project: project, Version: version})
	}
	if !main || len(distributions) != 11 {
		return nil, errors.New("pipx source distribution census is incomplete")
	}
	return distributions, nil
}

func writeSBOM(path, version, commit, binary, pnpmVersion, bunVersion string, distributions []artifactpypi.PipxRuntimeDistribution) error {
	build, err := buildinfo.ReadFile(binary)
	if err != nil || build.Main.Path == "" {
		return errors.New("read native helox build information")
	}
	components := make([]cyclonedxComponent, 0, len(build.Deps))
	for _, dependency := range build.Deps {
		if dependency == nil || dependency.Path == "" || dependency.Version == "" {
			return errors.New("native helox dependency build information is incomplete")
		}
		components = append(components, cyclonedxComponent{Type: "library", Name: dependency.Path, Version: dependency.Version})
	}
	components = append(components, cyclonedxComponent{Type: "application", Name: "pnpm", Version: pnpmVersion})
	components = append(components, cyclonedxComponent{Type: "application", Name: "bun", Version: bunVersion})
	for _, distribution := range distributions {
		kind, name := "library", "pypi/"+distribution.Project
		if distribution.Project == "pipx" {
			kind, name = "application", "pipx"
		}
		components = append(components, cyclonedxComponent{Type: kind, Name: name, Version: distribution.Version})
	}

	sort.Slice(components, func(left, right int) bool {
		if components[left].Name == components[right].Name {
			return components[left].Version < components[right].Version
		}
		return components[left].Name < components[right].Name
	})
	sbom := cyclonedxSBOM{
		BOMFormat:   "CycloneDX",
		SpecVersion: sbomSchema,
		Version:     1,
		Metadata: cyclonedxMetadata{Component: cyclonedxComponent{
			Type:    "application",
			Name:    "helox",
			Version: version,
			Properties: []cyclonedxProperty{
				{Name: "heliopause:source-commit", Value: commit},
				{Name: "heliopause:go-version", Value: build.GoVersion},
			},
		}},
		Components: components,
	}
	body, err := json.MarshalIndent(sbom, "", "  ")
	if err != nil {
		return err
	}
	return writeNewFile(path, append(body, '\n'))
}

func fileRecord(path string) (releaseRecord, error) {
	body, err := readRegularFile(path)
	if err != nil {
		return releaseRecord{}, err
	}
	sum := sha256.Sum256(body)
	return releaseRecord{Name: filepath.Base(path), SHA256: hex.EncodeToString(sum[:]), Size: int64(len(body))}, nil
}

func readRegularFile(path string) ([]byte, error) {
	if path == "" {
		return nil, errors.New("path is empty")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxAssetBytes {
		return nil, errors.New("path is not a bounded regular file")
	}
	return os.ReadFile(path)
}

func writeNewFile(path string, body []byte) error {
	if path == "" || len(body) == 0 {
		return errors.New("output path or content is invalid")
	}
	if _, err := os.Lstat(path); err == nil {
		return errors.New("output already exists or is unavailable")
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.New("output already exists or is unavailable")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(body); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}
