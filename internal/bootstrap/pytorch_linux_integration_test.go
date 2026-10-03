package bootstrap_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/bootstrap"
)

func TestLinuxPyTorchFullIntegration(t *testing.T) {
	if os.Getenv("HELOX_PYTORCH_FULL_INTEGRATION") != "1" {
		t.Skip("requires controlled Linux gVisor PyTorch qualification")
	}
	if runtime.GOOS != "linux" {
		t.Fatal("PyTorch full qualification requires Linux")
	}
	profile := os.Getenv("HELOX_PYTORCH_PROFILE")
	version := map[string]string{"cpu": "2.14.0+cpu", "cu126": "2.14.0+cu126", "cu130": "2.14.0+cu130", "cu132": "2.14.0+cu132"}[profile]
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
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	target := filepath.Join(root, "target")
	for _, relative := range []string{"lib/python3.14/site-packages", "bin", "share"} {
		if err := os.MkdirAll(filepath.Join(target, relative), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(target, "pyvenv.cfg"), []byte("version = 3.14.7\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), map[string]time.Duration{"cpu": 15 * time.Minute, "cu126": 40 * time.Minute, "cu130": 40 * time.Minute, "cu132": 40 * time.Minute}[profile])
	defer cancel()
	var stdout, stderr bytes.Buffer
	args := []string{"pip", "install", "torch@" + version, "--source", "pytorch:" + profile, "--target", target}
	prerequisite := os.Getenv("HELOX_PYTORCH_INSPECTION_PREREQUISITE")
	if prerequisite != "" {
		args = append(args, "--inspection-prerequisites", prerequisite)
	}
	err := bootstrap.Run(ctx, args, &stdout, &stderr)
	if err != nil {
		t.Fatalf("PyTorch %s install failed: %v\nstdout=%s\nstderr=%s", profile, err, stdout.String(), stderr.String())
	}
	if info, statErr := os.Stat(filepath.Join(target, "lib", "python3.14", "site-packages", "torch")); statErr != nil || !info.IsDir() {
		t.Fatalf("PyTorch %s target is missing: %v", profile, statErr)
	}
	if profile == "cpu" {
		for _, relative := range []string{"bin/isympy", "share/man/man1/isympy.1"} {
			if info, err := os.Lstat(filepath.Join(target, relative)); err != nil || !info.Mode().IsRegular() {
				t.Fatalf("scheme output %s unavailable: %v", relative, err)
			}
		}
	}
	staged, globErr := filepath.Glob(filepath.Join(root, "cache", "heliopause", "staging", "*", "manifest.json"))
	if globErr != nil || len(staged) != 1 {
		t.Fatalf("PyTorch %s staging manifest = %q, %v", profile, staged, globErr)
	}
	manifest, readErr := os.ReadFile(staged[0])
	if readErr != nil || !strings.Contains(string(manifest), `"source":"pytorch-`+profile+`"`) || !strings.Contains(string(manifest), `"name":"torch"`) {
		t.Fatalf("PyTorch %s source identity did not reach staged manifest: %v\n%s", profile, readErr, manifest)
	}
	if expected := os.Getenv("HELOX_PYTORCH_NOT_ATTESTED_MODULE"); expected != "" {
		paths, err := filepath.Glob(filepath.Join(root, "cache/heliopause/evidence/*/pypi-command-not-attested-*.json"))
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
			matches, err := filepath.Glob(filepath.Join(target, "lib/python3.14/site-packages", pattern))
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
		evidence, err := filepath.Glob(filepath.Join(root, "cache/heliopause/evidence/*/pypi-dynamic-import-supplemented-result.json"))
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
