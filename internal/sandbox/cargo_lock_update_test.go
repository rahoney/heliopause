package sandbox

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestCargoLockUpdateFixedCommandAdmission(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want bool
	}{
		{"existing-metadata", []string{"metadata", "--locked", "--format-version", "1"}, true},
		{"existing-add", []string{"add", "itoa@=1.0.17"}, true},
		{"complete-update", []string{"update"}, true},
		{"one-update", []string{"update", "-p", "itoa"}, true},
		{"locked-version-selector", []string{"update", "-p", "itoa@1.0.17"}, true},
		{"missing-selector", []string{"update", "-p"}, false},
		{"empty-selector", []string{"update", "-p", ""}, false},
		{"repeated-selector", []string{"update", "-p", "itoa", "-p", "serde"}, false},
		{"raw-long-flag", []string{"update", "--package", "itoa"}, false},
		{"positional-selector", []string{"update", "itoa"}, false},
		{"offline", []string{"update", "--offline"}, false},
		{"manifest-path", []string{"update", "--manifest-path", "/tmp/foreign/Cargo.toml"}, false},
		{"url-selector", []string{"update", "-p", "https://example.invalid/itoa"}, false},
		{"flag-selector", []string{"update", "-p", "--precise"}, false},
		{"invalid-version", []string{"update", "-p", "itoa@1.0"}, false},
		{"substituted-version", []string{"update", "-p", "itoa@1.0.17#foreign"}, false},
		{"build", []string{"build"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := validCargoResolverCommand(test.args); got != test.want {
				t.Fatalf("fixed command admission = %t; want %t", got, test.want)
			}
		})
	}
}

type cargoLockUpdateRunner struct {
	base     *cargoProjectTestRunner
	args     [][]string
	mutation string
}

func (r *cargoLockUpdateRunner) RunCargo(ctx context.Context, workspace string, environment []string, args ...string) ([]byte, error) {
	r.args = append(r.args, append([]string(nil), args...))
	if args[0] != "update" {
		return r.base.RunCargo(ctx, workspace, environment, args...)
	}
	if workspace == r.base.original || ValidateCargoResolverEnvironment(environment, strings.TrimPrefix(environment[0], "CARGO_HOME=")) != nil {
		r.base.t.Fatal("original source or ambient environment reached SDK")
	}
	r.base.workspaces = append(r.base.workspaces, workspace)
	r.base.homes = append(r.base.homes, strings.TrimPrefix(environment[0], "CARGO_HOME="))
	if r.base.failure != nil {
		return nil, r.base.failure
	}
	if err := os.WriteFile(filepath.Join(workspace, "Cargo.lock"), r.base.lock, 0o600); err != nil {
		return nil, err
	}
	if r.mutation != "" {
		if err := os.WriteFile(filepath.Join(workspace, r.mutation), []byte("changed\n"), 0o600); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func cargoLockUpdateFixture(t *testing.T) (string, domain.InstallContext, *cargoLockUpdateRunner, []domain.ProjectControlFile) {
	t.Helper()
	root, install, base := cargoProjectResolverFixture(t, true)
	manifest := bytes.ReplaceAll(base.manifest, []byte("'=1.0.17'"), []byte("'1.0'"))
	lock := bytes.ReplaceAll(base.lock, []byte("1.0.17"), []byte("1.0.16"))
	if err := os.WriteFile(filepath.Join(root, "Cargo.toml"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Cargo.lock"), lock, 0o600); err != nil {
		t.Fatal(err)
	}
	controls, err := cargoFrozenControls(map[string][]byte{"Cargo.toml": manifest, "Cargo.lock": lock})
	if err != nil {
		t.Fatal(err)
	}
	return root, install, &cargoLockUpdateRunner{base: base}, controls
}

func TestCargoLockUpdateBindsOriginalAndCompletePrivateSelection(t *testing.T) {
	for _, selector := range []string{"", "itoa", "itoa@1.0.16"} {
		t.Run(selector, func(t *testing.T) {
			root, install, runner, controls := cargoLockUpdateFixture(t)
			resolver, _ := NewCargoResolver(runner)
			update, err := resolver.ResolveProjectLockUpdate(context.Background(), install, controls, selector)
			if err != nil || !update.Valid() || update.HasRequestedResolution() || len(update.Resolution().Graph().Nodes()) != 0 || update.Snapshot().Context() != install || len(update.Snapshot().Dependencies()) != 1 || update.Snapshot().Dependencies()[0].Identity().Version() != "1.0.17" || !artifactcargo.EqualControls(update.OriginalControls(), controls) {
				t.Fatalf("complete frozen update binding lost: %v", err)
			}
			if len(runner.args) != 2 || runner.args[0][0] != "update" || strings.Join(runner.args[1], " ") != "metadata --locked --format-version 1" {
				t.Fatal("update selection or locked final graph replayed")
			}
			if selector == "" && len(runner.args[0]) != 1 || selector != "" && strings.Join(runner.args[0], " ") != "update -p "+selector {
				t.Fatal("requested selection was changed")
			}
			for _, control := range controls {
				body, err := os.ReadFile(filepath.Join(root, control.Name()))
				if err != nil || !bytes.Equal(body, control.Body()) {
					t.Fatal("selection published original controls before approval")
				}
			}
			assertCargoPrivateCleanup(t, runner.base)
		})
	}
}

func TestCargoLockUpdateRejectsUnboundSelectionBeforePublication(t *testing.T) {
	for _, test := range []string{"unknown-selector", "original-controls", "Cargo.toml", "src/main.rs", "Cargo.lock", "canceled", "trusted-failure"} {
		t.Run(test, func(t *testing.T) {
			_, install, runner, controls := cargoLockUpdateFixture(t)
			selector := ""
			ctx := context.Background()
			switch test {
			case "unknown-selector":
				selector = "missing"
			case "original-controls":
				controls[0], _ = domain.NewProjectControlFile(controls[0].Name(), []byte("changed"), true)
			case "Cargo.toml", "src/main.rs":
				runner.mutation = test
			case "Cargo.lock":
				runner.base.mutateMetadata = test
			case "canceled":
				canceled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = canceled
			case "trusted-failure":
				runner.base.failure = &isolatedCargoResolverFailure{cause: context.DeadlineExceeded, phase: "COMMAND", diagnostic: TraceDiagnostic{Reason: "STREAM_FAULT"}}
			}
			resolver, _ := NewCargoResolver(runner)
			update, err := resolver.ResolveProjectLockUpdate(ctx, install, controls, selector)
			if err == nil || update.Valid() {
				t.Fatal("unbound or failed update returned a valid proposal")
			}
			if (test == "unknown-selector" || test == "original-controls" || test == "canceled") && len(runner.args) != 0 {
				t.Fatal("invalid original request reached SDK")
			}
			if test == "trusted-failure" && !errors.Is(err, context.DeadlineExceeded) || test == "canceled" && !errors.Is(err, context.Canceled) {
				t.Fatal("primary cause lost")
			}
			assertCargoPrivateCleanup(t, runner.base)
		})
	}
}
