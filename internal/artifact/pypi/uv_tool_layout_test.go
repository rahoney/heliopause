package pypi

import (
	"bytes"
	"os"
	"reflect"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestUVToolLayoutPreservesNativeRecipeAndOriginalBytes(t *testing.T) {
	for _, fixture := range []string{"uv-installed.toml", "uv-upgraded.toml", "uv-explicit-index.toml"} {
		t.Run(fixture, func(t *testing.T) {
			body, err := os.ReadFile("testdata/persistent-tool/" + fixture)
			if err != nil {
				t.Fatal(err)
			}
			original, err := ParseUVToolReceipt(body)
			if err != nil {
				t.Fatal(err)
			}
			environment := "/tools/env with spaces/" + original.Requirement().Project()
			bin := "/tools/exposed commands"
			projected, err := original.WithUVLayout(environment, bin)
			if err != nil {
				t.Fatal(err)
			}
			if err := projected.ValidateLayout(environment, bin, environment+"/bin/python"); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(original.Body(), body) || reflect.DeepEqual(original.Commands(), projected.Commands()) {
				t.Fatal("original mutated or guest paths retained")
			}
			var before, after uvToolReceiptWire
			if err := toml.Unmarshal(body, &before); err != nil {
				t.Fatal(err)
			}
			if err := toml.Unmarshal(projected.Body(), &after); err != nil {
				t.Fatal(err)
			}
			after.Tool.Python = before.Tool.Python
			for i := range after.Tool.Entrypoints {
				after.Tool.Entrypoints[i].Path = before.Tool.Entrypoints[i].Path
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("requirement/options/command semantics changed")
			}
			var sourceWire, projectedWire map[string]any
			if err := toml.Unmarshal(body, &sourceWire); err != nil {
				t.Fatal(err)
			}
			if err := toml.Unmarshal(projected.Body(), &projectedWire); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(sourceWire["tool"].(map[string]any)["options"], projectedWire["tool"].(map[string]any)["options"]) {
				t.Fatal("omitted or explicit options changed")
			}
		})
	}
}

func TestUVToolLayoutRejectsWrongNamespaceAndOverlappingRoots(t *testing.T) {
	body, err := os.ReadFile("testdata/persistent-tool/uv-installed.toml")
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := ParseUVToolReceipt(body)
	if err != nil {
		t.Fatal(err)
	}
	for _, paths := range [][2]string{{"/tools/wrong", "/bin"}, {"/tools/haa-tool-fixture", "/tools/haa-tool-fixture/bin"}, {"/tools/haa-tool-fixture", "/tools"}, {"/", "/bin"}, {"/tools/haa-tool-fixture", "/bin/../bin"}, {"/tools/haa-tool-fixture", "/bin\n"}} {
		if result, err := receipt.WithUVLayout(paths[0], paths[1]); err == nil || result.Valid() {
			t.Fatalf("unsafe layout accepted: %q", paths)
		}
	}
	if result, err := (ToolReceipt{}).WithUVLayout("/tools/haa-tool-fixture", "/bin"); err == nil || result.Valid() {
		t.Fatal("empty receipt accepted")
	}
	pipxBody, err := os.ReadFile("testdata/persistent-tool/pipx-installed.json")
	if err != nil {
		t.Fatal(err)
	}
	// Project the historical format onto the current runtime; keep captured testdata unchanged.
	pipxBody = bytes.ReplaceAll(pipxBody, []byte("Python 3.14.7"), []byte("Python "+runtimeidentity.PythonVersion))
	pipx, err := ParsePipxToolReceipt(pipxBody)
	if err != nil {
		t.Fatal(err)
	}
	if result, err := pipx.WithUVLayout("/tools/haa-tool-fixture", "/bin"); err == nil || result.Valid() {
		t.Fatal("pipx metadata projected as UV")
	}
}

func TestUVToolSelectionLayoutNeverRetainsOriginalInterpreterOrExposure(t *testing.T) {
	body, err := os.ReadFile("testdata/persistent-tool/uv-installed.toml")
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := ParseUVToolReceipt(body)
	if err != nil {
		t.Fatal(err)
	}
	host, err := receipt.WithUVLayout("/user/tools/haa-tool-fixture", "/user/commands")
	if err != nil {
		t.Fatal(err)
	}
	environment, bin := "/haa-site/tools/haa-tool-fixture", "/haa-site/tool-bin"
	originalBody := host.Body()
	selected, err := host.WithUVSelectionLayout(environment, bin)
	if err != nil {
		t.Fatal(err)
	}
	if err := selected.ValidateLayout(environment, bin, "/usr/local/bin/python"); err != nil {
		t.Fatal(err)
	}
	if !selected.Requirement().SameConstraints(host.Requirement()) || !bytes.Equal(host.Body(), originalBody) {
		t.Fatal("original semantics changed")
	}
	var before, after uvToolReceiptWire
	if toml.Unmarshal(host.Body(), &before) != nil || toml.Unmarshal(selected.Body(), &after) != nil {
		t.Fatal("receipt wire")
	}
	after.Tool.Python = before.Tool.Python
	for i := range after.Tool.Entrypoints {
		if after.Tool.Entrypoints[i].Path == before.Tool.Entrypoints[i].Path {
			t.Fatal("original command exposure retained")
		}
		after.Tool.Entrypoints[i].Path = before.Tool.Entrypoints[i].Path
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("selection projection changed original recipe or commands")
	}
	repeated, err := selected.WithUVSelectionLayout(environment, bin)
	if err != nil || !bytes.Equal(repeated.Body(), selected.Body()) {
		t.Fatal("selection layout is not stable", err)
	}
}
