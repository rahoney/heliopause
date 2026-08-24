package bootstrap

import (
	"context"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestUnsupportedInstallResolverFailsBeforeHostResolution(t *testing.T) {
	t.Parallel()
	resolver, err := installDependencyResolver("darwin", "arm64", nil)
	if err != nil {
		t.Fatal(err)
	}
	source, _ := domain.NewSourceID("npm")
	reference, _ := domain.NewArtifactReference(source, "safe@1.0.0")
	target, _ := domain.NewInstallTarget("/tmp/heliopause-unsupported-target")
	installContext, _ := domain.NewInstallContext(target)
	if _, err := resolver.ResolveDependencies(context.Background(), reference, installContext); err == nil {
		t.Fatal("unsupported resolver returned success")
	}
}
