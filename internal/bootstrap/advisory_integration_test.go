//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
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
	for _, mode := range []string{"fresh-cache", "cache-reuse", "malware-block", "tampered-cache", "force-refresh"} {
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
			malware := mode == "malware-block" || mode == "force-refresh"
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
