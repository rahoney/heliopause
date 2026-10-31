//go:build linux && realcorpus

package sandbox

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
)

func TestBunPublicInspectionPreparationIntegration(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_BUN_INSPECTION_PREPARATION") != "1" {
		t.Fatal("declare ordinary-user Bun preparation qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
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
	for _, name := range []string{"single", "transitive", "peer", "multiple-versions"} {
		if !t.Run(name, func(t *testing.T) {
			var input artifactnpm.BunInspectionInput
			var inventories []artifactnpm.ArchiveInventory
			var intake string
			if name == "multiple-versions" {
				intake = t.TempDir()
				source, err := artifactnpm.NewPublicResolver(intake)
				if err != nil {
					t.Fatal(err)
				}
				selected, err := native.SelectProject(ctx, bunResolverInput(t, artifactnpm.BunInstall, "", `{"name":"haa-bun-multiple","private":true,"scripts":{"install":"exit 99"},"dependencies":{"is-number":"7.0.0","is-odd":"3.0.1"}}`))
				if err != nil {
					t.Fatal("actual multi-version native selection", err)
				}
				target, err := domain.NewInstallTarget("/tmp/haa-bun-multiple-inspection")
				if err != nil {
					t.Fatal(err)
				}
				install, err := domain.NewNPMProjectInstallContext(target)
				if err != nil {
					t.Fatal(err)
				}
				frozen, err := selected.BindInspectionContext(ctx, source, install)
				if err != nil {
					t.Fatal(err)
				}
				input, inventories = bunAcquireInspectionInputs(t, ctx, intake, source, frozen)
				versions := map[string]bool{}
				for _, a := range input.Artifacts() {
					if a.Identity().Name() == "is-number" {
						versions[a.Identity().Version()] = true
					}
				}
				if !versions["6.0.0"] || !versions["7.0.0"] || len(input.Artifacts()) != 3 {
					t.Fatal("native multi-version context missing")
				}
			} else {
				intake, input, inventories = bunPublicInspectionInputsWithRoot(t, ctx, name)
			}
			preparer, err := NewBunInspectionPreparer(native, intake)
			if err != nil {
				t.Fatal(err)
			}
			layout, err := preparer.PrepareInspection(ctx, input, inventories)
			if err != nil || !layout.MatchesInput(input) {
				t.Fatal("production Bun preparation", err)
			}
			instances := 0
			for _, artifact := range input.Artifacts() {
				placements, err := input.InstancesFor(artifact)
				if err != nil {
					t.Fatal(err)
				}
				for _, placement := range placements {
					if _, err := layout.Target(artifact, placement.Key); err != nil {
						t.Fatal(err)
					}
					instances++
				}
			}
			t.Logf("bun_preparation=%s archives=%d instances=%d entries=%d exact_controls=true native_offline=true; Artifact_execution/Policy/Host_install=NOT_EXERCISED", name, len(input.Artifacts()), instances, len(layout.Entries()))
		}) {
			return
		}
	}
}

func TestBunActualPreparationBoundary(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_BUN_INSPECTION_PREPARATION") != "1" {
		t.Fatal("declare ordinary-user Bun preparation boundary qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
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
	intake, input, inventories := bunPublicInspectionInputsWithRoot(t, ctx, "single")
	for _, mode := range []string{"complete", "command and cleanup", "cleanup acknowledgement", "incomplete observation", "forbidden observation"} {
		if !t.Run(mode, func(t *testing.T) {
			cause := errors.New("injected bun preparation failure")
			injections := 0
			runner := &pnpmNativeFaultRunner{CommandRunner: admissionAwareRunner(executor), mode: mode, nativeTool: bunToolGuestPath, cause: cause}
			native, err := NewBunResolver(runner, &pnpmCountingEndpoints{}, pnpmNativeFaultObserver{supervisor.Observer(), mode, &injections}, integrationCapabilityProbe(executor), &recordingResolverPolicyService{})
			if err != nil {
				t.Fatal(err)
			}
			preparer, err := NewBunInspectionPreparer(native, intake)
			if err != nil {
				t.Fatal(err)
			}
			layout, err := preparer.PrepareInspection(ctx, input, inventories)
			if mode == "complete" {
				if err != nil || !layout.MatchesInput(input) {
					t.Fatal("positive preparation boundary", err)
				}
				return
			}
			if err == nil || layout.Digest().String() != "" {
				t.Fatal("failed preparation retained output")
			}
			if (mode == "command and cleanup" || mode == "cleanup acknowledgement") && !errors.Is(err, cause) {
				t.Fatal("original cause lost", err)
			}
			wantCommand, wantCleanup, wantTrace := 0, 0, 0
			if mode == "command and cleanup" {
				wantCommand = 1
				wantCleanup = 1
			}
			if mode == "cleanup acknowledgement" {
				wantCleanup = 1
			}
			if mode == "incomplete observation" || mode == "forbidden observation" {
				wantTrace = 1
			}
			if runner.commandFaults != wantCommand || runner.cleanupFaults != wantCleanup || injections != wantTrace {
				t.Fatal("intended failure boundary not reached", runner.commandFaults, runner.cleanupFaults, injections)
			}
		}) {
			return
		}
	}
}
