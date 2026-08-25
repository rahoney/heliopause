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
	if err := os.WriteFile(runtimeLock, []byte(`{"schema_version":1,"gvisor":{"release":"release-20260810.0","commit":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}`), 0o600); err != nil {
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
	if manifest.Schema != manifestSchema || manifest.Version != "v1.2.3" || manifest.SourceCommit != strings.Repeat("b", 40) || manifest.WorkflowRun != workflowRun || len(manifest.Assets) != 6 {
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
	if err := os.WriteFile(runtimeLock, []byte(`{"schema_version":1,"gvisor":{"release":"release-20260810.0","commit":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}`), 0o600); err != nil {
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
