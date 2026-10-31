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

func TestPythonUVRunExecutionRequiresCurrentAdaptersAndRetainsCancelCause(t *testing.T) {
	if service, err := newUVRunExecutionService(nil, "cowsay", nil, nil, nil, nil, nil, nil); err == nil || service != nil {
		t.Fatal("missing actual native/current adapters became a service")
	}
	reference, err := artifactpypi.ParseReference("example@1.0")
	if err != nil {
		t.Fatal(err)
	}
	resolver := freshPythonUVRunExecutionResolver{native: &sandbox.UVResolver{}, command: "example"}
	cause := errors.New("caller canceled native run before source reader")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	selected, err := resolver.ResolveExecutionDependencies(ctx, reference)
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || len(selected.Graph().Nodes()) != 0 {
		t.Fatal("native selection lost cancellation or returned DATA", err)
	}
	if execution, err := (pythonUVRunExecutionBinder{}).BindExecutionEntry(ctx, application.ExecutionInspection{}, "example"); !errors.Is(err, cause) || execution.Valid() {
		t.Fatal("current binder lost cancellation or fabricated context", err)
	}
	for _, ctx := range []context.Context{nil, context.Background()} {
		if selected, err := resolver.ResolveExecutionDependencies(ctx, reference); err == nil || len(selected.Graph().Nodes()) != 0 {
			t.Fatal("unavailable native reader produced resolution")
		}
		if execution, err := (pythonUVRunExecutionBinder{}).BindExecutionEntry(ctx, application.ExecutionInspection{}, "example"); err == nil || execution.Valid() {
			t.Fatal("missing actual current approval became execution")
		}
	}
}

func TestPythonUVRunExecutionRejectsUnsupportedReferencesBeforeNative(t *testing.T) {
	resolver := freshPythonUVRunExecutionResolver{native: &sandbox.UVResolver{}, command: "example"}
	for _, row := range []struct{ source, locator string }{{"npm", "example"}, {"pypi", "example>=1"}, {"pypi", "example==1,>=1"}, {"pypi", "example[extra]"}, {"pypi", "https://example.invalid/tool.whl"}} {
		source, _ := domain.NewSourceID(row.source)
		reference, err := domain.NewArtifactReference(source, row.locator)
		if err != nil {
			t.Fatal(err)
		}
		if selected, err := resolver.ResolveExecutionDependencies(context.Background(), reference); err == nil || len(selected.Graph().Nodes()) != 0 {
			t.Fatal("unsupported reference accepted")
		}
	}
	for _, command := range []string{"", "../example", "sh -c example", "-V", "${example}"} {
		resolver.command = command
		reference, _ := artifactpypi.ParseReference("example@1.0")
		if selected, err := resolver.ResolveExecutionDependencies(context.Background(), reference); err == nil || len(selected.Graph().Nodes()) != 0 {
			t.Fatal("unsupported configured command accepted", command)
		}
	}
}
