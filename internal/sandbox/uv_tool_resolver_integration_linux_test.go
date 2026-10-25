//go:build linux

package sandbox

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

func TestUVPublicToolInstallResolverIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_TOOL_RESOLVER_INTEGRATION") != "1" {
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
	resolver, err := NewUVResolver(runner, systemNamedEndpointResolver{}, supervisor.Observer(), integrationPythonCapabilityProbe(runner), integrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Error(err)
		}
	}()
	requirement, err := artifactpypi.ParseUpgradeRequirement("cowsay==6.1")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	selected, err := resolver.ResolveToolInstall(ctx, requirement)
	cancel()
	if err != nil {
		t.Fatal("first public native tool install/source binding failed", err)
	}
	resolution := selected.Resolution()
	if len(resolution.Graph().Nodes()) != 1 || resolution.LockfileDigest().String() == "" || !strings.HasPrefix(resolution.RuntimeIdentity(), "uv-tool-install:uv:") {
		t.Fatal("native tool graph/provenance differs")
	}
	for _, node := range resolution.Graph().Nodes() {
		id := node.Artifact().Identity()
		if id.Name() != "cowsay" || id.Version() != "6.1" || id.Source() != artifactpypi.PublicPyPIProfile().Source() {
			t.Fatal("native tool root differs")
		}
	}
	controls := selected.Controls()
	if len(controls) != 4 {
		t.Fatal("tool provenance controls are incomplete")
	}
	var receipt artifactpypi.ToolReceipt
	for _, control := range controls {
		if !control.Present() || len(control.Body()) == 0 {
			t.Fatal("tool control absent")
		}
		if control.Name() == "uv-tool-receipt.toml" {
			receipt, err = artifactpypi.ParseUVToolReceipt(control.Body())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if !receipt.Valid() || len(receipt.Commands()) != 1 || receipt.Commands()[0].Name != "cowsay" || receipt.Requirement().Requirement() != "cowsay==6.1" {
		t.Fatal("native command receipt differs")
	}
	t.Logf("uv_tool_native_graph selector=%s nodes=%d commands=%d controls=%d runtime=%s", requirement.Requirement(), len(resolution.Graph().Nodes()), len(receipt.Commands()), len(controls), resolution.RuntimeIdentity())
	noCommand, _ := artifactpypi.ParseUpgradeRequirement("idna==3.11")
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
	refused, err := resolver.ResolveToolInstall(ctx, noCommand)
	cancel()
	if err == nil || len(refused.Controls()) != 0 || len(refused.Resolution().Graph().Nodes()) != 0 {
		t.Fatal("package without a tool command retained a successful selection")
	}
	t.Log("uv_tool_no_console_package_refused_without_selection")
}
