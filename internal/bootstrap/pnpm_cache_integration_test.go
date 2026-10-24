//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/hosttool"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationnpm "github.com/rahoney/heliopause/internal/verification/npm"
)

// Production composition through data staging only. No Host project is created
// and no native/cache pathname can replace the recorded inspection approval.
func TestPNPMPublicApprovedCacheIntegration(t *testing.T) {
	if os.Getenv("HELOX_PNPM_CACHE_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user authenticated PNPM cache qualification")
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
	launcher, err := hosttool.NewSystemObserverLauncher()
	if err != nil {
		t.Fatal(err)
	}
	observer, err := sandbox.NewObserverSupervisor(ctx, func(start context.Context, remote, output string) (sandbox.ObserverProcess, error) {
		return launcher.StartObserver(start, remote, output)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := observer.Close(); err != nil {
			t.Error(err)
		}
	}()
	policyService, err := newSystemResolverPolicyAdapter()
	if err != nil {
		t.Fatal(err)
	}
	native, err := sandbox.NewLinuxPNPMResolverWithExecutorAndPolicy(executor, observer.Observer(), policyService)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		name, manifest string
		count          int
	}{
		{"peer", `{"scripts":{"install":"exit 99"},"dependencies":{"react":"18.3.1","use-sync-external-store":"1.2.0"}}`, 4},
		{"empty", `{}`, 0},
	} {
		if !t.Run(item.name, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			intake, evidenceRoot, cacheRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache")
			source, err := artifactnpm.NewPublicResolver(intake)
			if err != nil {
				t.Fatal(err)
			}
			manifest, err := domain.NewProjectControlFile("package.json", []byte(item.manifest), true)
			if err != nil {
				t.Fatal(err)
			}
			absentLock, err := domain.NewProjectControlFile("pnpm-lock.yaml", nil, false)
			if err != nil {
				t.Fatal(err)
			}
			input, err := artifactnpm.NewPNPMSelectionInput(artifactnpm.PNPMInstall, "", []domain.ProjectControlFile{manifest, absentLock}, runtimeidentity.PNPMVersion)
			if err != nil {
				t.Fatal(err)
			}
			selected, err := native.SelectProject(ctx, input)
			if err != nil {
				t.Fatal(err)
			}
			target, err := domain.NewInstallTarget(filepath.Join(root, "host-project"))
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
			static, err := inspectionnpm.NewInspector(intake)
			if err != nil {
				t.Fatal(err)
			}
			prepare, err := sandbox.NewPNPMInspectionPreparer(native, intake)
			if err != nil {
				t.Fatal(err)
			}
			backend, err := sandbox.NewLinuxBackendWithExecutor(intake, executor, observer.Observer())
			if err != nil {
				t.Fatal(err)
			}
			inspector, err := inspectionnpm.NewPNPMProjectInspector(frozen, static, prepare, backend)
			if err != nil {
				t.Fatal(err)
			}
			store, err := local.NewStore(evidenceRoot)
			if err != nil {
				t.Fatal(err)
			}
			quarantine, err := application.NewProjectInspectService(source, verificationnpm.IntegrityVerifier{}, inspector, store, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			approved, err := quarantine.InspectProject(ctx, frozen.Snapshot())
			if err != nil || !approved.Valid() || len(approved.Inspected().Inspections()) != item.count {
				t.Fatal("public inspection approval failed", err)
			}
			cache, err := promotion.NewPNPMVerifiedCache(intake, evidenceRoot, cacheRoot, frozen, store, inspector)
			if err != nil {
				t.Fatal(err)
			}
			staged, err := cache.StageProject(ctx, approved)
			if err != nil {
				t.Fatal("stage actual approved data", err)
			}
			data, err := cache.OpenProjectCache(ctx, staged)
			if err != nil {
				t.Fatal("read actual approved data", err)
			}
			for _, control := range selected.Controls() {
				body, err := os.ReadFile(filepath.Join(data, control.Name()))
				if err != nil || !bytes.Equal(body, control.Body()) {
					t.Fatal("original selected controls changed", err)
				}
			}
			entries, err := os.ReadDir(data)
			if err != nil {
				t.Fatal(err)
			}
			want := 3
			if item.count == 0 {
				want = 2
			}
			if len(entries) != want {
				t.Fatal("data cache contains foreign files")
			}
			if item.count != 0 {

				acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
				for _, entry := range approved.Inspected().Inspections() {
					acquired[entry.Node()] = entry.Artifact()
				}
				if _, err := inspector.PreparedLayoutFor(frozen.Snapshot(), acquired); err != nil {
					t.Fatal("completed handoff missing", err)
				}
				cancelled, stop := context.WithCancel(ctx)
				stop()
				if _, err := inspector.InspectProject(cancelled, frozen.Snapshot(), acquired); !errors.Is(err, context.Canceled) {
					t.Fatal("cancelled request lost cause", err)
				}
				if _, err := inspector.PreparedLayoutFor(frozen.Snapshot(), acquired); err == nil {
					t.Fatal("failed next request retained previous handoff")
				}
				body, err := os.ReadFile(filepath.Join(data, "node_modules.tar"))
				if err != nil {
					t.Fatal(err)
				}
				view, err := artifactnpm.ParsePNPMLayoutArchive(ctx, body)
				if err != nil {
					t.Fatal(err)
				}
				for _, entry := range view.Entries() {
					if entry.Name == "node_modules/.pnpm-workspace-state-v1.json" {
						t.Fatal("discarded freshness cache reintroduced")
					}
				}
				file := filepath.Join(data, "node_modules.tar")
				body[len(body)/2] ^= 1
				if err := os.Chmod(file, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file, body, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(file, 0o444); err != nil {
					t.Fatal(err)
				}
				if _, err := cache.OpenProjectCache(ctx, staged); err == nil {
					t.Fatal("changed actual data tar admitted")
				}
			}
			if _, err := os.Lstat(target.String()); !os.IsNotExist(err) {
				t.Fatal("staging created a Host project", err)
			}
			t.Logf("pnpm_approved_cache fixture=%s public_entries=%d stored_evidence_entries=%d native_layout_data=%t sealed=true drift_tested=%t Host_install=NOT_RUN", item.name, item.count, item.count, item.count != 0, item.count != 0)
		}) {
			t.FailNow()
		}
	}
}
