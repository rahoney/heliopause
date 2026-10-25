package bootstrap

import (
	"context"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestBunProjectRequiresActualBindingsAndSelectedContext(t *testing.T) {
	if service, err := newBunProjectService(bunProjectBindings{}, artifactnpm.BunInstall, ""); err == nil || service != nil {
		t.Fatal("missing actual adapters accepted")
	}
	s := &bunProjectSession{}
	if approved, err := s.InspectProject(context.Background(), domain.ProjectDependencySnapshot{}); err == nil || approved.Valid() {
		t.Fatal("inspection accepted missing native context")
	}
	if staged, err := s.StageProject(context.Background(), domain.ProjectVerifiedSet{}); err == nil || staged.Valid() {
		t.Fatal("staging accepted missing native approval")
	}
	// Invalid input is rejected before touching the native adapter, and still
	// consumes this request. Retrying selection cannot reuse partial state.
	if _, err := s.ResolveProjectLockUpdate(context.Background(), domain.InstallContext{}, nil, ""); err == nil {
		t.Fatal("invalid selection input accepted")
	}
	if _, err := s.ResolveProjectLockUpdate(context.Background(), domain.InstallContext{}, nil, ""); err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatal("request repeated selection", err)
	}
}
