package cli_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestTerraformInitInvokesExactProviderResolver(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root, err := cli.New(&stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	resolver := &terraformResolverFixture{}
	if err := cli.AddTerraformInit(root, resolver); err != nil {
		t.Fatal(err)
	}
	root.SetArgs([]string{"terraform", "init", "hashicorp/aws@5.50.0"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !resolver.called || resolver.reference.Source().String() != "terraform-registry" || resolver.reference.Locator() != "hashicorp/aws@5.50.0" || !resolver.install.Valid() {
		t.Fatalf("Terraform resolution request = %#v %#v", resolver.reference, resolver.install)
	}
}

type terraformResolverFixture struct {
	called    bool
	reference domain.ArtifactReference
	install   domain.InstallContext
}

func (r *terraformResolverFixture) Resolve(_ context.Context, reference domain.ArtifactReference, install domain.InstallContext) (domain.DependencyResolution, error) {
	r.called, r.reference, r.install = true, reference, install
	return domain.DependencyResolution{}, nil
}
