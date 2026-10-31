package releaseinstall

import (
	"encoding/json"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func pipxRuntimeMetadataFixture(t *testing.T) (ReleaseManifest, ReleaseManifest, map[string]any, []byte) {
	t.Helper()
	_, _, images := syntheticPassiveMetadata(t)
	var value map[string]any
	if err := json.Unmarshal(images, &value); err != nil {
		t.Fatal(err)
	}
	value["schema"] = "helox.runtime-image-manifest/v3"
	value["tools"] = append(value["tools"].([]any), map[string]any{"name": "pipx", "tool_version": runtimeidentity.PipxVersion, "platform": "linux/amd64", "asset": runtimeidentity.PipxRuntimeAssetName, "sha256": runtimeidentity.PipxToolSHA256, "size": runtimeidentity.PipxToolSize, "notices_asset": "pipx-NOTICE.txt", "notices_sha256": runtimeidentity.PipxNoticesSHA256, "notices_size": runtimeidentity.PipxNoticesSize, "role": "pipx-runtime", "mode": "0444", "runtime_image": runtimeidentity.PythonImageReference})
	parse := func(data bool) ReleaseManifest {
		body, err := json.Marshal(pipxDataManifestFixture(t, data))
		if err != nil {
			t.Fatal(err)
		}
		m, err := ParseReleaseManifest(body)
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	return parse(true), parse(false), value, images
}

func TestPipxMetadataBindsDataRoleOwnerAndLegacyMembership(t *testing.T) {
	current, legacy, value, old := pipxRuntimeMetadataFixture(t)
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateRuntimeImages(body); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRuntimeImagesForRelease(body, current); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRuntimeImagesForRelease(old, legacy); err != nil {
		t.Fatal("legacy v2 rejected", err)
	}
	if err := ValidateRuntimeImagesForRelease(body, legacy); err == nil {
		t.Fatal("DATA metadata without DATA release accepted")
	}
	if err := ValidateRuntimeImagesForRelease(old, current); err == nil {
		t.Fatal("DATA release used legacy metadata")
	}
	for _, mode := range []string{"role", "mode", "runtime_image", "sha256", "size", "notices_sha256", "missing", "duplicate", "unknown-field"} {
		t.Run(mode, func(t *testing.T) {
			var candidate map[string]any
			if err := json.Unmarshal(body, &candidate); err != nil {
				t.Fatal(err)
			}
			tools := candidate["tools"].([]any)
			data := tools[2].(map[string]any)
			switch mode {
			case "role":
				data["role"] = "helox"
			case "mode":
				data["mode"] = "0555"
			case "runtime_image":
				data["runtime_image"] = runtimeidentity.NodeImageReference
			case "sha256":
				data["sha256"] = "changed"
			case "size":
				data["size"] = runtimeidentity.PipxToolSize + 1
			case "notices_sha256":
				data["notices_sha256"] = "changed"
			case "missing":
				candidate["tools"] = tools[:2]
			case "duplicate":
				candidate["tools"] = append(tools, tools[2])
			case "unknown-field":
				data["host_execution_allowed"] = true
			}
			encoded, err := json.Marshal(candidate)
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateRuntimeImagesForRelease(encoded, current); err == nil {
				t.Fatal("substituted DATA metadata accepted")
			}
		})
	}
}
