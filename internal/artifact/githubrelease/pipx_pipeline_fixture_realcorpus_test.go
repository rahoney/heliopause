//go:build linux && realcorpus

package githubrelease

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/testutil/releasefixture"
)

// This reuses the existing source/staging fixture and its unsigned proof. Only
// the selected DATA is real; native sandbox results remain explicit test doubles.
func NewPipxReleaseSetPipelineFixture(t *testing.T, zip, producerSBOM []byte) ReleaseSetPipelineFixture {
	t.Helper()
	notices, err := os.ReadFile(filepath.Join("..", "..", "..", "tools", "pipx", "NOTICE"))
	if err != nil {
		t.Fatal(err)
	}
	f := newReleaseSetFixture(t, func(metadata map[string]any) {
		assets := metadata["assets"].([]any)
		for name, body := range map[string][]byte{runtimeidentity.PipxRuntimeAssetName: zip, "pipx-NOTICE.txt": notices} {
			role := "release-metadata"
			if name == runtimeidentity.PipxRuntimeAssetName {
				role = "pipx-runtime"
			}
			assets = append(assets, map[string]any{"name": name, "platform": "linux/amd64", "role": role, "size": len(body), "sha256": fmt.Sprintf("%x", sha256.Sum256(body))})
		}
		metadata["assets"] = assets
	})
	f.files[runtimeidentity.PipxRuntimeAssetName] = append([]byte(nil), zip...)
	f.files["pipx-NOTICE.txt"] = notices
	// Rebind only this synthetic fixture's HAA identity. Do not edit the retained
	// producer output or claim that its old signed bundle authenticates this set.
	var sbom map[string]any
	if err := json.Unmarshal(producerSBOM, &sbom); err != nil {
		t.Fatal(err)
	}
	component := sbom["metadata"].(map[string]any)["component"].(map[string]any)
	component["version"] = f.metadata["version"]
	for _, raw := range component["properties"].([]any) {
		item := raw.(map[string]any)
		if item["name"] == "heliopause:source-commit" {
			item["value"] = f.metadata["source_commit"]
		}
	}
	body, err := json.Marshal(sbom)
	if err != nil {
		t.Fatal(err)
	}
	materials, err := releasefixture.BindCurrentRuntime(f.files[heliopauseManifestName], body, f.files["helox-runtime-images.json"])
	if err != nil {
		t.Fatal(err)
	}
	f.files[heliopauseManifestName], f.files["helox-release-sbom.cdx.json"], f.files["helox-runtime-images.json"] = materials.Manifest, materials.SBOM, materials.Images
	if err := json.Unmarshal(materials.Manifest, &f.metadata); err != nil {
		t.Fatal(err)
	}
	// No source freeze/resolve has occurred. Refresh the original fixture's
	// acquired numeric declarations to the same rebased bytes before first use.
	for i := range f.release.Assets {
		item := &f.release.Assets[i]
		if data, ok := f.files[item.Name]; ok {
			item.Size = int64(len(data))
			item.Digest = fmt.Sprintf("sha256:%x", sha256.Sum256(data))
		}
	}
	return ReleaseSetPipelineFixture{f.set, f.client.intakeRoot, f.install}
}
