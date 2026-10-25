//go:build linux

package bootstrap_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/bootstrap"
)

func TestLinuxUVProjectLockCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_PROJECT_CLI_INTEGRATION") != "1" {
		t.Skip("requires authenticated pinned UV project CLI environment")
	}
	ctx, root, cache, unrelatedVenv := uvSyncActualEnvironment(t)
	venvBefore := uvCLIRegularFiles(t, unrelatedVenv)
	project := filepath.Join(root, "lock-cli-project")
	if err := os.Mkdir(project, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile("../artifact/pypi/testdata/uv-0.13.4-empty-project.pyproject.toml")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := os.ReadFile("../artifact/pypi/testdata/uv-0.13.4-empty-project.uv.lock")
	if err != nil {
		t.Fatal(err)
	}
	manifest = bytes.Replace(manifest, []byte("dependencies = []"), []byte("dependencies = ['packaging==25.0']"), 1)
	manifestPath := filepath.Join(project, "pyproject.toml")
	if err := os.WriteFile(manifestPath, manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "uv.lock"), lock, 0o600); err != nil {
		t.Fatal(err)
	}
	note := filepath.Join(project, "user-note")
	if err := os.WriteFile(note, []byte("preserved user data"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VIRTUAL_ENV", unrelatedVenv)
	seen := map[string]bool{}
	apply := func(output string, explicitProject bool, count int) {
		t.Helper()
		before, err := os.Lstat(manifestPath)
		if err != nil {
			t.Fatal(err)
		}
		args := []string{"--lang", "ko", "--output", output, "uv", "lock", "--upgrade"}
		if explicitProject {
			args = append(args, "--project", project)
		}
		var out, diagnostic bytes.Buffer
		if err := bootstrap.Run(ctx, args, &out, &diagnostic); err != nil {
			t.Fatal("actual lock-only CLI failed", err, out.String(), diagnostic.String())
		}
		if output == "json" {
			var report struct {
				uvSyncActualReport
				Lock         string `json:"lock_update_status"`
				Installation string `json:"installation_status"`
			}
			if json.Unmarshal(out.Bytes(), &report) != nil || report.Schema != "helox.project-lock-update-result/v1" || report.Status != "COMPLETED" || report.Promotion != "COMPLETED" || report.Lock != "COMPLETED" || report.Installation != "NOT_REQUESTED" || report.Policy.Decision != "ALLOW" || report.Verified.Count != count || len(report.Entries) != count || report.DependencyFree != (count == 0) {
				t.Fatal("actual lock-only result incomplete or claims installation", out.String())
			}
			for _, entry := range report.Entries {
				if seen[entry.Run] || len(entry.Evidence) < 3 || len(entry.Checks) < 3 {
					t.Fatal("lock-only CLI skipped fresh full inspection")
				}
				seen[entry.Run] = true
				for _, check := range entry.Checks {
					if check.Required && check.Status != "COMPLETED" {
						t.Fatal("lock-only required check incomplete")
					}
				}
			}
			payload := actualUVSyncPayload(t, cache, report.uvSyncActualReport)
			body, err := os.ReadFile(filepath.Join(payload, "manifest.json"))
			var records struct {
				Target   struct{ Path string } `json:"target"`
				Controls []struct {
					Name    string
					Content []byte
				} `json:"project_controls"`
			}
			if err != nil || json.Unmarshal(body, &records) != nil || records.Target.Path != project || len(records.Controls) != 6 {
				t.Fatal("actual lock-only records incomplete", err)
			}
			for _, control := range records.Controls {
				if control.Name == "pyproject.toml" || control.Name == "uv.lock" {
					current, err := os.ReadFile(filepath.Join(project, control.Name))
					if err != nil || !bytes.Equal(current, control.Content) {
						t.Fatal("lock-only approved controls differ", err)
					}
				}
			}
		} else if json.Valid(out.Bytes()) || !strings.Contains(out.String(), "잠금파일 업데이트:") || !strings.Contains(out.String(), "패키지 설치: 요청하지 않음") || !strings.Contains(out.String(), "Manifest: sha256:") {
			t.Fatal("actual human lock-only result unclear", out.String())
		}
		current, err := os.Lstat(manifestPath)
		body, readErr := os.ReadFile(manifestPath)
		if err != nil || readErr != nil || !os.SameFile(before, current) || before.Mode() != current.Mode() || !bytes.Equal(body, manifest) {
			t.Fatal("lock-only CLI changed manifest", err, readErr)
		}
		if body, err := os.ReadFile(note); err != nil || string(body) != "preserved user data" {
			t.Fatal("lock-only CLI changed user data", err)
		}
		if !reflect.DeepEqual(venvBefore, uvCLIRegularFiles(t, unrelatedVenv)) {
			t.Fatal("lock-only CLI changed unrelated active venv")
		}
		if info, err := os.Lstat(filepath.Join(project, "uv.lock")); err != nil || info.Mode().Perm() != 0o600 {
			t.Fatal("lock-only CLI changed original lock mode", err)
		}
		t.Logf("actual_uv_project_lock_cli=PASS output=%s explicit_project=%t entries=%d lock-only completion/installation NOT_REQUESTED/current records", output, explicitProject, count)
	}
	apply("json", true, 1)
	if _, err := os.Lstat(filepath.Join(project, ".venv")); !os.IsNotExist(err) {
		t.Fatal("lock-only CLI provisioned default venv", err)
	}
	apply("json", true, 1)
	if err := os.Mkdir(filepath.Join(project, ".venv"), 0o700); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(project, ".venv", "foreign-user-data")
	if err := os.WriteFile(foreign, []byte("not a venv"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)
	apply("human", false, 1)
	manifest = bytes.Replace(manifest, []byte("dependencies = ['packaging==25.0']"), []byte("dependencies = []"), 1)
	if err := os.WriteFile(manifestPath, manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	apply("json", false, 0)
	if body, err := os.ReadFile(foreign); err != nil || string(body) != "not a venv" {
		t.Fatal("lock-only CLI changed arbitrary project venv", err)
	}
}
