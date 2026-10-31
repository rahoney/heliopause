//go:build linux && realcorpus

package bootstrap_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/bootstrap"
)

func TestPNPMProjectCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_PNPM_PROJECT_CLI_INTEGRATION") != "1" {
		t.Skip("requires installed authenticated PNPM CLI test client")
	}
	if os.Geteuid() == 0 {
		t.Fatal("PNPM CLI actual requires ordinary UID")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(root, "cache")
	t.Setenv("XDG_CACHE_HOME", cache)
	operations := map[string]bool{}
	for _, fixture := range []struct {
		name, manifest, code string
		entries              int
	}{
		{"single", `{"scripts":{"install":"exit 99"},"dependencies":{"is-number":"6.0.0"}}`, "node_modules/is-number/package.json", 1},
		{"peer", `{"scripts":{"install":"exit 99"},"dependencies":{"react":"18.3.1","use-sync-external-store":"1.2.0"}}`, "node_modules/react/package.json", 4},
		{"empty", `{}`, "", 0},
	} {
		if !t.Run(fixture.name, func(t *testing.T) {
			target := filepath.Join(root, fixture.name)
			if err := os.Mkdir(target, 0o700); err != nil {
				t.Fatal(err)
			}
			manifest := filepath.Join(target, "package.json")
			if err := os.WriteFile(manifest, []byte(fixture.manifest), 0o600); err != nil {
				t.Fatal(err)
			}
			note := filepath.Join(target, "user-note")
			if err := os.WriteFile(note, []byte("preserve user file\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			apply := func(operation, selector string, adopt bool) {
				args := []string{"--lang", "ko", "--output", "json", "pnpm", operation, "--target", target}
				if adopt {
					args = append(args, "--adopt")
				}
				if selector != "" {
					args = append(args, "--", selector)
				}
				var out, diagnostic bytes.Buffer
				if err := bootstrap.Run(ctx, args, &out, &diagnostic); err != nil {
					t.Fatalf("actual pnpm %s failed: %v stdout=%s diagnostics=%s", operation, err, out.String(), diagnostic.String())
				}
				var report struct {
					Schema    string                    `json:"schema_version"`
					Operation string                    `json:"operation"`
					Status    string                    `json:"operation_status"`
					Promotion string                    `json:"promotion_status"`
					Source    string                    `json:"source_id"`
					Policy    struct{ Decision string } `json:"policy"`
					Entries   []struct {
						Run    string `json:"run_id"`
						Checks []struct {
							Required   bool
							Capability string
							Status     string `json:"execution_status"`
						}
						Evidence []json.RawMessage `json:"evidence_references"`
					} `json:"entries"`
					Records struct {
						Manifest string `json:"manifest_id"`
						SBOM     string `json:"sbom_handle"`
						Staging  string `json:"staging_handle"`
						Count    int    `json:"entry_count"`
					} `json:"verified_set"`
				}
				if err := json.Unmarshal(out.Bytes(), &report); err != nil {
					t.Fatal("non-JSON CLI stdout", err, out.String())
				}
				wantOp, wantSchema := "INSTALL", "helox.project-install-result/v1"
				if operation == "update" {
					wantOp, wantSchema = "UPDATE", "helox.project-update-result/v1"
				}
				if report.Schema != wantSchema || report.Operation != wantOp || report.Status != "COMPLETED" || report.Promotion != "COMPLETED" || report.Source != "npm" || report.Policy.Decision != "ALLOW" || len(report.Entries) != fixture.entries || report.Records.Count != fixture.entries || !strings.HasPrefix(report.Records.Staging, "project-cache:") || report.Records.SBOM != report.Records.Staging+":sbom.cdx.json" {
					t.Fatal("incomplete actual CLI result", out.String())
				}
				for _, entry := range report.Entries {
					if entry.Run == "" || len(entry.Evidence) == 0 || len(entry.Checks) < 3 {
						t.Fatal("missing actual per-entry facts", out.String())
					}
					for _, check := range entry.Checks {
						if check.Required && (check.Status != "COMPLETED" || check.Capability != "SUPPORTED") {
							t.Fatal("incomplete required check", out.String())
						}
					}
				}
				body, err := os.ReadFile(filepath.Join(target, ".heliopause/manifest.json"))
				if err != nil {
					t.Fatal(err)
				}
				var record struct {
					Manifest  string `json:"manifest_id"`
					Operation string `json:"operation_id"`
				}
				if json.Unmarshal(body, &record) != nil || record.Manifest != report.Records.Manifest || record.Operation == "" || operations[record.Operation] {
					t.Fatal("actual Manifest binding or fresh operation differs", string(body))
				}
				operations[record.Operation] = true
				handle := strings.TrimPrefix(report.Records.Staging, "project-cache:")
				cachedManifest, err := os.ReadFile(filepath.Join(cache, "heliopause/pnpm-project-cache", handle, "records/manifest.json"))
				if err != nil || !bytes.Equal(cachedManifest, body) {
					t.Fatal("reported cache Manifest differs from target", err)
				}
				cached, err := os.ReadFile(filepath.Join(cache, "heliopause/pnpm-project-cache", handle, "records/sbom.cdx.json"))
				if err != nil {
					t.Fatal(err)
				}
				published, err := os.ReadFile(filepath.Join(target, ".heliopause/sbom.cdx.json"))
				if err != nil || !bytes.Equal(cached, published) {
					t.Fatal("reported SBOM differs from publication", err)
				}
			}
			apply("install", "", false)
			controlBefore, err := os.Lstat(manifest)
			if err != nil {
				t.Fatal(err)
			}
			var codeBefore os.FileInfo
			if fixture.code != "" {
				codeBefore, err = os.Stat(filepath.Join(target, fixture.code))
				if err != nil {
					t.Fatal(err)
				}
			}
			apply("install", "", false)
			controlAfter, err := os.Lstat(manifest)
			if err != nil || !os.SameFile(controlBefore, controlAfter) || controlAfter.Mode() != 0o600 {
				t.Fatal("CLI repeat changed original control inode/mode", err)
			}
			if fixture.code != "" {
				after, err := os.Stat(filepath.Join(target, fixture.code))
				if err != nil || !os.SameFile(codeBefore, after) {
					t.Fatal("CLI repeat rewrote native payload", err)
				}
			}
			if fixture.name == "single" {
				apply("add", "is-number@7.0.0", false)
				apply("update", "is-number", false)
				body, err := os.ReadFile(filepath.Join(target, fixture.code))
				if err != nil {
					t.Fatal(err)
				}
				var pkg struct{ Version string }
				if json.Unmarshal(body, &pkg) != nil || pkg.Version != "7.0.0" {
					t.Fatal("CLI add/update selected another version")
				}
				body, err = os.ReadFile(manifest)
				if err != nil {
					t.Fatal(err)
				}
				var doc map[string]json.RawMessage
				if err := json.Unmarshal(body, &doc); err != nil {
					t.Fatal(err)
				}
				doc["description"] = json.RawMessage(`"explicitly edited controls"`)
				body, err = json.Marshal(doc)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(manifest, body, 0o600); err != nil {
					t.Fatal(err)
				}
				var out, diagnostic bytes.Buffer
				if err := bootstrap.Run(ctx, []string{"--lang", "en", "--output", "json", "pnpm", "install", "--target", target}, &out, &diagnostic); err == nil || !bytes.Contains(out.Bytes(), []byte(`"operation_status":"FAILED"`)) {
					t.Fatal("implicit changed-control enrollment accepted", err, out.String())
				}
				if current, err := os.ReadFile(manifest); err != nil || !bytes.Equal(current, body) {
					t.Fatal("rejected edited control was changed", err)
				}
				apply("install", "", true)
			}
			if body, err := os.ReadFile(note); err != nil || string(body) != "preserve user file\n" {
				t.Fatal("unmanaged user file changed", err)
			}
			t.Logf("pnpm_actual_cli fixture=%s entries=%d complete_set=true fresh_operation=true records_bound=true repeat_inode_and_mode=true root_scripts=NOT_EXECUTED user_files_preserved=true", fixture.name, fixture.entries)
		}) {
			t.FailNow()
		}
	}
	// Explicit enrollment cannot bypass independently unproven native payload.
	native := filepath.Join(root, "unmanaged")
	if err := os.Mkdir(native, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(native, "package.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(native, "node_modules"), 0o700); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(native, "node_modules/user-file")
	if err := os.WriteFile(foreign, []byte("unproven native bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, diagnostic bytes.Buffer
	if err := bootstrap.Run(ctx, []string{"--lang", "en", "--output", "json", "pnpm", "install", "--target", native, "--adopt"}, &out, &diagnostic); err == nil || !bytes.Contains(out.Bytes(), []byte(`"promotion_status":"NOT_COMPLETED"`)) {
		t.Fatal("native adoption bypass accepted", err, out.String())
	}
	if body, err := os.ReadFile(foreign); err != nil || string(body) != "unproven native bytes" {
		t.Fatal("unproven native data deleted", err)
	}
	t.Logf("pnpm_actual_cli_fresh_operations=%d implicit_control_enrollment=REFUSED unproven_native_adoption=REFUSED", len(operations))
}
