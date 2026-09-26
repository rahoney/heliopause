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
	digest, _ := domain.NewSHA256Digest("0000000000000000000000000000000000000000000000000000000000000000")
	source, _ := domain.NewSourceID("go-proxy")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "example.com/module", "v1.2.3", "module")
	artifact, _ := domain.NewResolvedArtifact(identity, "https://proxy.golang.org/example.com/module/@v/v1.2.3.zip", "h1:fixture")
	mod, _ := domain.NewProjectControlDigest("go.mod", digest)
	sum, _ := domain.NewProjectControlDigest("go.sum", digest)
	return domain.NewProjectDependencySnapshot(installContext, source, []domain.ProjectControlDigest{mod, sum}, []domain.ResolvedArtifact{artifact}, digest)
}
