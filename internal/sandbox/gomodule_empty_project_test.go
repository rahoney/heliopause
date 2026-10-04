package sandbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestGoResolverFreezesExplicitDependencyFreeProject(t *testing.T) {
	for _, present := range []bool{false, true} {
		project := t.TempDir()
		if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.com/app\ngo 1.26.0\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if present {
			if err := os.WriteFile(filepath.Join(project, "go.sum"), nil, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		target, _ := domain.NewInstallTarget(project)
		install, _ := domain.NewInstallContext(target)
		resolver, err := NewGoModuleResolver(goEmptyProjectRunner{})
		if err != nil {
			t.Fatal(err)
		}
		snapshot, err := resolver.ResolveProjectDependencies(context.Background(), install)
		if err != nil || !snapshot.DependencyFree() {
			t.Fatalf("dependency-free project: %v", err)
		}
		_, err = os.Lstat(filepath.Join(project, "go.sum"))
		if present && err != nil || !present && !os.IsNotExist(err) {
			t.Fatal("resolver changed original sum presence")
		}
	}
}

type goEmptyProjectRunner struct{}

func (goEmptyProjectRunner) RunGo(_ context.Context, _ string, _ []string, args ...string) ([]byte, error) {
	switch strings.Join(args, " ") {
	case "mod download -json all":
		return nil, nil
	case "mod graph":
		return []byte("example.com/app go@1.26.0\ngo@1.26.0 toolchain@go1.26.0\n"), nil
	default:
		return nil, errors.New("unexpected empty-project command")
	}
}
