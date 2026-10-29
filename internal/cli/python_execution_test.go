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

func TestUVRunCLIParsesNativeAliasesAndLiteralBoundary(t *testing.T) {
	literal := []string{"--help", "--output", "human", "", "a b", "a\nb", "$(touch forbidden)", "--from", "foreign", "--show-output"}
	for _, row := range []struct {
		path, prefix       []string
		command, reference string
	}{
		{[]string{"uvx"}, []string{"cowsay@6.1", "--"}, "cowsay", "cowsay@6.1"},
		{[]string{"uv", "tool", "run"}, []string{"cowsay==6.1", "--"}, "cowsay", "cowsay@6.1"},
		{[]string{"uvx"}, []string{"--from", "python-slugify==8.0.4", "slugify", "--"}, "slugify", "python-slugify@8.0.4"},
		{[]string{"uv", "--lang", "ko", "tool", "run"}, []string{"--from=python-slugify@8.0.4", "slugify", "--"}, "slugify", "python-slugify@8.0.4"},
	} {
		var out bytes.Buffer
		root, err := New(&out, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		original := errors.New("original private native provider failure")
		runner := &executionRequestRecorder{cause: original}
		if err := AddUVRunExecution(root, runner); err != nil {
			t.Fatal(err)
		}
		args := append([]string{"--output", "json"}, row.path...)
		args = append(args, row.prefix...)
		args = append(args, literal...)
		source, err := PreparePresentation(root, args)
		if err != nil || source != "pypi" || SelectedPackageClient(root) != UVRunPackageClient || !SelectedPackageClientIsExecution(root) {
			t.Fatal("native alias preflight/route", args, source, err)
		}
		root.SetArgs(args)
		err = root.ExecuteContext(context.Background())
		var code interface{ ExitCode() int }
		if !errors.Is(err, original) || !errors.As(err, &code) || code.ExitCode() != 1 || len(runner.requests) != 1 {
			t.Fatal("current typed runner/error path differs", err)
		}
		request := runner.requests[0]
		if request.Reference().Locator() != row.reference || request.Reference().Source().String() != "pypi" || request.Command() != row.command || !slices.Equal(request.Arguments(), literal) {
			t.Fatal("literal/private request differs", request.Reference(), request.Command())
		}
		if strings.Contains(out.String(), "forbidden") || strings.Contains(out.String(), "private native provider") {
			t.Fatal("raw request/cause leaked into report")
		}
	}
}

func TestUVRunCLIRejectsUnsupportedBeforeRunner(t *testing.T) {
	for _, path := range [][]string{{"uvx"}, {"uv", "tool", "run"}} {
		for _, tail := range [][]string{
			{}, {"--from=", "cowsay"}, {"--from", "cowsay", "--from", "other", "cowsay"},
			{"--show-output", "--show-output=false", "cowsay"}, {"--with", "other", "cowsay"},
			{"--python", "3.14", "cowsay"}, {"--index-url", "https://example.invalid", "cowsay"},
			{"--isolated", "cowsay"}, {"--command", "cowsay", "cowsay"}, {"--stdin", "cowsay"},
			{"cowsay[extra]"}, {"cowsay>=1"}, {"cowsay==6.1,>=6"}, {"cowsay@latest"}, {"cowsay@6.2rc1"},
			{"./local"}, {"git+https://example.invalid/tool"}, {"https://example.invalid/tool.whl"},
			{"python"}, {"python3.14"}, {"python-3.14"}, {"pythonw"}, {"pypy@3.11"}, {"graalpy"},
			{"--from", "python-slugify@8.0.4", "../slugify"}, {"--from", "python-slugify@8.0.4", "sh -c slugify"},
		} {
			root, err := New(io.Discard, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			runner := &executionRequestRecorder{}
			if err := AddUVRunExecution(root, runner); err != nil {
				t.Fatal(err)
			}
			args := append(append([]string{}, path...), tail...)
			if _, err := PreparePresentation(root, args); err == nil {
				t.Fatal("unsupported request accepted", args)
			}
			if len(runner.requests) != 0 {
				t.Fatal("preflight invoked runner")
			}
		}
	}
	root, _ := New(io.Discard, io.Discard)
	if _, err := PreparePresentation(root, []string{"uv", "--lang", "ko", "tool", "--output", "json", "run", "--from", "first", "--from", "second", "command"}); err == nil {
		t.Fatal("interleaved global flags hid repeated --from")
	}
	for _, arguments := range [][]string{append([]string{"uvx", "cowsay"}, make([]string, 129)...), {"uvx", "cowsay", strings.Repeat("x", (32<<10)+1)}} {
		root, _ := New(io.Discard, io.Discard)
		if _, err := PreparePresentation(root, arguments); err == nil {
			t.Fatal("argv bound bypassed")
		}
	}
	if err := AddUVRunExecution(nil, &executionRequestRecorder{}); err == nil {
		t.Fatal("nil root accepted")
	}
}

func TestUVRunCLIStaticHelpHasNoExecutionOrInstallTarget(t *testing.T) {
	for _, path := range [][]string{{"uvx"}, {"uv", "tool", "run"}} {
		for _, language := range []string{"en", "ko"} {
			var out bytes.Buffer
			root, err := New(&out, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			runner := &executionRequestRecorder{}
			if err := AddUVRunExecution(root, runner); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"--lang", language}, path...)
			args = append(args, "--help")
			source, err := PreparePresentation(root, args)
			if err != nil || source != "" {
				t.Fatal("static help reached execution", source, err)
			}
			root.SetArgs(args)
			if err := root.ExecuteContext(context.Background()); err != nil {
				t.Fatal(err)
			}
			if len(runner.requests) != 0 || !strings.Contains(out.String(), "--from") || !strings.Contains(out.String(), "--show-output") || !strings.Contains(out.String(), "ASCII") || strings.Contains(out.String(), "--target") || strings.Contains(out.String(), "--bin-dir") {
				t.Fatal("help expanded install/runtime scope")
			}
		}
	}
}
