//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/hosttool"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationnpm "github.com/rahoney/heliopause/internal/verification/npm"
)

// This exercises complete dependency quarantine and owned temporary-target
// publication. Project-root lifecycle and public CLI have separate acceptance.
func TestNPMPublicWholeProjectQuarantineIntegration(t *testing.T) {
	if os.Getenv("HELOX_NPM_PROJECT_QUARANTINE_INTEGRATION") != "1" {
		t.Skip("requires declared protected npm whole-project quarantine")
	}
	if os.Geteuid() == 0 {
		t.Fatal("requires ordinary-user client and protected infrastructure")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	executor, supervisor, native := npmProjectIntegrationRuntime(t, ctx)
	for _, tc := range []struct {
		name, dependencies string
		placements         int
	}{
		{"nested", `"is-number":"7.0.0","is-odd":"3.0.1"`, 3},
		{"bin", `"semver":"7.7.2"`, 1},
		{"peer", `"react":"19.2.0","react-dom":"19.2.0"`, 3},
	} {
		if !t.Run(tc.name, func(t *testing.T) {
			started := time.Now()
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			intake := filepath.Join(root, "intake")
			source, err := artifactnpm.NewPublicResolver(intake)
			if err != nil {
				t.Fatal(err)
			}
			manifest, err := domain.NewProjectControlFile("package.json", []byte(`{"name":"haa-npm-whole-quarantine","private":true,"dependencies":{`+tc.dependencies+`}}`), true)
			if err != nil {
				t.Fatal(err)
			}
			lock, err := domain.NewProjectControlFile("package-lock.json", nil, false)
			if err != nil {
				t.Fatal(err)
			}
			input, err := artifactnpm.NewNPMProjectInput(artifactnpm.NPMProjectInstall, []domain.ProjectControlFile{manifest, lock}, runtimeidentity.NodeNPMVersion)
			if err != nil {
				t.Fatal(err)
			}
			selected, err := native.SelectProject(ctx, input)
			if err != nil {
				t.Fatal("native complete selection", err)
			}
			bound, err := source.BindNPMProjectLock(ctx, selected.Selection())
			if err != nil {
				t.Fatal("independent source binding", err)
			}
			target, err := domain.NewInstallTarget(filepath.Join(root, "project"))
			if err != nil {
				t.Fatal(err)
			}
			install, err := domain.NewNPMProjectInstallContext(target)
			if err != nil {
				t.Fatal(err)
			}
			frozen, err := bound.InspectionContext(install, selected.Controls(), sandbox.NPMProjectRuntimeIdentity())
			if err != nil {
				t.Fatal(err)
			}
			static, err := inspectionnpm.NewInspector(intake)
			if err != nil {
				t.Fatal(err)
			}
			backend, err := sandbox.NewLinuxBackendWithExecutor(intake, executor, supervisor.Observer(), newObservationResourceAdapter())
			if err != nil {
				t.Fatal(err)
			}
			inspector, err := inspectionnpm.NewNPMProjectInspector(frozen, static, backend, backend)
			if err != nil {
				t.Fatal(err)
			}
			store, err := local.NewStore(filepath.Join(root, "evidence"))
			if err != nil {
				t.Fatal(err)
			}
			operation, err := domain.NewOperationID()
			if err != nil {
				t.Fatal(err)
			}
			quarantine, err := application.NewProjectInspectService(source, verificationnpm.IntegrityVerifier{}, inspector, store, policy.M3{}, policy.M4{}, func() (domain.OperationID, error) { return operation, nil }, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			snapshot := frozen.Snapshot()
			approved, err := quarantine.InspectProject(ctx, snapshot)
			if err != nil || !approved.Valid() || approved.Inspected().Snapshot().GraphDigest() != snapshot.GraphDigest() {
				t.Fatal("complete npm dependency quarantine", err)
			}
			acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
			count := 0
			for _, entry := range approved.Inspected().Inspections() {
				node, err := domain.ProjectDependencyNodeID(entry.Artifact().Identity())
				if err != nil {
					t.Fatal(err)
				}
				acquired[node] = entry.Artifact()
				action, known := entry.HostInstallActionEvidence()
				if !known || action.Kind() != domain.EvidenceHostInstallActionNotRequired || entry.PolicyDecision().Decision() != domain.DecisionAllow {
					t.Fatal("npm Policy/action evidence missing")
				}
				required := 0
				for _, check := range entry.Checks() {
					if check.Required() {
						required++
					}
					if check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted {
						t.Fatal("incomplete npm quarantine check")
					}
				}
				if required != 3 {
					t.Fatal("required integrity/static/lifecycle coverage lost")
				}
			}
			acquiredInput, err := frozen.BindAcquired(snapshot, acquired)
			if err != nil {
				t.Fatal(err)
			}
			layout, err := inspector.PreparedLayoutFor(snapshot, acquired)
			if err != nil || !layout.MatchesInput(acquiredInput) {
				t.Fatal("completed exact npm layout unavailable", err)
			}
			for _, subject := range acquiredInput.Artifacts() {
				instances, err := acquiredInput.InstancesFor(subject)
				if err != nil {
					t.Fatal(err)
				}
				for _, instance := range instances {
					if _, err := layout.Target(subject, instance.Location); err != nil {
						t.Fatal(err)
					}
					count++
				}
			}
			if count != tc.placements {
				t.Fatalf("whole npm placement count: got %d want %d", count, tc.placements)
			}
			cache, err := promotion.NewNPMProjectVerifiedCache(intake, filepath.Join(root, "evidence"), filepath.Join(root, "cache"), frozen, store, inspector, evidence.Generator{}, operation)
			if err != nil {
				t.Fatal(err)
			}
			staged, err := cache.StageProject(ctx, approved)
			if err != nil || !staged.Valid() {
				t.Fatal("whole npm approved staging", err)
			}
			data, err := cache.OpenProjectCache(ctx, staged)
			if err != nil {
				t.Fatal("whole npm approved cache replay", err)
			}
			if err := os.Mkdir(target.String(), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(target.String(), "package.json"), manifest.Body(), 0o600); err != nil {
				t.Fatal(err)
			}
			owner, err := promotion.NewNPMProjectPromotion(intake, filepath.Join(root, "evidence"), filepath.Join(root, "cache"), filepath.Join(root, "state"), store, operation, func(snapshot domain.ProjectDependencySnapshot) (*promotion.NPMProjectVerifiedCache, error) {
				return cache, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			func() {
				guard, err := owner.BeginUpgrade(ctx, install, true)
				if err != nil {
					t.Fatal("npm original guard", err)
				}
				defer func() {
					if err := guard.Close(); err != nil {
						t.Error(err)
					}
				}()
				update, err := domain.NewProjectDependencyUpdateFromSnapshot(guard.Controls(), selected.Controls(), snapshot)
				if err != nil {
					t.Fatal(err)
				}
				if err := guard.Commit(ctx, update, staged); err != nil {
					t.Fatal("npm whole atomic publication", err)
				}
			}()
			if _, _, _, ok := owner.PublishedRecords(); !ok {
				t.Fatal("npm whole publication records missing")
			}
			actualLock, err := os.ReadFile(filepath.Join(target.String(), "package-lock.json"))
			if err != nil || !bytes.Equal(actualLock, selected.Controls()[1].Body()) {
				t.Fatal("npm published lock differs", err)
			}
			func() {
				guard, err := owner.BeginUpgrade(ctx, install, true)
				if err != nil {
					t.Fatal("npm independently retained old approval", err)
				}
				if err := guard.Close(); err != nil {
					t.Fatal(err)
				}
			}()
			// An explicitly edited control may be selected again, but the independently
			// approved old payload/cache remains mandatory. No unowned adoption occurs.
			edited := append(manifest.Body(), '\n')
			if err := os.WriteFile(filepath.Join(target.String(), "package.json"), edited, 0o600); err != nil {
				t.Fatal(err)
			}
			func() {
				guard, err := owner.BeginUpgrade(ctx, install, true)
				if err != nil {
					t.Fatal("npm edited control prevented guarded reselection", err)
				}
				if err := guard.Close(); err != nil {
					t.Fatal(err)
				}
			}()
			archive := filepath.Join(data, "node_modules.tar")
			cacheBytes, err := os.ReadFile(archive)
			if err != nil || len(cacheBytes) == 0 {
				t.Fatal("npm staged archive unavailable", err)
			}
			cacheBytes[len(cacheBytes)/2] ^= 1
			if err := os.Chmod(archive, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(archive, cacheBytes, 0o444); err != nil {
				t.Fatal(err)
			}
			if _, err := cache.OpenProjectCache(ctx, staged); err == nil {
				t.Fatal("tampered npm cache reopened")
			}
			if guard, err := owner.BeginUpgrade(ctx, install, true); err == nil {
				guard.Close()
				t.Fatal("tampered retained cache authorized target mutation")
			}
			preserved, err := os.ReadFile(filepath.Join(target.String(), "package.json"))
			if err != nil || !bytes.Equal(preserved, edited) {
				t.Fatal("failed npm guard changed original control", err)
			}
			for node := range acquired {
				delete(acquired, node)
				break
			}
			if _, err := inspector.PreparedLayoutFor(snapshot, acquired); err == nil {
				t.Fatal("partial request reused complete layout")
			}
			t.Logf("npm_project_quarantine case=%s archives=%d placements=%d required_per_archive=3 whole_source/trace/evidence/Policy=COMPLETE primary=NONE cache/atomic_temp_target/retained_approval/tamper_refusal=COMPLETE root_lifecycle/CLI=NOT_EXERCISED seconds=%.3f", tc.name, len(approved.Inspected().Inspections()), count, time.Since(started).Seconds())
		}) {
			return
		}
	}
}

func npmProjectIntegrationRuntime(t *testing.T, ctx context.Context) (sandbox.TrustedExecutor, *sandbox.ObserverSupervisor, *sandbox.NPMResolver) {
	t.Helper()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	})
	launcher, err := hosttool.NewSystemObserverLauncher()
	if err != nil {
		t.Fatal(err)
	}
	supervisor, err := sandbox.NewObserverSupervisor(ctx, func(start context.Context, remote, output string) (sandbox.ObserverProcess, error) {
		return launcher.StartObserver(start, remote, output)
	})
	if err != nil {
		t.Fatal(err)
	}
	policyService, err := newSystemResolverPolicyAdapter()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	})
	native, err := sandbox.NewLinuxNPMResolverWithExecutorAndPolicy(executor, supervisor.Observer(), policyService)
	if err != nil {
		t.Fatal(err)
	}
	return executor, supervisor, native
}

// End-to-end Application sequencing keeps native intent distinct from the
// already approved old target. It never executes a package manager on the Host.
func TestNPMPublicProjectWorkflowIntegration(t *testing.T) {
	if os.Getenv("HELOX_NPM_PROJECT_QUARANTINE_INTEGRATION") != "1" {
		t.Skip("requires declared protected npm workflow")
	}
	if os.Geteuid() == 0 {
		t.Fatal("requires ordinary-user client")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	executor, observer, native := npmProjectIntegrationRuntime(t, ctx)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	targetName := filepath.Join(root, "project")
	if err := os.Mkdir(targetName, 0o700); err != nil {
		t.Fatal(err)
	}
	writeManifest := func(body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(targetName, "package.json"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeManifest(`{"dependencies":{"is-number":"6.0.0"}}`)
	if err := os.WriteFile(filepath.Join(targetName, "user-notes"), []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	source, err := artifactnpm.NewPublicResolver(filepath.Join(root, "intake"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := local.NewStore(filepath.Join(root, "evidence"))
	if err != nil {
		t.Fatal(err)
	}
	static, err := inspectionnpm.NewInspector(filepath.Join(root, "intake"))
	if err != nil {
		t.Fatal(err)
	}
	backend, err := sandbox.NewLinuxBackendWithExecutor(filepath.Join(root, "intake"), executor, observer.Observer(), newObservationResourceAdapter())
	if err != nil {
		t.Fatal(err)
	}
	target, err := domain.NewInstallTarget(targetName)
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewNPMProjectInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	bindings := npmProjectBindings{filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache"), filepath.Join(root, "state"), source, store, static, native, backend}
	operations := map[domain.OperationID]bool{}
	apply := func(intent artifactnpm.NPMProjectIntent, count int) {
		t.Helper()
		started := time.Now()
		service, err := newNPMProjectService(bindings, intent)
		if err != nil {
			t.Fatal(err)
		}
		result, err := service.Apply(ctx, install)
		if err != nil || !result.Valid() || len(result.Inspected().Inspections()) != count {
			t.Fatal("complete npm workflow", intent, err)
		}
		if operations[service.operation] {
			t.Fatal("npm reused a previous inspection operation")
		}
		operations[service.operation] = true
		if _, _, _, ok := service.promoter.PublishedRecords(); !ok {
			t.Fatal("npm workflow publication records missing")
		}
		t.Logf("npm_workflow intent=%s archives=%d fresh_operation=true atomic_temp_target=true root_lifecycle/CLI=NOT_EXERCISED seconds=%.3f", intent, count, time.Since(started).Seconds())
	}
	apply(artifactnpm.NPMProjectInstall, 1)
	lockBefore, err := os.ReadFile(filepath.Join(targetName, "package-lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	payloadBefore, err := os.Stat(filepath.Join(targetName, "node_modules/is-number/package.json"))
	if err != nil {
		t.Fatal(err)
	}
	apply(artifactnpm.NPMProjectCI, 1)
	lockAfter, err := os.ReadFile(filepath.Join(targetName, "package-lock.json"))
	if err != nil || !bytes.Equal(lockBefore, lockAfter) {
		t.Fatal("npm ci rewrote lock bytes", err)
	}
	payloadAfter, err := os.Stat(filepath.Join(targetName, "node_modules/is-number/package.json"))
	if err != nil || !os.SameFile(payloadBefore, payloadAfter) {
		t.Fatal("unchanged npm ci replaced payload", err)
	}
	writeManifest(`{"dependencies":{"is-number":"7.0.0","is-odd":"3.0.1"}}`)
	apply(artifactnpm.NPMProjectInstall, 3)
	changedLock, err := os.ReadFile(filepath.Join(targetName, "package-lock.json"))
	if err != nil || bytes.Equal(lockBefore, changedLock) {
		t.Fatal("npm install did not reconcile edited manifest", err)
	}
	apply(artifactnpm.NPMProjectCI, 3)
	writeManifest(`{"dependencies":{"is-number":"6.0.0","is-odd":"3.0.1"}}`)
	oldPayload, err := os.ReadFile(filepath.Join(targetName, "node_modules/is-number/package.json"))
	if err != nil {
		t.Fatal(err)
	}
	denied, err := newNPMProjectService(bindings, artifactnpm.NPMProjectCI)
	if err != nil {
		t.Fatal(err)
	}
	if result, err := denied.Apply(ctx, install); err == nil || result.Valid() {
		t.Fatal("npm ci accepted mismatched manifest/lock")
	}
	if denied.cache != nil {
		t.Fatal("invalid npm ci reached cache/staging")
	}
	preservedLock, err := os.ReadFile(filepath.Join(targetName, "package-lock.json"))
	if err != nil || !bytes.Equal(preservedLock, changedLock) {
		t.Fatal("failed npm ci changed lock", err)
	}
	preservedPayload, err := os.ReadFile(filepath.Join(targetName, "node_modules/is-number/package.json"))
	if err != nil || !bytes.Equal(oldPayload, preservedPayload) {
		t.Fatal("failed npm ci changed old payload", err)
	}
	notes, err := os.ReadFile(filepath.Join(targetName, "user-notes"))
	if err != nil || string(notes) != "preserve" {
		t.Fatal("npm workflow changed unrelated user file", err)
	}
	t.Log("npm_workflow install/ci/edited_install/ci=PASS ci_mismatch=REFUSED old_target/user_file=PRESERVED root_lifecycle/CLI=NOT_EXERCISED")
}
