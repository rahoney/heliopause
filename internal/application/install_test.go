package application_test

import (
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestNewInstallRequestPreservesReferenceAndNewTargetContext(t *testing.T) {
	t.Parallel()
	source, _ := domain.NewSourceID("npm")
	reference, _ := domain.NewArtifactReference(source, "example@1.0.0")
	target, _ := domain.NewInstallTarget("/tmp/heliopause-target")
	context, _ := domain.NewInstallContext(target)
	request, err := application.NewInstallRequest(reference, context)
	if err != nil {
		t.Fatal(err)
	}
	if request.Reference() != reference || request.Context() != context {
		t.Fatalf("request = %#v", request)
	}
}

func TestNewInstallRequestPreservesNPMProjectContext(t *testing.T) {
	t.Parallel()
	source, _ := domain.NewSourceID("npm")
	reference, _ := domain.NewArtifactReference(source, "example@1.0.0")
	root, _ := domain.NewInstallTarget("/tmp/heliopause-project")
	context, _ := domain.NewNPMProjectInstallContext(root)
	request, err := application.NewInstallRequest(reference, context)
	if err != nil || request.Context().Mode() != domain.InstallNPMProject {
		t.Fatalf("request = %#v, %v", request, err)
	}
}
