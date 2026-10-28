//go:build linux && realcorpus

package sandbox

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/hosttool"
)

func TestLinuxPipxPublicInstallResolverIntegration(t *testing.T) {
	if os.Getenv("HELOX_PIPX_PUBLIC_RESOLVER_INTEGRATION") != "1" {
		t.Skip("requires explicit installed supply and observed public resolver namespace")
	}
	if os.Geteuid() == 0 {
		t.Fatal("ordinary caller required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Minute)
	defer cancel()
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	client, err := hosttool.NewSystemObservationResourceClient()
	if err != nil {
		t.Fatal(err)
	}
	resources := &pipxActualResources{ObservationResourceClient: integrationObservationResources{client: client}}
	resolver, err := NewLinuxPipxResolverWithExecutorAndPolicy(executor, supervisor.Observer(), integrationResolverPolicyService(t), resources)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Error(err)
		}
	}()
	requirement, _ := artifactpypi.ParseUpgradeRequirement("cowsay==6.1")
	selection, err := resolver.ResolveToolInstall(ctx, requirement)
	if err != nil {
		t.Fatal("public pipx selection/source/native binding failed", err)
	}
	resolution := selection.Resolution()
	if len(resolution.Graph().Nodes()) != 1 || len(selection.Controls()) != 4 || resolution.RuntimeIdentity() != "pipx-tool-install:"+PipxResolverRuntimeIdentity() || resolution.LockfileDigest().String() == "" {
		t.Fatal("public native graph/provenance differs")
	}
	var receipt artifactpypi.ToolReceipt
	for _, control := range selection.Controls() {
		t.Logf("pipx_public_control name=%s digest=%s", control.Name(), control.Digest().String())
		if control.Name() == "pipx-tool-receipt.json" {
			receipt, err = artifactpypi.ParsePipxToolReceipt(control.Body())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if receipt.Client() != "pipx" || receipt.Version() != "6.1" || len(receipt.Commands()) != 1 || receipt.Commands()[0].Name != "cowsay" || !receipt.Requirement().SameConstraints(requirement) {
		t.Fatal("native command differs")
	}
	for _, node := range resolution.Graph().Nodes() {
		id := node.Artifact().Identity()
		t.Logf("pipx_public_original name=%s version=%s locator=%q integrity=%q", id.Name(), id.Version(), node.Artifact().AcquisitionLocator(), node.Artifact().DeclaredIntegrity())
		if id.Name() != "cowsay" || id.Version() != "6.1" || id.Source() != artifactpypi.PublicPyPIProfile().Source() {
			t.Fatal("public source/root differs")
		}
	}
	t.Logf("public_native_pipx selector=%s graph=%d commands=%d controls=%d lock=%s runtime=%s", requirement.Requirement(), len(resolution.Graph().Nodes()), len(receipt.Commands()), len(selection.Controls()), resolution.LockfileDigest().String(), resolution.RuntimeIdentity())
	noCommand, _ := artifactpypi.ParseUpgradeRequirement("idna==3.11")
	refused, err := resolver.ResolveToolInstall(ctx, noCommand)
	if err == nil || len(refused.Controls()) != 0 || len(refused.Resolution().Graph().Nodes()) != 0 {
		t.Fatal("package without console command retained selection")
	}
	resources.mu.Lock()
	creates, registers, closes, cpu := resources.creates, resources.registers, resources.closes, resources.cpu
	resources.mu.Unlock()
	if creates != 2 || closes != 2 || registers != 2 || cpu == 0 || integrationObserverFaultReason(supervisor) != "" {
		t.Fatal("native resource accounting not reconciled")
	}
	t.Logf("public_native_pipx_no_command_refused error=%v creates=%d registers=%d closes=%d last_cpu_usec=%d reason=%s", err, creates, registers, closes, cpu, integrationObserverFaultReason(supervisor))
	t.Log("public live selection and independent original archive binding only; no inspection/Policy, target publication, original upgrade or CLI qualification")
	if !strings.Contains(resolution.RuntimeIdentity(), "backend:uv:") {
		t.Fatal("fixed backend absent")
	}
}
