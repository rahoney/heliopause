package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildManifestBindsOfficialImmutableImages(t *testing.T) {
	lock := runtimeLock{SchemaVersion: 5}
	bindSyntheticUV(&lock)
	bindSyntheticPNPM(&lock)
	bindSyntheticBun(&lock)
	bindSyntheticPipx(&lock)
	lock.RustImage.Reference = "rust:1.99.0-bookworm@sha256:" + strings.Repeat("d", 64)
	lock.RustImage.RustVersion = "1.99.0"
	lock.RustImage.Architecture = "amd64"
	lock.RustImage.Target = "x86_64-unknown-linux-gnu"
	lock.GoImage.Reference = "golang:1.26.8-bookworm@sha256:" + strings.Repeat("c", 64)
	lock.GoImage.GoVersion = "1.26.8"
	lock.GoImage.Architecture = "amd64"
	lock.NodeImage.Reference = "node:22.23.1-slim@sha256:" + strings.Repeat("a", 64)
	lock.NodeImage.NPMVersion = "10.9.8"
	lock.PythonImage.Reference = "python:3.14.7-slim-bookworm@sha256:" + strings.Repeat("b", 64)
	lock.PythonImage.PythonVersion = "3.14.7"
	lock.PythonImage.PipVersion = "26.2.1"
	manifest, err := buildManifest(lock)
	if err != nil {
		t.Fatalf("buildManifest: %v", err)
	}
	if manifest.Schema != manifestSchema || manifest.CustomGHCRImage || !manifest.Provenance.Required || len(manifest.Images) != 5 {
		t.Fatalf("manifest = %#v", manifest)
	}
	if manifest.Images[0].Repository != "docker.io/library/node" || manifest.Images[1].Repository != "docker.io/library/python" || manifest.Images[2].Repository != "docker.io/library/golang" || manifest.Images[3].Repository != "docker.io/library/rust" {
		t.Fatalf("image repositories = %#v", manifest.Images)
	}
	if manifest.Images[0].Version != "22.23.1" || manifest.Images[0].Version == lock.NodeImage.NPMVersion {
		t.Fatal("Node manifest version must identify Node, not npm")
	}
}

func TestRunRejectsMutableOrNonOfficialImages(t *testing.T) {
	root := t.TempDir()
	lockPath := filepath.Join(root, "runtimes.lock.json")
	outputPath := filepath.Join(root, "runtime-images.json")
	body := []byte(`{"schema_version":5,"go_image":{"reference":"golang:1.26.8-bookworm@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","go_version":"1.26.8","architecture":"amd64"},"node_image":{"reference":"node:latest","npm_version":"10.9.8"},"python_image":{"reference":"python:3.14.7-slim-bookworm@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","python_version":"3.14.7","pip_version":"26.2.1"},"rust_image":{"reference":"rust:1.99.0-bookworm@sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","rust_version":"1.99.0","architecture":"amd64","target":"x86_64-unknown-linux-gnu"}}`)
	var input runtimeLock
	if err := json.Unmarshal(body, &input); err != nil {
		t.Fatal(err)
	}
	bindSyntheticUV(&input)
	bindSyntheticPNPM(&input)
	bindSyntheticBun(&input)
	bindSyntheticPipx(&input)
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
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
	body := []byte(`{"schema_version":5,"go_image":{"reference":"golang:1.26.8-bookworm@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","go_version":"1.26.8","architecture":"amd64"},"node_image":{"reference":"node:22.23.1-slim@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","npm_version":"10.9.8"},"python_image":{"reference":"python:3.14.7-slim-bookworm@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","python_version":"3.14.7","pip_version":"26.2.1"},"rust_image":{"reference":"rust:1.99.0-bookworm@sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","rust_version":"1.99.0","architecture":"amd64","target":"x86_64-unknown-linux-gnu"}}`)
	var input runtimeLock
	if err := json.Unmarshal(body, &input); err != nil {
		t.Fatal(err)
	}
	bindSyntheticUV(&input)
	bindSyntheticPNPM(&input)
	bindSyntheticBun(&input)
	bindSyntheticPipx(&input)
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
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
	if manifest.Schema != manifestSchema || len(manifest.Images) != 5 {
		t.Fatalf("manifest = %#v", manifest)
	}
	if err := run([]string{"--runtime-lock", lockPath, "--output", outputPath}); err == nil {
		t.Fatal("preexisting output was overwritten")
	}
}

