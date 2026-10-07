package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestTerraformInitInvokesGuardedInstallation(t *testing.T) {
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
	if !strings.Contains(stdout.String(), "Providers: 2") {
		t.Fatal("complete project provider count was lost")
	}
}

type terraformResolverFixture struct {
	called    bool
	reference domain.ArtifactReference
	install   domain.InstallContext
}

func (r *terraformResolverFixture) Init(_ context.Context, reference domain.ArtifactReference, install domain.InstallContext) (application.TerraformInitResult, error) {
	r.called, r.reference, r.install = true, reference, install
	return application.TerraformInitResult{ProviderCount: 2}, nil
}
