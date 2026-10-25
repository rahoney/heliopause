//go:build linux

package sandbox

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

func TestUVPublicVirtualProjectResolverIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_PROJECT_RESOLVER_INTEGRATION") != "1" {
		t.Skip("requires fixed UV/Python/gVisor and authenticated typed policy helper")
	}
	if os.Geteuid() == 0 {
		t.Fatal("requires ordinary-user caller")
	}
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	runner := integrationRunner{t: t}

	native, err := NewUVResolver(runner, systemNamedEndpointResolver{}, supervisor.Observer(), integrationPythonCapabilityProbe(runner), integrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := NewUVProjectResolver(native)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Error(err)
		}
	}()
	install := uvSyncTestContext(t)
	for _, dependencies := range []string{"[]", `["packaging==25.0"]`} {
		manifest := []byte(fmt.Sprintf("[project]\nname='haa-public-project'\nversion='0.1.0'\nrequires-python='==%s'\ndependencies=%s\n", PinnedPythonRuntime().PythonVersion, dependencies))
		input, err := artifactpypi.ParseUVProjectInput(manifest, nil, false, artifactpypi.UVProjectSync, "")
		if err != nil {
			t.Fatal(err)
		}
		original, err := NewUVProjectOriginalControls(input)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		update, err := resolver.ResolveProjectLockUpdate(ctx, install, original, "")
		cancel()
		if err != nil {
			t.Fatalf("first actual virtual project selection failed dependencies=%s: %v", dependencies, err)
		}
		if !update.Valid() || update.HasRequestedResolution() || len(update.SelectedControls()) != 6 {
			t.Fatal("project selection fabricated a primary or lost controls")
		}
		expected := 1
		if dependencies == "[]" {
			expected = 0
		}
		if len(update.Snapshot().Dependencies()) != expected || update.Snapshot().DependencyFree() != (expected == 0) {
			t.Fatal("actual project selection differs from complete input")
		}
		var lock []byte
		for _, c := range update.SelectedControls() {
			if c.Name() == artifactpypi.UVNativeLockControl {
				lock = c.Body()
			}
		}
		upgradeInput, err := artifactpypi.ParseUVProjectInput(manifest, lock, true, artifactpypi.UVProjectLockUpgrade, "")
		if err != nil {
			t.Fatal(err)
		}
		upgradeOriginal, err := NewUVProjectOriginalControls(upgradeInput)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		upgraded, err := resolver.ResolveProjectLockUpdate(ctx, install, upgradeOriginal, "")
		cancel()
		if err != nil || !upgraded.Valid() || len(upgraded.Snapshot().Dependencies()) != expected {
			t.Fatal("actual project lock-upgrade selection failed", err)
		}
		t.Logf("actual_virtual_project_selection=PASS dependencies=%d sync+lock-upgrade; independent source binding; no Policy/Host install claim", expected)
	}
}