func TestManifestRejectsMissingOrSubstitutedGoImage(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "runtimes.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pinned runtimeLock
	if err := json.Unmarshal(body, &pinned); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*runtimeLock){
		func(lock *runtimeLock) { lock.GoImage.Reference = "" },
		func(lock *runtimeLock) { lock.GoImage.Reference = "golang:latest" },
		func(lock *runtimeLock) { lock.GoImage.Architecture = "arm64" },
		func(lock *runtimeLock) { lock.GoImage.GoVersion = "1.26.7" },
		func(lock *runtimeLock) {
			lock.GoImage.Reference = strings.Replace(lock.GoImage.Reference, "golang:", "untrusted:", 1)
		},
	} {
		lock := pinned
		mutate(&lock)
		if _, err := buildManifest(lock); err == nil {
			t.Fatal("unverified Go image accepted")
		}
	}
}

func TestManifestRejectsMissingOrMismatchedRuntimeBindings(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "runtimes.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pinned runtimeLock
	if err := json.Unmarshal(body, &pinned); err != nil {
		t.Fatal(err)
	}
	nodeVersion := strings.TrimPrefix(strings.SplitN(pinned.NodeImage.Reference, "-slim@", 2)[0], "node:")
	for name, mutate := range map[string]func(*runtimeLock){
		"missing Rust":             func(lock *runtimeLock) { lock.RustImage.Reference = "" },
		"substituted Rust version": func(lock *runtimeLock) { lock.RustImage.RustVersion = "1.0.0" },
		"wrong Rust architecture":  func(lock *runtimeLock) { lock.RustImage.Architecture = "arm64" },
		"wrong Rust target":        func(lock *runtimeLock) { lock.RustImage.Target = "aarch64-unknown-linux-gnu" },
		"Node npm used as image tag": func(lock *runtimeLock) {
			lock.NodeImage.Reference = strings.Replace(lock.NodeImage.Reference, nodeVersion, lock.NodeImage.NPMVersion, 1)
			lock.NodeImage.NPMVersion = "latest"
		},
		"Python version mismatches tag": func(lock *runtimeLock) { lock.PythonImage.PythonVersion = "3.13.0" },
		"Python tag adds repository path": func(lock *runtimeLock) {
			lock.PythonImage.Reference = strings.Replace(lock.PythonImage.Reference, "@", "/extra@", 1)
		},
		"uppercase digest": func(lock *runtimeLock) { lock.RustImage.Reference = strings.ToUpper(lock.RustImage.Reference) },
	} {
		t.Run(name, func(t *testing.T) {
			lock := pinned
			mutate(&lock)
			if _, err := buildManifest(lock); err == nil {
				t.Fatal("invalid runtime binding accepted")
			}
		})
	}
	manifest, err := buildManifest(pinned)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Images) != 5 || manifest.Images[0].Version != nodeVersion || manifest.Images[0].Version == pinned.NodeImage.NPMVersion || manifest.Images[3].Version != pinned.RustImage.RustVersion {
		t.Fatalf("runtime versions: %#v", manifest.Images)
	}
}

func bindSyntheticUV(lock *runtimeLock) {
	lock.UVImage.Reference = "ghcr.io/astral-sh/uv:0.1.2@sha256:" + strings.Repeat("e", 64)
	lock.UVImage.Version = "0.1.2"
	lock.UVImage.Architecture = "amd64"
}

func TestManifestRejectsOldSchemaAndMissingUV(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "runtimes.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pinned runtimeLock
	if err := json.Unmarshal(body, &pinned); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*runtimeLock){
		"old schema": func(l *runtimeLock) { l.SchemaVersion = 1 },
		"missing uv": func(l *runtimeLock) { l.UVImage.Reference = "" },
		"wrong publisher": func(l *runtimeLock) {
			l.UVImage.Reference = strings.Replace(l.UVImage.Reference, "astral-sh", "foreign", 1)
		},
		"wrong uv version":      func(l *runtimeLock) { l.UVImage.Version = "0.0.1" },
		"wrong uv architecture": func(l *runtimeLock) { l.UVImage.Architecture = "arm64" },
	} {
		t.Run(name, func(t *testing.T) {
			l := pinned
			mutate(&l)
			if _, err := buildManifest(l); err == nil {
				t.Fatal("unbound UV supply accepted")
			}
		})
	}
}

func bindSyntheticPNPM(lock *runtimeLock) {
	lock.PNPMTool.Version = "1.2.3"
	lock.PNPMTool.Architecture = "amd64"
	lock.PNPMTool.Tool.SHA256 = strings.Repeat("e", 64)
	lock.PNPMTool.Tool.Size = 768
	lock.PNPMTool.Notices.SHA256 = strings.Repeat("f", 64)
	lock.PNPMTool.Notices.Size = 1024
}

