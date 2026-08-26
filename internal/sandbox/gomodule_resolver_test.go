package sandbox

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	artifactgomodule "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type goModuleRunnerFixture struct{ calls []string }

func (r *goModuleRunnerFixture) RunGo(_ context.Context, _ string, environment []string, arguments ...string) ([]byte, error) {
	if err := artifactgomodule.ValidateResolverEnvironment(environment); err != nil {
		return nil, err
	}
	r.calls = append(r.calls, strings.Join(arguments, " "))
	h1 := func(value byte) string {
		return "h1:" + base64.StdEncoding.EncodeToString([]byte(strings.Repeat(string([]byte{value}), 32)))
	}
	if strings.Join(arguments, " ") == "mod download -json all" {
		return []byte(`{"Path":"example.com/mod","Version":"v1.2.3","GoMod":"/tmp/mod.mod","Zip":"/tmp/mod.zip","Sum":"` + h1('a') + `","GoModSum":"` + h1('b') + `","Origin":null}` + "\n"), nil
	}
	return []byte("example.com/mod@v1.2.3 example.com/mod@v1.2.3\n"), nil
}

func TestGoModuleResolverUsesCanonicalCommandsAndSource(t *testing.T) {
	runner := &goModuleRunnerFixture{}
	resolver, err := NewGoModuleResolver(runner)
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := artifactgomodule.ParseReference("example.com/mod@v1.2.3")
	target, _ := domain.NewInstallTarget("/workspace/project")
	install, _ := domain.NewInstallContext(target)
	resolution, err := resolver.ResolveDependencies(context.Background(), reference, install)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolution.Graph().Nodes()) != 1 || resolution.Graph().Nodes()[0].Artifact().Identity().Source() != artifactgomodule.Source() {
		t.Fatalf("resolution graph = %#v", resolution.Graph())
	}
	if len(runner.calls) != 2 || runner.calls[0] != "mod download -json all" || runner.calls[1] != "mod graph" {
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
