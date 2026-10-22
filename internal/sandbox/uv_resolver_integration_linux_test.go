//go:build linux

package sandbox

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// Run through the existing protected CI client path: the typed policy helper
// authenticates this actual test executable, not an unregistered go-test child.
func TestUVPublicNativeResolverIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_PUBLIC_RESOLVER_INTEGRATION") != "1" {
		t.Skip("requires actual pinned UV/Python/gVisor and authenticated network-policy service")
	}
	if os.Geteuid() == 0 {
		t.Fatal("requires ordinary-user typed-helper caller")
	}
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	runner := integrationRunner{t: t}
	resolver, err := NewUVResolver(runner, systemNamedEndpointResolver{}, supervisor.Observer(), integrationPythonCapabilityProbe(runner), integrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		locator, name, version string
		minimumNodes           int
	}{
		{"idna@3.11", "idna", "3.11", 1},
		{"requests@2.32.5", "requests", "2.32.5", 5},
	} {
		reference, err := artifactpypi.ParseReference(test.locator)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		result, err := resolver.ResolveDependencies(ctx, reference, domain.InstallContext{})
		cancel()
		// Preserve the first failed public input; do not repeat the same fault
		// against a larger graph or claim independent replay as live success.
		if err != nil {
			t.Fatalf("public native UV selection/binding %s failed: %v", test.locator, err)
		}
		if len(result.Graph().Nodes()) < test.minimumNodes || result.LockfileDigest().String() == "" || !strings.HasPrefix(result.RuntimeIdentity(), "uv:") || strings.Contains(result.RuntimeIdentity(), ";pip:") {
			t.Fatal("public native graph is incomplete or solver identity differs")
		}
		primary := false
		for _, node := range result.Graph().Nodes() {
			identity := node.Artifact().Identity()
			if identity.Source() != artifactpypi.PublicPyPIProfile().Source() {
				t.Fatal("public UV graph adopted a foreign source")
			}
			if node.Node() == result.Graph().Primary() {
				primary = identity.Name() == test.name && identity.Version() == test.version
			}
		}
		if !primary {
			t.Fatal("public UV graph root identity differs")
		}
		t.Logf("uv_public_graph reference=%s nodes=%d edges=%d runtime=%s", test.locator, len(result.Graph().Nodes()), len(result.Graph().Edges()), result.RuntimeIdentity())
	}
}

// Original metadata is an inert selection fixture, not enrollment authority.
// This qualifies the actual native producer and independent reader only.
func TestUVPublicNativeUpdateResolverIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_PUBLIC_UPDATE_RESOLVER_INTEGRATION") != "1" {
		t.Skip("requires pinned UV and authenticated typed policy helper")
	}
	if os.Geteuid() == 0 {
		t.Fatal("requires ordinary-user typed-helper caller")
	}
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	runner := integrationRunner{t: t}
	resolver, err := NewUVResolver(runner, systemNamedEndpointResolver{}, supervisor.Observer(), integrationPythonCapabilityProbe(runner), integrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	files := []artifactpypi.InstalledMetadataFile{
		{Directory: "packaging-24.2.dist-info", Body: []byte("Metadata-Version: 2.4\nName: packaging\nVersion: 24.2\n\n")},
		{Directory: "idna-3.11.dist-info", Body: []byte("Metadata-Version: 2.4\nName: idna\nVersion: 3.11\n\n")},
	}
	for _, test := range []struct {
		intent            artifactpypi.UVSelectionIntent
		selector, version string
	}{
		{artifactpypi.UVInstallSelection, "packaging", "24.2"},
		{artifactpypi.UVUpgradePackageSelection, "packaging==25.0", "25.0"},
		{artifactpypi.UVUpgradeSelection, "packaging==25.0", "25.0"},
	} {
		requirement, err := artifactpypi.ParseUpgradeRequirement(test.selector)
		if err != nil {
			t.Fatal(err)
		}
		input, err := artifactpypi.NewUpgradeInput(requirement, files, PinnedPythonRuntime().PythonVersion)
		if err != nil {
			t.Fatal(err)
		}
		original, err := input.Control()
		if err != nil {
			t.Fatal(err)
		}
		intent, err := artifactpypi.NewUVSelectionIntentControl(test.intent)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		result, err := resolver.ResolveDependencyUpdate(ctx, requirement.Reference(), domain.InstallContext{}, []domain.ProjectControlFile{original, intent})
		cancel()
		if err != nil {
			t.Fatalf("first actual UV installed-aware selection failed intent=%s: %v", test.intent, err)
		}
		nodes := result.Graph().Nodes()
		if len(nodes) != 1 || nodes[0].Artifact().Identity().Name() != "packaging" || nodes[0].Artifact().Identity().Version() != test.version || result.RuntimeIdentity() != UVResolverRuntimeIdentity() || len(result.ProjectControls()) != 4 {
			t.Fatal("native selection/graph/control binding differs", test.intent)
		}
		selected := map[string]domain.ProjectControlFile{}
		for _, control := range result.ProjectControls() {
			selected[control.Name()] = control
		}
		if selected[artifactpypi.UpgradeInputControlName].Digest() != original.Digest() || selected[artifactpypi.UVSelectionIntentControlName].Digest() != intent.Digest() || selected[artifactpypi.UVSelectionLockControlName].Digest() != result.LockfileDigest() {
			t.Fatal("original intent or selected lock changed")
		}
		t.Logf("uv_public_update intent=%s version=%s nodes=%d controls=%d runtime=%s", test.intent, test.version, len(nodes), len(selected), result.RuntimeIdentity())
	}
}
