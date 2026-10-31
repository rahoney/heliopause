package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func uvCLILockFixture(t *testing.T) string {
	t.Helper()
	project := uvCLIProjectFixture(t)
	if err := os.Remove(filepath.Join(project, ".venv")); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile("../artifact/pypi/testdata/uv-0.13.4-empty-project.uv.lock")
	if err != nil {
		t.Fatal(err)
	}
	// Project the historical format onto the current runtime; keep captured testdata unchanged.
	body = bytes.ReplaceAll(body, []byte("3.14.7"), []byte(runtimeidentity.PythonVersion))
	if err := os.WriteFile(filepath.Join(project, "uv.lock"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	return project
}

// Synthetic approval facts exercise presentation only; actual native approval
// and lock publication are qualified separately through production bootstrap.
func TestUVProjectLockCLIRoutesWithoutVenvAndReportsNoInstallation(t *testing.T) {
	for _, lang := range []string{"en", "ko"} {
		for _, output := range []string{"json", "human"} {
			t.Run(lang+output, func(t *testing.T) {
				project := uvCLILockFixture(t)
				t.Setenv("VIRTUAL_ENV", filepath.Join(project, "unrelated-environment"))
				args := []string{"--lang", lang, "--output", output, "uv", "lock", "--upgrade", "--project", project}
				if lang == "ko" {
					t.Chdir(project)
					args = args[:len(args)-2]
				}
				var out bytes.Buffer
				root, err := New(&out, io.Discard)
				if err != nil {
					t.Fatal(err)
				}
				if source, err := PreparePresentation(root, args); err != nil || source != "pypi" || SelectedPackageClient(root) != UVProjectPackageClient || SelectedPackageClientOperation(root) != "lock" {
					t.Fatal("wrong lock route", source, err)
				}
				recorder := &uvProjectCLIRecorder{t: t}
				if err := AddUVProject(root, recorder); err != nil {
					t.Fatal(err)
				}
				root.SetArgs(args)
				if err := root.ExecuteContext(context.Background()); err != nil {
					t.Fatal(err, out.String())
				}
				if len(recorder.requests) != 1 {
					t.Fatal("lock request repeated or missing")
				}
				request := recorder.requests[0]
				if request.Intent != artifactpypi.UVProjectLockUpgrade || request.ProjectRoot != project || request.Install.Target().String() != project || request.Install.Mode() != domain.InstallPythonProjectControls || request.Selector != "" || request.Adopt {
					t.Fatal("lock intent became environment installation", request)
				}
				if output == "json" {
					var report struct {
						Schema       string `json:"schema_version"`
						Operation    string
						Status       string              `json:"operation_status"`
						Lock         string              `json:"lock_update_status"`
						Installation string              `json:"installation_status"`
						Records      *machineVerifiedSet `json:"verified_set"`
					}
					if json.Unmarshal(out.Bytes(), &report) != nil || report.Schema != "helox.project-lock-update-result/v1" || report.Operation != "LOCK_UPDATE" || report.Status != "COMPLETED" || report.Lock != "COMPLETED" || report.Installation != "NOT_REQUESTED" || report.Records == nil || report.Records.ManifestID == "" {
						t.Fatal("lock result claims installation or lacks completed records", out.String())
					}
				} else if json.Valid(out.Bytes()) || !strings.Contains(out.String(), map[string]string{"en": "Package installation: not requested", "ko": "패키지 설치: 요청하지 않음"}[lang]) {
					t.Fatal("human lock-only result unclear", out.String())
				}
				if _, err := os.Lstat(filepath.Join(project, ".venv")); !os.IsNotExist(err) {
					t.Fatal("CLI lock route provisioned venv", err)
				}
			})
		}
	}
}

func TestUVProjectLockCLIRejectsUnsupportedIntentAndUnsafeOriginalBeforeApplication(t *testing.T) {
	for _, name := range []string{"missing upgrade", "disabled upgrade", "duplicate upgrade", "duplicate project", "target", "adopt", "argument", "missing lock", "shared lock", "symlink lock", "malformed lock", "empty project"} {
		t.Run(name, func(t *testing.T) {
			project := uvCLILockFixture(t)
			args := []string{"uv", "lock", "--upgrade", "--project", project}
			path := filepath.Join(project, "uv.lock")
			switch name {
			case "missing upgrade":
				args = []string{"uv", "lock", "--project", project}
			case "disabled upgrade":
				args[2] = "--upgrade=false"
			case "duplicate upgrade":
				args = append(args, "--upgrade=false")
			case "duplicate project":
				args = append(args, "--project", project)
			case "target":
				args = append(args, "--target", project)
			case "adopt":
				args = append(args, "--adopt")
			case "argument":
				args = append(args, "packaging")
			case "missing lock":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "shared lock":
				if err := os.Link(path, filepath.Join(project, "foreign-lock-copy")); err != nil {
					t.Fatal(err)
				}
			case "symlink lock":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("pyproject.toml", path); err != nil {
					t.Fatal(err)
				}
			case "malformed lock":
				if err := os.WriteFile(path, []byte("version=999"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "empty project":
				args[len(args)-1] = ""
			}
			root, err := New(io.Discard, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := PreparePresentation(root, args); err == nil {
				t.Fatal("invalid lock request accepted", name)
			}
		})
	}
}

func TestUVProjectLockCLINeverClaimsCompletionWithoutPublicationRecords(t *testing.T) {
	for _, missing := range []bool{false, true} {
		project := uvCLILockFixture(t)
		var out bytes.Buffer
		root, err := New(&out, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		args := []string{"--output", "json", "uv", "lock", "--upgrade", "--project", project}
		if _, err := PreparePresentation(root, args); err != nil {
			t.Fatal(err)
		}
		recorder := &uvProjectCLIRecorder{t: t, missing: missing}
		if !missing {
			recorder.cause = errors.New("synthetic lock publication failure")
		}
		if err := AddUVProject(root, recorder); err != nil {
			t.Fatal(err)
		}
		root.SetArgs(args)
		if err := root.ExecuteContext(context.Background()); err == nil {
			t.Fatal("lock failure returned success")
		}
		var report struct {
			Status       string              `json:"operation_status"`
			Lock         string              `json:"lock_update_status"`
			Installation string              `json:"installation_status"`
			Records      *machineVerifiedSet `json:"verified_set"`
		}
		if json.Unmarshal(out.Bytes(), &report) != nil || report.Status == "COMPLETED" || report.Lock == "COMPLETED" || report.Installation != "NOT_REQUESTED" || report.Records != nil {
			t.Fatal("failed lock operation reported completion", out.String())
		}
	}
}
