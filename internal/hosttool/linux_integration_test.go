package hosttool

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinuxTrustedHostExecutorIntegration(t *testing.T) {
	if os.Getenv("HELOX_HOSTTOOL_INTEGRATION") != "1" {
		t.Skip("set HELOX_HOSTTOOL_INTEGRATION=1 in the controlled Linux qualification environment")
	}
	t.Setenv("PATH", "/tmp/haa-hostile-path")
	t.Setenv("DOCKER_HOST", "tcp://127.0.0.1:2375")
	t.Setenv("DOCKER_CONTEXT", "hostile")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:9")
	executor, err := NewSystem(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	})
	for _, name := range []string{"docker", "runsc"} {
		path, err := executor.LookPath(name)
		if err != nil || !filepath.IsAbs(path) {
			t.Fatalf("trusted %s path=%q error=%v", name, path, err)
		}
	}
	version, err := executor.Output(context.Background(), "docker", "version", "--format", "{{.Server.Version}}")
	if err != nil || !atLeastVersion(strings.TrimSpace(string(version)), minimumDockerEngine) {
		t.Fatalf("trusted local Docker version=%q error=%v", version, err)
	}
	registration, err := executor.Output(context.Background(), "docker", "info", "--format", "{{json (index .Runtimes \"runsc-trace\")}}")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseRunscRegistration(registration); err != nil {
		t.Fatal(err)
	}
	runscVersion, err := executor.Output(context.Background(), "runsc", "--version")
	if err != nil || !strings.Contains(string(runscVersion), gVisorRelease) {
		t.Fatalf("runsc --version = %q, %v", runscVersion, err)
	}
	traceMeta, err := executor.Output(context.Background(), "runsc", "trace", "metadata")
	if err != nil {
		t.Fatalf("runsc trace metadata = %v", err)
	}
	for _, point := range []string{"syscall/open_result", "sentry/mount_topology_snapshot", "sentry/mount_topology_mutation"} {
		if !strings.Contains(string(traceMeta), point) {
			t.Fatalf("runsc trace metadata missing required point %q", point)
		}
	}
}
