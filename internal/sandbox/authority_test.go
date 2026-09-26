package sandbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestContainerCreationUsesExplicitNonAuthorityEnvironment(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "fake-github-token")
	t.Setenv("NPM_TOKEN", "fake-npm-token")
	t.Setenv("PYPI_TOKEN", "fake-pypi-token")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "fake-aws-secret")
	t.Setenv("OPENAI_API_KEY", "fake-openai-key")
	t.Setenv("CI_SECRET", "fake-ci-secret")
	t.Setenv("ARBITRARY_TOKEN", "fake-arbitrary-token")
	t.Setenv("SSH_AUTH_SOCK", "/tmp/fake-ssh-agent.sock")
	t.Setenv("DOCKER_HOST", "tcp://fake-host.invalid:2375")

	session, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	sets := [][]string{
		createArguments(session),
		pythonDynamicCreateArguments(session, artifactpypi.PublicPyPIProfile().ResourcePolicy()),
		githubELFCreateArguments(session),
		pypiCreateArguments("haa-resolver-test", []string{"--add-host", "pypi.org:127.0.0.1"}),
	}
	for _, arguments := range sets {
		assertExplicitNonAuthorityEnvironment(t, arguments)
		assertNoHostAuthorityMountOrNamespace(t, arguments)
	}
}

func TestCredentialPathCorpusIsTestOnlyAndNeverCreatesHostMount(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "credential-path-corpus.txt"))
	if err != nil {
		t.Fatal(err)
	}
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	arguments := []string{
		strings.Join(createArguments(session), " "),
		strings.Join(pythonDynamicCreateArguments(session, artifactpypi.PublicPyPIProfile().ResourcePolicy()), " "),
		strings.Join(githubELFCreateArguments(session), " "),
	}
	for _, path := range strings.Fields(string(body)) {
		if !strings.HasPrefix(path, "/") {
			t.Fatalf("credential corpus contains non-absolute path %q", path)
		}
		for _, command := range arguments {
			if strings.Contains(command, path) {
				t.Fatalf("production create command enumerates credential corpus path %q", path)
			}
		}
	}
}

func TestArtifactExecutionDoesNotRequestInteractiveHostDescriptors(t *testing.T) {
	arguments := boundaryExecArguments("0123456789abcdef", boundaryLaunchMode, "/bin/true")
	for _, value := range arguments {
		if value == "-i" || value == "-t" || value == "--interactive" || value == "--tty" {
			t.Fatalf("artifact execution requested inherited interactive descriptor: %#v", arguments)
		}
	}
}

func assertExplicitNonAuthorityEnvironment(t *testing.T, arguments []string) {
	t.Helper()
	joined := strings.Join(arguments, "\x00")
	for _, entry := range []string{
		"HOME=/tmp",
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"LANG=C",
		"LC_ALL=C",
	} {
		if !strings.Contains(joined, "--env\x00"+entry) {
			t.Fatalf("container environment is missing explicit entry %q: %#v", entry, arguments)
		}
	}
	for _, sentinel := range []string{"fake-github-token", "fake-npm-token", "fake-pypi-token", "fake-aws-secret", "fake-openai-key", "fake-ci-secret", "fake-arbitrary-token", "/tmp/fake-ssh-agent.sock", "tcp://fake-host.invalid:2375"} {
		if strings.Contains(joined, sentinel) {
			t.Fatalf("container creation inherited Host sentinel %q", sentinel)
		}
	}
}

func assertNoHostAuthorityMountOrNamespace(t *testing.T, arguments []string) {
	t.Helper()
	joined := strings.Join(arguments, " ")
	if !strings.Contains(joined, "--read-only") {
		t.Fatalf("container creation lacks immutable root filesystem: %q", joined)
	}
	for _, forbidden := range []string{"--mount", "--volume", " -v ", "--privileged", "--pid host", "--network host", "/var/run/docker.sock", "SSH_AUTH_SOCK"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("container creation exposes Host authority surface %q: %q", forbidden, joined)
		}
	}
}
