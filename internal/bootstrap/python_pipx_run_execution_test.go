package bootstrap

import (
	"context"
	"errors"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestPythonPipxRunRequiresCurrentAdaptersAndPreservesCancellation(t *testing.T) {
	if service, err := newPipxRunExecutionService(nil, "cowsay", nil, nil, nil, nil, nil, nil); err == nil || service != nil {
		t.Fatal("missing adapters became an execution service")
	}
	cause := errors.New("caller canceled pipx before selection")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	reference, _ := artifactpypi.ParseReference("example@1.0")
	r := freshPythonPipxRunExecutionResolver{native: &sandbox.PipxResolver{}, command: "example"}
	selected, err := r.ResolveExecutionDependencies(ctx, reference)
	if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || len(selected.Graph().Nodes()) != 0 {
		t.Fatal("native DATA survived cancellation", err)
	}
	if bound, err := (pythonPipxRunExecutionBinder{}).BindExecutionEntry(ctx, application.ExecutionInspection{}, "example"); !errors.Is(err, cause) || bound.Valid() {
		t.Fatal("canceled binding returned authority", err)
	}
	for _, ctx := range []context.Context{nil, context.Background()} {
		if bound, err := (pythonPipxRunExecutionBinder{}).BindExecutionEntry(ctx, application.ExecutionInspection{}, "example"); err == nil || bound.Valid() {
			t.Fatal("absent current approval became an execution")
		}
	}
}

func TestPythonPipxRunRejectsOtherSourcesAndUnboundedReferences(t *testing.T) {
	r := freshPythonPipxRunExecutionResolver{native: &sandbox.PipxResolver{}, command: "example"}
	for _, row := range []struct{ source, locator string }{{"npm", "example"}, {"pypi", "example>=1"}, {"pypi", "example[extra]"}, {"pypi", "https://example.invalid/foreign.whl"}} {
		source, _ := domain.NewSourceID(row.source)
		reference, _ := domain.NewArtifactReference(source, row.locator)
		if selected, err := r.ResolveExecutionDependencies(context.Background(), reference); err == nil || len(selected.Graph().Nodes()) != 0 {
			t.Fatal("unsupported native request admitted")
		}
	}
	for _, platform := range [][2]string{{"darwin", "arm64"}, {"windows", "amd64"}, {"linux", "arm64"}} {
		if pipxRunExecutionPlatform(platform[0], platform[1]) == nil {
			t.Fatal("unsupported runtime platform admitted")
		}
	}
}
