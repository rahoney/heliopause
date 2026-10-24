//go:build linux && realcorpus

package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	verificationnpm "github.com/rahoney/heliopause/internal/verification/npm"
)

// The installed typed helper must authenticate this actual client executable.
// Eight selections bind snapshots. Four selected cases additionally exercise
// ordinary public archive quarantine; Host installation is never performed.
func TestPNPMPublicNativeResolverIntegration(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_PNPM_PUBLIC_RESOLVER_INTEGRATION") != "1" {
		t.Fatal("requires ordinary-user authenticated public PNPM qualification")
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
	resolver, err := NewLinuxPNPMResolverWithExecutorAndPolicy(executor, supervisor.Observer(), integrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	intake := t.TempDir()
	source, err := artifactnpm.NewPublicResolver(intake)
	if err != nil {
		t.Fatal(err)
	}
	static, err := inspectionnpm.NewInspector(intake)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := NewLinuxBackendWithExecutor(intake, executor, supervisor.Observer())
	if err != nil {
		t.Fatal(err)
	}
	dynamic, err := inspectionnpm.NewDynamicInspector(backend)
	if err != nil {
		t.Fatal(err)
	}
	inspector, err := inspectionnpm.NewCompositeInspector(static, dynamic)
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
	qualify := func(t *testing.T, name string, input artifactnpm.PNPMSelectionInput, artifacts, instances, roots int) PNPMNativeSelection {
		t.Helper()
		selected, err := resolver.SelectProject(ctx, input)
		if err != nil {
			t.Fatalf("actual native selection %s failed: %v", name, err)
		}
		target, err := domain.NewInstallTarget("/tmp/haa-pnpm-unapproved-project")
		if err != nil {
			t.Fatal(err)
		}
		install, err := domain.NewNPMProjectInstallContext(target)
		if err != nil {
			t.Fatal(err)
		}
		snapshot, err := selected.BindProjectSnapshot(ctx, source, install)
		if err != nil {
			t.Fatalf("actual public binding %s failed: %v", name, err)
		}
		if !snapshot.Valid() || len(snapshot.Dependencies()) != artifacts || len(selected.Selection().Snapshots()) != instances || len(selected.Selection().Roots()) != roots || len(snapshot.ControlDigests()) != 2 || snapshot.GraphDigest().String() == "" {
			t.Fatal("native instances/public source binding differ")
		}
		for _, control := range selected.Controls() {
			sum := sha256.Sum256(control.Body())
			t.Logf("pnpm_selected_control name=%s bytes=%d sha256=%s", control.Name(), len(control.Body()), hex.EncodeToString(sum[:]))
		}
		t.Logf("pnpm_public_native case=%s artifacts=%d instances=%d roots=%d source_bound=true snapshot_bound=true host_install=NOT_EXECUTED", name, artifacts, instances, roots)
		if name == "single" || name == "transitive" || name == "peer" || name == "selected-update" {
			approved, err := quarantine.InspectProject(ctx, snapshot)
			if err != nil {
				var denied *application.ProjectPolicyFailure
				if errors.As(err, &denied) {
					t.Logf("pnpm_project_policy case=%s decision=%s reasons=%v", name, denied.Decision().Decision(), denied.Decision().Reasons())
					for _, entry := range denied.Inspected().Inspections() {
						fact, known := entry.HostInstallActionEvidence()
						t.Logf("pnpm_entry_policy name=%s version=%s decision=%s reasons=%v action_known=%t action_kind=%s", entry.Artifact().Identity().Name(), entry.Artifact().Identity().Version(), entry.PolicyDecision().Decision(), entry.PolicyDecision().Reasons(), known, fact.Kind())
						for _, check := range entry.Checks() {
							limit, _ := check.LimitationCode()
							t.Logf("pnpm_entry_check name=%s check=%s required=%t capability=%s status=%s limitation=%s", entry.Artifact().Identity().Name(), check.ID().String(), check.Required(), check.Capability(), check.Status(), limit)
						}
					}
				}
			}
			if err != nil || !approved.Valid() || approved.Inspected().Snapshot().GraphDigest() != snapshot.GraphDigest() || len(approved.Inspected().Inspections()) != artifacts {
				t.Fatalf("actual PNPM snapshot quarantine failed case=%s: %v", name, err)
			}
			for _, entry := range approved.Inspected().Inspections() {
				fact, known := entry.HostInstallActionEvidence()
				if !known || fact.Kind() != domain.EvidenceHostInstallActionNotRequired || fact.Identity() != entry.Artifact().Identity() || fact.Digest() != entry.Artifact().Digest() || len(entry.Checks()) != 3 || len(entry.Evidence()) != 4 || entry.PolicyDecision().Decision() != domain.DecisionAllow {
					t.Fatal("actual archive quarantine facts were lost")
				}
				for _, check := range entry.Checks() {
					if !check.Required() || check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted {
						t.Fatal("actual required quarantine check incomplete")
					}
				}
			}
			t.Logf("pnpm_project_quarantine case=%s entries=%d checks_per_entry=3 evidence_per_entry=4 action_fact=NOT_REQUIRED policy=ALLOW host_install=NOT_EXECUTED", name, artifacts)
		}
		return selected
	}
	var original PNPMNativeSelection
	for _, item := range []struct {
		name, selector, manifest    string
		intent                      artifactnpm.PNPMSelectionIntent
		artifacts, instances, roots int
	}{
		{"single", "", `{"dependencies":{"is-number":"6.0.0"}}`, artifactnpm.PNPMInstall, 1, 1, 1},
		{"transitive", "", `{"dependencies":{"is-odd":"3.0.1"}}`, artifactnpm.PNPMInstall, 2, 2, 1},
		{"peer", "", `{"dependencies":{"react":"18.3.1","use-sync-external-store":"1.2.0"}}`, artifactnpm.PNPMInstall, 4, 4, 2},
		{"add", "is-number@6.0.0", `{}`, artifactnpm.PNPMAdd, 1, 1, 1},
	} {
		passed := t.Run(item.name, func(t *testing.T) {
			input := pnpmSelectionInputFixture(t, item.intent, item.selector, item.manifest)
			selected := qualify(t, item.name, input, item.artifacts, item.instances, item.roots)
			if item.name == "single" {
				original = selected
			}
		})
		if !passed {
			t.FailNow()
		}
	}
	if !t.Run("frozen-install", func(t *testing.T) {
		input, err := artifactnpm.NewPNPMSelectionInput(artifactnpm.PNPMInstall, "", original.Controls(), runtimeidentity.PNPMVersion)
		if err != nil {
			t.Fatal(err)
		}
		selected := qualify(t, "frozen-install", input, 1, 1, 1)
		for i, c := range selected.Controls() {
			if !bytes.Equal(c.Body(), original.Controls()[i].Body()) {
				t.Fatal("frozen controls changed")
			}
		}
	}) {
		t.FailNow()
	}
	// Explicit inert update fixture: widen the constraint around the actual v6
	// lock. It grants no enrollment or Host publication authority.
	manifest := pnpmTestControl(t, "package.json", `{"dependencies":{"is-number":">=6.0.0 <8.0.0"}}`, true)
	old := original.Controls()[1].Body()
	if bytes.Count(old, []byte("specifier: 6.0.0")) != 1 {
		t.Fatal("actual seed shape changed")
	}
	lockBody := bytes.Replace(old, []byte("specifier: 6.0.0"), []byte("specifier: '>=6.0.0 <8.0.0'"), 1)
	lock, err := domain.NewProjectControlFile("pnpm-lock.yaml", lockBody, true)
	if err != nil {
		t.Fatal(err)
	}
	seed := []domain.ProjectControlFile{manifest, lock}
	var updated PNPMNativeSelection
	for _, item := range []struct {
		name, selector string
		noop           bool
	}{
		{"selected-update", "is-number", false}, {"update-no-op", "is-number", true}, {"full-update", "", false},
	} {
		if !t.Run(item.name, func(t *testing.T) {
			controls := seed
			if item.noop {
				controls = updated.Controls()
			}
			input, err := artifactnpm.NewPNPMSelectionInput(artifactnpm.PNPMUpdate, item.selector, controls, runtimeidentity.PNPMVersion)
			if err != nil {
				t.Fatal(err)
			}
			selected := qualify(t, item.name, input, 1, 1, 1)
			roots := selected.Selection().Roots()
			packages := selected.Selection().Packages()
			if roots[0].Specifier != ">=6.0.0 <8.0.0" || packages[0].Name != "is-number" || packages[0].Version != "7.0.0" {
				t.Fatal("native update did not preserve the supported range/select v7")
			}
			if item.noop {
				for i, c := range selected.Controls() {
					if !bytes.Equal(c.Body(), controls[i].Body()) {
						t.Fatal("native no-op changed controls")
					}
				}
			}
			if item.name == "selected-update" {
				updated = selected
			}
		}) {
			t.FailNow()
		}
	}
}
