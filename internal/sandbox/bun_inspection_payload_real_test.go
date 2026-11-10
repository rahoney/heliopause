//go:build linux && realcorpus

package sandbox

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/hosttool"
)

func TestBunInspectionPayloadRoundTripIntegration(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_BUN_INSPECTION_PAYLOAD") != "1" {
		t.Fatal("declare ordinary-user Bun payload qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
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
	preparer, err := NewBunInspectionPreparer(native, intake)
	if err != nil {
		t.Fatal(err)
	}
	layout, err := preparer.PrepareInspection(ctx, input, inventories)
	if err != nil || !layout.MatchesInput(input) {
		t.Fatal("production preparation", err)
	}
	payload, err := prepareBunInspectionPayload(ctx, layout)
	if err != nil {
		t.Fatal(err)
	}
	err = native.scope.run(ctx, false, func(ctx context.Context, container, network string, _ *closureVolume) error {
		if err := disconnectPNPMInspectionNetwork(ctx, native.scope.runner, container, network); err != nil {
			return err
		}
		return copyBunInspectionPayload(ctx, native.scope.runner, container, payload)
	})
	if err != nil {
		t.Fatal("fresh whole execution-data readback", err)
	}
	t.Logf("bun_execution_payload archives=%d layout_entries=%d payload_entries=%d bytes=%d full_readback=true fresh_session=true scripts/Artifact_execution/Policy/Host_install=NOT_EXERCISED", len(inventories), len(layout.Entries()), len(payload.entries), len(payload.body))
}
