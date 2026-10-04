package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type goUpdateRunner struct{ goModuleRunnerFixture }

func (r *goUpdateRunner) RunGo(ctx context.Context, directory string, environment []string, arguments ...string) ([]byte, error) {
	if strings.Join(arguments, " ") == "get example.com/mod@v1.2.3" {
		if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module example.com/app\n\ngo 1.25\nrequire example.com/mod v1.2.3\n"), 0o600); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(directory, "go.sum"), goFixtureSums(), 0o600); err != nil {
			return nil, err
		}
	}
	return r.goModuleRunnerFixture.RunGo(ctx, directory, environment, arguments...)
}

func TestGoUpdateFreezesOneSelectionIncludingAbsentOriginalSum(t *testing.T) {
	project := t.TempDir()
	mod := []byte("module example.com/app\n\ngo 1.25\n")
	if err := os.WriteFile(filepath.Join(project, "go.mod"), mod, 0o600); err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(project)
	install, _ := domain.NewInstallContext(target)
	original, err := readGoFrozenControls(project)
	if err != nil {
		t.Fatal(err)
	}
	runner := &goUpdateRunner{}
	resolver, _ := NewGoModuleResolver(runner)
	reference, _ := artifactgo.ParseReference("example.com/mod@v1.2.3")
	update, err := resolver.ResolveProjectDependencyUpdate(context.Background(), reference, install, original)
	if err != nil || !update.Valid() {
		t.Fatalf("update: %v", err)
	}
	if len(runner.calls) != 3 || runner.calls[0] != "get example.com/mod@v1.2.3" {
		t.Fatalf("extra/live selection: %v", runner.calls)
	}
	if update.OriginalControls()[1].Present() || !update.SelectedControls()[1].Present() || len(update.Snapshot().Dependencies()) != 1 || update.Resolution().LockfileDigest() != update.Snapshot().GraphDigest() {
		t.Fatal("selection/control bindings lost")
	}
	body, err := os.ReadFile(filepath.Join(project, "go.mod"))
	if err != nil || string(body) != string(mod) {
		t.Fatal("user module control changed")
	}
	if _, err := os.Lstat(filepath.Join(project, "go.sum")); !os.IsNotExist(err) {
		t.Fatal("user checksum control published during selection")
	}
}

func TestGoUpdateRejectsChangedPresenceBeforeExecution(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.com/app\ngo 1.25\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	original, err := readGoFrozenControls(project)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "go.sum"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(project)
	install, _ := domain.NewInstallContext(target)
	runner := &goUpdateRunner{}
	resolver, _ := NewGoModuleResolver(runner)
	reference, _ := artifactgo.ParseReference("example.com/mod@v1.2.3")
	update, err := resolver.ResolveProjectDependencyUpdate(context.Background(), reference, install, original)
	if err == nil || update.Valid() || len(runner.calls) != 0 {
		t.Fatalf("changed presence reached execution: %v", err)
	}
}
