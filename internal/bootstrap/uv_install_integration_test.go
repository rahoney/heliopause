package bootstrap_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
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
	args := []string{"--output", "json", "--lang", "ko", "uv", "pip", "install", "idna==3.11", "--target", venv}
	var stdout, stderr bytes.Buffer
	if err := bootstrap.Run(ctx, args, &stdout, &stderr); err != nil {
		t.Fatalf("actual UV install failed: %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
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
	if result.Schema != "helox.operation-result/v1" || result.Operation != "INSTALL" || result.Status != "COMPLETED" || result.Promotion != "COMPLETED" || result.Policy.Decision != "ALLOW" || result.Verified.Count != 1 || result.Verified.Manifest == "" || len(result.Checks) < 3 || len(result.Evidence) < 3 {
		t.Fatal("incomplete UV install", stdout.String())
	}
	for _, check := range result.Checks {
		if check.Required && check.Status != "COMPLETED" {
			t.Fatal("required check incomplete", stdout.String())
		}
	}
	for _, path := range []string{filepath.Join(site, "idna", "__init__.py"), filepath.Join(site, "idna-3.11.dist-info", "METADATA"), filepath.Join(venv, ".heliopause", "pypi-transaction.json")} {
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			t.Fatal("approved file missing", path, err)
		}
	}
	manifestPath := filepath.Join(cache, "heliopause", "staging", result.Verified.Manifest, "manifest.json")
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Resolver struct {
			Runtime string `json:"runtime_identity"`
		} `json:"resolver"`
	}
	if err := json.Unmarshal(body, &manifest); err != nil || manifest.Resolver.Runtime != sandbox.UVResolverRuntimeIdentity() {
		t.Fatal("solver identity is not the pinned UV selection", err, string(body))
	}
	if body, err := os.ReadFile(filepath.Join(venv, "pyvenv.cfg")); err != nil || !bytes.Equal(body, cfg) {
		t.Fatal("target configuration changed", err)
	}
	before := uvCLIRegularFiles(t, venv)
	evidenceBefore := uvCLIRegularFiles(t, filepath.Join(cache, "heliopause", "evidence"))
	stdout.Reset()
	stderr.Reset()
	if err := bootstrap.Run(ctx, args, &stdout, &stderr); err == nil {
		t.Fatal("nonempty target used fresh selection", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("INSTALL_PREPARATION_FAILED")) {
		t.Fatal("nonempty target failed at the wrong boundary", stdout.String(), stderr.String())
	}
	if !reflect.DeepEqual(before, uvCLIRegularFiles(t, venv)) || !reflect.DeepEqual(evidenceBefore, uvCLIRegularFiles(t, filepath.Join(cache, "heliopause", "evidence"))) {
		t.Fatal("refused repeat changed target files or inspection Evidence")
	}
	t.Logf("actual_uv_install status=%s promotion=%s entries=%d runtime=%s repeat=REFUSED target_and_evidence=UNCHANGED", result.Status, result.Promotion, result.Verified.Count, manifest.Resolver.Runtime)
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
