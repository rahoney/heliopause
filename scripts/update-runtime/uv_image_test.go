package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func uvProofDirectory(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "internal", "verification", "releaseprovenance", "testdata", "uv-0.13.4"))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestOfflineUVImageSupplyCommand(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"verify-uv-image", "--input", uvProofDirectory(t)}, &out); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Schema   string
		Identity struct{ Version string }
		Layers   []struct{ Size int64 }
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Schema != "helox.uv-image-supply/v1" || result.Identity.Version != "0.13.4" || len(result.Layers) != 2 || result.Layers[0].Size != 19395988 {
		t.Fatalf("unexpected result: %s", out.String())
	}
}

func TestOfflineUVImageSupplyRejectsIncompleteAndChangedInputs(t *testing.T) {
	for _, name := range []string{"relative", "missing-file", "symlink-file", "symlink-directory", "oversize-identity", "unknown-identity", "changed-config", "trailing-identity"} {
		t.Run(name, func(t *testing.T) {
			input := t.TempDir()
			for _, file := range []string{"identity.json", "bundle.json", "index.json", "manifest.json", "config.json"} {
				body, err := os.ReadFile(filepath.Join(uvProofDirectory(t), file))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(input, file), body, 0600); err != nil {
					t.Fatal(err)
				}
			}
			var err error
			switch name {
			case "relative":
				input = "."
			case "missing-file":
				err = os.Remove(filepath.Join(input, "manifest.json"))
			case "symlink-file":
				err = os.Remove(filepath.Join(input, "config.json"))
				if err == nil {
					err = os.Symlink(filepath.Join(uvProofDirectory(t), "config.json"), filepath.Join(input, "config.json"))
				}
			case "symlink-directory":
				link := filepath.Join(t.TempDir(), "input")
				err = os.Symlink(input, link)
				input = link
			case "oversize-identity":
				err = os.WriteFile(filepath.Join(input, "identity.json"), []byte(strings.Repeat("x", 4097)), 0600)
			case "unknown-identity":
				err = os.WriteFile(filepath.Join(input, "identity.json"), []byte(`{"unknown":true}`), 0600)
			case "changed-config":
				err = os.WriteFile(filepath.Join(input, "config.json"), []byte(`{}`), 0600)
			case "trailing-identity":
				body, readErr := os.ReadFile(filepath.Join(input, "identity.json"))
				if readErr != nil {
					t.Fatal(readErr)
				}
				err = os.WriteFile(filepath.Join(input, "identity.json"), append(body, []byte("\n{}")...), 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err := run([]string{"verify-uv-image", "--input", input}, &out); err == nil || out.Len() != 0 {
				t.Fatalf("invalid proof returned receipt: %v %s", err, out.String())
			}
		})
	}
}
