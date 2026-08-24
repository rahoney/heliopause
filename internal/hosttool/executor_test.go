package hosttool

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"
)

func TestVerifyExecutableRejectsHostileIdentity(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	writable := filepath.Join(directory, "docker")
	if err := os.WriteFile(writable, []byte("fixture"), 0o777); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(directory, "runsc")
	if err := os.Symlink(writable, symlink); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"docker", writable, symlink} {
		if _, err := verifyExecutable(path, ""); err == nil {
			t.Fatalf("verifyExecutable(%q) accepted an untrusted identity", path)
		}
	}
}

func TestVerifyExecutableRejectsDigestMismatch(t *testing.T) {
	t.Parallel()
	path := trustedFixtureExecutable(t)
	if _, err := verifyExecutable(path, strings.Repeat("0", 128)); err == nil {
		t.Fatal("verifyExecutable accepted a wrong runsc digest")
	}
}

func TestParseRunscRegistrationRejectsDaemonRuntimeMismatch(t *testing.T) {
	t.Parallel()
	for _, body := range [][]byte{
		[]byte(`{"path":"runsc"}`),
		[]byte(`{"path":"/usr/local/bin/../bin/runsc"}`),
		[]byte(`{"path":"/usr/local/bin/runsc"`),
	} {
		if _, err := parseRunscRegistration(body); err == nil {
			t.Fatalf("parseRunscRegistration(%q) accepted mismatched registration", body)
		}
	}
	if got, err := parseRunscRegistration([]byte(`{"path":"/usr/local/bin/runsc"}`)); err != nil || got != "/usr/local/bin/runsc" {
		t.Fatalf("valid registration path=%q error=%v", got, err)
	}
}

func TestExecutorRejectsChangedExecutableIdentity(t *testing.T) {
	t.Parallel()
	firstPath := trustedFixtureExecutable(t)
	first, err := verifyExecutable(firstPath, "")
	if err != nil {
		t.Fatal(err)
	}
	secondPath := ""
	for _, candidate := range []string{"/usr/bin/false", "/bin/false"} {
		if candidate != firstPath {
			if _, candidateErr := verifyExecutable(candidate, ""); candidateErr == nil {
				secondPath = candidate
				break
			}
		}
	}
	if secondPath == "" {
		t.Skip("no second root-owned fixture executable")
	}
	first.path = secondPath
	executor := &Executor{tools: map[string]identity{"docker": first}, clientHome: t.TempDir()}
	if _, err := executor.LookPath("docker"); err == nil {
		t.Fatal("executor accepted a replaced executable identity")
	}
}

func TestCommandUsesAbsoluteIdentityMinimalEnvironmentAndExplicitDockerEndpoint(t *testing.T) {
	t.Setenv("PATH", "/tmp/hostile-path")
	t.Setenv("DOCKER_HOST", "tcp://attacker.invalid:2375")
	t.Setenv("DOCKER_CONTEXT", "attacker")
	t.Setenv("HTTPS_PROXY", "http://attacker.invalid")
	path := trustedFixtureExecutable(t)
	tool, err := verifyExecutable(path, "")
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	_ = tool
	arguments := dockerArguments("unix:///run/docker.sock", home, []string{"version"})
	wantPrefix := []string{path, "--host", "unix:///run/docker.sock", "--config", home, "version"}
	gotArgs := append([]string{path}, arguments...)
	if !slices.Equal(gotArgs, wantPrefix) {
		t.Fatalf("command args = %#v, want %#v", gotArgs, wantPrefix)
	}
	environment := minimalEnvironment(home)
	joined := strings.Join(environment, "\n")
	for _, forbidden := range []string{"hostile-path", "DOCKER_HOST", "DOCKER_CONTEXT", "HTTPS_PROXY", "attacker.invalid"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("minimal environment inherited %q: %s", forbidden, joined)
		}
	}
	if len(environment) != 4 {
		t.Fatalf("environment = %#v", environment)
	}
}

func TestVerifyLocalSocketRejectsRemoteAndUnsafeEndpoint(t *testing.T) {
	t.Parallel()
	for _, endpoint := range []string{"tcp://127.0.0.1:2375", "ssh://host/run/docker.sock", "unix://relative.sock", "unix:///tmp/missing.sock"} {
		if _, err := verifyLocalSocket(endpoint); err == nil {
			t.Fatalf("verifyLocalSocket(%q) accepted unsafe endpoint", endpoint)
		}
	}
}

func TestVerifyLocalSocketAcceptsProtectedUserOwnedRuntimeDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket fixture")
	}
	directory, err := os.MkdirTemp(".", ".endpoint-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)
	if err := os.Chmod(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	path, err := filepath.Abs(filepath.Join(directory, "docker.sock"))
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	listener, listenErr := net.Listen("unix", "docker.sock")
	if err := os.Chdir(workingDirectory); err != nil {
		t.Fatal(err)
	}
	err = listenErr
	if err != nil {
		if errors.Is(err, syscall.EPERM) {
			t.Skip("test sandbox prohibits Unix socket creation")
		}
		t.Fatal(err)
	}
	defer listener.Close()
	got, err := verifyLocalSocket("unix://" + path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "unix://"+path {
		t.Fatalf("endpoint = %q", got)
	}
}

func trustedFixtureExecutable(t *testing.T) string {
	t.Helper()
	for _, path := range []string{"/usr/bin/true", "/bin/true"} {
		if _, err := verifyExecutable(path, ""); err == nil {
			return path
		}
	}
	t.Skip("no root-owned non-symlink fixture executable")
	return ""
}
