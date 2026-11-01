package cli_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestNPMProjectPreflightKeepsNativeIntentAndRejectsUnsupportedInputs(t *testing.T) {
	const manifest = `{"name":"fixture","dependencies":{"is-number":"7.0.0"}}`
	const lock = `{"name":"fixture","lockfileVersion":3,"packages":{"":{"name":"fixture","dependencies":{"is-number":"6.0.0"}}}}`
	for _, tc := range []struct {
		name, operation, manifest, lock string
		valid                           bool
	}{
		{"install without lock", "install", manifest, "", true},
		{"install reconciles stale lock", "install", manifest, lock, true},
		{"ci matching empty lock", "ci", `{}`, `{"lockfileVersion":3,"packages":{"":{}}}`, true},
		{"ci missing lock", "ci", manifest, "", false},
		{"ci mismatched lock", "ci", manifest, lock, false},
		{"malformed lock", "install", manifest, `{}`, false},
		{"missing manifest", "install", "", "", false},
		{"workspace", "install", `{"workspaces":["packages/*"]}`, "", false},
		{"arbitrary source", "install", `{"dependencies":{"foreign":"file:../other"}}`, "", false},
		{"symlink control", "install", manifest, "", false},
		{"unbound version", "install", manifest, "", false},
		{"repeated target", "install", manifest, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if tc.manifest != "" {
				if err = os.WriteFile(filepath.Join(target, "package.json"), []byte(tc.manifest), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if tc.lock != "" {
				if err = os.WriteFile(filepath.Join(target, "package-lock.json"), []byte(tc.lock), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if tc.name == "symlink control" {
				if err = os.Rename(filepath.Join(target, "package.json"), filepath.Join(target, "original.json")); err != nil {
					t.Fatal(err)
				}
				if err = os.Symlink("original.json", filepath.Join(target, "package.json")); err != nil {
					t.Fatal(err)
				}
			}
			command, err := cli.New(io.Discard, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			if tc.name != "unbound version" {
				if err = cli.SetNPMClientVersion(command, runtimeidentity.NodeNPMVersion); err != nil {
					t.Fatal(err)
				}
			}
			args := []string{"npm", tc.operation, "--target", target}
			if tc.name == "repeated target" {
				args = append(args, "--target", target)
			}
			source, err := cli.PreparePresentation(command, args)
			if (err == nil) != tc.valid {
				t.Fatal("npm preflight intent was changed", tc.name, source, err)
			}
			if tc.valid && (source != "npm" || !cli.NPMProjectRequested(command) || cli.SelectedPackageClientOperation(command) != tc.operation) {
				t.Fatal("npm complete-project intent lost", source)
			}
		})
	}
}

type unusedNPMProjectApplier struct{ calls int }

func (a *unusedNPMProjectApplier) ApplyNPM(context.Context, cli.NPMProjectRequest) (cli.NPMProjectOutcome, error) {
	a.calls++
	return cli.NPMProjectOutcome{}, nil
}

func TestNPMWholeRoutingPreservesExplicitPackageInstall(t *testing.T) {
	command, err := cli.New(io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	old := &npmCLIInstaller{}
	if err = cli.AddNPMInstall(command, old); err != nil {
		t.Fatal(err)
	}
	whole := &unusedNPMProjectApplier{}
	if err = cli.AddNPMProject(command, whole); err != nil {
		t.Fatal(err)
	}
	command.SetArgs([]string{"npm", "install", "demo-package@1.0.0"})
	_ = command.ExecuteContext(context.Background())
	if whole.calls != 0 || old.request.Reference().Locator() != "demo-package@1.0.0" {
		t.Fatal("explicit package install rerouted to complete-project request")
	}
}
