package sandbox

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactgomodule "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type goModuleRunnerFixture struct {
	calls  []string
	caches []string
}

type goModulePrivateMutationRunner struct {
	goModuleRunnerFixture
	workspace string
}

func (r *goModulePrivateMutationRunner) RunGo(ctx context.Context, directory string, environment []string, arguments ...string) ([]byte, error) {
	r.workspace = directory
	if strings.Join(arguments, " ") == "mod download -json all" {
		if err := os.WriteFile(filepath.Join(directory, "go.sum"), []byte("resolver-private updated sums\n"), 0o600); err != nil {
			return nil, err
		}
	}
	return r.goModuleRunnerFixture.RunGo(ctx, directory, environment, arguments...)
}

func TestGoModuleProjectResolverCannotMutateUserControlFiles(t *testing.T) {
	project := t.TempDir()
	mod := []byte("module example.com/app\n\ngo 1.25\nrequire example.com/mod v1.2.3\n")
	sum := goFixtureSums()
	for name, body := range map[string][]byte{"go.mod": mod, "go.sum": sum} {
		if err := os.WriteFile(filepath.Join(project, name), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	target, _ := domain.NewInstallTarget(project)
	install, _ := domain.NewInstallContext(target)
	runner := &goModulePrivateMutationRunner{}
	resolver, _ := NewGoModuleResolver(runner)
	if _, err := resolver.ResolveProjectDependencies(context.Background(), install); err != nil {
		t.Fatal(err)
	}
	if runner.workspace == project {
		t.Fatal("resolver ran inside the user project")
	}
	for name, want := range map[string][]byte{"go.mod": mod, "go.sum": sum} {
		got, err := os.ReadFile(filepath.Join(project, name))
		if err != nil || string(got) != string(want) {
			t.Fatalf("user %s was modified: %q (%v)", name, got, err)
		}
	}
	if _, err := os.Stat(runner.workspace); !os.IsNotExist(err) {
		t.Fatalf("private workspace survived: %v", err)
	}
}

func TestGoModuleResolverRejectsUnsafeControlsBeforeAnyCommand(t *testing.T) {
	for _, member := range []string{"replacement", "mod symlink", "sum symlink", "oversize", "directory"} {
		t.Run(member, func(t *testing.T) {
			project := t.TempDir()
			mod := "module example.com/app\n\ngo 1.25\nrequire example.com/mod v1.2.3\n"
			if member == "replacement" {
				mod += "replace example.com/mod => ../host-private\n"
			}
			if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte(mod), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(project, "go.sum"), []byte("fixture\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			name := "go.mod"
			if member == "sum symlink" {
				name = "go.sum"
			}
			if strings.HasSuffix(member, "symlink") || member == "directory" {
				if err := os.Remove(filepath.Join(project, name)); err != nil {
					t.Fatal(err)
				}
				var err error
				if member == "directory" {
					err = os.Mkdir(filepath.Join(project, name), 0o700)
				} else {
					err = os.Symlink(filepath.Join(t.TempDir(), "outside"), filepath.Join(project, name))
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			if member == "oversize" {
				if err := os.WriteFile(filepath.Join(project, name), []byte(strings.Repeat(" ", artifactgomodule.MaxProjectControlBytes+1)), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			target, _ := domain.NewInstallTarget(project)
			install, _ := domain.NewInstallContext(target)
			runner := &goModuleRunnerFixture{}
			resolver, _ := NewGoModuleResolver(runner)
			if _, err := resolver.ResolveProjectDependencies(context.Background(), install); err == nil {
				t.Fatal("unsafe project controls reached resolver")
			}
			if len(runner.calls) != 0 {
				t.Fatalf("Go ran before control validation: %v", runner.calls)
			}
		})
	}
}

func (r *goModuleRunnerFixture) RunGo(_ context.Context, _ string, environment []string, arguments ...string) ([]byte, error) {
	cache := ""
	for _, entry := range environment {
		if strings.HasPrefix(entry, "GOMODCACHE=") {
			cache = strings.TrimPrefix(entry, "GOMODCACHE=")
		}
	}
	if err := artifactgomodule.ValidateResolverEnvironmentForCache(environment, cache); err != nil {
		return nil, err
	}
	if info, err := os.Stat(cache); err != nil || !info.IsDir() {
		return nil, errors.New("private Go module cache is unavailable")
	}
	r.caches = append(r.caches, cache)
	r.calls = append(r.calls, strings.Join(arguments, " "))
	h1 := func(value byte) string {
		return "h1:" + base64.StdEncoding.EncodeToString([]byte(strings.Repeat(string([]byte{value}), 32)))
	}
	if strings.Join(arguments, " ") == "get example.com/mod@v1.2.3" {
		return nil, nil
	}
	if strings.Join(arguments, " ") == "mod download -json all" {
		return []byte(`{"Path":"example.com/mod","Version":"v1.2.3","GoMod":"/tmp/mod.mod","Zip":"/tmp/mod.zip","Sum":"` + h1('a') + `","GoModSum":"` + h1('b') + `","Origin":null}` + "\n"), nil
	}
	return []byte("example.com/app example.com/mod@v1.2.3\nexample.com/app go@1.25\ngo@1.25 toolchain@go1.25\n"), nil
}

func TestGoModuleResolverUsesCanonicalCommandsAndSource(t *testing.T) {
	runner := &goModuleRunnerFixture{}
	resolver, err := NewGoModuleResolver(runner)
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := artifactgomodule.ParseReference("example.com/mod@v1.2.3")
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.com/app\n\ngo 1.25\nrequire example.com/mod v1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "go.sum"), goFixtureSums(), 0o600); err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(project)
	install, _ := domain.NewInstallContext(target)
	resolution, err := resolver.ResolveDependencies(context.Background(), reference, install)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolution.Graph().Nodes()) != 1 || resolution.Graph().Nodes()[0].Artifact().Identity().Source() != artifactgomodule.Source() {
		t.Fatalf("resolution graph = %#v", resolution.Graph())
	}
	if len(runner.calls) != 3 || runner.calls[0] != "get example.com/mod@v1.2.3" || runner.calls[1] != "mod download -json all" || runner.calls[2] != "mod graph" {
		t.Fatalf("Go commands = %#v", runner.calls)
	}
}

func TestGoModuleResolverRejectsOtherSource(t *testing.T) {
	runner := &goModuleRunnerFixture{}
	resolver, _ := NewGoModuleResolver(runner)
	other, _ := domain.NewSourceID("pypi")
	reference, _ := domain.NewArtifactReference(other, "example.com/mod@v1.2.3")
	target, _ := domain.NewInstallTarget("/workspace/project")
	install, _ := domain.NewInstallContext(target)
	if _, err := resolver.ResolveDependencies(context.Background(), reference, install); err == nil {
		t.Fatal("Go resolver accepted another source")
	}
}

func TestGoModuleResolverFreezesWholeProjectWithoutPrimaryArtifact(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.com/app\n\ngo 1.25\nrequire example.com/mod v1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "go.sum"), goFixtureSums(), 0o600); err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(project)
	install, _ := domain.NewInstallContext(target)
	runner := &goModuleRunnerFixture{}
	resolver, err := NewGoModuleResolver(runner)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := resolver.ResolveProjectDependencies(context.Background(), install)
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Valid() || snapshot.Source() != artifactgomodule.Source() || len(snapshot.Dependencies()) != 1 || len(snapshot.ControlDigests()) != 2 {
		t.Fatalf("project snapshot = %#v", snapshot)
	}
	for _, cache := range runner.caches {
		if _, err := os.Stat(cache); !os.IsNotExist(err) {
			t.Fatalf("private Go module cache remained after resolution: %s (%v)", cache, err)
		}
	}
}

func TestGoModuleResolverRejectsProjectControlFileDrift(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.com/app\n\ngo 1.25\nrequire example.com/mod v1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "go.sum"), goFixtureSums(), 0o600); err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(project)
	install, _ := domain.NewInstallContext(target)
	resolver, err := NewGoModuleResolver(&goModuleDriftRunner{project: project})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.ResolveProjectDependencies(context.Background(), install); err == nil || !strings.Contains(err.Error(), "changed during resolution") {
		t.Fatalf("project drift error = %v", err)
	}
}

