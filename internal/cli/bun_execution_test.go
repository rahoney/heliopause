package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
)

type bunRuntimeRequestRecorder struct {
	executionRequestRecorder
	forced int
}

func (r *bunRuntimeRequestRecorder) RunBun(ctx context.Context, request application.ExecutionPreparationRequest) (application.ExecutionResult, error) {
	r.forced++
	return r.Run(ctx, request)
}

func TestBunRuntimeSelectionIsSeparateFromLiteralProgramArguments(t *testing.T) {
	for _, path := range [][]string{{"bunx"}, {"bun", "x"}} {
		var output bytes.Buffer
		root, err := New(&output, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		cause := errors.New("private selected runtime failure")
		runner := &bunRuntimeRequestRecorder{executionRequestRecorder: executionRequestRecorder{cause: cause}}
		if err := AddBunRunExecution(root, runner); err != nil {
			t.Fatal(err)
		}
		for _, force := range []string{"--bun", "--bun=false"} {
			args := append(append([]string{}, path...), force, "--package", "typescript@5.9.3", "tsc", "--", "--bun", "private argument")
			if _, err := PreparePresentation(root, args); err != nil {
				t.Fatal(err)
			}
			root.SetArgs(args)
			if err := root.ExecuteContext(context.Background()); !errors.Is(err, cause) {
				t.Fatal("runtime selection lost original cause", err)
			}
		}
		if runner.forced != 1 || len(runner.requests) != 2 || runner.requests[0].Arguments()[0] != "--bun" || runner.requests[1].Arguments()[0] != "--bun" {
			t.Fatal("Bun selection escaped into argv or remained selected for Node")
		}
	}
}

func TestBunRuntimeSelectionRejectsRepeatedFlagBeforeComposition(t *testing.T) {
	root, err := New(io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PreparePresentation(root, []string{"bunx", "--bun", "--bun=false", "typescript"}); err == nil {
		t.Fatal("conflicting runtime selections reached composition")
	}
}
