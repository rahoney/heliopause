package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestPipxRuntimeSourceOwnerRequiresEveryMetadataInput(t *testing.T) {
	for _, mode := range []string{"valid", "missing", "changed", "symlink", "input-count", "duplicate", "notice-size", "python-owner"} {
		t.Run(mode, func(t *testing.T) {
			root := runtimeOwnerFixture(t)
			body, err := os.ReadFile(filepath.Join(root, "scripts/runtimes.lock.json"))
			if err != nil {
				t.Fatal(err)
			}
			var lock map[string]any
			if json.Unmarshal(body, &lock) != nil {
				t.Fatal("lock")
			}
			pin := lock["pipx_tool"].(map[string]any)
			path := filepath.Join(root, "tools/pipx/supply/pipx/provenance.json")
			switch mode {
			case "missing":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "changed":
				if err := os.WriteFile(path, []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				other := filepath.Join(t.TempDir(), "proof")
				original, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(other, original, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(other, path); err != nil {
					t.Fatal(err)
				}
			case "input-count":
				pin["inputs"] = pin["inputs"].([]any)[:26]
			case "duplicate":
				inputs := pin["inputs"].([]any)
				inputs[1] = inputs[0]
			case "notice-size":
				pin["notices"].(map[string]any)["size"] = float64(1)
			case "python-owner":
				pin["runtime_image"] = "node_image"
			}
			err = checkPipxProofFiles(root, lock)
			if (err == nil) != (mode == "valid") {
				t.Fatalf("unexpected drift outcome: %v", err)
			}
		})
	}
}

func TestRuntimeOwnerKeyReferenceIsNotASelectedIdentity(t *testing.T) {
	value := map[string]any{"pipx_tool": map[string]any{"runtime_image": "python_image", "other": "python_image"}, "python_image": map[string]any{"reference": "python:1.2.3@sha256:reviewed"}}
	values := runtimeLockStrings(value)
	if len(values) != 2 || !slices.Contains(values, "python_image") || !slices.Contains(values, "python:1.2.3@sha256:reviewed") {
		t.Fatal("owner reference exclusion affected another field", values)
	}
	value = map[string]any{"pipx_tool": map[string]any{"runtime_image": "foreign_runtime"}}
	if !slices.Contains(runtimeLockStrings(value), "foreign_runtime") {
		t.Fatal("unknown reference bypassed literal guard")
	}
}
