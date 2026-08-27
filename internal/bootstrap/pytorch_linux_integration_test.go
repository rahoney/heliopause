package bootstrap_test

import (
	"bytes"
	"context"
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
	version := map[string]string{"cpu": "2.9.1+cpu", "cu126": "2.9.0+cu126"}[profile]
	if version == "" {
		t.Fatalf("unsupported PyTorch qualification profile %q", profile)
	}
	root := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	target := filepath.Join(root, "target")
	ctx, cancel := context.WithTimeout(context.Background(), map[string]time.Duration{"cpu": 15 * time.Minute, "cu126": 40 * time.Minute}[profile])
	defer cancel()
	var stdout, stderr bytes.Buffer
	err := bootstrap.Run(ctx, []string{"pip", "install", "torch@" + version, "--source", "pytorch:" + profile, "--target", target}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("PyTorch %s install failed: %v\nstdout=%s\nstderr=%s", profile, err, stdout.String(), stderr.String())
	}
	if info, statErr := os.Stat(filepath.Join(target, "site", "torch")); statErr != nil || !info.IsDir() {
		t.Fatalf("PyTorch %s target is missing: %v", profile, statErr)
	}
	staged, globErr := filepath.Glob(filepath.Join(root, "cache", "heliopause", "staging", "*", "manifest.json"))
	if globErr != nil || len(staged) != 1 {
		t.Fatalf("PyTorch %s staging manifest = %q, %v", profile, staged, globErr)
	}
	manifest, readErr := os.ReadFile(staged[0])
	if readErr != nil || !strings.Contains(string(manifest), `"source":"pytorch-`+profile+`"`) || !strings.Contains(string(manifest), `"name":"torch"`) {
		t.Fatalf("PyTorch %s source identity did not reach staged manifest: %v\n%s", profile, readErr, manifest)
	}
}
