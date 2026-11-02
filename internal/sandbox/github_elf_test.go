package sandbox

import (
	"testing"
)

func TestGitHubELFHandoffUsesBoundaryBootstrapExec(t *testing.T) {
	if !sameStrings(boundaryExecArguments("0123456789abcdef", boundaryELFHandoffMode, "/work/artifact"), []string{"exec", "--user", boundaryBootstrapUser, "0123456789abcdef", boundaryHelperPath, boundaryOriginELFHandoffMode, "/work/artifact"}) {
		t.Fatal("ELF handoff is not a fixed boundary bootstrap exec")
	}
}
