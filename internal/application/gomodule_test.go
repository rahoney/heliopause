package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	artifactgomodule "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
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
	service, err := application.NewGoModuleProjectResolutionService(goModuleProjectResolverFixture{}, &goModulePromoterFixture{}, goUnavailableProjectPipeline{}, goUnavailableProjectPipeline{})
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
	service, err := application.NewGoModuleGetService(&goModuleResolverFixture{}, promoter, goUnavailableProjectPipeline{}, goUnavailableProjectPipeline{})
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

func (*goModuleResolverFixture) ResolveProjectDependencyUpdate(context.Context, domain.ArtifactReference, domain.InstallContext, []domain.ProjectControlFile) (domain.ProjectDependencyUpdate, error) {
	return domain.ProjectDependencyUpdate{}, errGoModuleResolver
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

func (p *goModulePromoterFixture) Begin(context.Context, domain.InstallContext) (ports.ProjectMutationGuard, error) {
	return &goMutationGuardFixture{owner: p}, nil
}

type goMutationGuardFixture struct{ owner *goModulePromoterFixture }

func (*goMutationGuardFixture) Controls() []domain.ProjectControlFile { return nil }
func (*goMutationGuardFixture) VerifyUnchanged(context.Context) error { return nil }
func (g *goMutationGuardFixture) Commit(context.Context, domain.ProjectDependencyUpdate, domain.StagedProjectSet) error {
	g.owner.called = true
	return nil
}
func (*goMutationGuardFixture) Close() error { return nil }

type goUnavailableProjectPipeline struct{}

func (goUnavailableProjectPipeline) InspectProject(context.Context, domain.ProjectDependencySnapshot) (domain.ProjectVerifiedSet, error) {
	return domain.ProjectVerifiedSet{}, errors.New("inspection must be unreachable")
}
func (goUnavailableProjectPipeline) StageProject(context.Context, domain.ProjectVerifiedSet) (domain.StagedProjectSet, error) {
	return domain.StagedProjectSet{}, errors.New("staging must be unreachable")
}

func (g *goMutationGuardFixture) CommitSnapshot(context.Context, domain.ProjectDependencySnapshot, domain.StagedProjectSet) error {
	g.owner.called = true
	return nil
}
