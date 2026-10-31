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
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

type uvRootRequestRecorder struct {
	executionRequestRecorder
	selections []artifactpypi.UVRunRequirements
}

func (r *uvRootRequestRecorder) RunUVWithRequirements(ctx context.Context, request application.ExecutionPreparationRequest, selection artifactpypi.UVRunRequirements) (application.ExecutionResult, error) {
	r.selections = append(r.selections, selection)
	return r.executionRequestRecorder.Run(ctx, request)
}
func TestUVRunSupplementalCLIUsesWholeTypedRequestsAndLiteralBoundary(t *testing.T) {
	for _, prefix := range [][]string{
		{"uvx", "--with", "second[Two,One]@1.0", "--with=first", "tool[Feature]@1.0"},
		{"uv", "tool", "run", "--with", "second[two,one]==1.0", "--with", "first", "--from", "tool[feature]==1.0", "alternate"},
	} {
		var output bytes.Buffer
		root, err := New(&output, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		cause := errors.New("private supplemental execution failure")
		runner := &uvRootRequestRecorder{executionRequestRecorder: executionRequestRecorder{cause: cause}}
		if err := AddUVRunExecution(root, runner); err != nil {
			t.Fatal(err)
		}
		literal := []string{"--with", "private-literal", "", "a\nb", "--from", "other"}
		args := append([]string{"--output", "json"}, prefix...)
		args = append(args, "--")
		args = append(args, literal...)
		if _, err := PreparePresentation(root, args); err != nil {
			t.Fatal("valid repeatable requirements rejected", err)
		}
		root.SetArgs(args)
		if err := root.ExecuteContext(context.Background()); !errors.Is(err, cause) || len(runner.selections) != 1 || len(runner.requests) != 1 {
			t.Fatal("typed whole selection not dispatched", err)
		}
		if !slices.Equal(runner.selections[0].Requirements(), []string{"tool[feature]==1.0", "first", "second[one,two]==1.0"}) || !slices.Equal(runner.requests[0].Arguments(), literal) || strings.Contains(output.String(), "private-") {
			t.Fatal("root selection, literal argv or privacy changed")
		}
	}
}
func TestUVRunSupplementalCLIRejectsUnboundOrUnsafeBeforeIntake(t *testing.T) {
	for _, tail := range [][]string{
		{"--with=", "tool"}, {"--with", "tool", "tool"}, {"--with", "child", "--with", "CHILD@1.0", "tool"},
		{"--with", "child>=1", "tool"}, {"--with", "https://example.invalid/x.whl", "tool"}, {"--with", "./local", "tool"},
		{"--with", "python3.14", "tool"}, {"--with", "child", "--from", "tool", "--from", "other", "tool"},
		{"--with", "child", "--show-output", "--show-output=false", "tool"},
	} {
		root, err := New(io.Discard, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		runner := &uvRootRequestRecorder{}
		if err := AddUVRunExecution(root, runner); err != nil {
			t.Fatal(err)
		}
		if _, err := PreparePresentation(root, append([]string{"uvx"}, tail...)); err == nil {
			t.Fatal("unsafe or repeated selection accepted", tail)
		}
		if len(runner.requests) != 0 || len(runner.selections) != 0 {
			t.Fatal("preflight invoked intake consumer")
		}
	}
	root, _ := New(io.Discard, io.Discard)
	legacy := &executionRequestRecorder{}
	if err := AddUVRunExecution(root, legacy); err != nil {
		t.Fatal(err)
	}
	args := []string{"uvx", "--with", "child", "tool"}
	if _, err := PreparePresentation(root, args); err != nil {
		t.Fatal(err)
	}
	root.SetArgs(args)
	if err := root.ExecuteContext(context.Background()); err == nil || len(legacy.requests) != 0 {
		t.Fatal("legacy primary-only runner consumed supplemental roots")
	}
}
