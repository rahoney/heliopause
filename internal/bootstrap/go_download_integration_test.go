package bootstrap_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/bootstrap"
)

// The actual CLI composition owns source resolution, guard, complete
// inspection, verified-cache staging and retained approval. No go-get success
// is inferred from this independent dependency-free download gate.
func TestLinuxGoDependencyFreeDownloadIntegration(t *testing.T) {
	if os.Getenv("HELOX_GO_RESOLVER_PROJECT_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Fatal("Go qualification requires Linux amd64")
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, "project")
	if err := os.Mkdir(project, 0o700); err != nil {
		t.Fatal(err)
	}
	mod := []byte("module example.com/haa-fixture\n\ngo 1.26\n")
	if err := os.WriteFile(filepath.Join(project, "go.mod"), mod, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	t.Chdir(project)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	for attempt := 0; attempt < 2; attempt++ {
		var stdout, stderr bytes.Buffer
		if err := bootstrap.Run(ctx, []string{"go", "mod", "download"}, &stdout, &stderr); err != nil {
			t.Fatalf("actual download attempt %d: %v stdout=%s stderr=%s", attempt, err, stdout.String(), stderr.String())
		}
		if !strings.Contains(stdout.String(), "Source: go-proxy\nModules: 0\nGraph digest: ") {
			t.Fatalf("missing complete download result: %s", stdout.String())
		}
		got, err := os.ReadFile(filepath.Join(project, "go.mod"))
		if err != nil || string(got) != string(mod) {
			t.Fatal("download changed module controls")
		}
		sum, err := os.ReadFile(filepath.Join(project, "go.sum"))
		if err != nil || len(sum) != 0 {
			t.Fatal("download did not retain the normalized empty checksum controls")
		}
		t.Logf("actual_download_attempt=%d %s", attempt, strings.TrimSpace(stdout.String()))
	}
	state, err := filepath.Glob(filepath.Join(root, "cache", "heliopause", "go-projects", "*.json"))
	if err != nil || len(state) != 1 {
		t.Fatal("complete download did not retain one project approval")
	}
}

// Required positive product gate: a runner-only fixture or a successful empty
// download never substitutes for exact selection, approval and publication.
func TestLinuxGoGetDownloadIntegration(t *testing.T) {
	goGetDownloadIntegration(t, "github.com/spf13/pflag@v1.0.9", 1)
}

// This complete transitive graph exceeds the previous 10,000-file cache cap.
// Keep the actual CLI gate alongside the aggregate budget/security tests.
func TestLinuxGoTransitiveGetDownloadIntegration(t *testing.T) {
	goGetDownloadIntegration(t, "google.golang.org/grpc@v1.76.0", 41)
}

func goGetDownloadIntegration(t *testing.T, reference string, modules int) {
	t.Helper()
	if os.Getenv("HELOX_GO_RESOLVER_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Fatal("Go qualification requires Linux amd64")
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, "project")
	if err := os.Mkdir(project, 0o700); err != nil {
		t.Fatal(err)
	}
	mod := []byte("module example.com/haa-fixture\n\ngo 1.26\n")
	if err := os.WriteFile(filepath.Join(project, "go.mod"), mod, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	t.Chdir(project)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	var stdout, stderr bytes.Buffer
	getErr := bootstrap.Run(ctx, []string{"go", "get", reference}, &stdout, &stderr)
	if getErr != nil {
		got, readErr := os.ReadFile(filepath.Join(project, "go.mod"))
		_, sumErr := os.Lstat(filepath.Join(project, "go.sum"))
		_, lockErr := os.Lstat(filepath.Join(project, ".heliopause-go-transaction.lock"))
		approvals, stateErr := filepath.Glob(filepath.Join(root, "cache", "heliopause", "go-projects", "*.json"))
		if readErr != nil || string(got) != string(mod) || !os.IsNotExist(sumErr) || !os.IsNotExist(lockErr) || stateErr != nil || len(approvals) != 0 {
			t.Errorf("failed get changed original controls, retained approval or guard cleanup")
		}
		t.Fatalf("actual go get failed: %v stdout=%s stderr=%s", getErr, stdout.String(), stderr.String())
	}
	expected := fmt.Sprintf("Source: go-proxy\nModules: %d\n", modules)
	if !strings.Contains(stdout.String(), expected) {
		t.Fatalf("missing complete get result: %s", stdout.String())
	}
	selected, err := os.ReadFile(filepath.Join(project, "go.mod"))
	if err != nil || !strings.Contains(string(selected), strings.Replace(reference, "@", " ", 1)) {
		t.Fatal("approved exact module was not published")
	}
	stdout.Reset()
	stderr.Reset()
	if err := bootstrap.Run(ctx, []string{"go", "mod", "download"}, &stdout, &stderr); err != nil {
		t.Fatalf("actual retained managed download failed: %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), expected) {
		t.Fatal("managed download did not retain the complete module graph")
	}
}
