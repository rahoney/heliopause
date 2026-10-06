package cli_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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

func TestCargoBuildKeepsFixedSelectorAndFirstFailure(t *testing.T) {
	for _, args := range [][]string{{"cargo", "build"}, {"cargo", "build", "./..."}, {"cargo", "build", "--release"}, {"cargo", "build", "--target-dir", "/tmp/foreign"}} {
		t.Run(fmt.Sprint(args), func(t *testing.T) {
			t.Chdir(t.TempDir())
			var stdout, stderr bytes.Buffer
			root, err := cli.New(&stdout, &stderr)
			if err != nil {
				t.Fatal(err)
			}
			failure := errors.New("trusted build failure")
			builder := &cargoBuilderFixture{failure: failure}
			if err := cli.AddCargoBuild(root, builder); err != nil {
				t.Fatal(err)
			}
			root.SetArgs(args)
			err = root.ExecuteContext(context.Background())
			if len(args) == 2 {
				if !errors.Is(err, failure) || !builder.called || builder.selector != "default" || !builder.install.Valid() {
					t.Fatal("fixed build context or primary failure lost")
				}
			} else if err == nil || builder.called {
				t.Fatal("artifact-selected flags/selector reached builder")
			}
			if stdout.Len() != 0 {
				t.Fatal("failed build printed successful publication")
			}
		})
	}
}

type cargoBuilderFixture struct {
	called   bool
	selector string
	install  domain.InstallContext
	failure  error
}

func (b *cargoBuilderFixture) Build(_ context.Context, install domain.InstallContext, selector string) (domain.OperationResult, domain.PublishedProjectBuild, error) {
	b.called, b.install, b.selector = true, install, selector
	return domain.OperationResult{}, domain.PublishedProjectBuild{}, b.failure
}
