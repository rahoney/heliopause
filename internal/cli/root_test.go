package cli_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/cli"
)

func TestRootHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command, err := cli.New(&stdout, &stderr)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	command.SetArgs([]string{"--help"})

	if err = command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Usage:\n  helox") {
		t.Fatalf("help output missing usage: %q", stdout.String())
	}
	for _, commandName := range []string{"npm", "pip", "pypi", "github", "go", "cargo", "terraform", "doctor"} {
		if !strings.Contains(stdout.String(), commandName) {
			t.Fatalf("help output missing %q command: %q", commandName, stdout.String())
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestSourceHelpShowsStaticSubcommands(t *testing.T) {
	t.Parallel()

	tests := map[string][]string{
		"npm":       {"npm", "--help"},
		"pip":       {"pip", "--help"},
		"pypi":      {"pypi", "--help"},
		"github":    {"github", "--help"},
		"go":        {"go", "--help"},
		"cargo":     {"cargo", "--help"},
		"terraform": {"terraform", "--help"},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			command, err := cli.New(&stdout, &stderr)
			if err != nil {
				t.Fatal(err)
			}
			command.SetArgs(args)
			if err := command.ExecuteContext(context.Background()); err != nil {
				t.Fatalf("ExecuteContext error: %v", err)
			}
			if !strings.Contains(stdout.String(), "Available Commands:") {
				t.Fatalf("help output missing command list: %q", stdout.String())
			}
			if name == "pip" && !strings.Contains(stdout.String(), "install") {
				t.Fatalf("pip help missing install: %q", stdout.String())
			}
			if name != "pip" && name != "go" && name != "cargo" && name != "terraform" && !strings.Contains(stdout.String(), "inspect") {
				t.Fatalf("%s help missing inspect: %q", name, stdout.String())
			}
			if name == "go" && (!strings.Contains(stdout.String(), "get") || !strings.Contains(stdout.String(), "mod")) {
				t.Fatalf("go help missing module commands: %q", stdout.String())
			}
			if name == "cargo" && (!strings.Contains(stdout.String(), "add") || !strings.Contains(stdout.String(), "build")) {
				t.Fatalf("cargo help missing crate commands: %q", stdout.String())
			}
			if name == "terraform" && !strings.Contains(stdout.String(), "init") {
				t.Fatalf("terraform help missing init: %q", stdout.String())
			}
		})
	}
}

func TestNewRejectsNilWriters(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stdout io.Writer
		stderr io.Writer
	}{
		"stdout": {stdout: nil, stderr: io.Discard},
		"stderr": {stdout: io.Discard, stderr: nil},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := cli.New(test.stdout, test.stderr); err == nil {
				t.Fatal("New error = nil, want non-nil")
			}
		})
	}
}

func TestDoctorRendersEveryCheckAndFailsClosed(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	command, err := cli.New(&stdout, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if err := cli.AddDoctor(command, testDoctor{}); err != nil {
		t.Fatal(err)
	}
	command.SetArgs([]string{"doctor"})
	if err := command.ExecuteContext(context.Background()); err == nil {
		t.Fatal("incomplete doctor report returned success")
	}
	for _, want := range []string{"release-installation=OK (OK)", "trusted-host-runtime=UNAVAILABLE (TRUSTED_HOST_RUNTIME_UNAVAILABLE)"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("doctor output missing %q: %q", want, stdout.String())
		}
	}
}

type testDoctor struct{}

func (testDoctor) Diagnose(context.Context) cli.DoctorReport {
	return cli.DoctorReport{Healthy: false, Checks: []cli.DoctorCheck{
		{Name: "release-installation", Healthy: true, Detail: "OK"},
		{Name: "trusted-host-runtime", Detail: "TRUSTED_HOST_RUNTIME_UNAVAILABLE"},
	}}
}
