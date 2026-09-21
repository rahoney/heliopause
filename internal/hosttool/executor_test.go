package hosttool

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
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
		[]byte(`{"path":"/usr/local/bin/runsc","unknown":true}`),
		[]byte(`{"path":"/usr/local/bin/runsc"} {}`),
	} {
		if _, err := parseRunscRegistration(body); err == nil {
			t.Fatalf("parseRunscRegistration(%q) accepted mismatched registration", body)
		}
	}
}

func TestParseRunscRegistrationRequiresStrictSidecarPolicy(t *testing.T) {
	t.Parallel()
	valid := []byte(`{"path":"/usr/local/bin/runsc","runtimeArgs":["--sidecar-usage-policy=STRICT","--pod-init-config=/etc/heliopause/pod-init.json"],"status":{"features":"bounded-daemon-data"}}`)
	if got, err := parseRunscRegistration(valid); err != nil || got != "/usr/local/bin/runsc" {
		t.Fatalf("valid registration path=%q error=%v", got, err)
	}
	if _, err := parseRunscRegistration([]byte(`{"path":"/usr/local/bin/runsc"}`)); err == nil {
		t.Fatal("registration without runtime arguments was accepted")
	}
	for name, arguments := range map[string][]string{
		"empty":                            {},
		"strict absent":                    {"--pod-init-config=/etc/heliopause/pod-init.json"},
		"double dash default":              {"--sidecar-usage-policy=DEFAULT", "--pod-init-config=/etc/heliopause/pod-init.json"},
		"single dash default":              {"-sidecar-usage-policy=DEFAULT", "--pod-init-config=/etc/heliopause/pod-init.json"},
		"single dash strict":               {"-sidecar-usage-policy=STRICT", "--pod-init-config=/etc/heliopause/pod-init.json"},
		"separate strict value":            {"--sidecar-usage-policy", "STRICT", "--pod-init-config=/etc/heliopause/pod-init.json"},
		"malformed attached value":         {"--sidecar-usage-policy=STRICT=DEFAULT", "--pod-init-config=/etc/heliopause/pod-init.json"},
		"malformed separate value":         {"--sidecar-usage-policy", "DEFAULT", "--pod-init-config=/etc/heliopause/pod-init.json"},
		"duplicate strict":                 {"--sidecar-usage-policy=STRICT", "--sidecar-usage-policy=STRICT", "--pod-init-config=/etc/heliopause/pod-init.json"},
		"strict then default":              {"--sidecar-usage-policy=STRICT", "-sidecar-usage-policy=DEFAULT", "--pod-init-config=/etc/heliopause/pod-init.json"},
		"default then strict":              {"--sidecar-usage-policy=DEFAULT", "--sidecar-usage-policy=STRICT", "--pod-init-config=/etc/heliopause/pod-init.json"},
		"allow override double shorthand":  {"--sidecar-usage-policy=STRICT", "--allow-flag-override", "--pod-init-config=/etc/heliopause/pod-init.json"},
		"allow override single shorthand":  {"--sidecar-usage-policy=STRICT", "-allow-flag-override", "--pod-init-config=/etc/heliopause/pod-init.json"},
		"allow override double attached":   {"--sidecar-usage-policy=STRICT", "--allow-flag-override=true", "--pod-init-config=/etc/heliopause/pod-init.json"},
		"allow override single attached":   {"--sidecar-usage-policy=STRICT", "-allow-flag-override=false", "--pod-init-config=/etc/heliopause/pod-init.json"},
		"allow override separate":          {"--sidecar-usage-policy=STRICT", "--allow-flag-override", "false", "--pod-init-config=/etc/heliopause/pod-init.json"},
		"allow override single separate":   {"--sidecar-usage-policy=STRICT", "-allow-flag-override", "false", "--pod-init-config=/etc/heliopause/pod-init.json"},
		"strict consumed by debug command": {"--debug-command", "--sidecar-usage-policy=STRICT"},
		"strict consumed by pod init":      {"--pod-init-config", "--sidecar-usage-policy=STRICT"},
		"unknown flag before strict":       {"--unknown", "--sidecar-usage-policy=STRICT", "--pod-init-config=/etc/heliopause/pod-init.json"},
		"unknown flag after strict":        {"--sidecar-usage-policy=STRICT", "--pod-init-config=/etc/heliopause/pod-init.json", "--unknown"},
		"end of flags":                     {"--", "--sidecar-usage-policy=STRICT", "--pod-init-config=/etc/heliopause/pod-init.json"},
		"positional argument":              {"--sidecar-usage-policy=STRICT", "run", "--pod-init-config=/etc/heliopause/pod-init.json"},
		"strict looking pod init value":    {"--sidecar-usage-policy=STRICT", "--pod-init-config=/etc/sidecar-usage-policy.json"},
		"case variant":                     {"--sidecar-usage-policy=strict", "--pod-init-config=/etc/heliopause/pod-init.json"},
		"whitespace value":                 {"--sidecar-usage-policy=STRICT ", "--pod-init-config=/etc/heliopause/pod-init.json"},
	} {
		t.Run(name, func(t *testing.T) {
			body, err := json.Marshal(struct {
				Path        string   `json:"path"`
				RuntimeArgs []string `json:"runtimeArgs"`
			}{Path: "/usr/local/bin/runsc", RuntimeArgs: arguments})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := parseRunscRegistration(body); err == nil {
				t.Fatalf("parseRunscRegistration(%q) accepted non-canonical runtime arguments", arguments)
			}
		})
	}
}

