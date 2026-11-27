//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCargoPublicFetchCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_CARGO_FETCH_CLI_INTEGRATION") != "1" {
		t.Skip("requires declared protected Cargo fetch CLI")
	}
	if os.Geteuid() == 0 {
		t.Fatal("requires ordinary-user client")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, "project")
	if err = os.MkdirAll(filepath.Join(project, "src"), 0700); err != nil {
		t.Fatal(err)
	}
	manifest := []byte("[package]\nname='haa_fetch_project'\nversion='0.1.0'\nedition='2021'\n[dependencies]\nitoa='=1.0.17'\n")
	code := []byte("fn main(){let mut b=itoa::Buffer::new();assert_eq!(b.format(7),\"7\");}\n")
	buildScript := []byte("fn main(){panic!(\"fetch must not execute the build script\");}\n")
	for name, body := range map[string][]byte{"Cargo.toml": manifest, "src/main.rs": code, "build.rs": buildScript} {
		if err = os.WriteFile(filepath.Join(project, name), body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	originalManifest, err := os.Lstat(filepath.Join(project, "Cargo.toml"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Chdir(project)
	var lock []byte
	var lockInfo os.FileInfo
	for _, stage := range []string{"cold-first-lock", "warm-existing-lock"} {
		started := time.Now()
		var output, stderr bytes.Buffer
		if err = Run(ctx, []string{"--lang", "en", "--output", "json", "cargo", "fetch"}, &output, &stderr); err != nil {
			t.Fatal("public fetch", stage, err, output.String(), stderr.String())
		}
		var report struct {
			Operation string `json:"operation"`
			Scope     string `json:"inspection_scope"`
			Status    string `json:"operation_status"`
			Promotion string `json:"promotion_status"`
			Entries   []struct {
				Identity struct{ Name, Version string } `json:"identity"`
				Checks   []struct {
					Status   string `json:"execution_status"`
					Required bool   `json:"required"`
				} `json:"checks"`
			} `json:"entries"`
			Policy struct{ Decision string } `json:"policy"`
		}
		decoder := json.NewDecoder(&output)
		if err = decoder.Decode(&report); err != nil {
			t.Fatal(err)
		}
		if err = decoder.Decode(new(any)); err != io.EOF {
			t.Fatal("fetch emitted multiple documents", err)
		}
		if report.Operation != "FETCH" || report.Scope != "DEPENDENCY_PREPARATION" || report.Status != "COMPLETED" || report.Promotion != "COMPLETED" || report.Policy.Decision != "ALLOW" || len(report.Entries) != 1 || report.Entries[0].Identity.Name != "itoa" || report.Entries[0].Identity.Version != "1.0.17" {
			t.Fatal("fetch lost actual whole preparation", report)
		}
		checks := 0
		for _, check := range report.Entries[0].Checks {
			if check.Required {
				checks++
				if check.Status != "COMPLETED" {
					t.Fatal("fetch omitted required inspection", check)
				}
			}
		}
		if checks != 2 {
			t.Fatal("fetch integrity/static coverage changed", checks)
		}
		currentManifest, err := os.Lstat(filepath.Join(project, "Cargo.toml"))
		if err != nil || !os.SameFile(originalManifest, currentManifest) || originalManifest.Mode() != currentManifest.Mode() {
			t.Fatal("fetch replaced original manifest", err)
		}
		currentLock, err := os.ReadFile(filepath.Join(project, "Cargo.lock"))
		if err != nil {
			t.Fatal(err)
		}
		currentInfo, err := os.Lstat(filepath.Join(project, "Cargo.lock"))
		if err != nil {
			t.Fatal(err)
		}
		if stage == "warm-existing-lock" && (!bytes.Equal(lock, currentLock) || !os.SameFile(lockInfo, currentInfo)) {
			t.Fatal("fetch replaced unchanged lock")
		}
		lock, lockInfo = currentLock, currentInfo
		for _, name := range []string{"target", filepath.Join(".heliopause", "builds")} {
			if _, err = os.Lstat(filepath.Join(project, name)); !os.IsNotExist(err) {
				t.Fatal("fetch created compiled output", name, err)
			}
		}
		t.Logf("cargo_fetch_cli case=%s crates=1 required_checks=2 Policy=ALLOW compilation=NOT_RUN manifest=PRESERVED seconds=%.3f", stage, time.Since(started).Seconds())
	}
	// Existing independent approval must reject a changed controller cache before
	// either fetch or the later build can trust it. Restore only this owned fixture.
	metadataPath := filepath.Join(project, ".heliopause", "cargo-transaction.json")
	metadata, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatal(err)
	}
	var approval struct{ Cache string }
	projectHash := sha256.Sum256([]byte(project))
	approvalPath := filepath.Join(root, "cache", "heliopause", "cargo-projects", hex.EncodeToString(projectHash[:])+".json")
	approvalBytes, err := os.ReadFile(approvalPath)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(approvalBytes, &approval); err != nil {
		t.Fatal(err)
	}
	id, ok := strings.CutPrefix(approval.Cache, "project-cache:")
	if !ok || strings.ContainsAny(id, "/\\") {
		t.Fatal("invalid owned cache handle")
	}
	receipt := filepath.Join(root, "cache", "heliopause", "cargo-verified-cache", id, "receipt.json")
	saved, err := os.ReadFile(receipt)
	if err != nil {
		t.Fatal(err)
	}
	receiptInfo, err := os.Lstat(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(receipt, 0600); err != nil {
		t.Fatal(err)
	}
	tampered := bytes.Clone(saved)
	tampered[0] ^= 1
	if err = os.WriteFile(receipt, tampered, 0600); err != nil {
		t.Fatal(err)
	}
	var output, stderr bytes.Buffer
	if err = Run(ctx, []string{"--lang", "en", "--output", "json", "cargo", "fetch"}, &output, &stderr); err == nil {
		t.Fatal("fetch accepted tampered retained cache")
	}
	if current, err := os.ReadFile(filepath.Join(project, "Cargo.lock")); err != nil || !bytes.Equal(current, lock) {
		t.Fatal("cache refusal changed lock", err)
	}
	if current, err := os.ReadFile(metadataPath); err != nil || !bytes.Equal(current, metadata) {
		t.Fatal("cache refusal changed approval", err)
	}
	if current, err := os.ReadFile(approvalPath); err != nil || !bytes.Equal(current, approvalBytes) {
		t.Fatal("cache refusal changed independent approval", err)
	}
	if err = os.WriteFile(receipt, saved, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(receipt, receiptInfo.Mode().Perm()); err != nil {
		t.Fatal(err)
	}
	t.Log("cargo_fetch_cli case=cache-tamper result=REFUSED original_controls/approval=PRESERVED")
	// The build guard must consume the real approved dependency cache. Remove the
	// deliberate panic source before invoking the existing observed build path.
	if err = os.Remove(filepath.Join(project, "build.rs")); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	stderr.Reset()
	started := time.Now()
	if err = Run(ctx, []string{"--lang", "en", "--output", "human", "cargo", "build"}, &output, &stderr); err != nil {
		t.Fatal("existing observed build after fetch", err, output.String(), stderr.String())
	}
	entries, err := os.ReadDir(filepath.Join(project, ".heliopause", "builds"))
	if err != nil || len(entries) != 1 {
		t.Fatal("existing build did not publish exact output", err)
	}
	program, err := os.ReadFile(filepath.Join(project, ".heliopause", "builds", entries[0].Name(), "haa_fetch_project"))
	if err != nil || len(program) < 4 || !bytes.Equal(program[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatal("existing build did not consume prepared crate", err)
	}
	t.Logf("cargo_fetch_cli case=existing-build approved_dependencies=CONSUMED output=APPROVED_ELF seconds=%.3f", time.Since(started).Seconds())
}
