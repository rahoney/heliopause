package sandbox

import (
	"context"
	"os"
	"path/filepath"
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
	if err := ValidateCargoResolverEnvironment(environment, home); err != nil {
		return nil, context.Canceled
	}
	r.calls = append(r.calls, strings.Join(arguments, " "))
	return []byte(`{"packages":[{"id":"registry+https://github.com/rust-lang/crates.io-index#serde@1.0.200","name":"serde","version":"1.0.200","source":"registry+https://github.com/rust-lang/crates.io-index","checksum":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}],"resolve":{"nodes":[{"id":"registry+https://github.com/rust-lang/crates.io-index#serde@1.0.200","deps":[]}]}}`), nil
}

func TestCargoResolverEnvironmentRejectsRegistrySubstitution(t *testing.T) {
	environment, err := CargoResolverEnvironmentForHome("/private/tmp/haa-cargo-home")
	if err != nil || ValidateCargoResolverEnvironment(environment, "/private/tmp/haa-cargo-home") != nil {
		t.Fatalf("environment = %#v, error = %v", environment, err)
	}
	unsafe := append([]string(nil), environment...)
	unsafe[len(unsafe)-1] = "CARGO_REGISTRIES_CRATES_IO_INDEX=https://evil.example/"
	if err := ValidateCargoResolverEnvironment(unsafe, "/private/tmp/haa-cargo-home"); err == nil {
		t.Fatal("accepted substituted Cargo registry")
	}
}

func TestCargoResolverPinsCanonicalSparseRegistry(t *testing.T) {
	runner := &cargoRunnerFixture{}
	resolver, err := NewCargoResolver(runner)
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := artifactcargo.ParseReference("serde@1.0.200")
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "Cargo.toml"), []byte("[package]\nname = \"app\"\nversion = \"0.1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "Cargo.lock"), []byte("version = 3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(project)
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

func TestCargoResolverRejectsControlFileDrift(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "Cargo.toml"), []byte("[package]\nname = \"app\"\nversion = \"0.1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "Cargo.lock"), []byte("version = 3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(project)
	install, _ := domain.NewInstallContext(target)
	reference, _ := artifactcargo.ParseReference("serde@1.0.200")
	resolver, err := NewCargoResolver(&cargoDriftRunner{project: project})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.ResolveDependencies(context.Background(), reference, install); err == nil || !strings.Contains(err.Error(), "changed during resolution") {
		t.Fatalf("Cargo drift error = %v", err)
	}
}

type cargoDriftRunner struct{ project string }

func (r *cargoDriftRunner) RunCargo(_ context.Context, _ string, environment []string, _ ...string) ([]byte, error) {
	home := ""
	for _, entry := range environment {
		if strings.HasPrefix(entry, "CARGO_HOME=") {
			home = strings.TrimPrefix(entry, "CARGO_HOME=")
		}
	}
	if err := ValidateCargoResolverEnvironment(environment, home); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(r.project, "Cargo.lock"), []byte("changed\n"), 0o600); err != nil {
		return nil, err
	}
	return []byte(`{"packages":[{"id":"registry+https://github.com/rust-lang/crates.io-index#serde@1.0.200","name":"serde","version":"1.0.200","source":"registry+https://github.com/rust-lang/crates.io-index","checksum":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}],"resolve":{"nodes":[{"id":"registry+https://github.com/rust-lang/crates.io-index#serde@1.0.200","deps":[]}]}}`), nil
}
