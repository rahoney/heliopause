package sandbox

import (
	"errors"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestClosureFailureRetainsCauseAndIncompleteStatus(t *testing.T) {
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("installed file hash differs from RECORD")
	result, err := pythonClosureFailure(session, "verify prepared installation", cause)
	if !errors.Is(err, cause) || !strings.Contains(err.Error(), "verify prepared installation") {
		t.Fatalf("underlying failure lost: %v", err)
	}
	if result.Status() != domain.SandboxIncomplete {
		t.Fatal("closure failure became qualifying")
	}
	code, _ := result.LimitationCode()
	if code != "M5_PYPI_DYNAMIC_CLOSURE_INVALID" {
		t.Fatalf("wrong limitation: %s", code)
	}
}
