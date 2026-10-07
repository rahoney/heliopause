package bootstrap_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/bootstrap"
)

// Actual installed client exercises source authentication, fixed isolated probe,
// recorded Evidence, full project ALLOW, cache, initial/retained installation.
func TestLinuxTerraformInitIntegration(t *testing.T) {
	if os.Getenv("HELOX_TERRAFORM_INIT_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	for _, fixture := range []struct {
		name, reference, requirements, lock string
		count                               int
	}{
		{"official", "hashicorp/random@3.7.2", `random = { source = "hashicorp/random", version = "3.7.2" }`, "", 1},
		{"partner", "integrations/github@6.6.0", `github = { source = "integrations/github", version = "6.6.0" }`, "", 1},
		{"complete-project", "hashicorp/random@3.7.2", `random = { source = "hashicorp/random", version = "3.7.2" }
 github = { source = "integrations/github", version = "6.6.0" }`, `provider "registry.terraform.io/integrations/github" {
 version = "6.6.0"
 constraints = "6.6.0"
 hashes = ["zh:772edb5890d72b32868f9fdc0a9a1d4f4701d8e7f8acb37a7ac530d053c776e3"]
}
`, 2},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			root, e := filepath.EvalSymlinks(t.TempDir())
			if e != nil {
				t.Fatal(e)
			}
			project := filepath.Join(root, "project")
			if e := os.Mkdir(project, 0o700); e != nil {
				t.Fatal(e)
			}
			config := []byte("terraform {\n required_providers {\n" + fixture.requirements + "\n }\n}\n")
			if e := os.WriteFile(filepath.Join(project, "main.tf"), config, 0o600); e != nil {
				t.Fatal(e)
			}
			if fixture.lock != "" {
				if e := os.WriteFile(filepath.Join(project, ".terraform.lock.hcl"), []byte(fixture.lock), 0o600); e != nil {
					t.Fatal(e)
				}
			}
			t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
			t.Chdir(project)
			// Ambient mirror and CLI flags must never influence provider acquisition.
			mirror := filepath.Join(root, "terraform.rc")
			if e := os.WriteFile(mirror, []byte(`provider_installation { network_mirror { url = "https://example.invalid/mirror/" } }`), 0o600); e != nil {
				t.Fatal(e)
			}
			t.Setenv("TF_CLI_CONFIG_FILE", mirror)
			t.Setenv("TF_CLI_ARGS_init", "-plugin-dir=/foreign/cache")
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
			defer cancel()
			previousRuns := map[string]bool{}
			for attempt := 0; attempt < 2; attempt++ {
				var out, errs bytes.Buffer
				e := bootstrap.Run(ctx, []string{"terraform", "init", fixture.reference}, &out, &errs)
				if e != nil {
					if attempt == 0 {
						got, re := os.ReadFile(filepath.Join(project, ".terraform.lock.hcl"))
						if fixture.lock == "" {
							if !os.IsNotExist(re) {
								t.Error("failure published a new lock")
							}
						} else if re != nil || !bytes.Equal(got, []byte(fixture.lock)) {
							t.Error("failure changed original lock")
						}
					}
					if _, re := os.Lstat(filepath.Join(project, ".heliopause-terraform-transaction.lock")); !os.IsNotExist(re) {
						t.Error("guard leaked")
					}
					t.Fatalf("actual Terraform init attempt%d: %v stdout=%s stderr=%s", attempt, e, out.String(), errs.String())
				}
				if !strings.Contains(out.String(), fmt.Sprintf("Source: terraform-registry\nProviders: %d\nLock digest: ", fixture.count)) {
					t.Fatalf("complete count missing: %s", out.String())
				}
				got, re := os.ReadFile(filepath.Join(project, "main.tf"))
				if re != nil || !bytes.Equal(got, config) {
					t.Fatal("frozen configuration changed")
				}
				states, se := filepath.Glob(filepath.Join(root, "cache/heliopause/terraform-projects/*.json"))
				if se != nil || len(states) != 1 {
					t.Fatal("independent approval missing")
				}
				body, re := os.ReadFile(states[0])
				if re != nil {
					t.Fatal(re)
				}
				var state struct {
					Schema   int `json:"schema"`
					Approval struct {
						Policy  string `json:"policy"`
						Entries []struct {
							Name, Version, Run string
							Executable         string            `json:"executable"`
							ExecutableDigest   string            `json:"executable_digest"`
							Evidence           []json.RawMessage `json:"evidence"`
						} `json:"entries"`
					} `json:"approval"`
				}
				if json.Unmarshal(body, &state) != nil || state.Schema != 1 || len(state.Approval.Entries) != fixture.count || state.Approval.Policy == "" {
					t.Fatal("complete approval grammar/coverage missing")
				}
				runs := map[string]bool{}
				for _, entry := range state.Approval.Entries {
					if len(entry.Evidence) != 3 || entry.Run == "" || previousRuns[entry.Run] || runs[entry.Run] {
						t.Fatal("independent source/static/dynamic Evidence or Run binding missing")
					}
					runs[entry.Run] = true
					path := filepath.Join(project, ".terraform/providers/registry.terraform.io", strings.Replace(entry.Name, "_", "/", 1), entry.Version, "linux_amd64", entry.Executable)
					binary, re := os.ReadFile(path)
					if re != nil {
						t.Fatal(re)
					}
					digest := sha256.Sum256(binary)
					pins := map[string]string{"hashicorp_random": "b2e1c417324b1100b67360e044d69636043320ffe024d0546f18b74ed99a7c39", "integrations_github": "eb5a89fb6f8ac65d81ed98398a89155b21b1b072ef032e7cad34ca10b477884f"}
					if hex.EncodeToString(digest[:]) != entry.ExecutableDigest || entry.ExecutableDigest != pins[entry.Name] {
						t.Fatal("installed executable differs from independent public pin")
					}
					info, re := os.Lstat(path)
					if re != nil || info.Mode().Perm() != 0o555 {
						t.Fatal("executable installation mode differs")
					}
				}
				previousRuns = runs
				t.Logf("actual_terraform_init_attempt=%d %s", attempt, strings.TrimSpace(out.String()))
			}
		})
	}
}

