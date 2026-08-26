package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	artifactgomodule "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestGoModuleResolutionServiceUsesOnlyDependencyResolver(t *testing.T) {
	reference, err := artifactgomodule.ParseReference("example.com/module@v1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget("/tmp/haa-go-module-project")
	installContext, _ := domain.NewInstallContext(target)
	resolver := &goModuleResolverFixture{}
	service, err := application.NewGoModuleResolutionService(resolver)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Resolve(context.Background(), reference, installContext); err == nil || !errors.Is(err, errGoModuleResolver) {
		t.Fatalf("Resolve error = %v", err)
	}
}

var errGoModuleResolver = errors.New("resolver failed")

type goModuleResolverFixture struct{}

func (*goModuleResolverFixture) ResolveDependencies(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error) {
	return domain.DependencyResolution{}, errGoModuleResolver
}
