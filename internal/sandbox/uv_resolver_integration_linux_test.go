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
