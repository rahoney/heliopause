package sandbox

import (
	"context"
	"errors"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func bunResolverInput(t *testing.T, intent artifactnpm.BunSelectionIntent, selector, manifest string) artifactnpm.BunSelectionInput {
	t.Helper()
	control, err := domain.NewProjectControlFile("package.json", []byte(manifest), true)
	if err != nil {
		t.Fatal(err)
	}
	input, err := artifactnpm.NewBunSelectionInput(intent, selector, []domain.ProjectControlFile{control}, runtimeidentity.BunVersion)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func TestBunResolverPreflightHasNoResourceSideEffects(t *testing.T) {
	for _, runner := range []CommandRunner{nil, &recordingRunner{}, uvStreamOnlyRunner{}} {
		if resolver, err := NewBunResolver(runner, &pnpmCountingEndpoints{}, &recordingObserver{}, availableProbe, &recordingResolverPolicyService{}); err == nil || resolver != nil {
			t.Fatal("incomplete transport accepted")
		}
	}
	for _, mode := range []string{"zero", "cancel", "cancel-cause", "probe", "tool", "wrong-kind"} {
		t.Run(mode, func(t *testing.T) {
			runner := &pnpmControlTransport{}
			endpoints := &pnpmCountingEndpoints{}
			probes, tools := 0, 0
			cause := errors.New("fixed Bun unavailable")
			r, err := NewBunResolver(runner, endpoints, &recordingObserver{}, func(context.Context) (Capability, error) {
				probes++
				return Capability{Available: mode != "probe"}, nil
			}, &recordingResolverPolicyService{})
			if err != nil {
				t.Fatal(err)
			}
			r.scope.prepare = func(context.Context) (nativeProjectTool, error) { tools++; return nil, cause }
			input := bunResolverInput(t, artifactnpm.BunInstall, "", `{}`)
			if mode == "zero" {
				input = artifactnpm.BunSelectionInput{}
			}
			if mode == "wrong-kind" {
				r.scope.kind = nativeRuntimePNPM
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancel" {
				cancel()
			}
			if mode == "cancel-cause" {
				withCause, cancelCause := context.WithCancelCause(ctx)
				cancelCause(cause)
				ctx = withCause
			}
			selected, err := r.SelectProject(ctx, input)
			if err == nil || len(selected.Controls()) != 0 || selected.Selection().DependencyFree() || runner.calls != 0 || endpoints.calls != 0 {
				t.Fatal("refused input reached resources or exposed output", err)
			}
			if (mode == "cancel" || mode == "cancel-cause") && (!errors.Is(err, context.Canceled) || probes != 0 || tools != 0) {
				t.Fatal("cancellation cause/preflight changed")
			}
			if mode == "cancel-cause" && !errors.Is(err, cause) {
				t.Fatal("explicit cancellation cause lost")
			}
			if mode == "tool" && !errors.Is(err, cause) {
				t.Fatal("tool cause lost")
			}
		})
	}
}

func TestBunSelectionArgumentsKeepFixedBoundary(t *testing.T) {
	for _, test := range []struct {
		intent             artifactnpm.BunSelectionIntent
		selector, manifest string
	}{
		{artifactnpm.BunInstall, "", `{}`}, {artifactnpm.BunAdd, "@scope/probe@^1.0.0", `{}`}, {artifactnpm.BunUpdate, "probe", `{"dependencies":{"probe":"^1.0.0"}}`}, {artifactnpm.BunUpdate, "", `{}`},
	} {
		input := bunResolverInput(t, test.intent, test.selector, test.manifest)
		args, err := bunSelectionArguments(input)
		if err != nil || len(args) < 13 || args[0] != bunToolGuestPath || args[1] != "--cwd="+bunProjectDirectory || args[2] != string(test.intent) {
			t.Fatal("Bun arguments differ", err)
		}
		for _, option := range []string{"--lockfile-only", "--ignore-scripts", "--no-env-file", "--registry=https://registry.npmjs.org/", "--cache-dir=" + bunProjectRoot + "/cache", "--backend=copyfile", "--linker=hoisted", "--save-text-lockfile", "--no-progress", "--no-summary"} {
			if !strings.Contains(strings.Join(args, "\x00"), option) {
				t.Fatal("fixed option missing", option)
			}
		}
		if test.selector != "" && (len(args) != 15 || args[13] != "--" || args[14] != test.selector) {
			t.Fatal("selector escaped final delimiter")
		}
	}
	if _, err := bunSelectionArguments(artifactnpm.BunSelectionInput{}); err == nil {
		t.Fatal("unknown client became command")
	}
}
