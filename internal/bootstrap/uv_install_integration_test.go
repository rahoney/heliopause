package bootstrap_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/bootstrap"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// Like the NPM CLI test, this runs as the installed authenticated bootstrap
// test client. It exercises the production composition rather than test ports.
func TestLinuxUVInstallCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_INSTALL_CLI_INTEGRATION") != "1" {
		t.Skip("requires pinned native runtime and authenticated policy helper")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" || os.Geteuid() == 0 {
		t.Fatal("requires ordinary-user Linux amd64")
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(root, "cache")
	t.Setenv("XDG_CACHE_HOME", cache)
	venv := filepath.Join(root, "venv")
	site := filepath.Join(venv, "lib", "python3.14", "site-packages")
	for _, dir := range []string{site, filepath.Join(venv, "bin")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	cfg := []byte("home = /usr/local/bin\nimplementation = CPython\nuv = 0.13.4\nversion_info = " + sandbox.PinnedPythonRuntime().PythonVersion + "\ninclude-system-site-packages = false\n")
	if err := os.WriteFile(filepath.Join(venv, "pyvenv.cfg"), cfg, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var originalWheel []byte
	userFile := filepath.Join(venv, "user-note.txt")
	if err := os.WriteFile(userFile, []byte("preserve user data"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		name, spec, version, operation, intent string
		flags                                  []string
	}{
		{"fresh", "idna==3.10", "3.10", "INSTALL", "", nil},
		{"retained", "idna", "3.10", "INSTALL", "install", nil},
		{"root-upgrade", "idna==3.11", "3.11", "UPDATE", "upgrade-package", []string{"--upgrade-package", "idna"}},
		{"closure-noop", "idna==3.11", "3.11", "UPDATE", "upgrade", []string{"-U"}},
		{"new-root", "packaging==25.0", "25.0", "INSTALL", "install", nil},
	} {
		args := append([]string{"--output", "json", "--lang", "ko", "uv", "pip", "install", step.spec, "--target", venv}, step.flags...)
		var stdout, stderr bytes.Buffer
		if err := bootstrap.Run(ctx, args, &stdout, &stderr); err != nil {
			t.Fatalf("actual UV %s failed: %v stdout=%s stderr=%s", step.name, err, stdout.String(), stderr.String())
		}
		var result struct {
			Schema    string                    `json:"schema_version"`
			Operation string                    `json:"operation"`
			Status    string                    `json:"operation_status"`
			Promotion string                    `json:"promotion_status"`
			Policy    struct{ Decision string } `json:"policy"`
			Verified  struct {
				Manifest string `json:"manifest_id"`
				Count    int    `json:"entry_count"`
			} `json:"verified_set"`
			Checks []struct {
				Required bool
				Status   string `json:"execution_status"`
			} `json:"checks"`
			Evidence []struct{ ID, Handle string } `json:"evidence_references"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatal(err, stdout.String())
		}
		if result.Schema != "helox.operation-result/v1" || result.Operation != step.operation || result.Status != "COMPLETED" || result.Promotion != "COMPLETED" || result.Policy.Decision != "ALLOW" || result.Verified.Count != 1 || result.Verified.Manifest == "" || len(result.Checks) < 3 || len(result.Evidence) < 3 {
			t.Fatal("incomplete UV CLI operation", step.name, stdout.String())
		}
		if step.name == "fresh" {
			directory := filepath.Join(cache, "heliopause", "staging", result.Verified.Manifest, "artifacts")
			entries, err := os.ReadDir(directory)
			if err != nil || len(entries) != 1 || !strings.HasSuffix(entries[0].Name(), ".whl") {
				t.Fatal("authentic original wheel unavailable", err)
			}
			originalWheel, err = os.ReadFile(filepath.Join(directory, entries[0].Name()))
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(originalWheel)
			if entries[0].Name() != hex.EncodeToString(digest[:])+".whl" {
				t.Fatal("original staged wheel digest mismatch")
			}
		}
		for _, check := range result.Checks {
			if check.Required && check.Status != "COMPLETED" {
				t.Fatal("required check incomplete", stdout.String())
			}
		}
		project := "idna"
		if step.name == "new-root" {
			project = "packaging"
		}
		for _, path := range []string{filepath.Join(site, project, "__init__.py"), filepath.Join(site, project+"-"+step.version+".dist-info", "METADATA"), filepath.Join(venv, ".heliopause", "pypi-transaction.json")} {
			if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
				t.Fatal("approved file missing", path, err)
			}
		}
		body, err := os.ReadFile(filepath.Join(cache, "heliopause", "staging", result.Verified.Manifest, "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		var manifest struct {
			Resolver struct {
				Runtime string `json:"runtime_identity"`
			} `json:"resolver"`
			Controls []struct {
				Name    string `json:"name"`
				Content []byte `json:"content"`
			} `json:"project_controls"`
		}
		if err := json.Unmarshal(body, &manifest); err != nil || manifest.Resolver.Runtime != sandbox.UVResolverRuntimeIdentity() {
			t.Fatal("solver identity changed", err, string(body))
		}
		count := 0
		if step.intent != "" {
			count = 4
		}
		if len(manifest.Controls) != count {
			t.Fatal("selection controls not bound", step.name, string(body))
		}
		if step.intent != "" {
			intent := ""
			for _, control := range manifest.Controls {
				if control.Name == "uv-selection-intent.json" {
					var value struct{ Intent string }
					if err := json.Unmarshal(control.Content, &value); err != nil {
						t.Fatal(err)
					}
					intent = value.Intent
				}
			}
			if intent != step.intent {
				t.Fatal("wrong native intent", step.name, intent)
			}
		}
		if body, err := os.ReadFile(filepath.Join(venv, "pyvenv.cfg")); err != nil || !bytes.Equal(body, cfg) {
			t.Fatal("target configuration changed", err)
		}
		if body, err := os.ReadFile(userFile); err != nil || string(body) != "preserve user data" {
			t.Fatal("user file changed", err)
		}
		if step.name == "root-upgrade" {
			if _, err := os.Lstat(filepath.Join(site, "idna-3.10.dist-info", "METADATA")); !os.IsNotExist(err) {
				t.Fatal("obsolete owned version retained", err)
			}
		}
		if step.name == "new-root" {
			if _, err := os.Stat(filepath.Join(site, "idna-3.11.dist-info", "METADATA")); err != nil {
				t.Fatal("unrelated installed distribution removed", err)
			}
		}
		t.Logf("actual_uv_install step=%s operation=%s status=%s promotion=%s entries=%d controls=%d runtime=%s", step.name, result.Operation, result.Status, result.Promotion, result.Verified.Count, len(manifest.Controls), manifest.Resolver.Runtime)
	}
	// Unsupported installed requirements are refused before native selection.
	before := uvCLIRegularFiles(t, venv)
	evidenceBefore := uvCLIRegularFiles(t, filepath.Join(cache, "heliopause", "evidence"))
	foreign := filepath.Join(site, "foreign-1.0.dist-info")
	if err := os.Mkdir(foreign, 0700); err != nil {
		t.Fatal(err)
	}
	// Unsupported original requirements are rejected before an actual solve.
	if err := os.WriteFile(filepath.Join(foreign, "METADATA"), []byte("Name: foreign\nVersion: 1.0\nRequires-Dist: local @ file:///tmp/foreign\n\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := bootstrap.Run(ctx, []string{"--output", "json", "uv", "pip", "install", "idna", "--target", venv}, &stdout, &stderr); err == nil {
		t.Fatal("unsupported original metadata accepted")
	}
	if !bytes.Contains(stdout.Bytes(), []byte("UPDATE_ORIGINAL_STATE_UNSUPPORTED")) && !bytes.Contains(stdout.Bytes(), []byte("INSTALL_PREPARATION_FAILED")) {
		t.Fatal("wrong refusal boundary", stdout.String(), stderr.String())
	}
	if err := os.RemoveAll(foreign); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, uvCLIRegularFiles(t, venv)) || !reflect.DeepEqual(evidenceBefore, uvCLIRegularFiles(t, filepath.Join(cache, "heliopause", "evidence"))) {
		t.Fatal("refused original input changed target or Evidence")
	}
	uvNativeEnrollmentCLI(t, ctx, root, cache, originalWheel)
}

func uvNativeEnrollmentCLI(t *testing.T, ctx context.Context, root, cache string, wheel []byte) {
	t.Helper()
	venv := filepath.Join(root, "native-venv")
	uvNativeOriginalFixture(t, ctx, wheel, venv)
	version := strings.Split(sandbox.PinnedPythonRuntime().PythonVersion, ".")
	site := filepath.Join(venv, "lib", "python"+version[0]+"."+version[1], "site-packages")
	originalCfg, err := os.ReadFile(filepath.Join(venv, "pyvenv.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	userFile := filepath.Join(venv, "user-note.txt")
	if err := os.WriteFile(userFile, []byte("preserve native user data"), 0600); err != nil {
		t.Fatal(err)
	}
	// A real UV-generated original plus an unrecorded bytecode directory must
	// fail before new-set inspection, without publishing ownership or Evidence.
	cacheDir := filepath.Join(site, "idna", "__pycache__")
	if err := os.Mkdir(cacheDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "unrecorded.pyc"), []byte("never execute"), 0600); err != nil {
		t.Fatal(err)
	}
	before := uvCLIRegularFiles(t, venv)
	evidenceBefore := uvCLIRegularFiles(t, filepath.Join(cache, "heliopause", "evidence"))
	var stdout, stderr bytes.Buffer
	args := []string{"--output", "json", "uv", "pip", "install", "idna==3.11", "--upgrade", "--adopt", "--target", venv}
	if err := bootstrap.Run(ctx, args, &stdout, &stderr); err == nil || !bytes.Contains(stdout.Bytes(), []byte("UPDATE_ORIGINAL_STATE_UNSUPPORTED")) {
		t.Fatal("unrecorded native cache accepted or wrong refusal", err, stdout.String(), stderr.String())
	}
	if !reflect.DeepEqual(before, uvCLIRegularFiles(t, venv)) || !reflect.DeepEqual(evidenceBefore, uvCLIRegularFiles(t, filepath.Join(cache, "heliopause", "evidence"))) {
		t.Fatal("refused native original changed target or Evidence")
	}
	if err := os.RemoveAll(cacheDir); err != nil {
		t.Fatal(err)
	}
	for _, adopt := range []bool{true, false} {
		args := []string{"--output", "json", "uv", "pip", "install", "idna==3.11", "--upgrade", "--target", venv}
		if adopt {
			args = append(args, "--adopt")
		}
		stdout.Reset()
		stderr.Reset()
		if err := bootstrap.Run(ctx, args, &stdout, &stderr); err != nil {
			t.Fatal("actual UV native enrollment/replay failed", adopt, err, stdout.String(), stderr.String())
		}
		var result struct {
			Operation string                    `json:"operation"`
			Status    string                    `json:"operation_status"`
			Promotion string                    `json:"promotion_status"`
			Policy    struct{ Decision string } `json:"policy"`
			Verified  struct {
				Manifest string `json:"manifest_id"`
				Count    int    `json:"entry_count"`
			} `json:"verified_set"`
			Checks []struct {
				Required bool
				Status   string `json:"execution_status"`
			} `json:"checks"`
			Evidence []struct{ Handle string } `json:"evidence_references"`
		}
		if json.Unmarshal(stdout.Bytes(), &result) != nil || result.Operation != "UPDATE" || result.Status != "COMPLETED" || result.Promotion != "COMPLETED" || result.Policy.Decision != "ALLOW" || result.Verified.Count != 1 || len(result.Checks) < 3 || len(result.Evidence) < 3 {
			t.Fatal("incomplete native enrollment", stdout.String())
		}
		for _, check := range result.Checks {
			if check.Required && check.Status != "COMPLETED" {
				t.Fatal("native required check incomplete")
			}
		}
		body, err := os.ReadFile(filepath.Join(cache, "heliopause", "staging", result.Verified.Manifest, "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		var manifest struct {
			Controls []struct {
				Name string `json:"name"`
			} `json:"project_controls"`
			Resolver struct {
				Runtime string `json:"runtime_identity"`
			} `json:"resolver"`
		}
		if json.Unmarshal(body, &manifest) != nil || manifest.Resolver.Runtime != sandbox.UVResolverRuntimeIdentity() {
			t.Fatal("native enrollment selection identity mismatch")
		}
		expected := 4
		if adopt {
			expected = 5
		}
		if len(manifest.Controls) != expected {
			t.Fatal("native proof not bound to Manifest", string(body))
		}
		if _, err := os.Stat(filepath.Join(site, "idna-3.11.dist-info", "METADATA")); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(filepath.Join(site, "idna-3.10.dist-info", "METADATA")); !os.IsNotExist(err) {
			t.Fatal("native old metadata retained", err)
		}
		if _, err := os.Stat(filepath.Join(venv, ".heliopause", "pypi-transaction.json")); err != nil {
			t.Fatal(err)
		}
		if cfg, err := os.ReadFile(filepath.Join(venv, "pyvenv.cfg")); err != nil || !bytes.Equal(cfg, originalCfg) {
			t.Fatal("native original configuration changed", err)
		}
		if body, err := os.ReadFile(userFile); err != nil || string(body) != "preserve native user data" {
			t.Fatal("native user data changed", err)
		}
		t.Logf("actual_uv_native_enrollment adopt=%t status=%s promotion=%s entries=%d controls=%d", adopt, result.Status, result.Promotion, result.Verified.Count, len(manifest.Controls))
	}
}

func uvCLIRegularFiles(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	result := map[string][32]byte{}
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			t.Fatalf("unexpected non-regular test output: %s", path)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[path] = sha256.Sum256(body)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return result
}
