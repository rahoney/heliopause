//go:build linux

package bootstrap_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/bootstrap"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestLinuxUVProjectCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_PROJECT_CLI_INTEGRATION") != "1" {
		t.Skip("requires authenticated pinned UV project CLI environment")
	}
	ctx, root, cache, oldVenv := uvSyncActualEnvironment(t)
	project := filepath.Join(root, "project")
	if err := os.Mkdir(project, 0o700); err != nil {
		t.Fatal(err)
	}
	venv := filepath.Join(project, ".venv")
	if err := os.Rename(oldVenv, venv); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(project, "pyproject.toml")
	body := []byte("[project]\nname='haa-cli-project'\nversion='0.1.0'\nrequires-python='==" + sandbox.PinnedPythonRuntime().PythonVersion + "'\ndependencies=[]\n")
	if err := os.WriteFile(manifest, body, 0o644); err != nil {
		t.Fatal(err)
	}
	note := filepath.Join(venv, "user-note")
	if err := os.WriteFile(note, []byte("preserve user data"), 0o600); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	apply := func(operation, selector string, count int) {
		t.Helper()
		args := []string{"--lang", "ko", "--output", "json", "uv", operation, "--project", project}
		if selector != "" {
			args = append(args, selector)
		}
		var out, diagnostic bytes.Buffer
		if err := bootstrap.Run(ctx, args, &out, &diagnostic); err != nil {
			t.Fatalf("actual UV project CLI %s failed: %v stdout=%s diagnostics=%s", operation, err, out.String(), diagnostic.String())
		}
		var report uvSyncActualReport
		if json.Unmarshal(out.Bytes(), &report) != nil || report.Status != "COMPLETED" || report.Promotion != "COMPLETED" || report.Policy.Decision != "ALLOW" || report.Verified.Manifest == "" || report.Verified.Count != count || len(report.Entries) != count || report.DependencyFree != (count == 0) {
			t.Fatal("incomplete actual project CLI result", out.String())
		}
		wantSchema := "helox.project-update-result/v1"
		if operation == "add" {
			wantSchema = "helox.project-install-result/v1"
		}
		if report.Schema != wantSchema {
			t.Fatal("native intent report schema changed", out.String())
		}
		for _, entry := range report.Entries {
			if seen[entry.Run] || len(entry.Evidence) < 3 || len(entry.Checks) < 3 {
				t.Fatal("project CLI skipped fresh full inspection")
			}
			seen[entry.Run] = true
			for _, check := range entry.Checks {
				if check.Required && check.Status != "COMPLETED" {
					t.Fatal("required check incomplete")
				}
			}
		}
		payload := actualUVSyncPayload(t, cache, report)
		var records struct {
			Controls []struct {
				Name, SHA256 string
				Content      []byte
			} `json:"project_controls"`
		}
		document, err := os.ReadFile(filepath.Join(payload, "manifest.json"))
		if err != nil || json.Unmarshal(document, &records) != nil || len(records.Controls) != 6 {
			t.Fatal("actual project provenance missing", err)
		}
		for _, control := range records.Controls {
			if control.Name == "pyproject.toml" || control.Name == "uv.lock" {
				current, err := os.ReadFile(filepath.Join(project, control.Name))
				if err != nil || !bytes.Equal(current, control.Content) {
					t.Fatal("actual CLI published controls differ from approved records", err)
				}
			}
		}
		if current, err := os.ReadFile(note); err != nil || string(current) != "preserve user data" {
			t.Fatal("actual CLI changed user data", err)
		}
		t.Logf("actual_uv_project_cli=PASS operation=%s entries=%d manifest=%s", operation, count, report.Verified.Manifest)
	}
	apply("add", "packaging==24.2", 1)
	apply("add", "packaging==25.0", 1)
	apply("sync", "", 1)
	apply("add", "idna==3.10", 2)
	// Same current-directory/default target route, with human presentation.
	{
		t.Chdir(project)
		var out, diagnostic bytes.Buffer
		if err := bootstrap.Run(ctx, []string{"--lang", "ko", "--output", "human", "uv", "sync"}, &out, &diagnostic); err != nil || json.Valid(out.Bytes()) || !strings.Contains(out.String(), "Python") {
			t.Fatal("actual human/default project route", err, out.String(), diagnostic.String())
		}
		t.Log("actual_uv_project_cli=PASS human/default-project/default-venv same-version sync")
	}
	current, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(current), "\n")
	start, end := -1, -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "dependencies") {
			start = i
			for j := i; j < len(lines); j++ {
				if strings.Contains(lines[j], "]") {
					end = j
					break
				}
			}
			break
		}
	}
	if start < 0 || end < start {
		t.Fatal("native dependency array missing")
	}
	lines = append(append(lines[:start], "dependencies = []"), lines[end+1:]...)
	if err := os.WriteFile(manifest, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	apply("sync", "", 0)
	apply("sync", "", 0)
	// A native-looking unowned distribution must not silently become removable.
	directory := filepath.Join(venv, "lib/python3.14/site-packages/unknown-1.0.dist-info")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	metadata := filepath.Join(directory, "METADATA")
	unknown := []byte("Metadata-Version: 2.4\nName: unknown\nVersion: 1.0\n\n")
	if err := os.WriteFile(metadata, unknown, 0o600); err != nil {
		t.Fatal(err)
	}
	var out, diagnostic bytes.Buffer
	err = bootstrap.Run(ctx, []string{"--lang", "ko", "--output", "json", "uv", "sync", "--project", project}, &out, &diagnostic)
	var failed struct {
		Status string                `json:"operation_status"`
		Error  struct{ Code string } `json:"error"`
	}
	if err == nil || json.Unmarshal(out.Bytes(), &failed) != nil || failed.Status == "COMPLETED" || failed.Error.Code != "UPDATE_ADOPTION_REQUIRED" {
		t.Fatal("unowned original CLI refusal", err, out.String(), diagnostic.String())
	}
	if current, err := os.ReadFile(metadata); err != nil || !bytes.Equal(current, unknown) {
		t.Fatal("unowned original changed", err)
	}
	t.Log("actual_uv_project_cli=PASS unowned original UPDATE_ADOPTION_REQUIRED; no implicit adoption")
}
