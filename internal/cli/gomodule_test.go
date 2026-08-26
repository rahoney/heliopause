package cli_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestGoModuleDownloadInvokesProjectResolver(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root, err := cli.New(&stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	resolver := &goModuleProjectResolverFixture{}
	if err := cli.AddGoModuleDownload(root, resolver); err != nil {
		t.Fatal(err)
	}
	root.SetArgs([]string{"go", "mod", "download"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !resolver.called || !resolver.context.Valid() {
		t.Fatalf("project resolver request = %#v", resolver.context)
	}
}

type goModuleProjectResolverFixture struct {
	called  bool
	context domain.InstallContext
}

func (r *goModuleProjectResolverFixture) Resolve(_ context.Context, installContext domain.InstallContext) (domain.ProjectDependencySnapshot, error) {
	r.called = true
	r.context = installContext
	return domain.ProjectDependencySnapshot{}, nil
}
