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

// The pinned Cargo fixture was captured with a bounded, offline diagnostic
// directory source. It proves metadata grammar, not production acquisition or
// observer coverage. No crate code was compiled or executed for the capture.
func TestCargoResolverActualMetadataAndLock(t *testing.T) {
	body, err := os.ReadFile("../artifact/cargo/testdata/metadata-v1-rust-1.99.0.json")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := os.ReadFile("../artifact/cargo/testdata/metadata-v1-rust-1.99.0.lock")
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	manifest := []byte("[package]\nname=\"haa_schema_fixture\"\nversion=\"0.1.0\"\nedition=\"2021\"\n[dependencies]\nitoa=\"=1.0.17\"\n")
	for name, data := range map[string][]byte{"Cargo.toml": manifest, "Cargo.lock": lock} {
		if err := os.WriteFile(filepath.Join(project, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	runner := &cargoActualMetadataRunner{body: body}
	resolver, err := NewCargoResolver(runner)
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := artifactcargo.ParseReference("itoa@1.0.17")
	target, _ := domain.NewInstallTarget(project)
	install, _ := domain.NewInstallContext(target)
	resolution, err := resolver.ResolveDependencies(context.Background(), reference, install)
	if err != nil {
		t.Fatalf("actual Cargo metadata with matching frozen lock: %v", err)
	}
	nodes := resolution.Graph().Nodes()
	if len(nodes) != 1 || nodes[0].Artifact().Identity().Name() != "itoa" || nodes[0].Artifact().DeclaredIntegrity() != "sha256=92ecc6618181def0457392ccd0ee51198e065e016d1d527a7ac1b6dc7c1f09d2" {
		t.Fatalf("actual Cargo graph lost exact lock identity: nodes=%d", len(nodes))
	}
}

type cargoActualMetadataRunner struct{ body []byte }

func (r *cargoActualMetadataRunner) RunCargo(_ context.Context, directory string, _ []string, _ ...string) ([]byte, error) {
	return []byte(strings.ReplaceAll(string(r.body), "/fixture", directory)), nil
}
