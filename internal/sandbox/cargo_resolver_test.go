package sandbox

import (
	"context"
	"strings"
	"testing"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type cargoRunnerFixture struct{ calls []string }

func (r *cargoRunnerFixture) RunCargo(_ context.Context, _ string, environment []string, arguments ...string) ([]byte, error) {
	home := ""
	for _, entry := range environment {
		if strings.HasPrefix(entry, "CARGO_HOME=") {
			home = strings.TrimPrefix(entry, "CARGO_HOME=")
		}
	}
	expected, err := CargoResolverEnvironmentForHome(home)
	if err != nil || strings.Join(environment, "\n") != strings.Join(expected, "\n") {
		return nil, context.Canceled
	}
	r.calls = append(r.calls, strings.Join(arguments, " "))
	return []byte(`{"packages":[{"id":"registry+https://github.com/rust-lang/crates.io-index#serde@1.0.200","name":"serde","version":"1.0.200","source":"registry+https://github.com/rust-lang/crates.io-index","checksum":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}],"resolve":{"nodes":[{"id":"registry+https://github.com/rust-lang/crates.io-index#serde@1.0.200","deps":[]}]}}`), nil
}

func TestCargoResolverPinsCanonicalSparseRegistry(t *testing.T) {
	runner := &cargoRunnerFixture{}
	resolver, err := NewCargoResolver(runner)
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := artifactcargo.ParseReference("serde@1.0.200")
	target, _ := domain.NewInstallTarget("/workspace/project")
	install, _ := domain.NewInstallContext(target)
	resolution, err := resolver.ResolveDependencies(context.Background(), reference, install)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolution.Graph().Nodes()) != 1 || resolution.Graph().Nodes()[0].Artifact().Identity().Source() != artifactcargo.Source() {
		t.Fatalf("Cargo graph = %#v", resolution.Graph())
	}
	if len(runner.calls) != 1 || runner.calls[0] != "metadata --locked --format-version 1" {
		t.Fatalf("Cargo calls = %#v", runner.calls)
	}
}

func TestCargoResolverRejectsNonCargoSource(t *testing.T) {
	runner := &cargoRunnerFixture{}
	resolver, _ := NewCargoResolver(runner)
	other, _ := domain.NewSourceID("go-proxy")
	reference, _ := domain.NewArtifactReference(other, "serde@1.0.200")
	target, _ := domain.NewInstallTarget("/workspace/project")
	install, _ := domain.NewInstallContext(target)
	if _, err := resolver.ResolveDependencies(context.Background(), reference, install); err == nil {
		t.Fatal("Cargo resolver accepted another source")
	}
}
