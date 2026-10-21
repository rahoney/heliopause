package bootstrap_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/bootstrap"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestLinuxPyTorchFullIntegration(t *testing.T) {
	if os.Getenv("HELOX_PYTORCH_FULL_INTEGRATION") != "1" {
		t.Skip("requires controlled Linux gVisor PyTorch qualification")
	}
	if runtime.GOOS != "linux" {
		t.Fatal("PyTorch full qualification requires Linux")
	}
	profile := os.Getenv("HELOX_PYTORCH_PROFILE")
	version := map[string]string{"cpu": runtimeidentity.PyTorchCPUVersion, "cu126": runtimeidentity.PyTorchCU126Version, "cu130": runtimeidentity.PyTorchCU130Version, "cu132": runtimeidentity.PyTorchCU132Version}[profile]
	if version == "" {
		t.Fatalf("unsupported PyTorch qualification profile %q", profile)
	}
	root := t.TempDir()
	if evidenceRoot := os.Getenv("HELOX_INTEGRATION_EVIDENCE_ROOT"); evidenceRoot != "" {
		if err := os.MkdirAll(evidenceRoot, 0o700); err != nil {
			t.Fatal(err)
		}
		var err error
		root, err = os.MkdirTemp(evidenceRoot, "pytorch-"+profile+"-")
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("retained integration evidence: %s", root)
	}
	cache, err := pytorchIntegrationCache(root, profile, os.Getenv("HELOX_INTEGRATION_CACHE_ROOT"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", cache)
	if os.Getenv("HELOX_INTEGRATION_CACHE_ROOT") != "" {
		t.Logf("retained integration cache: %s", cache)
	}
	target := filepath.Join(root, "target")
	pythonParts := strings.Split(runtimeidentity.PythonVersion, ".")
	pythonMinor := strings.Join(pythonParts[:2], ".")
	for _, relative := range []string{filepath.Join("lib", "python"+pythonMinor, "site-packages"), "bin", "share"} {
		if err := os.MkdirAll(filepath.Join(target, relative), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(target, "pyvenv.cfg"), []byte("version = "+runtimeidentity.PythonVersion+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), map[string]time.Duration{"cpu": 15 * time.Minute, "cu126": 40 * time.Minute, "cu130": 40 * time.Minute, "cu132": 40 * time.Minute}[profile])
	defer cancel()
	if source, ok := artifactpypi.PyTorchProfile("pytorch:" + profile); ok {
		policy := source.ResourcePolicy()
		t.Logf("pytorch_os_pressure_capacity logical_cpus=%d runtime_memory_bytes=%d runtime_tmpfs_bytes=%d promotion_tmpfs_bytes=%d capacity_sufficiency=NOT_ESTABLISHED", runtime.NumCPU(), policy.RuntimeMemory(), policy.RuntimeTmpfs(), policy.PromotionTmpfs())
	}
	stopPressureDiagnostic := startPyTorchPressureDiagnostic(t, ctx)
	defer stopPressureDiagnostic()
	var stdout, stderr bytes.Buffer
	args := []string{"pip", "install", "torch@" + version, "--source", "pytorch:" + profile, "--target", target}
	prerequisite := os.Getenv("HELOX_PYTORCH_INSPECTION_PREREQUISITE")
	if plan := os.Getenv("HELOX_PYTORCH_QUALIFICATION_PLAN"); plan != "" {
		if prerequisite != "" {
			t.Fatal("qualification plan and manual prerequisite pin are mutually exclusive")
		}
		var receipt []byte
		prerequisite, receipt, err = bootstrap.PreparePyTorchQualification(ctx, profile, version, target, plan)
		if err != nil {
			t.Fatalf("explicit qualification preparation failed: %v", err)
		}
		if err := os.WriteFile(filepath.Join(root, "qualification-plan.json"), receipt, 0o600); err != nil {
			t.Fatal(err)
		}
		t.Logf("explicit qualification metadata/input frozen: %s", receipt)
	}
	if prerequisite != "" {
		args = append(args, "--inspection-prerequisites", prerequisite)
	}
	err = bootstrap.Run(ctx, args, &stdout, &stderr)
	if err != nil {
		t.Fatalf("PyTorch %s install failed: %v\nstdout=%s\nstderr=%s", profile, err, stdout.String(), stderr.String())
	}
	if info, statErr := os.Stat(filepath.Join(target, "lib", "python"+pythonMinor, "site-packages", "torch")); statErr != nil || !info.IsDir() {
		t.Fatalf("PyTorch %s target is missing: %v", profile, statErr)
	}
	if profile == "cpu" {
		for _, relative := range []string{"bin/isympy", "share/man/man1/isympy.1"} {
			if info, err := os.Lstat(filepath.Join(target, relative)); err != nil || !info.Mode().IsRegular() {
				t.Fatalf("scheme output %s unavailable: %v", relative, err)
			}
		}
	}
	staged, globErr := filepath.Glob(filepath.Join(cache, "heliopause", "staging", "*", "manifest.json"))
	if globErr != nil || len(staged) != 1 {
		t.Fatalf("PyTorch %s staging manifest = %q, %v", profile, staged, globErr)
	}
	manifest, readErr := os.ReadFile(staged[0])
	if readErr != nil || !strings.Contains(string(manifest), `"source":"pytorch-`+profile+`"`) || !strings.Contains(string(manifest), `"name":"torch"`) {
		t.Fatalf("PyTorch %s source identity did not reach staged manifest: %v\n%s", profile, readErr, manifest)
	}
	if expected := os.Getenv("HELOX_PYTORCH_NOT_ATTESTED_MODULE"); expected != "" {
		paths, err := filepath.Glob(filepath.Join(cache, "heliopause/evidence/*/pypi-command-not-attested-*.json"))
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var ev struct {
				Summary string `json:"summary"`
			}
			if err := json.Unmarshal(data, &ev); err != nil {
				t.Fatal(err)
			}
			var bounded struct {
				Module        string `json:"module"`
				Disposition   string `json:"disposition"`
				Functionality bool   `json:"functionality_attested"`
				Enforcement   bool   `json:"later_execution_enforced"`
			}
			if err := json.Unmarshal([]byte(ev.Summary), &bounded); err != nil {
				t.Fatal(err)
			}
			if bounded.Module != expected {
				continue
			}
			if found || bounded.Disposition != "NOT_ATTESTED" || bounded.Functionality || bounded.Enforcement {
				t.Fatal("invalid bounded command evidence", ev.Summary)
			}
			id := strings.TrimSuffix(filepath.Base(path), ".json")
			if !strings.Contains(string(manifest), id) || !strings.Contains(stdout.String(), id) {
				t.Fatal("command limitation missing from result/promotion references")
			}
			found = true
			t.Logf("bounded command evidence retained in result/promotion: %s", ev.Summary)
		}
		if !found {
			t.Fatal("expected NOT_ATTESTED command evidence missing")
		}
	}
	if prerequisite != "" {
		var pin struct {
			Project string `json:"project"`
			SHA256  string `json:"sha256"`
		}
		if err := json.Unmarshal([]byte(prerequisite), &pin); err != nil || pin.Project == "" || pin.SHA256 == "" {
			t.Fatal("invalid qualification prerequisite pin")
		}
		var stagedManifest struct {
			Entries []struct {
				Name string `json:"name"`
			} `json:"entries"`
		}
		if err := json.Unmarshal(manifest, &stagedManifest); err != nil {
			t.Fatal(err)
		}
		for _, entry := range stagedManifest.Entries {
			if entry.Name == pin.Project {
				t.Fatal("inspection input was promoted as a requested dependency")
			}
		}
		for _, pattern := range []string{pin.Project, pin.Project + ".libs", pin.Project + "-*.dist-info"} {
			matches, err := filepath.Glob(filepath.Join(target, "lib", "python"+pythonMinor, "site-packages", pattern))
			if err != nil || len(matches) != 0 {
				t.Fatalf("inspection-only target content %s: %v %v", pattern, matches, err)
			}
		}
		// NumPy's generated entry-point scripts must not escape its inspection
		// volume either; the generic original manifest remains authoritative.
		if pin.Project == "numpy" {
			for _, name := range []string{"f2py", "numpy-config"} {
				if _, err := os.Lstat(filepath.Join(target, "bin", name)); !os.IsNotExist(err) {
					t.Fatalf("inspection script leaked: %s (%v)", name, err)
				}
			}
		}
		evidence, err := filepath.Glob(filepath.Join(cache, "heliopause/evidence/*/pypi-dynamic-import-supplemented-result.json"))
		if err != nil || len(evidence) != 1 {
			t.Fatalf("supplemented evidence: %v %v", evidence, err)
		}
		document, err := os.ReadFile(evidence[0])
		if err != nil {
			t.Fatal(err)
		}
		var record struct {
			Name    string `json:"name"`
			Summary string `json:"summary"`
		}
		if err := json.Unmarshal(document, &record); err != nil || record.Name == "torch" || !strings.Contains(record.Summary, pin.SHA256) || !strings.Contains(record.Summary, `"original_environment":"NOT_ATTESTED"`) {
			t.Fatalf("inspection-input scope not retained: %s %v", document, err)
		}
		t.Logf("inspection input excluded from %d promoted entries; separate supplemented evidence retained", len(stagedManifest.Entries))
	}
}

func TestLinuxPyTorchQualificationPreparationIntegration(t *testing.T) {
	if os.Getenv("HELOX_PYTORCH_PREPARATION_INTEGRATION") != "1" {
		t.Skip("requires explicit installed qualification metadata preflight")
	}
	if runtime.GOOS != "linux" {
		t.Fatal("qualification metadata preflight requires Linux")
	}
	root := t.TempDir()
	if parent := os.Getenv("HELOX_INTEGRATION_EVIDENCE_ROOT"); parent != "" {
		if err := os.MkdirAll(parent, 0o700); err != nil {
			t.Fatal(err)
		}
		var err error
		root, err = os.MkdirTemp(parent, "qualification-preflight-")
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	target := filepath.Join(root, "target")
	raw, receipt, err := bootstrap.PreparePyTorchQualification(ctx, "cu126", runtimeidentity.PyTorchCU126Version, target, "cuda-bindings-numpy")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "qualification-plan.json"), receipt, 0o600); err != nil {
		t.Fatal(err)
	}
	var pin map[string]string
	if err := json.Unmarshal([]byte(raw), &pin); err != nil {
		t.Fatal(err)
	}
	if pin["project"] != "numpy" || pin["version"] != "2.4.6" || pin["target_sha256"] == "" {
		t.Fatal("explicit prepared pin unavailable")
	}
	// An intentionally unselected caller pin must fail on metadata, without
	// downloading the main multi-GiB graph. Payload hashes are never expectations.
	pin["target_sha256"] = strings.Repeat("0", 64)
	stale, err := json.Marshal(pin)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err = bootstrap.Run(ctx, []string{"pip", "install", "torch@" + runtimeidentity.PyTorchCU126Version, "--source", "pytorch:cu126", "--target", target, "--inspection-prerequisites", string(stale)}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "inspection prerequisite target is not selected") {
		t.Fatalf("unexpected stale-pin outcome: %v", err)
	}
	intake := filepath.Join(root, "cache/heliopause/intake")
	entries, readErr := os.ReadDir(intake)
	if readErr != nil && !os.IsNotExist(readErr) {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatal("stale pin reached main payload acquisition")
	}
	t.Logf("official selected metadata/input retained: %s; main intake=0; root=%s", receipt, root)
}

// The optional qualification cache is fresh per invocation. Private mutation
// and the final target remain together; production verification and copying
// still bind cache content to the exact inspected set across filesystems.
func pytorchIntegrationCache(root, profile, parent string) (string, error) {
	if parent == "" {
		return filepath.Join(root, "cache"), nil
	}
	if !filepath.IsAbs(parent) || filepath.Clean(parent) != parent {
		return "", errors.New("qualification cache parent must be canonical and absolute")
	}
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil || resolved != parent {
		return "", errors.New("qualification cache parent must have no symlink components")
	}
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
		return "", errors.New("qualification cache parent must be a private existing directory")
	}
	return os.MkdirTemp(parent, "pytorch-"+profile+"-")
}