func TestLinuxTerraformInitNegativeIntegration(t *testing.T) {
	if os.Getenv("HELOX_TERRAFORM_INIT_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	for _, kind := range []string{"lock-checksum-mismatch", "remote-module", "foreign-provider-cache", "concurrent-init"} {
		t.Run(kind, func(t *testing.T) {
			root, e := filepath.EvalSymlinks(t.TempDir())
			if e != nil {
				t.Fatal(e)
			}
			project := filepath.Join(root, "project")
			if e := os.Mkdir(project, 0o700); e != nil {
				t.Fatal(e)
			}
			config := []byte("terraform {\n required_providers {\n random = { source = \"hashicorp/random\", version = \"3.7.2\" }\n }\n}\n")
			if kind == "remote-module" {
				config = append(config, []byte("module \"outside\" { source = \"https://example.invalid/module\" }\n")...)
			}
			if e := os.WriteFile(filepath.Join(project, "main.tf"), config, 0o600); e != nil {
				t.Fatal(e)
			}
			lock := []byte(nil)
			if kind == "lock-checksum-mismatch" {
				lock = []byte("provider \"registry.terraform.io/hashicorp/random\" {\n version = \"3.7.2\"\n hashes = [\"zh:" + strings.Repeat("0", 64) + "\"]\n}\n")
				if e := os.WriteFile(filepath.Join(project, ".terraform.lock.hcl"), lock, 0o600); e != nil {
					t.Fatal(e)
				}
			}
			if kind == "foreign-provider-cache" {
				if e := os.Mkdir(filepath.Join(project, ".terraform"), 0o755); e != nil {
					t.Fatal(e)
				}
			}
			if kind == "concurrent-init" {
				if e := os.WriteFile(filepath.Join(project, ".heliopause-terraform-transaction.lock"), []byte("held by independent fixture"), 0o600); e != nil {
					t.Fatal(e)
				}
			}
			t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
			t.Chdir(project)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			var out, errs bytes.Buffer
			e = bootstrap.Run(ctx, []string{"terraform", "init", "hashicorp/random@3.7.2"}, &out, &errs)
			if e == nil || strings.Contains(out.String(), "Providers:") {
				t.Fatal("invalid project installed")
			}
			got, re := os.ReadFile(filepath.Join(project, "main.tf"))
			if re != nil || !bytes.Equal(got, config) {
				t.Fatal("original configuration changed")
			}
			got, re = os.ReadFile(filepath.Join(project, ".terraform.lock.hcl"))
			if len(lock) == 0 {
				if !os.IsNotExist(re) {
					t.Fatal("negative published lock")
				}
			} else if re != nil || !bytes.Equal(got, lock) {
				t.Fatal("original lock not preserved")
			}
			states, se := filepath.Glob(filepath.Join(root, "cache/heliopause/terraform-projects/*.json"))
			if se != nil || len(states) != 0 {
				t.Fatal("negative published independent approval")
			}
			if kind == "concurrent-init" {
				got, re := os.ReadFile(filepath.Join(project, ".heliopause-terraform-transaction.lock"))
				if re != nil || string(got) != "held by independent fixture" {
					t.Fatal("another init guard was removed")
				}
			} else if _, re := os.Lstat(filepath.Join(project, ".heliopause-terraform-transaction.lock")); !os.IsNotExist(re) {
				t.Fatal("failed init leaked guard")
			}
			t.Logf("actual_terraform_negative=%s primary=%v", kind, e)
		})
	}
}
