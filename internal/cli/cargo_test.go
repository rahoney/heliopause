package cli_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestCargoAddInvokesResolverWithCanonicalReference(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root, err := cli.New(&stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	resolver := &cargoResolverFixture{}
	if err := cli.AddCargoAdd(root, resolver); err != nil {
		t.Fatal(err)
	}
	root.SetArgs([]string{"cargo", "add", "serde@1.0.200"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !resolver.called || resolver.reference.Source().String() != "crates-io" || resolver.reference.Locator() != "serde@1.0.200" || !resolver.install.Valid() {
		t.Fatalf("Cargo resolution request = %#v %#v", resolver.reference, resolver.install)
	}
}

type cargoResolverFixture struct {
	called    bool
	reference domain.ArtifactReference
	install   domain.InstallContext
}

func (r *cargoResolverFixture) Resolve(_ context.Context, reference domain.ArtifactReference, install domain.InstallContext) (domain.DependencyResolution, error) {
	r.called, r.reference, r.install = true, reference, install
	return domain.DependencyResolution{}, nil
}
