package sandbox

import (
	"context"
	"errors"
	"reflect"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
)

func TestPNPMRunSelectionKeepsOnlyFixedSealedMissingCommand(t *testing.T) {
	reference, _ := artifactnpm.ParseReference("typescript@5.9.3")
	args, err := pnpmRunSelectionArguments(reference)
	if err != nil || len(args) != 17 || args[0] != pnpmToolGuestPath || args[2] != pnpmProjectRoot || args[3] != "dlx" || args[14] != reference.Locator() || !reflect.DeepEqual(args[15:], []string{"--", pnpmRunQueryPath}) {
		t.Fatal("native one-off ABI differs", args, err)
	}
	for _, required := range []string{"--ignore-scripts", "--ignore-pnpmfile", "--ignore-workspace", "--registry=https://registry.npmjs.org/", "--reporter=silent"} {
		found := false
		for _, arg := range args {
			found = found || arg == required
		}
		if !found {
			t.Fatal("fixed query lost restriction", required)
		}
	}
	endpoints := &pnpmCountingEndpoints{}
	called := 0
	r := &PNPMResolver{runner: &recordingRunner{}, endpoints: endpoints, observer: &recordingObserver{}, policy: &recordingResolverPolicyService{}, probe: func(context.Context) (Capability, error) { called++; return Capability{Available: true}, nil }, prepare: func(context.Context) (*pnpmPreparedTool, error) { called++; return nil, errors.New("must not prepare") }}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if selected, err := r.SelectToolRun(ctx, reference); !errors.Is(err, context.Canceled) || selected.Root() != "" {
		t.Fatal("canceled native result survived", err)
	}
	for _, locator := range []string{"node@24.0.0", "bun", "rust", "typescript@latest"} {
		reference, _ := artifactnpm.ParseReference(locator)
		if args, err := pnpmRunSelectionArguments(reference); err == nil || args != nil {
			t.Fatal("provisioning reached native argv")
		}
		if selected, err := r.SelectToolRun(context.Background(), reference); err == nil || selected.Root() != "" {
			t.Fatal("invalid input reached native selection")
		}
	}
	if called != 0 || endpoints.calls != 0 {
		t.Fatal("rejected input touched infrastructure")
	}
}

func TestPNPMRunPreparationDiagnosticDoesNotAcceptOtherFailures(t *testing.T) {
	diagnostic := []byte("ERR_PNPM_DLX_COMMAND_NOT_FOUND\nCommand \"" + pnpmRunQueryPath + "\" not found")
	if !pnpmRunMissingTerminal(1, diagnostic) {
		t.Fatal("fixed producer diagnostic missing")
	}
	for _, code := range []int{0, 2, 127} {
		if pnpmRunMissingTerminal(code, diagnostic) {
			t.Fatal("different terminal became preparation DATA")
		}
	}
	for _, output := range [][]byte{nil, []byte("ERR_PNPM_DLX_COMMAND_NOT_FOUND\nCommand \"foreign\" not found"), []byte("Command \"" + pnpmRunQueryPath + "\" not found"), make([]byte, 4097)} {
		if pnpmRunMissingTerminal(1, output) {
			t.Fatal("different or unbounded diagnostic accepted")
		}
	}
}
