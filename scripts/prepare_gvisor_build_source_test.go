package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGvisorSourceSDKPreparation(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("gVisor SDK preparation runs on Linux amd64 builders")
	}
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("source preparation fixture requires the builder's jq tool")
	}
	script, err := filepath.Abs("prepare-gvisor-build-source.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"coordinated synthetic SDK", "partial owner", "wrong patch", "SDK downgrade", "independent selector", "changed SDK patch"} {
		t.Run(scenario, func(t *testing.T) {
			base := t.TempDir()
			upstream := filepath.Join(base, "upstream")
			workspace := filepath.Join(base, "workspace")
			write := func(root, path, body string) {
				t.Helper()
				full := filepath.Join(root, path)
				if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			goMod := "module gvisor.dev/gvisor\n\ngo 1.0.0\n"
			if scenario == "independent selector" {
				goMod += "toolchain go1.2.3\n"
			}
			write(upstream, "go.mod", goMod)
			write(upstream, "README", "before\n")
			write(upstream, "MODULE.bazel", "go_sdk.from_file(\n    go_mod = \"//:go.mod\",\n)\n")
			write(upstream, "tools/go_types_memoize.patch", "synthetic SDK patch\n")
			git := func(args ...string) string {
				t.Helper()
				cmd := exec.Command("git", args...)
				cmd.Dir = upstream
				output, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("fixture Git failed: %v\n%s", err, output)
				}
				return strings.TrimSpace(string(output))
			}
			git("init", "-q")
			git("add", "--all")
			git("-c", "user.name=SDK fixture", "-c", "user.email=sdk-fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "SDK 준비 fixture 기준선")
			commit := git("rev-parse", "HEAD")
			write(upstream, "README", "after\n")
			if scenario == "changed SDK patch" {
				write(upstream, "tools/go_types_memoize.patch", "unreviewed mutation\n")
			}
			patch := git("diff", "--binary") + "\n"
			write(workspace, "tools/gvisor/fixture.patch", patch)
			sum := sha256.Sum256([]byte(patch))
			patchSHA := hex.EncodeToString(sum[:])
			if scenario == "wrong patch" {
				patchSHA = strings.Repeat("a", 64)
			}
			selected := "1.99.1"
			if scenario == "SDK downgrade" {
				selected = "0.1.0"
			}
			imageVersion := selected
			if scenario == "partial owner" {
				imageVersion = "1.99.0"
			}
			lock := map[string]any{"gvisor": map[string]any{"source_repository": "https://github.com/google/gvisor.git", "commit": commit, "patch": map[string]any{"path": "tools/gvisor/fixture.patch", "sha256": patchSHA}}, "go_image": map[string]any{"go_version": selected, "reference": "golang:" + imageVersion + "-bookworm@sha256:" + strings.Repeat("b", 64)}}
			body, err := json.Marshal(lock)
			if err != nil {
				t.Fatal(err)
			}
			write(workspace, "runtime.json", string(body))
			destination := filepath.Join(base, "prepared")
			cmd := exec.Command("sh", script, filepath.Join(workspace, "runtime.json"), destination)
			cmd.Dir = workspace
			cmd.Env = append(os.Environ(), "GVISOR_LOCAL_REPO="+upstream)
			output, runErr := cmd.CombinedOutput()
			if scenario != "coordinated synthetic SDK" {
				if runErr == nil {
					t.Fatal("invalid SDK preparation accepted")
				}
				return
			}
			if runErr != nil {
				t.Fatalf("known source rejected: %v\n%s", runErr, output)
			}
			actual, err := os.ReadFile(filepath.Join(destination, "go.mod"))
			if err != nil {
				t.Fatal(err)
			}
			want := strings.Replace(goMod, "go 1.0.0", "go "+selected, 1)
			if string(actual) != want {
				t.Fatal("SDK selector not changed in the one declared field")
			}
			original, err := os.ReadFile(filepath.Join(upstream, "go.mod"))
			if err != nil || string(original) != goMod {
				t.Fatal("upstream source changed", err)
			}
			var manifest struct {
				Go      string `json:"selected_go"`
				Minimum string `json:"upstream_go_minimum"`
			}
			record, err := os.ReadFile(destination + ".haa-source.json")
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(record, &manifest); err != nil {
				t.Fatal(err)
			}
			if manifest.Go != selected || manifest.Minimum != "1.0.0" {
				t.Fatal("source/SDK identity missing")
			}
		})
	}
}
