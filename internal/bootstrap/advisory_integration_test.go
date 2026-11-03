//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/inspection/advisory"
	"github.com/rahoney/heliopause/internal/sandbox"
	"github.com/rahoney/heliopause/internal/storage"
)

// The advisory response seed is explicitly controlled DATA, not a claim that
// packaging is malicious. The protected production CLI separately performs the
// real public wheel acquisition, native checks, Policy and installation.
func TestAdvisoryProductionCLIQuarantineIntegration(t *testing.T) {
	if os.Getenv("HELOX_ADVISORY_CLI_INTEGRATION") != "1" {
		t.Skip("requires declared protected production CLI advisory qualification")
	}
	if os.Geteuid() == 0 {
		t.Fatal("requires ordinary-user client")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	type check struct {
		ID       string `json:"id"`
		Status   string `json:"execution_status"`
		Required bool   `json:"required"`
	}
	for _, mode := range []string{"fresh-cache", "cache-reuse", "malware-block", "tampered-cache", "force-refresh", "malware-human"} {
		if !t.Run(mode, func(t *testing.T) {
			started := time.Now()
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
			t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
			cacheRoot := filepath.Join(root, "cache", "heliopause")
			backend, err := storage.NewLocal(ctx, cacheRoot, domain.DefaultStoragePolicy())
			if err != nil {
				t.Fatal(err)
			}
			operation, _ := domain.NewOperationID()
			seedContext := ports.WithStorage(ports.WithStorageOperation(ctx, operation), backend)
			malware := mode == "malware-block" || mode == "force-refresh" || mode == "malware-human"
			requests := 0
			seed, err := advisory.NewInspector(&advisoryCacheOriginal{}, &http.Client{Transport: advisoryCacheHTTP(func(request *http.Request) (*http.Response, error) {
				requests++
				body := `{"results":[{}]}`
				if malware {
					body = `{"results":[{"vulns":[{"id":"MAL-0000-helox-controlled","modified":"2026-10-20T00:00:00Z"}]}]}`
					if request.Method == http.MethodGet {
						body = `{"schema_version":"1.9.2","id":"MAL-0000-helox-controlled","modified":"2026-10-20T00:00:00Z","affected":[{"package":{"ecosystem":"PyPI","name":"packaging"},"versions":["24.2"]}]}`
					}
				}
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
			})})
			if err != nil {
				t.Fatal(err)
			}
			source, _ := domain.NewSourceID("pypi")
			identity, _ := domain.NewResolvedArtifactIdentity(source, "packaging", "24.2", "wheel")
			digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
			artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:controlled:advisory-data", 1)
			if err != nil {
				t.Fatal(err)
			}
			seedReport, seedErr := seed.Inspect(seedContext, artifact)
			closeErr := backend.Close()
			if seedErr != nil || closeErr != nil || requests != 1 && !malware || requests != 2 && malware {
				t.Fatal("controlled response seed failed", seedErr, closeErr, requests)
			}
			originalProof := ""
			for _, fact := range seedReport.Evidence() {
				if fact.Kind() == "advisory-response" && originalProof == "" {
					originalProof = fact.Summary()
				}
			}
			if originalProof == "" {
				t.Fatal("seed has no response proof")
			}
			if mode == "tampered-cache" {
				allocations, err := os.ReadDir(filepath.Join(cacheRoot, "advisory-cache"))
				if err != nil || len(allocations) != 1 {
					t.Fatal("seed allocation missing", err)
				}
				if err := os.WriteFile(filepath.Join(cacheRoot, "advisory-cache", allocations[0].Name(), "response.json"), []byte("changed controlled DATA"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			venv := filepath.Join(root, "venv")
			pythonVersion := strings.SplitN(sandbox.PinnedPythonRuntime().PythonVersion, ".", 3)
			if len(pythonVersion) != 3 {
				t.Fatal("pinned Python version is invalid")
			}
			site := filepath.Join(venv, "lib", "python"+strings.Join(pythonVersion[:2], "."), "site-packages")
			for _, directory := range []string{site, filepath.Join(venv, "bin")} {
				if err := os.MkdirAll(directory, 0700); err != nil {
					t.Fatal(err)
				}
			}
			for path, body := range map[string]string{filepath.Join(venv, "pyvenv.cfg"): "version = " + sandbox.PinnedPythonRuntime().PythonVersion + "\n", filepath.Join(site, "user-note.txt"): "preserve original user file\n", filepath.Join(root, "requirements.txt"): "packaging==24.2\n"} {
				if err := os.WriteFile(path, []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "malware-human" {
				before := requirementsTargetTree(t, venv)

				textCommand := exec.CommandContext(ctx, "/usr/libexec/heliopause/helox", "--lang", "en", "--output", "human", "--color", "never", "pip", "install", "-r", filepath.Join(root, "requirements.txt"), "--target", venv)
				var text, textDiagnostic bytes.Buffer
				textCommand.Stdout, textCommand.Stderr = &text, &textDiagnostic
				textErr := textCommand.Run()
				t.Logf("advisory_cli_text case=%s text=%s diagnostic=%s", mode, bytes.TrimSpace(text.Bytes()), bytes.TrimSpace(textDiagnostic.Bytes()))
				var exit *exec.ExitError
				if !errors.As(textErr, &exit) || exit.ExitCode() != 1 || !strings.Contains(text.String(), "MAL-0000-helox-controlled") || !strings.Contains(text.String(), "unknown severity") || !reflect.DeepEqual(before, requirementsTargetTree(t, venv)) {
					t.Fatal("actual explicit-human advisory detail/denial lost", textErr, text.String())
				}

				t.Logf("advisory_cli_human_complete case=%s target=PRESERVED exit=1 seconds=%.3f", mode, time.Since(started).Seconds())
				return
			}
			count := 1
			if mode == "cache-reuse" {
				count = 2
			}
			for iteration := range count {
				before := requirementsTargetTree(t, venv)
				args := []string{"--lang", "en", "--output", "json", "pip", "install", "-r", filepath.Join(root, "requirements.txt"), "--target", venv}
				if mode == "force-refresh" {
					args = append([]string{"--refresh-advisories"}, args...)
				}
				command := exec.CommandContext(ctx, "/usr/libexec/heliopause/helox", args...)
				var output, diagnostic bytes.Buffer
				command.Stdout, command.Stderr = &output, &diagnostic
				cause := command.Run()
				t.Logf("advisory_cli_result case=%s iteration=%d json=%s diagnostic=%s", mode, iteration, bytes.TrimSpace(output.Bytes()), bytes.TrimSpace(diagnostic.Bytes()))
				var report struct {
					Status    string `json:"operation_status"`
					Promotion string `json:"promotion_status"`
					Entries   []struct {
						Identity struct {
							Source                 string `json:"source_id"`
							Name, Version, Variant string
						} `json:"identity"`
						Digest struct {
							Value string `json:"value"`
						} `json:"digest"`
						Run      string                        `json:"run_id"`
						Checks   []check                       `json:"checks"`
						Evidence []struct{ ID, Handle string } `json:"evidence_references"`
						Policy   struct {
							Decision string
							Reasons  []string
						} `json:"policy"`
						Detail *struct {
							Status  string
							Records []struct{ ID, Impact string }
						} `json:"advisory_details"`
					} `json:"entries"`

					Failures []struct {
						Checks []check `json:"checks"`
					} `json:"inspection_results"`
				}
				decoder := json.NewDecoder(&output)
				if err := decoder.Decode(&report); err != nil {
					t.Fatal("production CLI has no structured result", err, cause)
				}
				if decoder.Decode(new(any)) != io.EOF {
					t.Fatal("multiple CLI result documents")
				}
				denied := mode == "malware-block" || mode == "tampered-cache"
				if denied {
					if cause == nil || report.Status != "FAILED" || report.Promotion != "NOT_COMPLETED" || !reflect.DeepEqual(before, requirementsTargetTree(t, venv)) {
						t.Fatal("denied advisory changed target or lost failure", cause, report)
					}
				} else if cause != nil || report.Status != "COMPLETED" || report.Promotion != "COMPLETED" {
					t.Fatal("normal/refresh CLI failed", cause, report)
				}
				if mode == "tampered-cache" {
					incomplete := false
					for _, failure := range report.Failures {
						for _, c := range failure.Checks {
							incomplete = incomplete || c.ID == "dependency-advisory" && c.Required && c.Status == "INCOMPLETE"
						}
					}
					if !incomplete {
						t.Fatal("tampered cache lost its required incomplete check", report)
					}
					continue
				}
				if len(report.Entries) != 1 {
					t.Fatal("whole public package set missing", report)
				}
				entry := report.Entries[0]
				if entry.Identity.Source != "pypi" || entry.Identity.Name != "packaging" || entry.Identity.Version != "24.2" {
					t.Fatal("unexpected selected public package", entry.Identity)
				}
				advisoryComplete := false
				for _, c := range entry.Checks {
					if c.Required && c.Status != "COMPLETED" {
						t.Fatal("required native check incomplete", c)
					}
					advisoryComplete = advisoryComplete || c.ID == "dependency-advisory" && c.Required && c.Status == "COMPLETED"
				}
				if !advisoryComplete {
					t.Fatal("advisory check missing")
				}
				if mode == "malware-block" {
					if entry.Policy.Decision != "BLOCK" || !strings.Contains(fmt.Sprint(entry.Policy.Reasons), "M13_ADVISORY_MALICIOUS") {
						t.Fatal("no-CVSS malware fixture did not block", entry.Policy)
					}
					if entry.Detail == nil || entry.Detail.Status != "AVAILABLE" || len(entry.Detail.Records) != 1 || entry.Detail.Records[0].ID != "MAL-0000-helox-controlled" || entry.Detail.Records[0].Impact != "AFFECTED" {
						t.Fatal("actual recorded advisory detail missing", entry.Detail)
					}
				} else if entry.Policy.Decision != "ALLOW" {
					t.Fatal("normal public package not allowed", entry.Policy)
				}
				run, err := domain.ParseRunID(entry.Run)
				if err != nil {
					t.Fatal(err)
				}
				subject, err := domain.NewResolvedArtifactIdentity(source, entry.Identity.Name, entry.Identity.Version, entry.Identity.Variant)
				if err != nil {
					t.Fatal(err)
				}
				currentDigest, err := domain.NewSHA256Digest(entry.Digest.Value)
				if err != nil {
					t.Fatal(err)
				}
				evidence, err := local.NewStore(filepath.Join(cacheRoot, "evidence"))
				if err != nil {
					t.Fatal(err)
				}
				states := 0
				proof := ""
				for _, ref := range entry.Evidence {
					id, err := domain.NewEvidenceID(ref.ID)
					if err != nil {
						t.Fatal(err)
					}
					reference, err := domain.NewEvidenceReference(id, ref.Handle)
					if err != nil {
						t.Fatal(err)
					}
					fact, _, err := evidence.ReadReference(ctx, run, reference, subject, currentDigest)
					if err != nil {
						t.Fatal("current-subject Evidence unavailable", err)
					}
					if fact.Kind() == "workspace-state" {
						states++
					}
					if fact.Kind() == "advisory-response" && proof == "" {
						proof = fact.Summary()
					}
				}
				if states != 2 || proof == "" {
					t.Fatal("native two-state inspection or advisory proof missing", states)
				}
				if mode != "force-refresh" && proof != originalProof || mode == "force-refresh" && proof == originalProof {
					t.Fatal("cache hit changed original fetch proof or refresh reused it")
				}
				marker, err := os.ReadFile(filepath.Join(site, "user-note.txt"))
				if err != nil || string(marker) != "preserve original user file\n" {
					t.Fatal("original user marker changed", err)
				}
				t.Logf("advisory_cli_verified case=%s iteration=%d current_checks=%d stored_states=%d policy=%s installation_requested=%t user_marker_preserved=true", mode, iteration, len(entry.Checks), states, entry.Policy.Decision, !denied)
			}
			t.Logf("advisory_cli_case=%s seconds=%.3f controlled_advisory=%t public_artifact=packaging@24.2", mode, time.Since(started).Seconds(), mode != "force-refresh")
		}) {
			t.FailNow()
		}
	}
}

// Reuse the production client, native selectors and cache owner. Only advisory
// DATA is controlled; the small public packages are never represented as malware.
func TestAdvisoryPublicLanguageProductionCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_ADVISORY_CLI_INTEGRATION") != "1" {
		t.Skip("requires declared protected production CLI advisory qualification")
	}
	if os.Geteuid() == 0 {
		t.Fatal("requires ordinary-user client")
	}
	type advisoryDetail struct {
		Status  string
		Records []struct{ ID, Impact string }
	}
	type subject struct {
		Source        string `json:"source_id"`
		Name, Version string
	}
	type packageResult struct {
		Identity subject `json:"identity"`
		Checks   []struct {
			ID       string `json:"id"`
			Required bool   `json:"required"`
			Status   string `json:"execution_status"`
		} `json:"checks"`
		Policy struct {
			Decision string
			Reasons  []string
		} `json:"policy"`
		Detail *advisoryDetail `json:"advisory_details"`
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	for _, fixture := range []struct {
		language, source, ecosystem, name, version string
		args                                       []string
		files                                      map[string]string
	}{
		{"npm", "npm", "npm", "is-number", "7.0.0", []string{"npm", "install", "is-number@7.0.0"}, map[string]string{"user-note.txt": "preserve user file\n", "package.json": `{ "name": "haa-advisory-fixture", "version": "1.0.0", "private": true, "dependencies": {} }`, "package-lock.json": `{"name":"haa-advisory-fixture","version":"1.0.0","lockfileVersion":3,"requires":true,"packages":{"":{"name":"haa-advisory-fixture","version":"1.0.0"}}}`}},
		{"go", "go-proxy", "Go", "github.com/google/uuid", "v1.6.0", []string{"go", "mod", "tidy", "--adopt"}, map[string]string{"go.mod": "module example.com/haa_advisory_fixture\n\ngo 1.26.0\n\nrequire github.com/google/uuid v1.6.0\n", "main.go": "package main\nimport _ \"github.com/google/uuid\"\nfunc init(){panic(\"tidy must not execute\")}\nfunc main(){}\n"}},
		{"cargo", "crates-io", "crates.io", "itoa", "1.0.17", []string{"cargo", "fetch"}, map[string]string{"Cargo.toml": "[package]\nname='haa_advisory_fixture'\nversion='0.1.0'\nedition='2021'\n[dependencies]\nitoa='=1.0.17'\n", "src/main.rs": "fn main(){}\n", "build.rs": "fn main(){panic!(\"fetch must not execute\");}\n"}},
	} {
		for _, malware := range []bool{false, true} {
			mode := fixture.language + map[bool]string{false: "-normal", true: "-controlled-malware"}[malware]
			if !t.Run(mode, func(t *testing.T) {
				started := time.Now()
				root, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
				t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
				t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
				project := filepath.Join(root, "project")
				if err := os.Mkdir(project, 0700); err != nil {
					t.Fatal(err)
				}
				for name, body := range fixture.files {
					target := filepath.Join(project, name)
					if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(target, []byte(body), 0600); err != nil {
						t.Fatal(err)
					}
				}
				cacheRoot := filepath.Join(root, "cache", "heliopause")
				backend, err := storage.NewLocal(ctx, cacheRoot, domain.DefaultStoragePolicy())
				if err != nil {
					t.Fatal(err)
				}
				operation, _ := domain.NewOperationID()
				seedContext := ports.WithStorage(ports.WithStorageOperation(ctx, operation), backend)
				requests := 0
				seed, err := advisory.NewInspector(&advisoryCacheOriginal{}, &http.Client{Transport: advisoryCacheHTTP(func(request *http.Request) (*http.Response, error) {
					requests++
					body := `{"results":[{}]}`
					if malware {
						body = `{"results":[{"vulns":[{"id":"MAL-0000-helox-controlled","modified":"2026-10-20T00:00:00Z"}]}]}`
						if request.Method == http.MethodGet {
							record := map[string]any{"schema_version": "1.9.2", "id": "MAL-0000-helox-controlled", "modified": "2026-10-20T00:00:00Z", "affected": []any{map[string]any{"package": map[string]string{"ecosystem": fixture.ecosystem, "name": fixture.name}, "versions": []string{fixture.version}}}}
							encoded, err := json.Marshal(record)
							if err != nil {
								t.Fatal(err)
							}
							body = string(encoded)
						}
					}
					return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
				})})
				if err != nil {
					t.Fatal(err)
				}
				source, _ := domain.NewSourceID(fixture.source)
				identity, _ := domain.NewResolvedArtifactIdentity(source, fixture.name, fixture.version, "controlled-seed")
				digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
				artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:controlled:advisory-data", 1)
				if err != nil {
					t.Fatal(err)
				}
				_, seedErr := seed.Inspect(seedContext, artifact)
				closeErr := backend.Close()
				if seedErr != nil || closeErr != nil || requests != 1 && !malware || requests != 2 && malware {
					t.Fatal("controlled advisory seed", seedErr, closeErr, requests)
				}
				before := requirementsTargetTree(t, project)
				args := append([]string{"--lang", "en", "--output", "json"}, fixture.args...)
				command := exec.CommandContext(ctx, "/usr/libexec/heliopause/helox", args...)
				command.Dir = project
				var output, diagnostic bytes.Buffer
				command.Stdout, command.Stderr = &output, &diagnostic
				cause := command.Run()
				t.Logf("advisory_public_cli case=%s json=%s diagnostic=%s", mode, bytes.TrimSpace(output.Bytes()), bytes.TrimSpace(diagnostic.Bytes()))
				var report struct {
					packageResult
					Schema    string           `json:"schema_version"`
					Status    string           `json:"operation_status"`
					Promotion string           `json:"promotion_status"`
					Entries   []packageResult  `json:"entries"`
					Details   []advisoryDetail `json:"advisory_details"`
					Plans     []struct {
						Subject subject `json:"subject"`
					} `json:"check_plans"`
					Attribution []struct {
						Source, Package, Version string
						Decision                 string   `json:"entry_policy_decision"`
						Reasons                  []string `json:"entry_policy_reasons"`
					} `json:"dependency_policy_attribution"`
				}

				decoder := json.NewDecoder(&output)
				if err := decoder.Decode(&report); err != nil {
					t.Fatal("production CLI result missing", cause, err)
				}
				if decoder.Decode(new(any)) != io.EOF {
					t.Fatal("multiple result documents")
				}
				var entry packageResult
				deniedPromotion := "NOT_COMPLETED"
				if fixture.language == "npm" {
					if report.Schema != "helox.operation-result/v2" || len(report.Plans) != 1 || len(report.Attribution) != 1 {
						t.Fatal("incomplete npm install subject/attribution", report)
					}
					entry = report.packageResult
					entry.Identity = report.Plans[0].Subject
					attribution := report.Attribution[0]
					if attribution.Source != entry.Identity.Source || attribution.Package != entry.Identity.Name || attribution.Version != entry.Identity.Version {
						t.Fatal("npm attribution subject mismatch", attribution, entry.Identity)
					}
					entry.Policy.Decision, entry.Policy.Reasons = attribution.Decision, attribution.Reasons
					deniedPromotion = "NOT_PERFORMED"
				} else {
					if len(report.Entries) != 1 {
						t.Fatal("incomplete selected public graph", report)
					}
					entry = report.Entries[0]
				}

				if entry.Identity.Source != fixture.source || entry.Identity.Name != fixture.name || entry.Identity.Version != fixture.version {
					t.Fatal("selected identity changed", entry.Identity)
				}
				required := 0
				advisoryComplete := false
				for _, check := range entry.Checks {
					if check.Required {
						required++
						if check.Status != "COMPLETED" {
							t.Fatal("required check incomplete", check)
						}
					}
					advisoryComplete = advisoryComplete || check.ID == "dependency-advisory" && check.Required && check.Status == "COMPLETED"
				}
				if required < 3 || !advisoryComplete {
					t.Fatal("native original/advisory checks missing", entry.Checks)
				}
				if malware {
					if fixture.language != "npm" && entry.Detail != nil {
						report.Details = []advisoryDetail{*entry.Detail}
					}
					if cause == nil || entry.Policy.Decision != "BLOCK" || !strings.Contains(fmt.Sprint(entry.Policy.Reasons), "M13_ADVISORY_MALICIOUS") || report.Promotion != deniedPromotion || !reflect.DeepEqual(before, requirementsTargetTree(t, project)) {
						t.Fatal("controlled denial lost or target changed", cause, report)
					}
					if len(report.Details) != 1 || report.Details[0].Status != "AVAILABLE" || len(report.Details[0].Records) != 1 || report.Details[0].Records[0].ID != "MAL-0000-helox-controlled" || report.Details[0].Records[0].Impact != "AFFECTED" {
						t.Fatal("recorded advisory detail missing", report.Details)
					}
				} else if cause != nil || report.Status != "COMPLETED" || report.Promotion != "COMPLETED" || entry.Policy.Decision != "ALLOW" {
					t.Fatal("normal public dependency refused", cause, report)
				}
				t.Logf("advisory_public_cli_complete case=%s source=%s package=%s version=%s required_checks=%d policy=%s target_preserved=%t seconds=%.3f", mode, fixture.source, fixture.name, fixture.version, required, entry.Policy.Decision, reflect.DeepEqual(before, requirementsTargetTree(t, project)), time.Since(started).Seconds())
			}) {
				return
			}
		}
	}
}
