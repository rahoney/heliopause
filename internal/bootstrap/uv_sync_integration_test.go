package bootstrap_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/bootstrap"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type uvSyncActualReport struct {
	Schema         string                    `json:"schema_version"`
	Status         string                    `json:"operation_status"`
	Promotion      string                    `json:"promotion_status"`
	Graph          string                    `json:"graph_digest"`
	DependencyFree bool                      `json:"dependency_free"`
	Policy         struct{ Decision string } `json:"policy"`
	Entries        []struct {
		Identity struct{ Name, Version string } `json:"identity"`
		Digest   struct{ Value string }         `json:"digest"`
		Run      string                         `json:"run_id"`
		Checks   []struct {
			Required bool
			Status   string `json:"execution_status"`
		} `json:"checks"`
		Evidence []struct{ Handle string } `json:"evidence_references"`
	} `json:"entries"`
	Verified struct {
		Manifest string `json:"manifest_id"`
		SBOM     string `json:"sbom_handle"`
		Staging  string `json:"staging_handle"`
		Count    int    `json:"entry_count"`
	} `json:"verified_set"`
}

func uvSyncActualEnvironment(t *testing.T) (context.Context, string, string, string) {
	t.Helper()
	if os.Getenv("HELOX_UV_SYNC_CLI_INTEGRATION") != "1" {
		t.Skip("requires authenticated native UV/Python/gVisor integration")
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
	for _, directory := range []string{site, filepath.Join(venv, "bin")} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(venv, "pyvenv.cfg"), []byte("version = "+sandbox.PinnedPythonRuntime().PythonVersion+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	t.Cleanup(cancel)
	return ctx, root, cache, venv
}

func runActualUVSync(t *testing.T, ctx context.Context, source, target string, flags ...string) uvSyncActualReport {
	t.Helper()
	args := append([]string{"--lang", "ko", "--output", "json", "uv", "pip", "sync", source, "--target", target}, flags...)
	var out, diagnostic bytes.Buffer
	if err := bootstrap.Run(ctx, args, &out, &diagnostic); err != nil {
		t.Fatalf("actual sync failed: %v stdout=%s stderr=%s", err, out.String(), diagnostic.String())
	}
	var report uvSyncActualReport
	if json.Unmarshal(out.Bytes(), &report) != nil || report.Schema != "helox.project-update-result/v1" || report.Status != "COMPLETED" || report.Promotion != "COMPLETED" || report.Policy.Decision != "ALLOW" || report.Graph == "" || report.Verified.Manifest == "" || report.Verified.Count != len(report.Entries) || report.DependencyFree != (len(report.Entries) == 0) {
		t.Fatal("incomplete actual sync report", out.String())
	}
	for _, entry := range report.Entries {
		if entry.Run == "" || len(entry.Checks) < 3 || len(entry.Evidence) < 3 {
			t.Fatal("entry inspection coverage missing", out.String())
		}
		for _, check := range entry.Checks {
			if check.Required && check.Status != "COMPLETED" {
				t.Fatal("required entry check incomplete", out.String())
			}
		}
	}
	t.Logf("actual_uv_sync entries=%d dependency_free=%t graph=%s manifest=%s", len(report.Entries), report.DependencyFree, report.Graph, report.Verified.Manifest)
	return report
}

func actualUVSyncPayload(t *testing.T, cache string, report uvSyncActualReport) string {
	t.Helper()
	id, ok := strings.CutPrefix(report.Verified.Staging, "project-cache:")
	if !ok {
		t.Fatal("wrong actual cache handle")
	}
	if _, err := domain.ParseRunID(id); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(cache, "heliopause", "staging", id, "payload")
	body, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Schema    string
		Operation string            `json:"operation_id"`
		Primary   json.RawMessage   `json:"primary"`
		Controls  []json.RawMessage `json:"project_controls"`
		Manifest  string            `json:"manifest_id"`
	}
	if json.Unmarshal(body, &doc) != nil || doc.Schema != "helox.verified-project-manifest/v1" || doc.Operation == "" || len(doc.Primary) != 0 || len(doc.Controls) < 4 || doc.Manifest != report.Verified.Manifest {
		t.Fatal("actual primary-free Manifest incomplete", string(body))
	}
	return root
}

func TestLinuxUVSyncCompleteCLIIntegration(t *testing.T) {
	ctx, root, cache, venv := uvSyncActualEnvironment(t)
	source := filepath.Join(root, "requirements.txt")
	site := filepath.Join(venv, "lib", "python3.14", "site-packages")
	user := filepath.Join(venv, "user-note.txt")
	if err := os.WriteFile(user, []byte("preserve exact user data"), 0o600); err != nil {
		t.Fatal(err)
	}
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(source, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("idna==3.10\npackaging==25.0\n")
	first := runActualUVSync(t, ctx, source, venv)
	if len(first.Entries) != 2 {
		t.Fatal("disconnected root missing")
	}
	payload := actualUVSyncPayload(t, cache, first)
	var oldWheel []byte
	for _, entry := range first.Entries {
		if entry.Identity.Name == "idna" {
			var err error
			oldWheel, err = os.ReadFile(filepath.Join(payload, "artifacts", entry.Digest.Value+".whl"))
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(oldWheel)
			if hex.EncodeToString(sum[:]) != entry.Digest.Value {
				t.Fatal("actual native seed digest differs")
			}
		}
	}
	if len(oldWheel) == 0 {
		t.Fatal("missing authentic native seed")
	}
	second := runActualUVSync(t, ctx, source, venv)
	if second.Verified.Manifest == first.Verified.Manifest || len(second.Entries) != 2 {
		t.Fatal("same-set did not get fresh records")
	}
	for _, entry := range second.Entries {
		for _, old := range first.Entries {
			if entry.Run == old.Run {
				t.Fatal("same-set reused old inspection Run")
			}
		}
	}
	write("idna==3.11\n")
	replaced := runActualUVSync(t, ctx, source, venv)
	if len(replaced.Entries) != 1 {
		t.Fatal("removed package retained")
	}
	for _, path := range []string{filepath.Join(site, "packaging-25.0.dist-info", "METADATA"), filepath.Join(site, "idna-3.10.dist-info", "METADATA")} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatal("obsolete original retained", path, err)
		}
	}
	write("# blank preserves installed state\n")
	before := uvCLIRegularFiles(t, venv)
	evidenceBefore := uvCLIRegularFiles(t, filepath.Join(cache, "heliopause", "evidence"))
	var out, diagnostic bytes.Buffer
	if err := bootstrap.Run(ctx, []string{"--output", "json", "uv", "pip", "sync", source, "--target", venv}, &out, &diagnostic); err != nil || !bytes.Contains(out.Bytes(), []byte(`"operation_status":"SKIPPED"`)) {
		t.Fatal("blank sync did not skip", err, out.String())
	}
	if !reflect.DeepEqual(before, uvCLIRegularFiles(t, venv)) || !reflect.DeepEqual(evidenceBefore, uvCLIRegularFiles(t, filepath.Join(cache, "heliopause", "evidence"))) {
		t.Fatal("blank skip mutated target or created Evidence")
	}
	empty := runActualUVSync(t, ctx, source, venv, "--allow-empty-requirements")
	if !empty.DependencyFree {
		t.Fatal("explicit empty was not approved")
	}
	actualUVSyncPayload(t, cache, empty)
	if _, err := os.Lstat(filepath.Join(venv, ".heliopause", "pypi-transaction.json")); !os.IsNotExist(err) {
		t.Fatal("explicit empty retained ownership", err)
	}
	pylock := filepath.Join(root, "pylock.toml")
	if err := os.WriteFile(pylock, []byte("lock-version='1.0'\ncreated-by='uv'\nrequires-python='>=3.14.7'\npackages=[]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if report := runActualUVSync(t, ctx, pylock, venv); !report.DependencyFree {
		t.Fatal("explicit empty pylock not approved")
	}
	if body, err := os.ReadFile(user); err != nil || string(body) != "preserve exact user data" {
		t.Fatal("user file changed", err)
	}
	// A genuine UV original excluded from the desired set must be proved too.
	native := filepath.Join(root, "native")
	uvNativeOriginalFixture(t, ctx, oldWheel, native)
	write("packaging==25.0\n")
	before = uvCLIRegularFiles(t, native)
	evidenceBefore = uvCLIRegularFiles(t, filepath.Join(cache, "heliopause", "evidence"))
	out.Reset()
	diagnostic.Reset()
	if err := bootstrap.Run(ctx, []string{"--output", "json", "uv", "pip", "sync", source, "--target", native}, &out, &diagnostic); err == nil || !bytes.Contains(out.Bytes(), []byte("UPDATE_ADOPTION_REQUIRED")) {
		t.Fatal("native original was removed without explicit proof", err, out.String())
	}
	if !reflect.DeepEqual(before, uvCLIRegularFiles(t, native)) || !reflect.DeepEqual(evidenceBefore, uvCLIRegularFiles(t, filepath.Join(cache, "heliopause", "evidence"))) {
		t.Fatal("refused adoption changed target or Evidence")
	}
	adopted := runActualUVSync(t, ctx, source, native, "--adopt")
	actualUVSyncPayload(t, cache, adopted)
	if _, err := os.Lstat(filepath.Join(native, "lib", "python3.14", "site-packages", "idna-3.10.dist-info", "METADATA")); !os.IsNotExist(err) {
		t.Fatal("excluded proven native owner remained", err)
	}
}

func uvSyncCLIClosureSource() ([]byte, int, error) {
	body, err := os.ReadFile(filepath.Join("..", "artifact", "pypi", "testdata", "uv-0.13.4-requests.pylock.toml"))
	if err != nil {
		return nil, 0, err
	}
	lock, err := artifactpypi.ParseUVProjectLock(body, sandbox.PinnedPythonRuntime().PythonVersion)
	if err != nil {
		return nil, 0, err
	}
	var source strings.Builder
	for _, project := range lock.Packages() {
		fmt.Fprintf(&source, "%s==%s\n", project.Name, project.Version)
	}
	return []byte(source.String()), len(lock.Packages()), nil
}
func TestUVSyncCLIFixtureSources(t *testing.T) {
	body, count, err := uvSyncCLIClosureSource()
	if err != nil || count != 5 || len(body) == 0 {
		t.Fatal("tracked complete-source fixture unavailable", count, err)
	}
}

func TestLinuxUVSyncClosureCLIIntegration(t *testing.T) {
	ctx, root, cache, venv := uvSyncActualEnvironment(t)
	body, count, err := uvSyncCLIClosureSource()
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "requirements.txt")
	if err := os.WriteFile(source, body, 0o600); err != nil {
		t.Fatal(err)
	}
	report := runActualUVSync(t, ctx, source, venv)
	if len(report.Entries) != count {
		t.Fatal("whole dependency closure not inspected")
	}
	payload := actualUVSyncPayload(t, cache, report)
	data, err := os.ReadFile(filepath.Join(payload, "sbom.cdx.json"))
	if err != nil {
		t.Fatal(err)
	}
	var bom struct {
		Components   []json.RawMessage `json:"components"`
		Dependencies []struct {
			Depends []string `json:"dependsOn"`
		} `json:"dependencies"`
	}
	if json.Unmarshal(data, &bom) != nil || len(bom.Components) != count || len(bom.Dependencies) != count {
		t.Fatal("actual whole-closure SBOM incomplete")
	}
	edges := 0
	for _, dependency := range bom.Dependencies {
		edges += len(dependency.Depends)
	}
	if edges != count-1 {
		t.Fatal("actual wheel dependency edges were lost", edges)
	}
}
