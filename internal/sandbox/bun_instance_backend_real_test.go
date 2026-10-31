//go:build linux && realcorpus

package sandbox

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
)

func TestBunPublicInstanceBackendIntegration(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_BUN_INSTANCE_BACKEND") != "1" {
		t.Fatal("declare ordinary-user Bun bound-instance qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	native, err := NewLinuxBunResolverWithExecutorAndPolicy(executor, supervisor.Observer(), newIntegrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	intake, input, inventories := bunPublicInspectionInputsWithRoot(t, ctx, "peer")
	prepare, err := NewBunInspectionPreparer(native, intake)
	if err != nil {
		t.Fatal(err)
	}
	layout, err := prepare.PrepareInspection(ctx, input, inventories)
	if err != nil || !layout.MatchesInput(input) {
		t.Fatal("production source/layout preparation", err)
	}
	introducer, err := NewDockerArtifactIntroducer(intake, executor)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := NewBackend(executor, introducer, supervisor.Observer(), integrationCapabilityProbe(executor))
	if err != nil {
		t.Fatal(err)
	}
	sessions := map[string]bool{}
	for _, subject := range input.Artifacts() {
		instances, err := input.InstancesFor(subject)
		if err != nil {
			t.Fatal(err)
		}
		for _, instance := range instances {
			result, err := backend.ExecuteBunInstance(ctx, layout, subject, instance.Key)
			if err != nil || result.Status() != domain.SandboxCompleted {
				t.Fatal("production bound Bun instance", err, result.Status())
			}
			session := result.SessionID().String()
			if session == "" || sessions[session] {
				t.Fatal("instance session reused")
			}
			sessions[session] = true
			completed := 0
			for _, fact := range result.Observations() {
				switch fact.Subject() {
				case "lifecycle-completed":
					completed++
				case "filesystem-outside-workspace", "honeytoken-access", "network-attempt", "process-exec-unexpected":
					t.Fatal("script-free public source gained suspicious observation", fact.Subject(), fact.Count())
				}
			}
			if completed != 1 {
				t.Fatal("exact instance completion missing")
			}
		}
	}
	if len(sessions) != 4 {
		t.Fatal("whole peer instance set missing")
	}
	t.Logf("bun_production_instance archives=%d exact_instances=%d fresh_sessions=%d whole_payload_readback=true trace/cleanup_before_return=true npm_Node_lane=true explicit_Bun_runtime/Policy/Host_install=NOT_EXERCISED", len(inventories), len(sessions), len(sessions))
}
