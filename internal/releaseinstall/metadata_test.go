package releaseinstall

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/testutil/releasefixture"
)

func publishedMetadata(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "docs", "planning", "evidence", "m13-003-release-build", "v0.1.0", name))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestPassiveSyntheticReleaseMetadataMatchesCompiledRuntime(t *testing.T) {
	manifest, sbom, images := syntheticPassiveMetadata(t)
	if manifest.SHA256() == "" {
		t.Fatal("strict view lacks its exact input digest")
	}
	if err := ValidateReleaseSBOM(sbom, manifest); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRuntimeImages(images); err != nil {
		t.Fatal(err)
	}
}

func TestPassiveReleaseMetadataRejectsAmbiguityAndIdentitySubstitution(t *testing.T) {
	manifest, sbom, images := syntheticPassiveMetadata(t)
	mutate := func(body []byte, edit func(map[string]any)) []byte {
		var value map[string]any
		if err := json.Unmarshal(body, &value); err != nil {
			t.Fatal(err)
		}
		edit(value)
		result, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	imageChange := func(edit func(map[string]any)) []byte {
		return mutate(images, func(m map[string]any) { edit(m["images"].([]any)[0].(map[string]any)) })
	}
	toolChange := func(edit func(map[string]any)) []byte {
		return mutate(images, func(m map[string]any) { edit(m["tools"].([]any)[0].(map[string]any)) })
	}
	for name, body := range map[string][]byte{
		"empty":                 nil,
		"unknown":               bytes.Replace(images, []byte(`"schema":`), []byte(`"other":1,"schema":`), 1),
		"duplicate":             bytes.Replace(images, []byte(`"schema":`), []byte(`"schema":"wrong","schema":`), 1),
		"trailing":              append(append([]byte(nil), images...), []byte(" null")...),
		"custom":                mutate(images, func(m map[string]any) { m["custom_ghcr_image"] = true }),
		"missing-custom":        mutate(images, func(m map[string]any) { delete(m, "custom_ghcr_image") }),
		"missing-provenance":    mutate(images, func(m map[string]any) { delete(m, "provenance") }),
		"missing-image":         mutate(images, func(m map[string]any) { m["images"] = m["images"].([]any)[1:] }),
		"duplicate-image":       mutate(images, func(m map[string]any) { list := m["images"].([]any); list[1] = list[0] }),
		"wrong-reference":       imageChange(func(m map[string]any) { m["reference"] = "node:latest" }),
		"wrong-repository":      imageChange(func(m map[string]any) { m["repository"] = "other" }),
		"wrong-digest":          imageChange(func(m map[string]any) { m["digest"] = "sha256:bad" }),
		"wrong-version":         imageChange(func(m map[string]any) { m["runtime_version"] = "0.0.0" }),
		"old-image-schema":      mutate(images, func(m map[string]any) { m["schema"] = "helox.runtime-image-manifest/v1" }),
		"missing-native-tool":   mutate(images, func(m map[string]any) { delete(m, "tools") }),
		"duplicate-native-tool": mutate(images, func(m map[string]any) { v := m["tools"].([]any); m["tools"] = append(v, v[0]) }),
		"tool-name":             toolChange(func(m map[string]any) { m["name"] = "other" }),
		"tool-version":          toolChange(func(m map[string]any) { m["tool_version"] = "0.0.1" }),
		"tool-platform":         toolChange(func(m map[string]any) { m["platform"] = "linux/arm64" }),
		"tool-asset":            toolChange(func(m map[string]any) { m["asset"] = "../pnpm" }),
		"tool-hash":             toolChange(func(m map[string]any) { m["sha256"] = "bad" }),
		"tool-size":             toolChange(func(m map[string]any) { m["size"] = 1 }),
	} {
		t.Run("images/"+name, func(t *testing.T) {
			if err := ValidateRuntimeImages(body); err == nil {
				t.Fatal("invalid runtime image metadata accepted")
			}
		})
	}
	for name, body := range map[string][]byte{
		"empty":      nil,
		"excessive":  bytes.Repeat([]byte(" "), 8<<20+1),
		"case-alias": bytes.Replace(sbom, []byte(`"bomFormat":`), []byte(`"bomformat":`), 1),
		"unknown":    bytes.Replace(sbom, []byte(`"bomFormat":`), []byte(`"other":1,"bomFormat":`), 1),
		"duplicate":  bytes.Replace(sbom, []byte(`"bomFormat":`), []byte(`"bomFormat":"wrong","bomFormat":`), 1),
		"wrong-version": mutate(sbom, func(m map[string]any) {
			m["metadata"].(map[string]any)["component"].(map[string]any)["version"] = "v9.0.0"
		}),
		"duplicate-property": mutate(sbom, func(m map[string]any) {
			props := m["metadata"].(map[string]any)["component"].(map[string]any)["properties"].([]any)
			props[1] = props[0]
		}),
		"wrong-source": mutate(sbom, func(m map[string]any) {
			m["metadata"].(map[string]any)["component"].(map[string]any)["properties"].([]any)[0].(map[string]any)["value"] = "bad"
		}),
		"duplicate-library": mutate(sbom, func(m map[string]any) { components := m["components"].([]any); components[1] = components[0] }),
		"active-component":  mutate(sbom, func(m map[string]any) { m["components"].([]any)[0].(map[string]any)["type"] = "application" }),
	} {
		t.Run("sbom/"+name, func(t *testing.T) {
			if err := ValidateReleaseSBOM(body, manifest); err == nil {
				t.Fatal("invalid SBOM accepted")
			}
		})
	}
	if err := ValidateReleaseSBOM(sbom, ReleaseManifest{}); err == nil {
		t.Fatal("zero source accepted")
	}
}

// These current-runtime materials are deliberately unsigned. The committed
// published files and actual signature tests retain their original bytes.
func syntheticPassiveMetadata(t *testing.T) (ReleaseManifest, []byte, []byte) {
	t.Helper()
	materials, err := releasefixture.BindCurrentRuntime(publishedManifest(t), publishedMetadata(t, sbomName), publishedMetadata(t, runtimeImagesName))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := ParseReleaseManifest(materials.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	return manifest, materials.SBOM, materials.Images
}