func TestGoModuleExactResolverRejectsProjectControlFileDrift(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.com/app\n\ngo 1.25\nrequire example.com/mod v1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "go.sum"), goFixtureSums(), 0o600); err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(project)
	install, _ := domain.NewInstallContext(target)
	reference, _ := artifactgomodule.ParseReference("example.com/mod@v1.2.3")
	resolver, err := NewGoModuleResolver(&goModuleExactDriftRunner{project: project})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.ResolveDependencies(context.Background(), reference, install); err == nil || !strings.Contains(err.Error(), "changed during resolution") {
		t.Fatalf("exact project drift error = %v", err)
	}
}

type goModuleExactDriftRunner struct {
	project string
	calls   int
}

func (r *goModuleExactDriftRunner) RunGo(_ context.Context, _ string, environment []string, arguments ...string) ([]byte, error) {
	cache := ""
	for _, entry := range environment {
		if strings.HasPrefix(entry, "GOMODCACHE=") {
			cache = strings.TrimPrefix(entry, "GOMODCACHE=")
		}
	}
	if err := artifactgomodule.ValidateResolverEnvironmentForCache(environment, cache); err != nil {
		return nil, err
	}
	r.calls++
	if r.calls == 1 {
		return nil, nil
	}
	if r.calls == 3 {
		if err := os.WriteFile(filepath.Join(r.project, "go.sum"), []byte("changed\n"), 0o600); err != nil {
			return nil, err
		}
		return []byte("example.com/app example.com/mod@v1.2.3\nexample.com/app go@1.25\ngo@1.25 toolchain@go1.25\n"), nil
	}
	h1 := func(value byte) string {
		return "h1:" + base64.StdEncoding.EncodeToString([]byte(strings.Repeat(string([]byte{value}), 32)))
	}
	return []byte(`{"Path":"example.com/mod","Version":"v1.2.3","GoMod":"/tmp/mod.mod","Zip":"/tmp/mod.zip","Sum":"` + h1('a') + `","GoModSum":"` + h1('b') + `","Origin":null}` + "\n"), nil
}

