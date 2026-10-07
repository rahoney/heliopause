package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// This explicit gate consumes the registered observer, pinned OCI toolchain
// and authenticated resolver network helper; unit/stock-OCI PASS is separate.
func TestLinuxCargoSourceProjectSnapshotIntegration(t *testing.T) {
	if os.Getenv("HELOX_CARGO_RESOLVER_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	runner := integrationRunner{t: t}
	isolated, err := newIsolatedCargoRunner(runner, cargoNamedEndpointResolver{}, supervisor.Observer(), func(ctx context.Context) (CargoCapability, error) { return ProbeCargo(ctx, runner) }, integrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	project := cargoSourceFixture(t)
	manifest := []byte("[package]\nname='haa_schema_fixture'\nversion='0.1.0'\nedition='2021'\n[dependencies]\nitoa='=1.0.17'\n")
	lock, err := os.ReadFile("../artifact/cargo/testdata/metadata-v1-rust-1.99.0.lock")
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string][]byte{"Cargo.toml": manifest, "Cargo.lock": lock} {
		if err := os.WriteFile(filepath.Join(project, name), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	target, _ := domain.NewInstallTarget(project)
	install, _ := domain.NewInstallContext(target)
	resolver, _ := NewCargoResolver(isolated)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	snapshot, err := resolver.ResolveProjectDependencies(ctx, install)
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Valid() || snapshot.Context() != install || snapshot.Source() != artifactcargo.Source() || len(snapshot.Dependencies()) != 1 {
		t.Fatal("actual isolated project is incomplete")
	}
	crate := snapshot.Dependencies()[0]
	if crate.Identity().Name() != "itoa" || crate.Identity().Version() != "1.0.17" || crate.DeclaredIntegrity() != "sha256=92ecc6618181def0457392ccd0ee51198e065e016d1d527a7ac1b6dc7c1f09d2" {
		t.Fatal("exact public selection changed")
	}
	t.Logf("actual_cargo_graph_sha256=%s crates=%d", snapshot.GraphDigest(), len(snapshot.Dependencies()))
}
