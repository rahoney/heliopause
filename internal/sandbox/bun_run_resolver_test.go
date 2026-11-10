package sandbox

import (
	"context"
	"errors"
	"reflect"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
)

func TestBunRunSelectionUsesSealedMissingCommandAndRefusesBeforeIO(t *testing.T) {
	reference, _ := artifactnpm.ParseReference("typescript@5.9.3")
	args, err := bunRunSelectionArguments(reference)
	want := []string{bunToolGuestPath, "--cwd=" + bunProjectRoot, "x", "--ignore-scripts", "--registry=https://registry.npmjs.org/", "--package=typescript@5.9.3", bunRunQueryPath}
	if err != nil || !reflect.DeepEqual(args, want) {
		t.Fatal("native one-off argv differs", args, err)
	}
	called := 0
	endpoints := &pnpmCountingEndpoints{}
	r := &BunResolver{scope: &nativeProjectScope{kind: nativeRuntimeBun, runner: &recordingRunner{}, endpoints: endpoints, observer: &recordingObserver{}, policy: &recordingResolverPolicyService{}, probe: func(context.Context) (Capability, error) { called++; return Capability{Available: true}, nil }, prepare: func(context.Context) (nativeProjectTool, error) { called++; return nil, errors.New("must not prepare") }}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if cache, err := r.SelectToolRun(ctx, reference); !errors.Is(err, context.Canceled) || cache.Root() != "" {
		t.Fatal("canceled native DATA survived", err)
	}
	for _, locator := range []string{"bun", "node", "typescript@latest", "typescript@^5"} {
		reference, _ := artifactnpm.ParseReference(locator)
		if args, err := bunRunSelectionArguments(reference); err == nil || args != nil {
			t.Fatal("refused input reached native argv")
		}
		if cache, err := r.SelectToolRun(context.Background(), reference); err == nil || cache.Root() != "" {
			t.Fatal("refused input exposed DATA")
		}
	}
	if called != 0 || endpoints.calls != 0 {
		t.Fatal("rejected input touched infrastructure")
	}
	var absent *BunResolver
	if _, err := absent.SelectToolRun(context.Background(), reference); err == nil {
		t.Fatal("nil resolver accepted")
	}
}

func TestBunRunDiagnosticIsOnlyBoundedPreparationData(t *testing.T) {
	diagnostic := []byte("error: Package typescript does not provide a binary named " + bunRunQueryPath)
	if !bunRunMissingTerminal(1, diagnostic) {
		t.Fatal("fixed producer diagnostic missing")
	}
	for _, code := range []int{0, 2, 127} {
		if bunRunMissingTerminal(code, diagnostic) {
			t.Fatal("different terminal accepted")
		}
	}
	for _, output := range [][]byte{nil, []byte("does not provide a binary named foreign"), make([]byte, 4097)} {
		if bunRunMissingTerminal(1, output) {
			t.Fatal("different or unbounded output accepted")
		}
	}
}
