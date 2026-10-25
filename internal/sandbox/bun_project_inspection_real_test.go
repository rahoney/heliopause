//go:build linux && realcorpus

package sandbox

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/hosttool"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/policy"
	verificationnpm "github.com/rahoney/heliopause/internal/verification/npm"
)

func TestBunPublicProjectQuarantineIntegration(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_BUN_PROJECT_QUARANTINE") != "1" {
		t.Fatal("declare ordinary-user complete Bun quarantine qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
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
	for _, name := range []string{"transitive", "peer"} {
		if !t.Run(name, func(t *testing.T) {
			intake, initial, _ := bunPublicInspectionInputsWithRoot(t, ctx, name)
			frozen := initial.Context()
			source, err := artifactnpm.NewPublicResolver(intake)
			if err != nil {
				t.Fatal(err)
			}
			static, err := inspectionnpm.NewInspector(intake)
			if err != nil {
				t.Fatal(err)
			}
			prepare, err := NewBunInspectionPreparer(native, intake)
			if err != nil {
				t.Fatal(err)
			}
			backend, err := NewLinuxBackendWithExecutor(intake, executor, supervisor.Observer())
			if err != nil {
				t.Fatal(err)
			}
			inspector, err := inspectionnpm.NewBunProjectInspector(frozen, static, prepare, backend)
			if err != nil {
				t.Fatal(err)
			}
			store, err := local.NewStore(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			quarantine, err := application.NewProjectInspectService(source, verificationnpm.IntegrityVerifier{}, inspector, store, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			snapshot := frozen.Snapshot()
			approved, err := quarantine.InspectProject(ctx, snapshot)
			if err != nil || !approved.Valid() || approved.Inspected().Snapshot().GraphDigest() != snapshot.GraphDigest() || len(approved.Inspected().Inspections()) != len(initial.Artifacts()) {
				t.Fatal("complete Bun project quarantine", err)
			}
			acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
			for _, entry := range approved.Inspected().Inspections() {
				node, err := domain.ProjectDependencyNodeID(entry.Artifact().Identity())
				if err != nil {
					t.Fatal(err)
				}
				acquired[node] = entry.Artifact()
			}
			input, err := frozen.BindAcquired(snapshot, acquired)
			if err != nil {
				t.Fatal(err)
			}
			prepared, err := inspector.PreparedLayoutFor(snapshot, acquired)
			if err != nil || !prepared.MatchesInput(input) {
				t.Fatal("completed exact request handoff", err)
			}
			count := 0
			for _, entry := range approved.Inspected().Inspections() {
				instances, err := input.InstancesFor(entry.Artifact())
				if err != nil {
					t.Fatal(err)
				}
				count += len(instances)
				action, known := entry.HostInstallActionEvidence()
				if !known || action.Kind() != domain.EvidenceHostInstallActionNotRequired || action.Identity() != entry.Artifact().Identity() || action.Digest() != entry.Artifact().Digest() || entry.PolicyDecision().Decision() != domain.DecisionAllow || len(entry.Checks()) != 3+len(instances) || len(entry.Evidence()) != 4+2*len(instances) {
					t.Fatal("archive facts/recorded evidence/Policy lost")
				}
				required := 0
				for _, check := range entry.Checks() {
					if check.Required() {
						required++
					}
					if check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted {
						t.Fatal("incomplete quarantine check")
					}
				}
				if required != 3 {
					t.Fatal("required integrity/static/lifecycle coverage lost")
				}
			}
			for node := range acquired {
				delete(acquired, node)
				break
			}
			if _, err := inspector.PreparedLayoutFor(snapshot, acquired); err == nil {
				t.Fatal("partial request reused completed layout")
			}
			t.Logf("bun_project_quarantine fixture=%s archives=%d exact_instances=%d required_per_archive=3 facts=EXACT evidence=RECORDED entry_set_policy=ALLOW primary=NONE host_action=NOT_REQUIRED pinned_Node_Bun_runtime_supply=true native_Bun_JS=SEPARATE_B6b_FIXTURE Host_install=NOT_EXERCISED", name, len(initial.Artifacts()), count)
		}) {
			return
		}
	}
}
