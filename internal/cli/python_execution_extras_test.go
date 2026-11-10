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

type uvExtraRequestRecorder struct {
	executionRequestRecorder
	selections []artifactpypi.UVRunRequirement
}

func (r *uvExtraRequestRecorder) RunUVWithExtras(ctx context.Context, request application.ExecutionPreparationRequest, selection artifactpypi.UVRunRequirement) (application.ExecutionResult, error) {
	r.selections = append(r.selections, selection)
	return r.executionRequestRecorder.Run(ctx, request)
}

func TestUVRunExtrasCLIUsesTypedSelectionAndLiteralProgramBoundary(t *testing.T) {
	for _, prefix := range [][]string{
		{"uvx", "tool[Fea_ture]@1.0"},
		{"uv", "tool", "run", "tool[fea-ture]==1.0"},
		{"uvx", "--from", "tool[fea-ture]==1.0", "alternate"},
		{"uv", "tool", "run", "--from=tool[fea-ture]@1.0", "alternate"},
	} {
		var out bytes.Buffer
		root, err := New(&out, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		cause := errors.New("private extra provider failure")
		runner := &uvExtraRequestRecorder{executionRequestRecorder: executionRequestRecorder{cause: cause}}
		if err := AddUVRunExecution(root, runner); err != nil {
			t.Fatal(err)
		}
		literal := []string{"--with", "literal", "", "a\nb", "--from", "private"}
		args := append([]string{"--output", "json"}, prefix...)
		args = append(args, "--")
		args = append(args, literal...)
		if _, err := PreparePresentation(root, args); err != nil {
			t.Fatal(err)
		}
		root.SetArgs(args)
		if err := root.ExecuteContext(context.Background()); !errors.Is(err, cause) || len(runner.selections) != 1 || len(runner.requests) != 1 {
			t.Fatal("typed extra selection did not reach its current consumer", err)
		}
		if runner.selections[0].Requirement() != "tool[fea-ture]==1.0" || runner.requests[0].Reference().Locator() != "tool@1.0" || !slices.Equal(runner.requests[0].Arguments(), literal) || strings.Contains(out.String(), "private") {
			t.Fatal("selection, literal args or output privacy changed")
		}
	}
}
