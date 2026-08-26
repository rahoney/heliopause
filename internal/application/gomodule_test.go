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

func TestGoModuleProjectResolutionRejectsInvalidSnapshot(t *testing.T) {
	target, _ := domain.NewInstallTarget("/tmp/haa-go-project")
	installContext, _ := domain.NewInstallContext(target)
	service, err := application.NewGoModuleProjectResolutionService(goModuleProjectResolverFixture{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Resolve(context.Background(), installContext); err == nil {
		t.Fatal("accepted invalid project snapshot")
	}
}

func TestGoModuleGetDoesNotPromoteWhenResolutionFails(t *testing.T) {
	reference, _ := artifactgomodule.ParseReference("example.com/module@v1.2.3")
	target, _ := domain.NewInstallTarget("/tmp/haa-go-project")
	installContext, _ := domain.NewInstallContext(target)
	promoter := &goModulePromoterFixture{}
	service, err := application.NewGoModuleGetService(&goModuleResolverFixture{}, promoter)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(context.Background(), reference, installContext); err == nil || !errors.Is(err, errGoModuleResolver) {
		t.Fatalf("Get error = %v", err)
	}
	if promoter.called {
		t.Fatal("Go project promotion ran after failed resolution")
	}
}

var errGoModuleResolver = errors.New("resolver failed")

type goModuleResolverFixture struct{}

func (*goModuleResolverFixture) ResolveDependencies(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error) {
	return domain.DependencyResolution{}, errGoModuleResolver
}

type goModuleProjectResolverFixture struct{}

func (goModuleProjectResolverFixture) ResolveProjectDependencies(context.Context, domain.InstallContext) (domain.ProjectDependencySnapshot, error) {
	return domain.ProjectDependencySnapshot{}, nil
}

type goModulePromoterFixture struct{ called bool }

func (p *goModulePromoterFixture) PromoteProjectDependency(context.Context, domain.ArtifactReference, domain.InstallContext) error {
	p.called = true
	return nil
}
