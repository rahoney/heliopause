//go:build linux

package sandbox

import (
	"context"
	"fmt"
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

func TestUVPublicToolUpgradeResolverIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_TOOL_UPGRADE_RESOLVER_INTEGRATION") != "1" {
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
	// These old metadata controls are synthetic DATA, never an authentic native
	// adoption or Host ownership proof. All selected public wheels are bound anew.
	for _, c := range []struct{ name, original, positional, old string }{
		{"bounded-update", ">=6,<7", "cowsay==6.1", "6.0"},
		{"retained-exact-noop", "==6.1", "cowsay", "6.1"},
	} {
		t.Run(c.name, func(t *testing.T) {
			body := []byte(fmt.Sprintf(`[tool]
requirements = [{ name = "cowsay", specifier = %q }]
python = "/usr/local/bin/python"
entrypoints = [{ name = "cowsay", install-path = "/user/commands/cowsay", from = "cowsay" }]
[tool.options]
index-url = "https://pypi.org/simple/"
link-mode = "copy"
no-sources = true
no-build = true
`, c.original))
			receipt, err := artifactpypi.ParseUVToolReceipt(body)
			if err != nil {
				t.Fatal(err)
			}
			request, err := artifactpypi.ParseUpgradeRequirement(c.positional)
			if err != nil {
				t.Fatal(err)
			}
			directory := "cowsay-" + c.old + ".dist-info"
			metadata := []byte("Metadata-Version: 2.4\nName: cowsay\nVersion: " + c.old + "\n")
			input, err := artifactpypi.NewUVToolUpgradeInput(receipt, request, []artifactpypi.InstalledMetadataFile{{Directory: directory, Body: metadata}}, PinnedPythonRuntime().PythonVersion)
			if err != nil {
				t.Fatal(err)
			}
			wheels := []artifactpypi.UVInstalledWheelControls{{DistInfo: directory, Metadata: metadata, Wheel: []byte("Wheel-Version: 1.0\nTag: py3-none-any\n")}}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			selected, err := resolver.ResolveToolUpgrade(ctx, input, wheels)
			cancel()
			if err != nil {
				t.Fatal("native metadata-only upgrade/source binding failed", err)
			}
			resolution := selected.Resolution()
			if len(resolution.Graph().Nodes()) != 1 || !strings.HasPrefix(resolution.RuntimeIdentity(), "uv-tool-upgrade:uv:") || resolution.LockfileDigest().String() == "" || len(selected.Controls()) != 6 {
				t.Fatal("upgrade graph/provenance differs")
			}
			for _, n := range resolution.Graph().Nodes() {
				if n.Artifact().Identity().Name() != "cowsay" || n.Artifact().Identity().Version() != "6.1" {
					t.Fatal("wrong selected main version")
				}
			}
			original, _ := input.Control()
			boundOriginal := false
			for _, control := range selected.Controls() {
				if control.Name() == original.Name() {
					boundOriginal = control.Digest() == original.Digest()
				}
				if control.Name() == "uv-tool-receipt.toml" {
					final, err := artifactpypi.ParseUVToolReceipt(control.Body())
					if err != nil || !final.Requirement().SameConstraints(receipt.Requirement()) {
						t.Fatal("native original constraint changed", err)
					}
				}
			}
			if !boundOriginal {
				t.Fatal("original input not bound")
			}
			t.Logf("uv_tool_upgrade_data_selection case=%s original=%s positional=%s selected=6.1 controls=6", c.name, receipt.Requirement().Requirement(), request.Requirement())
		})
	}
}
