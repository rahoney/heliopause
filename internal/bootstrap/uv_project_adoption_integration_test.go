//go:build linux

package bootstrap_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rahoney/heliopause/internal/bootstrap"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestLinuxUVProjectNativeAdoptionCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_PROJECT_CLI_INTEGRATION") != "1" {
		t.Skip("requires authenticated pinned UV project CLI environment")
	}
	ctx, root, cache, donor := uvSyncActualEnvironment(t)
	source := filepath.Join(root, "requirements.txt")
	if err := os.WriteFile(source, []byte("idna==3.10\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	seed := runActualUVSync(t, ctx, source, donor)
	if len(seed.Entries) != 1 || seed.Entries[0].Identity.Name != "idna" {
		t.Fatal("native seed missing")
	}
	payload := actualUVSyncPayload(t, cache, seed)
	wheel, err := os.ReadFile(filepath.Join(payload, "artifacts", seed.Entries[0].Digest.Value+".whl"))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(wheel)
	if hex.EncodeToString(sum[:]) != seed.Entries[0].Digest.Value {
		t.Fatal("native seed digest mismatch")
	}
	for _, operation := range []string{"add", "sync"} {
		t.Run(operation, func(t *testing.T) {
			project := filepath.Join(root, "native-"+operation)
			if err := os.Mkdir(project, 0o700); err != nil {
				t.Fatal(err)
			}
			venv := filepath.Join(project, ".venv")
			uvNativeOriginalFixture(t, ctx, wheel, venv)
			deps := "[]"
			if operation == "sync" {
				deps = "['packaging==25.0']"
			}
			manifest := []byte("[project]\nname='haa-native-project'\nversion='0.1.0'\nrequires-python='==" + sandbox.PinnedPythonRuntime().PythonVersion + "'\ndependencies=" + deps + "\n")
			if err := os.WriteFile(filepath.Join(project, "pyproject.toml"), manifest, 0o644); err != nil {
				t.Fatal(err)
			}
			note := filepath.Join(venv, "user-note")
			if err := os.WriteFile(note, []byte("preserved"), 0o600); err != nil {
				t.Fatal(err)
			}
			args := []string{"--lang", "en", "--output", "json", "uv", operation, "--project", project}
			if operation == "add" {
				args = append(args, "packaging==25.0")
			}
			before := uvCLIRegularFiles(t, project)
			evidenceBefore := uvCLIRegularFiles(t, filepath.Join(cache, "heliopause", "evidence"))
			var out, diagnostic bytes.Buffer
			if err := bootstrap.Run(ctx, args, &out, &diagnostic); err == nil || !bytes.Contains(out.Bytes(), []byte("UPDATE_ADOPTION_REQUIRED")) {
				t.Fatal("unowned native accepted", err, out.String(), diagnostic.String())
			}
			if !reflect.DeepEqual(before, uvCLIRegularFiles(t, project)) || !reflect.DeepEqual(evidenceBefore, uvCLIRegularFiles(t, filepath.Join(cache, "heliopause", "evidence"))) {
				t.Fatal("refused adoption mutated original or Evidence")
			}
			out.Reset()
			diagnostic.Reset()
			if err := bootstrap.Run(ctx, append(args, "--adopt"), &out, &diagnostic); err != nil {
				t.Fatal("native adoption failed", err, out.String(), diagnostic.String())
			}
			var report uvSyncActualReport
			if json.Unmarshal(out.Bytes(), &report) != nil || report.Status != "COMPLETED" || report.Promotion != "COMPLETED" || report.Policy.Decision != "ALLOW" || report.Verified.Count != 1 || len(report.Entries) != 1 || report.Entries[0].Identity.Name != "packaging" {
				t.Fatal("native adoption incomplete", out.String())
			}
			for _, check := range report.Entries[0].Checks {
				if check.Required && check.Status != "COMPLETED" {
					t.Fatal("current required inspection incomplete")
				}
			}
			if len(report.Entries[0].Evidence) < 3 {
				t.Fatal("current Evidence missing")
			}
			payload := actualUVSyncPayload(t, cache, report)
			body, err := os.ReadFile(filepath.Join(payload, "manifest.json"))
			var record struct {
				Controls []struct {
					Name    string
					Content []byte
				} `json:"project_controls"`
			}
			if err != nil || json.Unmarshal(body, &record) != nil || len(record.Controls) != 7 {
				t.Fatal("adoption provenance missing", err, string(body))
			}
			proof := false
			for _, control := range record.Controls {
				if control.Name == "pypi-update-native-ownership.json" {
					proof = len(control.Content) > 0
				}
				if control.Name == "pyproject.toml" || control.Name == "uv.lock" {
					current, err := os.ReadFile(filepath.Join(project, control.Name))
					if err != nil || !bytes.Equal(current, control.Content) {
						t.Fatal("approved control publication mismatch", err)
					}
				}
			}
			if !proof {
				t.Fatal("authenticated complete-original proof missing")
			}
			if _, err := os.Lstat(filepath.Join(venv, "lib/python3.14/site-packages/idna-3.10.dist-info/METADATA")); !os.IsNotExist(err) {
				t.Fatal("unselected authenticated original retained", err)
			}
			if body, err := os.ReadFile(note); err != nil || string(body) != "preserved" {
				t.Fatal("user file changed", err)
			}
			if _, err := os.Lstat(filepath.Join(project, "pypi-update-native-ownership.json")); !os.IsNotExist(err) {
				t.Fatal("proof published as project file", err)
			}
			t.Logf("actual_uv_project_adoption=PASS operation=%s native unselected idna authenticated then retired; current packaging inspected; seven immutable controls; no-adopt unchanged", operation)
		})
	}
}
