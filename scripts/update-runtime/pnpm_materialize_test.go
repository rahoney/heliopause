package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPNPMNativePublicationRefusesReplacementAndUnsafeParents(t *testing.T) {
	for _, mode := range []string{"existing", "symlink", "symlink-parent", "writable-parent", "missing-parent"} {
		t.Run(mode, func(t *testing.T) {
			parent := t.TempDir()
			output := filepath.Join(parent, "pnpm-linux-amd64")
			foreign := filepath.Join(t.TempDir(), "foreign")
			writeTest(t, foreign, []byte("foreign"))
			switch mode {
			case "existing":
				writeTest(t, output, []byte("existing"))
			case "symlink":
				if err := os.Symlink(foreign, output); err != nil {
					t.Fatal(err)
				}
			case "symlink-parent":
				link := filepath.Join(t.TempDir(), "linked")
				if err := os.Symlink(parent, link); err != nil {
					t.Fatal(err)
				}
				output = filepath.Join(link, "pnpm-linux-amd64")
			case "writable-parent":
				if err := os.Chmod(parent, 0777); err != nil {
					t.Fatal(err)
				}
			case "missing-parent":
				output = filepath.Join(parent, "missing", "pnpm-linux-amd64")
			}
			if err := publishPNPMAsset(output, []byte("verified-native-fixture"), 0555); err == nil {
				t.Fatal("unsafe publication succeeded")
			}
			body, err := os.ReadFile(foreign)
			if err != nil || string(body) != "foreign" {
				t.Fatal("foreign file changed")
			}
			entries, err := os.ReadDir(parent)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if entry.Name() != "pnpm-linux-amd64" {
					t.Fatalf("temporary payload leaked: %s", entry.Name())
				}
			}
			if mode == "existing" {
				body, err := os.ReadFile(output)
				if err != nil || string(body) != "existing" {
					t.Fatal("existing output was overwritten")
				}
			}
		})
	}
}

func TestPNPMMaterializationOwnerRejectsAmbiguousOrUnboundedPins(t *testing.T) {
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
			pin := lock["pnpm_tool"].(map[string]any)
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
				pin["provenance"].(map[string]any)["bundle_path"] = "../foreign"
			case "native-path":
				pin["tool"].(map[string]any)["path"] = "dist/other"
			}
			body, err := json.Marshal(lock)
			if err != nil {
				t.Fatal(err)
			}
			if name == "duplicate" {
				body = bytes.Replace(body, []byte(`"pnpm_version":`), []byte(`"pnpm_version":"0.0.1","pnpm_version":`), 1)
			}
			writeTest(t, path, body)
			if _, err := readPNPMMaterializationOwner(root); err == nil {
				t.Fatal("invalid selected owner accepted")
			}
		})
	}
}

func TestPNPMMaterializationRejectsInvalidPathsBeforeWriting(t *testing.T) {
	root, _ := fixture(t)
	output := filepath.Join(t.TempDir(), "pnpm-linux-amd64")
	archive := filepath.Join(t.TempDir(), "pnpm-linux-x64.tar.gz")
	for _, mode := range []string{"relative-root", "relative-archive", "relative-output", "source-output", "wrong-name", "extra-arg", "owner-drift"} {
		t.Run(mode, func(t *testing.T) {
			args := []string{"materialize-pnpm", "--root", root, "--archive", archive, "--output", output, "--notices-output", filepath.Join(filepath.Dir(output), "pnpm-NOTICE.txt")}
			switch mode {
			case "relative-root":
				args[2] = "."
			case "relative-archive":
				args[4] = "archive.tar.gz"
			case "relative-output":
				args[6] = "pnpm-linux-amd64"
			case "source-output":
				args[6] = filepath.Join(root, "pnpm-linux-amd64")
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
