package sandbox

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

func TestUVRunPreparationBindsPublicRequestAndNeverSelectedProgram(t *testing.T) {
	requirement, err := artifactpypi.ParseUpgradeRequirement("probe@1.0")
	if err != nil {
		t.Fatal(err)
	}
	args, err := uvRunPreparationArguments(requirement, "probe")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{uvToolGuestPath, "tool", "run", "--isolated", "--from", "probe==1.0", "--offline", "--no-index", "--find-links", pipxNativeWheels, "--no-build", "--no-config", "--cache-dir", "/haa-site/uv-run-cache", "--no-python-downloads", "--no-managed-python", "--python", "/usr/local/bin/python", "--link-mode", "copy", "--no-sources", "/dev/null"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatal("native request/closure/interpreter/terminal differs", args)
	}
	for _, arg := range args {
		if arg == "probe" {
			t.Fatal("selected program passed to native execution")
		}
	}
	for _, bad := range []string{"", "../probe", "sh -c probe", "python;probe", "/usr/bin/probe", "-V", "${probe}", "héllo"} {
		if out, err := uvRunPreparationArguments(requirement, bad); err == nil || len(out) != 0 {
			t.Fatal("unsafe command admitted", bad)
		}
	}
	rangeRequest, _ := artifactpypi.ParseUpgradeRequirement("probe>=1")
	if _, err := uvRunPreparationArguments(rangeRequest, "probe"); err == nil {
		t.Fatal("range expanded initial execution subset")
	}
}

func TestUVRunFixedTerminalIsOnlyBoundedNativeDiagnostic(t *testing.T) {
	// A controlled child supplies an ordinary nonzero ProcessState. This is
	// not native UV, observation completeness, approval or selected-program proof.
	err := exec.Command("/bin/sh", "-c", "exit 2").Run()
	var child *exec.ExitError
	if !errors.As(err, &child) {
		t.Fatal("controlled nonzero fixture", err)
	}
	terminal := func(text string) error {
		return &exec.ExitError{ProcessState: child.ProcessState, Stderr: []byte(text)}
	}
	expected := terminal("error: /dev/null: Permission denied")
	if !uvRunPreparedTerminal(nil, expected) {
		t.Fatal("bounded inert diagnostic rejected")
	}
	for _, row := range []struct {
		output []byte
		err    error
	}{
		{nil, nil}, {[]byte("program output"), expected}, {nil, errors.New("/dev/null Permission denied")},
		{nil, terminal("/dev/null no permission")}, {nil, terminal("Permission denied elsewhere")},
		{nil, terminal(strings.Repeat("x", 64<<10) + "/dev/null Permission denied")},
	} {
		if uvRunPreparedTerminal(row.output, row.err) {
			t.Fatal("different/incomplete terminal admitted")
		}
	}
}

