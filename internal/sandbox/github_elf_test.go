package sandbox

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestGitHubELFCreateUsesRootInitializerAndUserArtifactExec(t *testing.T) {
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
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf("GitHub ELF create command missing %q: %q", required, joined)
		}
	}
	if strings.Contains(joined, "--user "+boundaryExecUser) || strings.Contains(joined, "docker cp") {
		t.Fatalf("GitHub ELF helper initializer is not root-owned OCI setup: %q", joined)
	}
	if !sameStrings(boundaryExecArguments("0123456789abcdef", boundaryELFHandoffMode, "/work/artifact"), []string{"exec", "--user", boundaryExecUser, "0123456789abcdef", boundaryHelperPath, boundaryELFHandoffMode, "/work/artifact"}) {
		t.Fatal("ELF handoff is not a user-scoped boundary exec")
	}
}
