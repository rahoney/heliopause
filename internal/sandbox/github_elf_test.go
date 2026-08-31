package sandbox

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestGitHubELFCreateUsesRootInitializerAndBoundaryBootstrapExec(t *testing.T) {
	sessionID, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	arguments := githubELFCreateArguments(sessionID)
	joined := strings.Join(arguments, " ")
	for _, required := range []string{
		"--tmpfs " + boundaryHelperMount,
		boundaryContainerCommand(),
		"--read-only",
		"--network none",
		"--cap-drop ALL",
		"--cap-add SETUID",
		"--cap-add SETGID",
		"--cap-add SETPCAP",
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf("GitHub ELF create command missing %q: %q", required, joined)
		}
	}
	if strings.Contains(joined, "--user "+boundaryBootstrapUser) || strings.Contains(joined, "docker cp") {
		t.Fatalf("GitHub ELF helper initializer is not root-owned OCI setup: %q", joined)
	}
	if !sameStrings(boundaryExecArguments("0123456789abcdef", boundaryELFHandoffMode, "/work/artifact"), []string{"exec", "--user", boundaryBootstrapUser, "0123456789abcdef", boundaryHelperPath, boundaryELFHandoffMode, "/work/artifact"}) {
		t.Fatal("ELF handoff is not a fixed boundary bootstrap exec")
	}
}
