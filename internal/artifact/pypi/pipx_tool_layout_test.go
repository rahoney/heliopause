package pypi

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestPipxLayoutPreservesActualNativeRecipeAndSeparateInterpreter(t *testing.T) {
	for _, fixture := range []string{"pipx-installed.json", "pipx-upgraded.json"} {
		t.Run(fixture, func(t *testing.T) {
			body := nativeToolFixture(t, fixture)
			receipt, err := ParsePipxToolReceipt(body)
			if err != nil {
				t.Fatal(err)
			}
			environment, bin, python := "/user/tools with spaces/haa-tool-fixture", "/user/exposed commands", "/held/runtime/python"
			projected, err := receipt.WithPipxLayout(environment, bin, python)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(receipt.Body(), body) || projected.InterpreterPath() != python || projected.Version() != receipt.Version() || !projected.Requirement().SameConstraints(receipt.Requirement()) {
				t.Fatal("original receipt, source interpreter or native selection changed")
			}
			links := map[string]string{}
			for i, c := range projected.Commands() {
				if c.Name != receipt.Commands()[i].Name || c.Path != environment+"/bin/"+c.Name {
					t.Fatal("retired/new commands or internal script location changed")
				}
				links[bin+"/"+c.Name] = c.Path
			}
			if err := projected.ValidateExposedCommands(environment, bin, python, links); err != nil {
				t.Fatal(err)
			}
			links[bin+"/foreign"] = environment + "/bin/foreign"
			if projected.ValidateExposedCommands(environment, bin, python, links) == nil {
				t.Fatal("projection granted unclaimed external exposure")
			}
			if projected.ValidateLayout(environment, bin, environment+"/bin/python") == nil {
				t.Fatal("source interpreter confused with environment interpreter")
			}
			var before, after map[string]any
			if err := json.Unmarshal(body, &before); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(projected.Body(), &after); err != nil {
				t.Fatal(err)
			}
			after["source_interpreter"] = before["source_interpreter"]
			after["main_package"].(map[string]any)["app_paths"] = before["main_package"].(map[string]any)["app_paths"]
			if !reflect.DeepEqual(before, after) {
				t.Fatal("backend, options, requirement, version or exposure semantics changed")
			}
			upgrade, err := projected.UpgradeRequirement()
			if err != nil || upgrade.Requirement() != "haa-tool-fixture" || !upgrade.AcceptsVersion("2.0.0") {
				t.Fatal("native unconstrained pipx upgrade semantics changed", err)
			}
		})
	}
}

func TestPipxLayoutPreservesOmittedAndNullOpaqueFields(t *testing.T) {
	for _, omit := range []bool{false, true} {
		var wire map[string]any
		if err := json.Unmarshal(nativeToolFixture(t, "pipx-installed.json"), &wire); err != nil {
			t.Fatal(err)
		}
		main := wire["main_package"].(map[string]any)
		// Parsing an opaque recipe is not permission to replay it. Projection
		// keeps these inert bytes; the fixed resolver must never execute them.
		main["pip_args"] = []any{"--index-url=https://unapproved.invalid/simple/", "--find-links=/host/private"}
		for _, key := range []string{"cooldown_days", "lock_file", "completion_paths"} {
			main[key] = nil
			if omit {
				delete(main, key)
			}
		}
		body, err := json.Marshal(wire)
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := ParsePipxToolReceipt(body)
		if err != nil {
			t.Fatal(err)
		}
		projected, err := receipt.WithPipxLayout("/tools/haa-tool-fixture", "/commands", "/fixed/python")
		if err != nil {
			t.Fatal(err)
		}
		var after map[string]any
		if err := json.Unmarshal(projected.Body(), &after); err != nil {
			t.Fatal(err)
		}
		afterMain := after["main_package"].(map[string]any)
		if !reflect.DeepEqual(afterMain["pip_args"], main["pip_args"]) {
			t.Fatal("opaque recipe was rewritten")
		}
		for _, key := range []string{"cooldown_days", "lock_file", "completion_paths"} {
			value, present := afterMain[key]
			if present == omit || value != nil {
				t.Fatal("omission/null changed", key)
			}
		}
	}
}

func TestPipxLayoutRejectsWrongClientChangedClaimsAndUnsafeRoots(t *testing.T) {
	receipt, err := ParsePipxToolReceipt(nativeToolFixture(t, "pipx-installed.json"))
	if err != nil {
		t.Fatal(err)
	}
	uv, err := ParseUVToolReceipt(nativeToolFixture(t, "uv-installed.toml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []ToolReceipt{{}, uv} {
		if result, err := r.WithPipxLayout("/tools/haa-tool-fixture", "/commands", "/fixed/python"); err == nil || result.Valid() {
			t.Fatal("wrong receipt accepted")
		}
	}
	for _, paths := range [][3]string{
		{"/tools/wrong", "/commands", "/fixed/python"},
		{"/tools/haa-tool-fixture", "/tools/haa-tool-fixture/bin", "/fixed/python"},
		{"/tools/haa-tool-fixture", "/tools", "/fixed/python"},
		{"/tools/haa-tool-fixture", "/commands/../commands", "/fixed/python"},
		{"/tools/haa-tool-fixture", "/commands", "/"},
		{"/tools/haa-tool-fixture", "/commands", "relative/python"},
		{"/tools/haa-tool-fixture", "/commands", "/fixed/../python"},
		{"/tools/haa-tool-fixture", "/commands", "/fixed/python\n"},
	} {
		if result, err := receipt.WithPipxLayout(paths[0], paths[1], paths[2]); err == nil || result.Valid() {
			t.Fatal("unsafe or mismatched layout accepted", paths)
		}
	}
	for _, field := range []string{"version", "interpreter", "environment", "commands", "body"} {
		changed := receipt
		switch field {
		case "version":
			changed.version = "9.9.9"
		case "interpreter":
			changed.interpreter = "/foreign/python"
		case "environment":
			changed.environment = "foreign"
		case "commands":
			changed.commands = []ToolCommand{{Name: "foreign", Path: "/foreign"}}
		case "body":
			changed.body = []byte("{}")
		}
		if result, err := changed.WithPipxLayout("/tools/haa-tool-fixture", "/commands", "/fixed/python"); err == nil || result.Valid() {
			t.Fatal("changed cached receipt claims accepted", field)
		}
	}
}
