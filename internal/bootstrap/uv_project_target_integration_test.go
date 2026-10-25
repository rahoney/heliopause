//go:build linux

package bootstrap_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/bootstrap"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestLinuxUVProjectExplicitTargetCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_PROJECT_CLI_INTEGRATION") != "1" {
		t.Skip("requires authenticated pinned UV project CLI environment")
	}
	ctx, root, cache, venv := uvSyncActualEnvironment(t)
	project := filepath.Join(root, "separate-project")
	if err := os.Mkdir(project, 0o700); err != nil {
		t.Fatal(err)
	}
	body := []byte("[project]\nname='haa-explicit-target'\nversion='0.1.0'\nrequires-python='==" + sandbox.PinnedPythonRuntime().PythonVersion + "'\ndependencies=[]\n")
	if err := os.WriteFile(filepath.Join(project, "pyproject.toml"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VIRTUAL_ENV", filepath.Join(root, "wrong-active-environment"))
	var out, diagnostic bytes.Buffer
	if err := bootstrap.Run(ctx, []string{"--lang", "en", "--output", "json", "uv", "sync", "--project", project, "--target", venv}, &out, &diagnostic); err != nil {
		t.Fatal("explicit existing venv was not selected", err, out.String(), diagnostic.String())
	}
	var report uvSyncActualReport
	if json.Unmarshal(out.Bytes(), &report) != nil || report.Status != "COMPLETED" || report.Promotion != "COMPLETED" || !report.DependencyFree || report.Verified.Count != 0 {
		t.Fatal("incomplete explicit target result", out.String())
	}
	payload := actualUVSyncPayload(t, cache, report)
	document, err := os.ReadFile(filepath.Join(payload, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Target struct{ Path string } `json:"target"`
	}
	if json.Unmarshal(document, &record) != nil || record.Target.Path != venv {
		t.Fatal("approved record target differs from explicit venv", string(document))
	}
	if _, err := os.Lstat(filepath.Join(project, ".venv")); !os.IsNotExist(err) {
		t.Fatal("explicit target provisioned default venv", err)
	}
	t.Log("actual_uv_project_cli=PASS explicit external target overrides absent project/.venv and unrelated VIRTUAL_ENV; approved record target exact")
}