func TestRegisteredRunscPathMustBeCanonicalHAAInstallation(t *testing.T) {
	if err := validateRegisteredRunscPath("/usr/local/bin/runsc"); err == nil {
		t.Fatal("non-canonical Docker runtime path accepted")
	}
	if err := validateRegisteredRunscPath(runtimeidentity.LocalRunscPath); err != nil {
		t.Fatal(err)
	}
}

func validGVisorBundleManifest() runtimeidentity.LocalGVisorBundleManifest {
	members := make([]runtimeidentity.GVisorBundleMember, len(runtimeidentity.GVisorRuntimeBundleMembers))
	for i, member := range runtimeidentity.GVisorRuntimeBundleMembers {
		members[i] = runtimeidentity.GVisorBundleMember(member)
	}
	return runtimeidentity.LocalGVisorBundleManifest{
		SchemaVersion: runtimeidentity.LocalGVisorBundleSchema, Architecture: runtime.GOARCH,
		GVisorCommit: runtimeidentity.GVisorCommit, GVisorPatchSHA256: runtimeidentity.GVisorPatchSHA256,
		BazelVersion: runtimeidentity.BazelVersion, BazelBinarySHA512: runtimeidentity.BazelLinuxX8664SHA512,
		Members: members,
	}
}

func TestBundleMemberDigestMismatchRejected(t *testing.T) {
	manifest := validGVisorBundleManifest()
	manifest.Members[0].SHA512 = strings.Repeat("0", 128)
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtimeidentity.ParseLocalGVisorBundleManifest(body, runtime.GOARCH); err == nil {
		t.Fatal("changed bundle member hash was accepted")
	}
}

func TestChangedManifestRejectedDuringRevalidation(t *testing.T) {
	manifest := validGVisorBundleManifest()
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	digest, _, err := validateLocalGVisorBundleManifestBody(body, "amd64", "")
	if err != nil {
		t.Fatal(err)
	}
	changed := append([]byte(nil), body...)
	changed[len(changed)-2] ^= 1
	if _, _, err := validateLocalGVisorBundleManifestBody(changed, "amd64", digest); err == nil {
		t.Fatal("changed manifest retained its validated identity")
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

func TestOrdinaryExecutorDoesNotRegisterFirewallTools(t *testing.T) {
	t.Parallel()
	tool, err := verifyExecutable(trustedFixtureExecutable(t), "")
	if err != nil {
		t.Fatal(err)
	}
	executor := &Executor{tools: map[string]identity{"docker": tool}, clientHome: t.TempDir()}
	for _, name := range []string{"iptables", "nft"} {
		if _, err := executor.LookPath(name); err == nil {
			t.Fatalf("ordinary executor exposed %s", name)
		}
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
