package bootstrap

import (
	"context"
	"errors"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestPythonExecutionPreparationRejectsMissingAuthorityAndUnsupportedSelection(t *testing.T) {
	if service, err := newPythonExecutionPreparationService(nil, nil, nil, nil, nil, nil); err == nil || service != nil {
		t.Fatal("incomplete concrete preparation was configured")
	}
	if execution, err := newPythonConsoleExecutionContext(domain.OperationID{}, domain.DependencyResolution{}, domain.VerifiedSet{}, artifactpypi.ConsoleExecutionEntry{}); err == nil || execution.Valid() {
		t.Fatal("missing measured declaration became an execution context")
	}
	resolver := freshPythonToolExecutionResolver{native: &sandbox.UVResolver{}}
	for _, row := range []struct{ source, locator string }{{"npm", "example"}, {"pypi", "example>=1"}, {"pypi", "example[extra]"}, {"pypi", "https://example.invalid/tool.whl"}} {
		source, _ := domain.NewSourceID(row.source)
		reference, err := domain.NewArtifactReference(source, row.locator)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := resolver.ResolveExecutionDependencies(context.Background(), reference); err == nil {
			t.Fatal("unsupported reference reached native selection")
		}
	}
	reference, err := artifactpypi.ParseReference("example@1.0")
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("caller stopped Python selection")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if _, err := resolver.ResolveExecutionDependencies(ctx, reference); !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal("caller selection cause lost", err)
	}
}
