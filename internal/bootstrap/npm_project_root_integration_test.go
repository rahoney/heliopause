//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
)

func TestNPMPublicProjectRootCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_NPM_PROJECT_QUARANTINE_INTEGRATION") != "1" {
		t.Skip("requires declared protected npm local-root CLI")
	}
	if os.Geteuid() == 0 {
		t.Fatal("requires ordinary-user client")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	for _, tc := range []struct {
		name, key, code          string
		dependencies, incomplete bool
	}{
		{"postinstall", "postinstall", "if(!require('is-number')(3))process.exit(42)", true, false},
		{"dependencies", "dependencies", "process.exit(37)", true, true},
		{"postprepare", "postprepare", "process.exit(37)", true, true},
		{"empty-dependencies", "postinstall", "process.exit(0)", false, false},
		{"conditional-network", "postinstall", "if(process.env.GITHUB_TOKEN&&require('fs').existsSync('/tmp/project/.env')){const s=require('net').connect(9,'127.0.0.1');s.on('error',()=>{});}", false, false},
	} {
		if !t.Run(tc.name, func(t *testing.T) {
			started := time.Now()
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(root, "project")
			if err := os.Mkdir(target, 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
			t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
			manifest := map[string]any{"name": "haa-local-root", "private": true, "scripts": map[string]string{tc.key: "node hook.js"}}
			if tc.dependencies {
				manifest["dependencies"] = map[string]string{"is-number": "7.0.0"}
			}
			body, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(target, "package.json"), body, 0o600); err != nil {
				t.Fatal(err)
			}
			code := []byte(tc.code + "\n")
			if err := os.WriteFile(filepath.Join(target, "hook.js"), code, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(target, ".env"), []byte("fixture-only dummy\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			var output, stderr bytes.Buffer
			cause := Run(ctx, []string{"--lang", "en", "--output", "json", "npm", "install", "--target", target}, &output, &stderr)
			if cause != nil && stderr.Len() != 0 {
				t.Logf("npm_root_cli_diagnostic=%s", stderr.String())
			}
			if cause == nil {
				t.Fatal("local install action gained automatic Host execution")
			}
			var report struct {
				Status    string `json:"operation_status"`
				Promotion string `json:"promotion_status"`
				Lifecycle struct {
					Checked  bool   `json:"checked"`
					Required bool   `json:"required"`
					Status   string `json:"inspection_status"`
					Result   struct {
						Status string `json:"operation_status"`
						Run    struct {
							ID string `json:"id"`
						} `json:"run"`
						Evidence []struct {
							ID     string `json:"id"`
							Handle string `json:"handle"`
						} `json:"evidence_references"`
						RawChecks []struct {
							ID       string `json:"id"`
							Status   string `json:"execution_status"`
							Required bool   `json:"required"`
							Profile  string `json:"profile_sha256"`
						} `json:"checks"`
						Policy struct {
							Decision string   `json:"decision"`
							Reasons  []string `json:"reasons"`
						} `json:"policy"`
						Artifact struct {
							Identity struct {
								Source  string `json:"source_id"`
								Variant string `json:"variant"`
								Name    string `json:"name"`
								Version string `json:"version"`
							} `json:"resolved_identity"`
							Digest struct {
								Value string `json:"value"`
							} `json:"digest"`
						} `json:"artifact"`
					} `json:"result"`
				} `json:"project_lifecycle"`
			}
			decoder := json.NewDecoder(&output)
			if err := decoder.Decode(&report); err != nil {
				t.Fatal("root CLI produced no structured installation result", err, cause, stderr.String())
			}
			if err := decoder.Decode(new(any)); err != io.EOF {
				t.Fatal("root CLI produced multiple result documents", err)
			}
			if report.Status != "FAILED" || report.Promotion != "NOT_COMPLETED" || !report.Lifecycle.Checked || !report.Lifecycle.Required || report.Lifecycle.Result.Artifact.Identity.Source != "project-local" || report.Lifecycle.Result.Artifact.Identity.Variant != "npm-source" {
				t.Fatalf("root result lost its local scope or failed installation: %+v; cause=%v", report, cause)
			}
			if len(report.Lifecycle.Result.RawChecks) != 4 {
				t.Fatal("root verification/static/dynamic checks missing", report.Lifecycle.Result.RawChecks, cause)
			}
			lifecycleStatus := ""
			for _, check := range report.Lifecycle.Result.RawChecks {
				if check.Required == (check.ID == "npm-project-root-observation-facts") {
					t.Fatal("root fact interpretation/required coverage roles changed", check.ID)
				}
				if check.ID == "npm-project-root-lifecycle" {
					lifecycleStatus = check.Status
				}
			}
			if tc.incomplete {
				var exited interface{ ExitCode() int }
				if !errors.As(cause, &exited) || exited.ExitCode() != 37 {
					t.Fatal("root hook failure lost actual exit37", cause)
				}
				t.Log("npm_root_hook_exit=37 original_cause=PRESERVED")
				if lifecycleStatus != "INCOMPLETE" || report.Lifecycle.Status != "INCOMPLETE" {
					t.Fatal("native root hook did not expose its actual nonzero execution", lifecycleStatus, report.Lifecycle.Status, cause)
				}
			} else {
				if lifecycleStatus != "COMPLETED" || report.Lifecycle.Status != "COMPLETED" || report.Lifecycle.Result.Policy.Decision != "MANUAL_REVIEW" {
					t.Fatal("normal root observation was incomplete or granted Host execution", report.Lifecycle, cause)
				}
				found := false
				for _, reason := range report.Lifecycle.Result.Policy.Reasons {
					// M3 findings precede M4; an observed Artifact exec remains manual review.
					if reason == "M4_HOST_LIFECYCLE_UNSUPPORTED" || reason == "M3_UNEXPECTED_PROCESS" {
						found = true
					}
				}
				if !found {
					t.Fatal("existing M4 Host action restriction was lost", report.Lifecycle.Result.Policy.Reasons)
				}
			}
			if tc.name == "conditional-network" {
				// M3 retains the first review reason when both process and
				// network findings exist. Verify the actual network Evidence.
				run, err := domain.ParseRunID(report.Lifecycle.Result.Run.ID)
				if err != nil {
					t.Fatal(err)
				}
				identity := report.Lifecycle.Result.Artifact.Identity
				source, err := domain.NewSourceID(identity.Source)
				if err != nil {
					t.Fatal(err)
				}
				subject, err := domain.NewResolvedArtifactIdentity(source, identity.Name, identity.Version, identity.Variant)
				if err != nil {
					t.Fatal(err)
				}
				digest, err := domain.NewSHA256Digest(report.Lifecycle.Result.Artifact.Digest.Value)
				if err != nil {
					t.Fatal(err)
				}
				store, err := local.NewStore(filepath.Join(root, "cache", "heliopause", "evidence"))
				if err != nil {
					t.Fatal(err)
				}
				profile := ""
				for _, check := range report.Lifecycle.Result.RawChecks {
					if check.ID == "npm-project-root-observation-facts" {
						profile = check.Profile
					}
				}
				if profile == "" {
					t.Fatal("root observation profile missing")
				}
				states, network := 0, uint64(0)
				for _, reference := range report.Lifecycle.Result.Evidence {
					id, err := domain.NewEvidenceID(reference.ID)
					if err != nil {
						t.Fatal(err)
					}
					ref, err := domain.NewEvidenceReference(id, reference.Handle)
					if err != nil {
						t.Fatal(err)
					}
					fact, _, err := store.ReadReference(ctx, run, ref, subject, digest)
					if err != nil {
						t.Fatal("current root Evidence unavailable", err)
					}
					if fact.Kind() != "workspace-state" && fact.Kind() != "npm-project-root-observation" {
						continue
					}
					if fact.ProfileDigest().String() != profile || fact.CheckID().String() != "npm-project-root-observation-facts" {
						t.Fatal("root Evidence/check/profile binding differs")
					}
					if fact.Kind() == "workspace-state" {
						states++
						continue
					}
					var summary struct {
						Counts map[string]uint64 `json:"counts"`
					}
					if err := json.Unmarshal([]byte(fact.Summary()), &summary); err != nil {
						t.Fatal(err)
					}
					network += summary.Counts["NETWORK:network-attempt"]
				}
				if states != 2 || network == 0 {
					t.Fatal("root conditional activation/current persisted environments missing", states, network)
				}
			}
			for _, name := range []string{"node_modules", "package-lock.json"} {
				if _, err := os.Lstat(filepath.Join(target, name)); !os.IsNotExist(err) {
					t.Fatal("root inspection changed the original target", name, err)
				}
			}
			original, err := os.ReadFile(filepath.Join(target, "package.json"))
			if err != nil || !bytes.Equal(original, body) {
				t.Fatal("root inspection changed original manifest", err)
			}
			original, err = os.ReadFile(filepath.Join(target, "hook.js"))
			if err != nil || !bytes.Equal(original, code) {
				t.Fatal("root inspection changed local source", err)
			}
			original, err = os.ReadFile(filepath.Join(target, ".env"))
			if err != nil || string(original) != "fixture-only dummy\n" {
				t.Fatal("root inspection changed original dummy env", err)
			}
			if tc.name == "conditional-network" {
				t.Log("npm_root_conditional network=OBSERVED stored_states=2 original_target=PRESERVED")
			}
			t.Logf("npm_root_cli case=%s local_identity=true required_checks=3 lifecycle=%s installation=REFUSED original_target=PRESERVED seconds=%.3f", tc.name, lifecycleStatus, time.Since(started).Seconds())
		}) {
			return
		}
	}
}

// Full public composition, separate from native/Application-only coverage.
func TestNPMPublicProjectCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_NPM_PROJECT_QUARANTINE_INTEGRATION") != "1" {
		t.Skip("requires declared protected npm whole-project CLI")
	}
	if os.Geteuid() == 0 {
		t.Fatal("requires ordinary-user client")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "project")
	if err = os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	manifest := []byte(`{"name":"haa-npm-cli","private":true,"dependencies":{"is-number":"7.0.0"}}`)
	if err = os.WriteFile(filepath.Join(target, "package.json"), manifest, 0600); err != nil {
		t.Fatal(err)
	}
	var lock []byte
	var payload os.FileInfo
	for _, operation := range []string{"install", "ci"} {
		started := time.Now()
		var output, stderr bytes.Buffer
		if err = Run(ctx, []string{"--lang", "en", "--output", "json", "npm", operation, "--target", target}, &output, &stderr); err != nil {
			t.Fatal("public npm CLI", operation, err, stderr.String(), output.String())
		}
		var report struct {
			Status    string `json:"operation_status"`
			Promotion string `json:"promotion_status"`
			Lifecycle struct {
				Checked, Required bool
				Status            string `json:"inspection_status"`
			} `json:"project_lifecycle"`
			Policy struct{ Decision string } `json:"policy"`
		}
		decoder := json.NewDecoder(&output)
		if err = decoder.Decode(&report); err != nil {
			t.Fatal(err)
		}
		if err = decoder.Decode(new(any)); err != io.EOF {
			t.Fatal("multiple public result documents", err)
		}
		if report.Status != "COMPLETED" || report.Promotion != "COMPLETED" || report.Policy.Decision != "ALLOW" || !report.Lifecycle.Checked || report.Lifecycle.Required || report.Lifecycle.Status != "NOT_REQUIRED" {
			t.Fatal("public npm completion/report mismatch", report)
		}
		current, err := os.ReadFile(filepath.Join(target, "package-lock.json"))
		if err != nil {
			t.Fatal(err)
		}
		currentPayload, err := os.Lstat(filepath.Join(target, "node_modules"))
		if err != nil || !currentPayload.IsDir() {
			t.Fatal("public npm omitted approved payload", err)
		}
		if operation == "ci" && (!bytes.Equal(lock, current) || !os.SameFile(payload, currentPayload)) {
			t.Fatal("ci rewrote lock or unchanged payload")
		}
		lock, payload = current, currentPayload
		installed, err := os.ReadFile(filepath.Join(target, "node_modules", "is-number", "package.json"))
		if err != nil {
			t.Fatal(err)
		}
		var pkg struct{ Name, Version string }
		if json.Unmarshal(installed, &pkg) != nil || pkg.Name != "is-number" || pkg.Version != "7.0.0" {
			t.Fatal("public CLI installed wrong package")
		}
		t.Logf("npm_public_cli operation=%s Policy=ALLOW publication=COMPLETED root=NOT_REQUIRED seconds=%.3f", operation, time.Since(started).Seconds())
	}
}
