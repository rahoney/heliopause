package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
)

type executionRequestRecorder struct {
	requests []application.ExecutionPreparationRequest
	cause    error
}

func (r *executionRequestRecorder) Run(_ context.Context, request application.ExecutionPreparationRequest) (application.ExecutionResult, error) {
	r.requests = append(r.requests, request)
	return application.ExecutionResult{}, r.cause
}

func TestExecutionCLIParsesPackageBoundaryBeforeComposition(t *testing.T) {
	for _, path := range [][]string{{"npx"}, {"npm", "exec"}} {
		var out bytes.Buffer
		root, err := New(&out, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		cause := errors.New("original private provider cause")
		runner := &executionRequestRecorder{cause: cause}
		if err := AddNPMExecution(root, runner); err != nil {
			t.Fatal(err)
		}
		literal := []string{"--help", "--output", "human", "", "a b", "a\nb", "$(touch forbidden)", "--command", "--show-output"}
		args := append([]string{"--lang", "ko", "--output", "json"}, path...)
		args = append(args, "--command", "tsc", "typescript@5.9.3", "--")
		args = append(args, literal...)
		source, err := PreparePresentation(root, args)
		if err != nil || source != "npm" || SelectedPackageClient(root) != NPXPackageClient {
			t.Fatal("invalid route", source, err)
		}
		root.SetArgs(args)
		err = root.ExecuteContext(context.Background())
		var coded interface{ ExitCode() int }
		if len(runner.requests) != 1 || !errors.Is(err, cause) || !errors.As(err, &coded) || coded.ExitCode() != 1 {
			t.Fatal("request/error not routed", err)
		}
		request := runner.requests[0]
		if request.Reference().Locator() != "typescript@5.9.3" || request.Command() != "tsc" || !slices.Equal(request.Arguments(), literal) {
			t.Fatal("literal program arguments were interpreted as HAA options")
		}
		if strings.Contains(out.String(), "original private provider cause") || strings.Contains(out.String(), "touch forbidden") || !strings.Contains(out.String(), `"schema":"helox.execution-result/v1"`) {
			t.Fatal("private request/cause leaked or report missing")
		}
	}
}

func TestExecutionCLIRejectsInvalidIntentBeforeComposition(t *testing.T) {
	for _, tail := range [][]string{
		{}, {"--command", "", "typescript"}, {"--command", "bad/name", "typescript"}, {"--command", "tsc", "--command=tsc", "typescript"},
		{"--show-output", "--show-output", "typescript"}, {"--target", "/tmp/target", "typescript"}, {"-c", "echo hi"}, {"--package", "typescript", "tsc"},
		{"--node-options", "x", "typescript"}, {"--stdin", "typescript"}, {"--tty", "typescript"}, {"./local"}, {"https://example.invalid/package.tgz"},
		{"typescript", "bad\x00arg"}, {"typescript", string([]byte{255})}, {"typescript", strings.Repeat("a", 32*1024+1)},
	} {
		root, err := New(io.Discard, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		args := append([]string{"npx"}, tail...)
		if _, err := PreparePresentation(root, args); err == nil {
			t.Fatalf("invalid execution accepted: case with %d args", len(tail))
		}
	}
	root, _ := New(io.Discard, io.Discard)
	args := append([]string{"npx", "typescript"}, make([]string, 129)...)
	if _, err := PreparePresentation(root, args); err == nil {
		t.Fatal("argv count overflow accepted")
	}
}

func TestExecutionCLIHelpAndLiteralDelimiter(t *testing.T) {
	for _, path := range [][]string{{"npx"}, {"npm", "exec"}} {
		root, err := New(io.Discard, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		args := append(append([]string{}, path...), "--help")
		if source, err := PreparePresentation(root, args); err != nil || source != "" {
			t.Fatal("HAA help prepared a runtime", source, err)
		}
		root, err = New(io.Discard, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		runner := &executionRequestRecorder{cause: errors.New("fixture refusal")}
		if err := AddNPMExecution(root, runner); err != nil {
			t.Fatal(err)
		}
		args = append(append([]string{}, path...), "typescript", "--", "--", "")
		if _, err := PreparePresentation(root, args); err != nil {
			t.Fatal(err)
		}
		root.SetArgs(args)
		_ = root.ExecuteContext(context.Background())
		if len(runner.requests) != 1 || !slices.Equal(runner.requests[0].Arguments(), []string{"--", ""}) {
			t.Fatal("literal delimiter/empty argument lost")
		}
	}
}
