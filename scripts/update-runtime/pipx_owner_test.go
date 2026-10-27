package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPipxOwnerRejectsAmbiguousAndIncompletePins(t *testing.T) {
	root, _ := fixture(t)
	filename := filepath.Join(root, runtimeFile)
	original, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readPipxMaterializationOwner(root); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"old-schema", "missing", "unknown", "case", "duplicate", "architecture", "runtime", "input-count", "input-duplicate", "input-path", "input-hash", "zip-size", "zip-path", "notices-path", "version"} {
		t.Run(name, func(t *testing.T) {
			var lock map[string]any
			if err := json.Unmarshal(original, &lock); err != nil {
				t.Fatal(err)
			}
			pin := lock["pipx_tool"].(map[string]any)
			inputs := pin["inputs"].([]any)
			switch name {
			case "old-schema":
				lock["schema_version"] = 4
			case "missing":
				delete(pin, "tool")
			case "unknown":
				pin["trusted"] = true
			case "case":
				pin["Runtime_image"] = pin["runtime_image"]
				delete(pin, "runtime_image")
			case "architecture":
				pin["architecture"] = "arm64"
			case "runtime":
				pin["runtime_image"] = "node_image"
			case "input-count":
				pin["inputs"] = inputs[:26]
			case "input-duplicate":
				inputs[1] = inputs[0]
			case "input-path":
				inputs[0].(map[string]any)["path"] = "../escape"
			case "input-hash":
				inputs[0].(map[string]any)["sha256"] = "invalid"
			case "zip-size":
				pin["tool"].(map[string]any)["size"] = 9 << 20
			case "zip-path":
				pin["tool"].(map[string]any)["path"] = "other.zip"
			case "notices-path":
				pin["notices"].(map[string]any)["path"] = "tools/other/NOTICE"
			case "version":
				pin["pipx_version"] = "0.0.1"
			}
			body, err := json.Marshal(lock)
			if err != nil {
				t.Fatal(err)
			}
			if name == "duplicate" {
				body = bytes.Replace(body, []byte(`"pipx_version":`), []byte(`"pipx_version":"0.0.1","pipx_version":`), 1)
			}
			writeTest(t, filename, body)
			if _, err := readPipxMaterializationOwner(root); err == nil {
				t.Fatal("invalid owner accepted")
			}
		})
	}
}

func TestPipxProductionMaterializerRefusesIncompleteInputWithoutOutput(t *testing.T) {
	root, _ := fixture(t)
	parent := t.TempDir()
	input := t.TempDir()
	var output bytes.Buffer
	err := materializePipx([]string{"--root", root, "--input", input, "--output", filepath.Join(parent, "pipx-runtime-linux-amd64.zip"), "--notices-output", filepath.Join(parent, "pipx-NOTICE.txt")}, &output)
	if err == nil || output.Len() != 0 {
		t.Fatal("incomplete input produced a release candidate")
	}
	files, err := os.ReadDir(parent)
	if err != nil || len(files) != 0 {
		t.Fatal("failed materializer wrote output")
	}
}

func TestCurrentNativeOwnerSchemaConsumers(t *testing.T) {
	root, _ := fixture(t)
	if _, err := readPNPMMaterializationOwner(root); err != nil {
		t.Fatal("pnpm current owner", err)
	}
	if _, err := readBunMaterializationOwner(root); err != nil {
		t.Fatal("Bun current owner", err)
	}
	if _, err := readPipxMaterializationOwner(root); err != nil {
		t.Fatal("pipx current owner", err)
	}
	filename := filepath.Join(root, runtimeFile)
	body, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	body = bytes.Replace(body, []byte(`"schema_version": 5`), []byte(`"schema_version": 4`), 1)
	writeTest(t, filename, body)
	if _, err := readPNPMMaterializationOwner(root); err == nil {
		t.Fatal("pnpm accepted retired schema")
	}
	if _, err := readBunMaterializationOwner(root); err == nil {
		t.Fatal("Bun accepted retired schema")
	}
	if _, err := readPipxMaterializationOwner(root); err == nil {
		t.Fatal("pipx accepted retired schema")
	}
}
