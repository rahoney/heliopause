//go:build linux && realcorpus

package sandbox

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
)

type pnpmInstanceActualObserver struct{ pnpmPrepareTraceObserver }

func (o *pnpmInstanceActualObserver) StartPNPMTargetLifecycleProfile(ctx context.Context, id string) (TraceReader, error) {
	observer, ok := o.TraceObserver.(pnpmTargetTraceObserver)
	if !ok {
		return nil, errors.New("fixed target profile unavailable")
	}
	reader, err := observer.StartPNPMTargetLifecycleProfile(ctx, id)
	if err != nil {
		return nil, err
	}
	wrapped := &pnpmPrepareTrace{TraceReader: reader, kinds: map[string]uint64{}}
	o.traces = append(o.traces, wrapped)
	return wrapped, nil
}

func TestPNPMActualProductionInstanceExecution(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_PNPM_NATIVE_RUNTIME") != "1" || os.Getenv("HELOX_PNPM_PUBLIC_METADATA_FIXTURE") != "1" || os.Getenv("HELOX_PNPM_INSTANCE_EXECUTION_FIXTURE") != "1" {
		t.Fatal("declare ordinary-user PNPM native/public and instance qualification")
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
	observer := &pnpmInstanceActualObserver{pnpmPrepareTraceObserver: pnpmPrepareTraceObserver{TraceObserver: supervisor.Observer()}}
	native, err := NewLinuxPNPMResolverWithExecutorAndPolicy(executor, observer, integrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	sessions := map[string]bool{}
	for _, fixture := range pnpmOfflineFixtures() {
		if !t.Run(fixture.name, func(t *testing.T) {
			data := loadPNPMOfflineInputs(t, ctx, fixture)
			preparer, err := NewPNPMInspectionPreparer(native, data.intake)
			if err != nil {
				t.Fatal(err)
			}
			bound, err := preparer.PrepareInspection(ctx, data.inspection, data.inventories)
			if err != nil {
				t.Fatal("production prepare", err)
			}
			introducer, err := NewDockerArtifactIntroducer(data.intake, executor)
			if err != nil {
				t.Fatal(err)
			}
			backend, err := NewBackend(executor, introducer, observer, integrationCapabilityProbe(executor))
			if err != nil {
				t.Fatal(err)
			}
			count := 0
			for _, subject := range data.inspection.Artifacts() {
				instances, err := data.inspection.InstancesFor(subject)
				if err != nil {
					t.Fatal(err)
				}
				for _, instance := range instances {
					before := len(observer.traces)
					result, err := backend.ExecutePNPMInstance(ctx, bound, subject, instance.Key)
					if err != nil || result.Status() != domain.SandboxCompleted {
						t.Fatal("fresh production instance", result.Status(), err)
					}
					id := result.SessionID().String()
					if id == "" || sessions[id] {
						t.Fatal("session identity reused")
					}
					sessions[id] = true
					if len(observer.traces) != before+1 || !observer.traces[before].complete {
						t.Fatal("actual instance has no independent complete trace")
					}
					for _, kind := range []string{"process-exec-unexpected", "filesystem-outside-workspace", "honeytoken-access", "network-attempt"} {
						if observer.traces[before].kinds[kind] != 0 {
							t.Fatal("unexpected public instance observation", kind)
						}
					}
					count++
				}
			}
			t.Logf("production_pnpm_instance fixture=%s archives=%d instances=%d complete=true fresh_sessions=true minimal_payload_readback=true network=none Policy=NOT_RUN Host_install=NOT_RUN", fixture.name, len(data.inventories), count)
		}) {
			t.FailNow()
		}
	}
	// These are explicitly controlled structural inputs. Public authenticity is
	// covered only by the producer/SRI cases above, never these fixture constructors.
	for _, item := range []struct {
		name, packageName string
		failed            bool
	}{
		{"controlled-success", "is-number", false},
		{"controlled-exit37", "is-number", true},
		{"controlled-long-name", strings.Repeat("a", 130), false},
	} {
		if !t.Run(item.name, func(t *testing.T) {
			failed := item.failed
			install := "true"
			if failed {
				install = "exit 37"
			}
			bound, input := pnpmBoundPayloadNamedScriptInputs(t, true, item.packageName, map[string]string{"preinstall": "true", "install": install, "postinstall": "true"})
			backend, err := NewBackend(executor, &recordingIntroducer{}, observer, integrationCapabilityProbe(executor))
			if err != nil {
				t.Fatal(err)
			}
			before := len(observer.traces)
			result, err := backend.ExecutePNPMInstance(ctx, bound, input.Artifacts()[0], input.Context().Lock().Snapshots()[0].Key)
			if failed {
				var exited *exec.ExitError
				if err == nil || !errors.As(err, &exited) || exited.ExitCode() != 37 || result.Status() != domain.SandboxIncomplete {
					t.Fatal("first script exit lost", err, result.Status())
				}
			} else if err != nil || result.Status() != domain.SandboxCompleted {
				t.Fatal("controlled execution failed", err)
			}
			if len(observer.traces) != before+1 || !observer.traces[before].complete {
				t.Fatal("controlled trace incomplete")
			}
			trace := observer.traces[before]
			want := uint64(3)
			if failed {
				want = 2
			}
			if trace.kinds["process-exec-unexpected"] != want {
				t.Fatal("script shells lost artifact attribution", trace.kinds)
			}
			for _, kind := range []string{"filesystem-outside-workspace", "honeytoken-access", "network-attempt"} {
				if trace.kinds[kind] != 0 {
					t.Fatal("controlled unexpected access", kind)
				}
			}
			if failed {
				found := false
				for _, fact := range result.Observations() {
					if fact.Category() == domain.ObservationProcess && fact.Subject() == "process-exec-unexpected" {
						found = true
					}
				}
				if !found {
					t.Fatal("failed script discarded observed process facts")
				}
			}
			t.Logf("production_pnpm_controlled failed=%t original_exit_preserved=true complete_trace=true artifact_script_shells=%d public_authenticity=NOT_CLAIMED Policy=NOT_RUN Host_install=NOT_RUN", failed, want)
		}) {
			t.FailNow()
		}
	}
}
