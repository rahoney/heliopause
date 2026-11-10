package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBunMaterializationOwnerRejectsAmbiguousOrUnboundedPins(t *testing.T) {
	root, _ := fixture(t)
	path := filepath.Join(root, runtimeFile)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"schema", "missing", "unknown", "case-alias", "duplicate", "architecture", "archive-url", "archive-size", "tool-size", "proof-path", "native-path"} {
		t.Run(name, func(t *testing.T) {
			var lock map[string]any
			if err := json.Unmarshal(original, &lock); err != nil {
				t.Fatal(err)
			}
			pin := lock["bun_tool"].(map[string]any)
			switch name {
			case "schema":
				lock["schema_version"] = 2
			case "missing":
				delete(pin, "tool")
			case "unknown":
				pin["unexpected"] = true
			case "case-alias":
				pin["Architecture"] = "amd64"
				delete(pin, "architecture")
			case "architecture":
				pin["architecture"] = "arm64"
			case "archive-url":
				pin["archive"].(map[string]any)["url"] = "https://foreign.invalid/archive"
			case "archive-size":
				pin["archive"].(map[string]any)["size"] = 1 << 30
			case "tool-size":
				pin["tool"].(map[string]any)["size"] = 0
			case "proof-path":
				pin["publisher"].(map[string]any)["checksums_path"] = "../foreign"
			case "native-path":
				pin["tool"].(map[string]any)["path"] = "dist/other"
			}
			body, err := json.Marshal(lock)
			if err != nil {
				t.Fatal(err)
			}
			if name == "duplicate" {
				body = bytes.Replace(body, []byte(`"bun_version":`), []byte(`"bun_version":"0.0.1","bun_version":`), 1)
			}
			writeTest(t, path, body)
			if _, err := readBunMaterializationOwner(root); err == nil {
				t.Fatal("invalid selected owner accepted")
			}
		})
	}
}

func TestBunMaterializationRejectsInvalidPathsBeforeWriting(t *testing.T) {
	root, _ := fixture(t)
	output := filepath.Join(t.TempDir(), "bun-linux-amd64")
	archive := filepath.Join(t.TempDir(), "bun-linux-x64.tar.gz")
	for _, mode := range []string{"relative-root", "relative-archive", "relative-output", "source-output", "wrong-name", "extra-arg", "owner-drift"} {
		t.Run(mode, func(t *testing.T) {
			args := []string{"materialize-bun", "--root", root, "--archive", archive, "--output", output, "--notices-output", filepath.Join(filepath.Dir(output), "bun-NOTICE.txt")}
			switch mode {
			case "relative-root":
				args[2] = "."
			case "relative-archive":
				args[4] = "archive.tar.gz"
			case "relative-output":
				args[6] = "bun-linux-amd64"
			case "source-output":
				args[6] = filepath.Join(root, "bun-linux-amd64")
			case "wrong-name":
				args[6] = filepath.Join(filepath.Dir(output), "wrong-name")
			case "extra-arg":
				args = append(args, "unexpected")
			case "owner-drift":
				writeTest(t, filepath.Join(root, runtimeFile), []byte("changed owner"))
			}
			var report bytes.Buffer
			if err := run(args, &report); err == nil || report.Len() != 0 {
				t.Fatalf("invalid materialization returned success: %v %s", err, report.String())
			}
			if _, err := os.Lstat(output); !os.IsNotExist(err) {
				t.Fatal("invalid materialization wrote output")
			}
		})
	}
}
