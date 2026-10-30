package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
)

func TestPNPMRunCLIUsesNativeAliasesAndLiteralArgumentBoundary(t *testing.T) {
	for _, path := range [][]string{{"pnpm", "dlx"}, {"pnx"}, {"pnpx"}} {
		for _, explicit := range []bool{false, true} {
			var output bytes.Buffer
			root, err := New(&output, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			cause := errors.New("private native provider diagnostic")
			runner := &executionRequestRecorder{cause: cause}
			if err := AddPNPMRunExecution(root, runner); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"--lang", "ko", "--output", "json"}, path...)
			wantCommand := ""
			if explicit {
				args = append(args, "--package", "typescript@5.9.3", "tsc")
				wantCommand = "tsc"
			} else {
				args = append(args, "typescript@5.9.3")
			}
			literal := []string{"--help", "", "a b", "a\nb", "$(touch forbidden)", "--package", "--show-output"}
			args = append(args, "--")
			args = append(args, literal...)
			if source, err := PreparePresentation(root, args); err != nil || source != "npm" || SelectedPackageClient(root) != PNPMRunPackageClient {
				t.Fatal("native alias route differs", source, err)
			}
			root.SetArgs(args)
			err = root.ExecuteContext(context.Background())
			if !errors.Is(err, cause) || len(runner.requests) != 1 {
				t.Fatal("current runner or primary error lost", err)
			}
			request := runner.requests[0]
			if request.Reference().Locator() != "typescript@5.9.3" || request.Command() != wantCommand || !slices.Equal(request.Arguments(), literal) {
				t.Fatal("package, command or literal arguments changed")
			}
			if strings.Contains(output.String(), "private native") || strings.Contains(output.String(), "touch forbidden") || !strings.Contains(output.String(), `"schema":"helox.execution-result/v1"`) {
				t.Fatal("private arguments/diagnostic leaked or execution report missing")
			}
		}
	}
}

func TestPNPMRunCLIRejectsProvisioningAndUnsafeOptionsBeforeComposition(t *testing.T) {
	for _, tail := range [][]string{
		{}, {"--package", "", "tsc"}, {"--package", "typescript", "bad/name"},
		{"--package", "typescript", "--package=typescript", "tsc"}, {"--show-output", "--show-output", "typescript"},
		{"--command", "tsc", "typescript"}, {"--target", "/tmp/target", "typescript"}, {"--shell-mode", "typescript"},
		{"--registry", "https://example.invalid", "typescript"}, {"--cache-dir", "/tmp/cache", "typescript"},
		{"node"}, {"node@22.0.0"}, {"pnpm"}, {"bun"}, {"deno"}, {"rust"}, {"npm"}, {"yarn"},
		{"typescript@latest"}, {"typescript@^5"}, {"./local"}, {"https://example.invalid/pkg.tgz"},
		{"typescript", "bad\x00arg"},
	} {
		root, err := New(io.Discard, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		args := append([]string{"pnpm", "dlx"}, tail...)
		if _, err := PreparePresentation(root, args); err == nil {
			t.Fatal("unsafe PNPM execution reached composition", tail)
		}
	}
}

func TestPNPMRunCLIHelpRequiresNoRuntimeOrInstallTarget(t *testing.T) {
	for _, path := range [][]string{{"pnpm", "dlx"}, {"pnx"}, {"pnpx"}} {
		var output bytes.Buffer
		root, err := New(&output, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		args := append(append([]string{}, path...), "--help")
		if source, err := PreparePresentation(root, args); err != nil || source != "" {
			t.Fatal("static alias help prepared a runtime", source, err)
		}
		root.SetArgs(args)
		if err := root.Execute(); err != nil || strings.Contains(output.String(), "--target") || !strings.Contains(output.String(), "--package") {
			t.Fatal("execution help exposed installation or lost package selection", err)
		}
	}
}