func TestUVRunSelectionUsesNativeIdentityAndBindsCompleteUnapprovedDATA(t *testing.T) {
	frozen, facts, _ := uvRunBindingFixture(t)
	requirement, _ := artifactpypi.ParseUpgradeRequirement("probe@1.0")
	original := cloneUVRunFacts(facts)
	selected, err := uvRunSelectionFromCensus(context.Background(), requirement, frozen, "probe", facts)
	if err != nil {
		t.Fatal(err)
	}
	resolution := selected.Resolution()
	if !selected.Entry().Valid() || selected.Entry().Command() != "probe" || resolution.RuntimeIdentity() != "uv-tool-run-isolated:"+UVResolverRuntimeIdentity() || resolution.LockfileDigest().String() == "" || len(resolution.Graph().Nodes()) != 1 || len(resolution.ProjectControls()) != 0 {
		t.Fatal("native selection differs or invents install controls")
	}
	if !reflect.DeepEqual(facts, original) {
		t.Fatal("native originals changed")
	}
	reversed := cloneUVRunFacts(facts)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	reordered, err := uvRunSelectionFromCensus(context.Background(), requirement, frozen, "probe", reversed)
	if err != nil || reordered.Resolution().LockfileDigest() != resolution.LockfileDigest() {
		t.Fatal("enumeration order changed selection fingerprint", err)
	}
	bare, _ := artifactpypi.ParseUpgradeRequirement("probe")
	other, err := uvRunSelectionFromCensus(context.Background(), bare, frozen, "probe", facts)
	if err != nil || other.Resolution().LockfileDigest() == resolution.LockfileDigest() {
		t.Fatal("request omitted from fingerprint", err)
	}
	changed := cloneUVRunFacts(facts)
	found := false
	for i := range changed {
		if strings.HasSuffix(changed[i].Name, "/RECORD") {
			changed[i].SHA256 = strings.Repeat("a", 64)
			found = true
		}
	}
	if !found {
		t.Fatal("native RECORD fixture absent")
	}
	unapproved, err := uvRunSelectionFromCensus(context.Background(), requirement, frozen, "probe", changed)
	if err != nil || unapproved.Resolution().LockfileDigest() == resolution.LockfileDigest() {
		t.Fatal("unapproved native RECORD became ownership or escaped DATA fingerprint", err)
	}
	cause := errors.New("original execution request cancellation")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if rejected, err := uvRunSelectionFromCensus(ctx, requirement, frozen, "probe", facts); !errors.Is(err, cause) || rejected.Entry().Valid() || len(rejected.Resolution().Graph().Nodes()) != 0 {
		t.Fatal("cancellation lost or partial result published", err)
	}
	for _, row := range []struct {
		ctx     context.Context
		request artifactpypi.UpgradeRequirement
		input   *pipxNativeInput
		command string
		data    []uvRunCensusFact
	}{
		{nil, requirement, frozen, "probe", facts}, {context.Background(), requirement, nil, "probe", facts},
		{context.Background(), artifactpypi.UpgradeRequirement{}, frozen, "probe", facts},
		{context.Background(), requirement, frozen, "missing", facts}, {context.Background(), requirement, frozen, "probe", nil},
	} {
		rejected, err := uvRunSelectionFromCensus(row.ctx, row.request, row.input, row.command, row.data)
		if err == nil || rejected.Entry().Valid() || len(rejected.Resolution().Graph().Nodes()) != 0 {
			t.Fatal("incomplete/foreign selection published")
		}
	}
	frozen.files[0].body = bytes.Replace(frozen.files[0].body, []byte("probe"), []byte("other"), 1)
	if rejected, err := uvRunSelectionFromCensus(context.Background(), requirement, frozen, "probe", facts); err == nil || rejected.Entry().Valid() {
		t.Fatal("changed original accepted")
	}
}

func TestUVRunResolverRejectsBeforeAnyNativeOrNetworkWork(t *testing.T) {
	requirement, _ := artifactpypi.ParseUpgradeRequirement("probe@1.0")
	for _, kind := range []string{"nil context", "nil resolver", "nil reader", "empty request", "unsafe command", "range", "canceled"} {
		t.Run(kind, func(t *testing.T) {
			_, runner := uvReaderFixture(t)
			endpoints := &uvCountingEndpoints{}
			resolver, err := NewUVResolver(runner, endpoints, &recordingObserver{}, availablePythonProbe, &recordingResolverPolicyService{})
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			request := requirement
			command := "probe"
			cause := errors.New("retained request cause")
			switch kind {
			case "nil context":
				ctx = nil
			case "nil resolver":
				resolver = nil
			case "nil reader":
				resolver = &UVResolver{}
			case "empty request":
				request = artifactpypi.UpgradeRequirement{}
			case "unsafe command":
				command = "../probe"
			case "range":
				request, _ = artifactpypi.ParseUpgradeRequirement("probe>=1")
			case "canceled":
				var cancel context.CancelCauseFunc
				ctx, cancel = context.WithCancelCause(ctx)
				cancel(cause)
			}
			selected, err := resolver.ResolveToolRun(ctx, request, command)
			if err == nil || selected.Entry().Valid() || len(selected.Resolution().Graph().Nodes()) != 0 || len(runner.calls) != 0 || len(runner.inputCalls) != 0 || endpoints.calls != 0 {
				t.Fatal("invalid request reached native/network or returned selection", err)
			}
			if kind == "canceled" && !errors.Is(err, cause) {
				t.Fatal("original cancellation cause lost", err)
			}
		})
	}
}

func TestUVRunPreparationRejectsConjoinedExactConstraint(t *testing.T) {
	request, err := artifactpypi.ParseUpgradeRequirement("probe==1.0,>=1.0")
	if err != nil {
		t.Fatal("supported requirement parser fixture", err)
	}
	if args, err := uvRunPreparationArguments(request, "probe"); err == nil || len(args) != 0 {
		t.Fatal("conjoined range escaped bare/exact execution subset")
	}
}