type goModuleDriftRunner struct {
	project string
	calls   int
}

func (r *goModuleDriftRunner) RunGo(_ context.Context, _ string, environment []string, arguments ...string) ([]byte, error) {
	cache := ""
	for _, entry := range environment {
		if strings.HasPrefix(entry, "GOMODCACHE=") {
			cache = strings.TrimPrefix(entry, "GOMODCACHE=")
		}
	}
	if err := artifactgomodule.ValidateResolverEnvironmentForCache(environment, cache); err != nil {
		return nil, err
	}
	r.calls++
	if r.calls == 2 {
		if err := os.WriteFile(filepath.Join(r.project, "go.sum"), []byte("changed\n"), 0o600); err != nil {
			return nil, err
		}
		return []byte("example.com/app example.com/mod@v1.2.3\nexample.com/app go@1.25\ngo@1.25 toolchain@go1.25\n"), nil
	}
	h1 := func(value byte) string {
		return "h1:" + base64.StdEncoding.EncodeToString([]byte(strings.Repeat(string([]byte{value}), 32)))
	}
	return []byte(`{"Path":"example.com/mod","Version":"v1.2.3","GoMod":"/tmp/mod.mod","Zip":"/tmp/mod.zip","Sum":"` + h1('a') + `","GoModSum":"` + h1('b') + `","Origin":null}` + "\n"), nil
}

func goFixtureSums() []byte {
	h1 := func(value byte) string {
		return "h1:" + base64.StdEncoding.EncodeToString([]byte(strings.Repeat(string([]byte{value}), 32)))
	}
	return []byte("example.com/mod v1.2.3 " + h1('a') + "\nexample.com/mod v1.2.3/go.mod " + h1('b') + "\n")
}
