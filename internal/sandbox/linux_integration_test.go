package sandbox

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestLinuxGVisorLifecycleIntegration(t *testing.T) {
	if os.Getenv("HELOX_GVISOR_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor runtime")
	}
	root := t.TempDir()
	path := filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa", "tarball.tgz")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	writer := tar.NewWriter(gzipWriter)
	body := `{"name":"tiny","version":"1.2.3"}`
	if err := writer.WriteHeader(&tar.Header{Name: "package/package.json", Size: int64(len(body)), Mode: 0o600}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	observer, err := NewSharedObserver("/run/heliopause-observer/haa-output.sock")
	if err != nil {
		t.Fatal(err)
	}
	helperPath := os.Getenv("HELOX_GVISOR_HELPER")
	if helperPath == "" {
		t.Fatal("HELOX_GVISOR_HELPER is required")
	}
	helper := exec.Command(helperPath, "/run/heliopause-observer/gvisor-remote.sock", "/run/heliopause-observer/haa-output.sock")
	if err := helper.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = helper.Process.Kill(); _ = helper.Wait() }()
	runner := integrationRunner{t: t}
	introducer, err := NewDockerArtifactIntroducer(root, runner)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := NewBackend(runner, introducer, observer, Probe)
	if err != nil {
		t.Fatal(err)
	}
	result, err := backend.Execute(context.Background(), integrationRequest(t))
	if err != nil {
		t.Fatal(err)
	}
	if result.Status() != domain.SandboxCompleted {
		code, _ := result.LimitationCode()
		t.Fatalf("Sandbox result = %q/%q", result.Status(), code)
	}
}

func TestLinuxNPMResolverNetworkPolicyIntegration(t *testing.T) {
	if os.Getenv("HELOX_NPM_RESOLVER_INTEGRATION") != "1" {
		t.Skip("requires privileged Linux Docker firewall integration")
	}
	if os.Geteuid() != 0 {
		t.Fatal("resolver network policy integration requires explicit CAP_NET_ADMIN elevation")
	}
	resolver, err := NewNPMResolver(integrationRunner{t: t}, systemEndpointResolver{})
	if err != nil {
		t.Fatal(err)
	}
	reference, err := artifactnpm.ParseReference("is-number@7.0.0")
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget("/tmp/heliopause-resolver-target")
	installContext, _ := domain.NewInstallContext(target)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	resolution, err := resolver.ResolveDependencies(ctx, reference, installContext)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolution.Graph().Nodes()) == 0 || resolution.Graph().Primary().String() == "" || resolution.RuntimeIdentity() == "" || resolution.LockfileDigest().String() == "" {
		t.Fatalf("resolver resolution = %#v", resolution)
	}
}

type integrationRunner struct {
	t *testing.T
}

func (r integrationRunner) Output(ctx context.Context, binary string, arguments ...string) ([]byte, error) {
	r.t.Helper()
	command := exec.CommandContext(ctx, binary, arguments...)
	output, err := command.Output()
	if err != nil {
		stderr := ""
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			stderr = strings.TrimSpace(string(exitError.Stderr))
		}
		r.t.Logf("command failed: %s %q: %v; stdout=%q; stderr=%q", binary, arguments, err, strings.TrimSpace(string(output)), stderr)
		// This is integration-test-only diagnostic output. Production adapter
		// errors remain sanitized and never carry command output.
		fmt.Fprintf(os.Stderr, "integration command failed: %s %q: %v; stdout=%q; stderr=%q\n", binary, arguments, err, strings.TrimSpace(string(output)), stderr)
	}
	if err == nil && binary == "docker" && len(arguments) == 2 && arguments[0] == "wait" && strings.TrimSpace(string(output)) != "0" {
		logs, logsErr := exec.CommandContext(ctx, "docker", "logs", arguments[1]).CombinedOutput()
		r.t.Logf("container exited with %q; logs=%q; logs error=%v", strings.TrimSpace(string(output)), strings.TrimSpace(string(logs)), logsErr)
	}
	return output, err
}

func (r integrationRunner) RunInput(ctx context.Context, input io.Reader, binary string, arguments ...string) error {
	r.t.Helper()
	command := exec.CommandContext(ctx, binary, arguments...)
	command.Stdin = input
	output, err := command.CombinedOutput()
	if err != nil {
		r.t.Logf("input command failed: %s %q: %v; output=%q", binary, arguments, err, strings.TrimSpace(string(output)))
	}
	return err
}

func integrationRequest(t *testing.T) domain.SandboxRequest {
	t.Helper()
	source, _ := domain.NewSourceID("npm")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "tiny", "1.2.3", "tarball")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:tarball", 1)
	if err != nil {
		t.Fatal(err)
	}
	request, err := domain.NewSandboxRequest(artifact)
	if err != nil {
		t.Fatal(err)
	}
	return request
}
