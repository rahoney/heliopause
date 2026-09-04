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
	if strings.Contains(command, "docker cp") || strings.Contains(command, "--user "+boundaryBootstrapUser) {
		t.Fatalf("initializer has an invalid helper installation surface: %q", command)
	}
}

func TestBoundaryExecUsesFixedRootBootstrapBeforeDemotingTarget(t *testing.T) {
	for _, test := range []struct {
		mode   string
		origin string
	}{
		{boundaryLaunchMode, boundaryOriginLaunchMode},
		{boundaryPythonHandoffMode, boundaryOriginPythonHandoffMode},
		{boundaryELFHandoffMode, boundaryOriginELFHandoffMode},
	} {
		arguments := boundaryExecArguments("0123456789abcdef", test.mode, "/bin/true")
		want := []string{"exec", "--user", boundaryBootstrapUser, "0123456789abcdef", boundaryHelperPath, test.origin, "/bin/true"}
		if !sameStrings(arguments, want) {
			t.Fatalf("boundary exec for %q = %#v, want %#v", test.mode, arguments, want)
		}
	}
	if !strings.Contains(boundaryHelper, "exec "+boundarySetprivPath+" "+boundaryDemotionArguments) {
		t.Fatalf("boundary helper does not irreversibly demote requested targets: %q", boundaryHelper)
	}
	for _, required := range []string{
		"exec " + boundaryHelperPath + " " + boundaryLaunchMode,
		"exec " + boundaryHelperPath + " " + boundaryPythonHandoffMode,
		"exec " + boundaryHelperPath + " " + boundaryELFHandoffMode,
	} {
		if !strings.Contains(boundaryHelper, required) {
			t.Fatalf("boundary helper does not self-exec canonical mode %q: %q", required, boundaryHelper)
		}
	}
	if strings.Contains(boundaryReadinessScript, "NoNewPrivs") {
		t.Fatalf("readiness relies on unsupported /proc NoNewPrivs: %q", boundaryReadinessScript)
	}
	if !strings.Contains(boundaryHelper, "  -c) shift; already_demoted; exec /bin/sh -c") ||
		strings.Contains(boundaryHelper, "--origin-c") {
		t.Fatalf("npm script-shell is not a direct in-container handoff: %q", boundaryHelper)
	}
	for _, required := range []string{
		"done < /proc/self/status",
		`Uid) [ "$#" -eq 4 ]`,
		`Gid) [ "$#" -eq 4 ]`,
		`Groups) [ "$#" -eq 0 ]`,
		`CapInh) [ "$#" -eq 1 ]`,
		`CapPrm) [ "$#" -eq 1 ]`,
		`CapEff) [ "$#" -eq 1 ]`,
		`CapBnd) [ "$#" -eq 1 ]`,
		`CapAmb) [ "$#" -eq 1 ]`,
		`1:1:1:1:1:1:1:1`,
	} {
		if !strings.Contains(boundaryHelper, required) {
			t.Fatalf("already-demoted npm handoff validation missing %q: %q", required, boundaryHelper)
		}
	}
	if strings.Contains(boundaryHelper, "-c) shift; demote") {
		t.Fatalf("already-demoted npm handoff still invokes privileged demotion: %q", boundaryHelper)
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
