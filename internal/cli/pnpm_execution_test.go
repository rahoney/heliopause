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
	testNativeNodeRunCLIUsesNativeAliasesAndLiteralArgumentBoundary(t, false)
}

func TestBunRunCLIUsesNativeAliasesAndLiteralArgumentBoundary(t *testing.T) {
	testNativeNodeRunCLIUsesNativeAliasesAndLiteralArgumentBoundary(t, true)
}

func testNativeNodeRunCLIUsesNativeAliasesAndLiteralArgumentBoundary(t *testing.T, bun bool) {
	t.Helper()
	paths := [][]string{{"pnpm", "dlx"}, {"pnx"}, {"pnpx"}}
	id, add := PNPMRunPackageClient, AddPNPMRunExecution
	if bun {
		paths = [][]string{{"bunx"}, {"bun", "x"}}
		id, add = BunRunPackageClient, AddBunRunExecution
	}
	for _, path := range paths {
		for _, explicit := range []bool{false, true} {
			var output bytes.Buffer
			root, err := New(&output, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			cause := errors.New("private native provider diagnostic")
			runner := &executionRequestRecorder{cause: cause}
			if err := add(root, runner); err != nil {
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
			if source, err := PreparePresentation(root, args); err != nil || source != "npm" || SelectedPackageClient(root) != id {
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
	testNativeNodeRunCLIRejectsProvisioningAndUnsafeOptionsBeforeComposition(t, false)
}

func TestBunRunCLIRejectsProvisioningAndUnsafeOptionsBeforeComposition(t *testing.T) {
	testNativeNodeRunCLIRejectsProvisioningAndUnsafeOptionsBeforeComposition(t, true)
}

func testNativeNodeRunCLIRejectsProvisioningAndUnsafeOptionsBeforeComposition(t *testing.T, bun bool) {
	t.Helper()
	paths := [][]string{{"pnpm", "dlx"}, {"pnx"}, {"pnpx"}}
	if bun {
		paths = [][]string{{"bunx"}, {"bun", "x"}}
	}
	for _, tail := range [][]string{
		{}, {"--package", "", "tsc"}, {"--package", "typescript", "bad/name"},
		{"--package", "typescript", "--package=typescript", "tsc"}, {"--show-output", "--show-output", "typescript"},
		{"--command", "tsc", "typescript"}, {"--target", "/tmp/target", "typescript"}, {"--shell-mode", "typescript"}, {"--bun", "typescript"},
		{"--registry", "https://example.invalid", "typescript"}, {"--cache-dir", "/tmp/cache", "typescript"},
		{"node"}, {"node@22.0.0"}, {"pnpm"}, {"bun"}, {"deno"}, {"rust"}, {"npm"}, {"yarn"},
		{"typescript@latest"}, {"typescript@^5"}, {"./local"}, {"https://example.invalid/pkg.tgz"},
		{"typescript", "bad\x00arg"},
	} {
		root, err := New(io.Discard, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		args := append(paths[0], tail...)
		if _, err := PreparePresentation(root, args); err == nil {
			t.Fatal("unsafe PNPM execution reached composition", tail)
		}
	}
}

func TestPNPMRunCLIHelpRequiresNoRuntimeOrInstallTarget(t *testing.T) {
	testNativeNodeRunCLIHelpRequiresNoRuntimeOrInstallTarget(t, false)
}

func TestBunRunCLIHelpRequiresNoRuntimeOrInstallTarget(t *testing.T) {
	testNativeNodeRunCLIHelpRequiresNoRuntimeOrInstallTarget(t, true)
}

func testNativeNodeRunCLIHelpRequiresNoRuntimeOrInstallTarget(t *testing.T, bun bool) {
	t.Helper()
	paths := [][]string{{"pnpm", "dlx"}, {"pnx"}, {"pnpx"}}
	if bun {
		paths = [][]string{{"bunx"}, {"bun", "x"}}
	}
	for _, path := range paths {
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
