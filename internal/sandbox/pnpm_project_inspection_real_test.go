//go:build linux && realcorpus

package sandbox

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
)

func TestPNPMActualProductionProjectInspection(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_PNPM_NATIVE_RUNTIME") != "1" || os.Getenv("HELOX_PNPM_PUBLIC_METADATA_FIXTURE") != "1" || os.Getenv("HELOX_PNPM_INSTANCE_EXECUTION_FIXTURE") != "1" {
		t.Fatal("declare ordinary-user PNPM public project qualification")
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
			static, err := inspectionnpm.NewInspector(data.intake)
			if err != nil {
				t.Fatal(err)
			}
			prepare, err := NewPNPMInspectionPreparer(native, data.intake)
			if err != nil {
				t.Fatal(err)
			}
			introducer, err := NewDockerArtifactIntroducer(data.intake, executor)
			if err != nil {
				t.Fatal(err)
			}
			backend, err := NewBackend(executor, introducer, observer, integrationCapabilityProbe(executor))
			if err != nil {
				t.Fatal(err)
			}
			inspector, err := inspectionnpm.NewPNPMProjectInspector(data.inspection.Context(), static, prepare, backend)
			if err != nil {
				t.Fatal(err)
			}
			acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
			for _, subject := range data.inspection.Artifacts() {
				node, err := domain.ProjectDependencyNodeID(subject.Identity())
				if err != nil {
					t.Fatal(err)
				}
				acquired[node] = subject
			}
			before := len(observer.traces)
			reports, err := inspector.InspectProject(ctx, data.inspection.Context().Snapshot(), acquired)
			if err != nil || len(reports) != len(acquired) {
				t.Fatal("production project inspection failed", err)
			}
			expected := 0
			observed := 0
			for node, subject := range acquired {
				instances, err := data.inspection.InstancesFor(subject)
				if err != nil {
					t.Fatal(err)
				}
				expected += len(instances)
				report := reports[node]
				for _, check := range report.Executions() {
					if check.Status() != domain.ExecutionCompleted {
						t.Fatal("public project incomplete", check.ID(), check.Status())
					}
				}
				if len(report.Findings()) != 0 {
					t.Fatal("unexpected public project finding", report.Findings())
				}
				bindings := 0
				for _, evidence := range report.Evidence() {
					if evidence.Identity() != subject.Identity() || evidence.Digest() != subject.Digest() {
						t.Fatal("foreign evidence subject")
					}
					if evidence.Kind() != "pnpm-instance-binding" {
						continue
					}
					var binding struct{ Instance, Session, Layout, Runtime string }
					if json.Unmarshal([]byte(evidence.Summary()), &binding) != nil || len(binding.Instance) != 64 || binding.Session == "" || len(binding.Layout) != 64 || len(binding.Runtime) != 64 || sessions[binding.Session] {
						t.Fatal("missing/reused actual instance binding")
					}
					sessions[binding.Session] = true
					bindings++
					observed++
				}
				if bindings != len(instances) {
					t.Fatal("peer instance coverage collapsed")
				}
			}
			if observed != expected || len(observer.traces) != before+expected+1 {
				t.Fatal("prepare/instance actual trace coverage mismatch", expected, len(observer.traces)-before)
			}
			for _, trace := range observer.traces[before:] {
				if !trace.complete {
					t.Fatal("incomplete actual trace")
				}
				for _, kind := range []string{"process-exec-unexpected", "filesystem-outside-workspace", "honeytoken-access", "network-attempt"} {
					if trace.kinds[kind] != 0 {
						t.Fatal("unexpected actual observation", kind)
					}
				}
			}
			t.Logf("production_pnpm_project fixture=%s archives=%d exact_instances=%d actual_fresh_sessions=%d static_all=true source_subjects_bound=true required_dynamic=COMPLETED Policy=NOT_RUN Host_install=NOT_RUN", fixture.name, len(acquired), expected, observed)
		}) {
			t.FailNow()
		}
	}
}
