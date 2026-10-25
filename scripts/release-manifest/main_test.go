package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildReleaseProducesBoundManifestAndSBOM(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "helox-linux-amd64")
	buildNativeHeloxTest(t, binary)
	assets := releaseAssets(t, root, binary)
	runtimeLock := filepath.Join(root, "runtimes.lock.json")
	if err := os.WriteFile(runtimeLock, syntheticReleaseLock(t, assets), 0o600); err != nil {
		t.Fatal(err)
	}
	sbomPath := filepath.Join(root, "helox-release-sbom.cdx.json")
	runtimeImagesPath := filepath.Join(root, "helox-runtime-images.json")
	if err := os.WriteFile(runtimeImagesPath, []byte(`{"schema":"helox.runtime-image-manifest/v1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "helox-release-manifest.json")
	workflowRun := "https://github.com/rahoney/heliopause/actions/runs/123"
	if err := buildRelease("v1.2.3", strings.Repeat("b", 40), workflowRun, runtimeLock, sbomPath, runtimeImagesPath, manifestPath, assets); err != nil {
		t.Fatal(err)
	}
	var manifest releaseManifest
	body, err := os.ReadFile(manifestPath)
	if err != nil || json.Unmarshal(body, &manifest) != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if manifest.Schema != manifestSchema || manifest.Version != "v1.2.3" || manifest.SourceCommit != strings.Repeat("b", 40) || manifest.WorkflowRun != workflowRun || len(manifest.Assets) != 11 {
		t.Fatalf("manifest = %#v", manifest)
	}
	if manifest.RuntimeLock.GVisorCommit != strings.Repeat("a", 40) || manifest.SBOM.Name != filepath.Base(sbomPath) || manifest.SBOM.SHA256 == "" {
		t.Fatalf("manifest bindings = %#v", manifest)
	}
	if manifest.RuntimeImages.Name != filepath.Base(runtimeImagesPath) || manifest.RuntimeImages.SHA256 == "" {
		t.Fatalf("runtime image binding = %#v", manifest.RuntimeImages)
	}
	var sbom cyclonedxSBOM
	sbomBody, err := os.ReadFile(sbomPath)
	if err != nil || json.Unmarshal(sbomBody, &sbom) != nil {
		t.Fatalf("read sbom: %v", err)
	}
	if sbom.BOMFormat != "CycloneDX" || sbom.SpecVersion != sbomSchema || sbom.Metadata.Component.Name != "helox" || len(sbom.Components) == 0 {
		t.Fatalf("sbom = %#v", sbom)
	}
}

func TestBuildReleaseRejectsIncompleteOrPreexistingOutput(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "helox-linux-amd64")
	buildNativeHeloxTest(t, binary)
	assets := releaseAssets(t, root, binary)
	runtimeLock := filepath.Join(root, "runtimes.lock.json")
	if err := os.WriteFile(runtimeLock, syntheticReleaseLock(t, assets), 0o600); err != nil {
		t.Fatal(err)
	}
	workflowRun := "https://github.com/rahoney/heliopause/actions/runs/123"
	runtimeImagesPath := filepath.Join(root, "runtime-images.json")
	if err := os.WriteFile(runtimeImagesPath, []byte("runtime-images"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := buildRelease("v1.2.3", strings.Repeat("b", 40), workflowRun, runtimeLock, filepath.Join(root, "sbom.json"), runtimeImagesPath, filepath.Join(root, "manifest.json"), assets[:5]); err == nil {
		t.Fatal("incomplete asset set was accepted")
	}
	if err := os.WriteFile(filepath.Join(root, "existing.json"), []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := buildRelease("v1.2.3", strings.Repeat("b", 40), workflowRun, runtimeLock, filepath.Join(root, "sbom.json"), runtimeImagesPath, filepath.Join(root, "existing.json"), assets); err == nil {
		t.Fatal("preexisting manifest output was accepted")
	}
	if err := buildRelease("v1.2.3", strings.Repeat("b", 40), "https://attacker.invalid/run/123", runtimeLock, filepath.Join(root, "other-sbom.json"), runtimeImagesPath, filepath.Join(root, "other-manifest.json"), assets); err == nil {
		t.Fatal("unexpected workflow run URL was accepted")
	}
}

func buildNativeHeloxTest(t *testing.T, output string) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "build", "-trimpath", "-buildvcs=true", "-o", output, "./cmd/helox")
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build native helox: %v\n%s", err, output)
	}
}

func releaseAssets(t *testing.T, root, binary string) []string {
	t.Helper()
	names := []string{
		"helox-linux-amd64",
		"helox-linux-arm64",
		"helox-darwin-amd64",
		"helox-darwin-arm64",
		"haa_gvisor_observer-linux-amd64",
		"haa-network-policy-helper-linux-amd64",
		"haa-release-installer-linux-amd64",
		"pnpm-linux-amd64",
		"pnpm-NOTICE.txt",
		"helox-linux-amd64.deb",
		"helox-linux-amd64.rpm",
	}
	paths := make([]string, 0, len(names))
	for _, name := range names {
		path := filepath.Join(root, name)
		if name == "helox-linux-amd64" {
			paths = append(paths, binary)
			continue
		}
		body := []byte("fixture-" + name)
		if err := os.WriteFile(path, body, 0o700); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	return paths
}

// Synthetic owner bytes for producer mechanics; no upstream or runtime approval.
func syntheticReleaseLock(t *testing.T, paths []string) []byte {
	t.Helper()
	var lock runtimeLock
	lock.SchemaVersion = 4
	lock.GVisor.Release = "release-20260810.0"
	lock.GVisor.Commit = strings.Repeat("a", 40)
	lock.PNPMTool.Version = "1.2.3"
	for _, path := range paths {
		switch filepath.Base(path) {
		case "pnpm-linux-amd64":
			record, err := fileRecord(path)
			if err != nil {
				t.Fatal(err)
			}
			lock.PNPMTool.Tool.SHA256 = record.SHA256
			lock.PNPMTool.Tool.Size = record.Size
		case "pnpm-NOTICE.txt":
			record, err := fileRecord(path)
			if err != nil {
				t.Fatal(err)
			}
			lock.PNPMTool.Notices.SHA256 = record.SHA256
			lock.PNPMTool.Notices.Size = record.Size
		}
	}
	body, err := json.Marshal(lock)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestPNPMReleaseAssetsMustMatchSelectedOwnerBeforeMetadataWrites(t *testing.T) {
	for _, name := range []string{"pnpm-linux-amd64", "pnpm-NOTICE.txt"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			binary := filepath.Join(root, "helox-linux-amd64")
			if err := os.WriteFile(binary, []byte("not a Go binary; binding rejection must precede buildinfo"), 0600); err != nil {
				t.Fatal(err)
			}
			assets := releaseAssets(t, root, binary)
			lock := filepath.Join(root, "runtimes.lock.json")
			if err := os.WriteFile(lock, syntheticReleaseLock(t, assets), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, name), []byte("substituted after materialization"), 0600); err != nil {
				t.Fatal(err)
			}
			sbom, manifest := filepath.Join(root, "sbom.json"), filepath.Join(root, "manifest.json")
			err := buildRelease("v1.2.3", strings.Repeat("b", 40), "https://github.com/rahoney/heliopause/actions/runs/123", lock, sbom, filepath.Join(root, "images.json"), manifest, assets)
			if err == nil || !strings.Contains(err.Error(), "pnpm release asset differs") {
				t.Fatalf("missing source-owner rejection: %v", err)
			}
			for _, path := range []string{sbom, manifest} {
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					t.Fatal("substituted pnpm produced metadata")
				}
			}
		})
	}
}
