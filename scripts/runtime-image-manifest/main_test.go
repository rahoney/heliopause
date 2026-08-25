package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildManifestBindsOfficialImmutableImages(t *testing.T) {
	lock := runtimeLock{SchemaVersion: 1}
	lock.NodeImage.Reference = "node:22.23.1-slim@sha256:" + strings.Repeat("a", 64)
	lock.NodeImage.NPMVersion = "10.9.8"
	lock.PythonImage.Reference = "python:3.14.7-slim-bookworm@sha256:" + strings.Repeat("b", 64)
	lock.PythonImage.PythonVersion = "3.14.7"
	lock.PythonImage.PipVersion = "26.2.1"
	manifest, err := buildManifest(lock)
	if err != nil {
		t.Fatalf("buildManifest: %v", err)
	}
	if manifest.Schema != manifestSchema || manifest.CustomGHCRImage || !manifest.Provenance.Required || len(manifest.Images) != 2 {
		t.Fatalf("manifest = %#v", manifest)
	}
	if manifest.Images[0].Repository != "docker.io/library/node" || manifest.Images[1].Repository != "docker.io/library/python" {
		t.Fatalf("image repositories = %#v", manifest.Images)
	}
}

func TestRunRejectsMutableOrNonOfficialImages(t *testing.T) {
	root := t.TempDir()
	lockPath := filepath.Join(root, "runtimes.lock.json")
	outputPath := filepath.Join(root, "runtime-images.json")
	body := []byte(`{"schema_version":1,"node_image":{"reference":"node:latest","npm_version":"10.9.8"},"python_image":{"reference":"python:3.14.7-slim-bookworm@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","python_version":"3.14.7","pip_version":"26.2.1"}}`)
	if err := os.WriteFile(lockPath, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--runtime-lock", lockPath, "--output", outputPath}); err == nil {
		t.Fatal("mutable runtime image was accepted")
	}
}

func TestRunWritesNoOverwriteManifest(t *testing.T) {
	root := t.TempDir()
	lockPath := filepath.Join(root, "runtimes.lock.json")
	outputPath := filepath.Join(root, "runtime-images.json")
	body := []byte(`{"schema_version":1,"node_image":{"reference":"node:22.23.1-slim@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","npm_version":"10.9.8"},"python_image":{"reference":"python:3.14.7-slim-bookworm@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","python_version":"3.14.7","pip_version":"26.2.1"}}`)
	if err := os.WriteFile(lockPath, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--runtime-lock", lockPath, "--output", outputPath}); err != nil {
		t.Fatalf("run: %v", err)
	}
	var manifest runtimeImageManifest
	encoded, err := os.ReadFile(outputPath)
	if err != nil || json.Unmarshal(encoded, &manifest) != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if manifest.Schema != manifestSchema || len(manifest.Images) != 2 {
		t.Fatalf("manifest = %#v", manifest)
	}
	if err := run([]string{"--runtime-lock", lockPath, "--output", outputPath}); err == nil {
		t.Fatal("preexisting output was overwritten")
	}
}