func TestManifestRequiresSeparateBoundedPNPMTool(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "runtimes.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pinned runtimeLock
	if err := json.Unmarshal(body, &pinned); err != nil {
		t.Fatal(err)
	}
	manifest, err := buildManifest(pinned)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Images) != 5 || len(manifest.Tools) != 3 || manifest.Tools[0].Name != "pnpm" || manifest.Tools[0].SHA256 != pinned.PNPMTool.Tool.SHA256 {
		t.Fatalf("native tool/image identity mixed: %+v", manifest)
	}
	for name, mutate := range map[string]func(*runtimeLock){
		"missing":      func(l *runtimeLock) { l.PNPMTool.Version = "" },
		"architecture": func(l *runtimeLock) { l.PNPMTool.Architecture = "arm64" },
		"hash":         func(l *runtimeLock) { l.PNPMTool.Tool.SHA256 = "" },
		"size":         func(l *runtimeLock) { l.PNPMTool.Tool.Size = 64<<20 + 1 },
	} {
		t.Run(name, func(t *testing.T) {
			l := pinned
			mutate(&l)
			if _, err := buildManifest(l); err == nil {
				t.Fatal("unbound native tool accepted")
			}
		})
	}
}

func bindSyntheticBun(lock *runtimeLock) {
	lock.BunTool.Version = "1.2.3"
	lock.BunTool.Architecture = "amd64"
	lock.BunTool.Tool.SHA256 = strings.Repeat("e", 64)
	lock.BunTool.Tool.Size = 768
	lock.BunTool.Notices.SHA256 = strings.Repeat("f", 64)
	lock.BunTool.Notices.Size = 1024
}

func TestManifestRequiresSeparateBoundedBunTool(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "runtimes.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pinned runtimeLock
	if err := json.Unmarshal(body, &pinned); err != nil {
		t.Fatal(err)
	}
	manifest, err := buildManifest(pinned)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Images) != 5 || len(manifest.Tools) != 3 || manifest.Tools[1].Name != "bun" || manifest.Tools[1].SHA256 != pinned.BunTool.Tool.SHA256 {
		t.Fatalf("native tool/image identity mixed: %+v", manifest)
	}
	for name, mutate := range map[string]func(*runtimeLock){
		"missing":      func(l *runtimeLock) { l.BunTool.Version = "" },
		"architecture": func(l *runtimeLock) { l.BunTool.Architecture = "arm64" },
		"hash":         func(l *runtimeLock) { l.BunTool.Tool.SHA256 = "" },
		"size":         func(l *runtimeLock) { l.BunTool.Tool.Size = 128<<20 + 1 },
	} {
		t.Run(name, func(t *testing.T) {
			l := pinned
			mutate(&l)
			if _, err := buildManifest(l); err == nil {
				t.Fatal("unbound native tool accepted")
			}
		})
	}
}

func bindSyntheticPipx(lock *runtimeLock) {
	lock.PipxTool.Version = "1.2.3"
	lock.PipxTool.Architecture = "amd64"
	lock.PipxTool.RuntimeImage = "python_image"
	lock.PipxTool.Tool.SHA256 = strings.Repeat("f", 64)
	lock.PipxTool.Tool.Size = 1024
	lock.PipxTool.Notices.SHA256 = strings.Repeat("e", 64)
	lock.PipxTool.Notices.Size = 1024
}

func TestManifestRequiresExplicitPipxDataRoleAndPythonBinding(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "runtimes.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pinned runtimeLock
	if err := json.Unmarshal(body, &pinned); err != nil {
		t.Fatal(err)
	}
	manifest, err := buildManifest(pinned)
	if err != nil {
		t.Fatal(err)
	}
	data := manifest.Tools[2]
	if manifest.Schema != "helox.runtime-image-manifest/v3" || len(manifest.Tools) != 3 || data.Name != "pipx" || data.Role != "pipx-runtime" || data.Mode != "0444" || data.RuntimeImage != pinned.PythonImage.Reference || data.Asset != "pipx-runtime-linux-amd64.zip" || data.SHA256 != pinned.PipxTool.Tool.SHA256 {
		t.Fatal("pipx DATA became executable or lost owner binding")
	}
	for name, mutate := range map[string]func(*runtimeLock){
		"missing":      func(l *runtimeLock) { l.PipxTool.Version = "" },
		"architecture": func(l *runtimeLock) { l.PipxTool.Architecture = "arm64" },
		"runtime":      func(l *runtimeLock) { l.PipxTool.RuntimeImage = "node_image" },
		"hash":         func(l *runtimeLock) { l.PipxTool.Tool.SHA256 = "" },
		"size":         func(l *runtimeLock) { l.PipxTool.Tool.Size = 8<<20 + 1 },
		"notices":      func(l *runtimeLock) { l.PipxTool.Notices.Size = 64<<10 + 1 },
	} {
		candidate := pinned
		mutate(&candidate)
		if _, err := buildManifest(candidate); err == nil {
			t.Fatal("invalid pipx DATA owner accepted", name)
		}
	}
}
