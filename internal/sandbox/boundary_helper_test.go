package sandbox

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestBoundaryContainerCommandInstallsImmutableHelperBeforeServingExecs(t *testing.T) {
	command := boundaryContainerCommand()
	for _, required := range []string{
		"tmp=/haa-runtime/.haa-boundary.tmp",
		"chown 0:0 \"$tmp\"",
		"chmod 0555 \"$tmp\"",
		"mv \"$tmp\" " + boundaryHelperPath,
		"exec " + boundarySetprivPath + " " + boundaryDemotionArguments + " /bin/sleep infinity",
	} {
		if !strings.Contains(command, required) {
			t.Fatalf("initializer missing %q: %q", required, command)
		}
	}
	if strings.Contains(command, "docker cp") || strings.Contains(command, "--user "+boundaryExecUser) {
		t.Fatalf("initializer has an invalid helper installation surface: %q", command)
	}
}

func TestAwaitBoundaryHelperUsesOnlyWrappedUserExecAndFailsClosed(t *testing.T) {
	runner := &recordingRunner{errors: []error{errors.New("not ready")}}
	if err := awaitBoundaryHelper(context.Background(), runner, "0123456789abcdef"); err != nil {
		t.Fatalf("awaitBoundaryHelper() = %v", err)
	}
	if len(runner.calls) != 2 || !sameStrings(runner.calls[0].arguments, boundaryReadinessArguments("0123456789abcdef")) || !sameStrings(runner.calls[1].arguments, boundaryReadinessArguments("0123456789abcdef")) {
		t.Fatalf("helper readiness calls = %#v", runner.calls)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := awaitBoundaryHelper(ctx, &recordingRunner{errors: []error{errors.New("not ready")}}, "0123456789abcdef"); err == nil {
		t.Fatal("awaitBoundaryHelper() accepted unavailable helper")
	}
}
