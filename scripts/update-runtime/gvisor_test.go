package main

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGvisorCaptureUsesActualCompilerAndOutput(t *testing.T) {
	root, state := fixture(t)
	selected, err := selectedVersion(root, "go")
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(state, "upstream")
	writeTest(t, filepath.Join(source, "go.mod"), []byte("module gvisor.dev/gvisor\n\ngo "+selected+"\n"))
	writeTest(t, filepath.Join(source, "README.md"), []byte("synthetic upstream\n"))
	sdkPatch := []byte("synthetic SDK patch; not upstream evidence\n")
	writeTest(t, filepath.Join(source, "tools/go_types_memoize.patch"), sdkPatch)
	gitTest(t, source, "init", "-q")
	gitTest(t, source, "add", "--all")
	commitTest(t, source)
	head, err := command(source, nil, "git", "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	var runtime map[string]any
	if err := loadJSON(filepath.Join(root, runtimeFile), &runtime); err != nil {
		t.Fatal(err)
	}
	gvisor := runtime["gvisor"].(map[string]any)
	originalCommit := gvisor["commit"].(string)
	gvisor["commit"] = strings.TrimSpace(string(head))
	haaPatch := []byte("diff --git a/README.md b/README.md\n--- a/README.md\n+++ b/README.md\n@@ -1 +1 @@\n-synthetic upstream\n+synthetic patched source\n")
	patchPath := "tools/gvisor/synthetic.patch"
	writeTest(t, filepath.Join(root, patchPath), haaPatch)
	gvisor["patch"].(map[string]any)["path"] = patchPath
	gvisor["patch"].(map[string]any)["sha256"] = hash(haaPatch)
	writeTest(t, filepath.Join(source, "README.md"), []byte("synthetic patched source\n"))
	body, err := json.MarshalIndent(runtime, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeTest(t, filepath.Join(root, runtimeFile), append(body, '\n'))
	moduleBody, err := os.ReadFile(filepath.Join(source, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	prep := gvisorPreparation{1, gvisor["commit"].(string), gvisor["patch"].(map[string]any)["sha256"].(string), selected, "1.26.3", hash(sdkPatch), hash(moduleBody)}
	manifest := source + ".haa-source.json"
	if err := saveJSON(manifest, prep); err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(source, "MODULE.bazel.lock")
	writeTest(t, module, []byte("{\"synthetic_lock\":true}\n"))
	buildRoot := filepath.Join(state, "built")
	program := filepath.Join(state, "minimal.go")
	writeTest(t, program, []byte("package main\nfunc main(){}\n"))
	binary := filepath.Join(state, "minimal")
	// The inspected gVisor target is Linux amd64 even on a native macOS checker.
	buildEnv := append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	if _, err := command(root, buildEnv, "go", "build", "-trimpath", "-o", binary, program); err != nil {
		t.Fatal(err)
	}
	binaryBody, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range bundlePaths {
		contents := binaryBody
		if path == "gvisor-bin/gvisor-sentry-prewarmer" {
			contents = []byte("#!/bin/sh\nexit 0\n")
		}
		writeTest(t, filepath.Join(buildRoot, path), contents)
	}
	// Bazel's output directory is a symlink. Members themselves must be real.
	alias := filepath.Join(state, "bazel-bin")
	if err := os.Symlink(buildRoot, alias); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(state, "captured.json")
	args := func(destination string) []string {
		return []string{"--root", root, "--built-dir", alias, "--module-lock", module, "--source-manifest", manifest, "--output", destination}
	}
	if err := captureGvisor(args(output), io.Discard); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(filepath.Join(root, runtimeFile))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, append(body, '\n')) {
		t.Fatal("capture modified checkout")
	}
	var candidate struct {
		Gvisor struct {
			Build struct {
				Module string `json:"bazel_module_lock_sha256"`
			} `json:"build"`
			Bundle struct {
				Members []struct {
					Path string `json:"path"`
					Size int    `json:"size"`
					SHA  string `json:"sha512"`
				} `json:"members"`
			} `json:"runtime_bundle"`
		} `json:"gvisor"`
	}
	captured, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(captured, &candidate); err != nil {
		t.Fatal(err)
	}
	lockBytes, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Gvisor.Build.Module != hash(lockBytes) {
		t.Fatal("module lock identity not captured")
	}
	for i, m := range candidate.Gvisor.Bundle.Members {
		contents, err := os.ReadFile(filepath.Join(buildRoot, bundlePaths[i]))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha512.Sum512(contents)
		if m.Path != bundlePaths[i] || m.Size != len(contents) || m.SHA != hex.EncodeToString(sum[:]) {
			t.Fatal("member identity not observed from actual bytes")
		}
	}
	input, err := readGvisorCapture(output)
	if err != nil {
		t.Fatal(err)
	}
	proposal := plan{Root: root, Component: "go", Version: selected, GvisorCapture: &input}
	imported, err := gvisorCaptureChanges(proposal, state, original)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(imported, captured) {
		t.Fatal("actual captured metadata was not linked to the coordinated patch")
	}
	// Exercise the public plan/apply path with changed outputs at the same SDK.
	// Both the lock and generated identity must change, with full Go qualification.
	observerPath := filepath.Join(root, "tools/gvisor-observer/BUILD")
	observer, err := os.ReadFile(observerPath)
	if err != nil {
		t.Fatal(err)
	}
	writeTest(t, observerPath, bytes.ReplaceAll(observer, []byte(originalCommit), []byte(prep.Commit)))
	if _, err := command(root, nil, "go", "run", "./scripts/generate-runtime-lock.go"); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", "--", runtimeFile, generatedFile, "tools/gvisor-observer/BUILD", patchPath)
	commitTest(t, root)
	planState := filepath.Join(state, "observed-plan")
	if err := run([]string{"plan", "--root", root, "--state", planState, "--component", "go", "--version", selected, "--gvisor-capture", output}, io.Discard); err != nil {
		t.Fatal(err)
	}
	var frozen plan
	if err := loadJSON(filepath.Join(planState, "plan.json"), &frozen); err != nil {
		t.Fatal(err)
	}
	if len(frozen.Changes) != 2 || !reflect.DeepEqual(frozen.RequiredQualification, qualification("go")) {
		t.Fatal("same-SDK output change omitted coordinated generation or qualification")
	}
	if err := run([]string{"apply", "--root", root, "--state", planState}, io.Discard); err != nil {
		t.Fatal(err)
	}
	applied, err := os.ReadFile(filepath.Join(planState, "candidate", runtimeFile))
	if err != nil || !bytes.Equal(applied, captured) {
		t.Fatal("frozen observed outputs were not applied to the isolated candidate")
	}
	tampered := input
	tampered.Lock = append(append([]byte{}, input.Lock...), ' ')
	proposal.GvisorCapture = &tampered
	if _, err := gvisorCaptureChanges(proposal, state, original); err == nil {
		t.Fatal("changed capture accepted without matching receipt")
	}
	proposal.GvisorCapture = &input
	proposal.Version = "1.99.1"
	if _, err := gvisorCaptureChanges(proposal, state, original); err == nil {
		t.Fatal("unrelated selected SDK accepted")
	}
	if err := captureGvisor(args(output), io.Discard); err == nil {
		t.Fatal("existing candidate overwritten")
	}
	for _, scenario := range []string{"wrong SDK", "wrong patch", "changed SDK selector", "changed source", "extra source", "extra member", "member alias", "wrong compiler"} {
		t.Run(scenario, func(t *testing.T) {
			changed := prep
			restore := func() {}
			switch scenario {
			case "wrong SDK":
				changed.Go = "1.99.1"
			case "wrong patch":
				changed.SDKPatch = strings.Repeat("a", 64)
			case "changed SDK selector":
				writeTest(t, filepath.Join(source, "go.mod"), []byte("changed"))
				restore = func() { writeTest(t, filepath.Join(source, "go.mod"), moduleBody) }
			case "changed source":
				path := filepath.Join(source, "README.md")
				writeTest(t, path, []byte("undeclared source change\n"))
				restore = func() { writeTest(t, path, []byte("synthetic patched source\n")) }
			case "extra source":
				path := filepath.Join(source, "undeclared.go")
				writeTest(t, path, []byte("package main\n"))
				restore = func() {
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
				}
			case "extra member":
				path := filepath.Join(buildRoot, "unexpected")
				writeTest(t, path, []byte("x"))
				restore = func() {
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
				}
			case "member alias":
				path := filepath.Join(buildRoot, "runsc")
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(binary, path); err != nil {
					t.Fatal(err)
				}
				restore = func() {
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
					writeTest(t, path, binaryBody)
				}
			case "wrong compiler":
				path := filepath.Join(buildRoot, "runsc")
				writeTest(t, path, []byte("not a Go binary"))
				restore = func() { writeTest(t, path, binaryBody) }
			}
			defer restore()
			if err := saveJSON(manifest, changed); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := saveJSON(manifest, prep); err != nil {
					t.Fatal(err)
				}
			}()
			destination := filepath.Join(state, "rejected-"+strings.ReplaceAll(scenario, " ", "-")+".json")
			if err := captureGvisor(args(destination), io.Discard); err == nil {
				t.Fatal("invalid capture admitted")
			}
			if _, err := os.Stat(destination); !os.IsNotExist(err) {
				t.Fatal("invalid capture wrote output")
			}
		})
	}
}
