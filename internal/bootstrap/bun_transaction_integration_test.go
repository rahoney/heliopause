//go:build linux && realcorpus

package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/hosttool"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestBunPublicProjectTransactionIntegration(t *testing.T) {
	if os.Getenv("HELOX_BUN_PROJECT_TRANSACTION_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user authenticated Bun transaction qualification")
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
	native, err := sandbox.NewLinuxBunResolverWithExecutorAndPolicy(executor, observer.Observer(), policyService)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		name, manifest, code string
		count                int
	}{
		{"single", `{"scripts":{"install":"exit 99"},"dependencies":{"is-number":"6.0.0"}}`, "node_modules/is-number/package.json", 1},
		{"peer", `{"scripts":{"install":"exit 99"},"dependencies":{"react":"18.3.1","use-sync-external-store":"1.2.0"}}`, "node_modules/react/package.json", 4},
		{"empty", `{}`, "", 0},
	} {
		if !t.Run(item.name, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			intake, evidenceRoot, cacheRoot, stateRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache"), filepath.Join(root, "state")
			source, err := artifactnpm.NewPublicResolver(intake)
			if err != nil {
				t.Fatal(err)
			}
			store, err := local.NewStore(evidenceRoot)
			if err != nil {
				t.Fatal(err)
			}
			static, err := inspectionnpm.NewInspector(intake)
			if err != nil {
				t.Fatal(err)
			}
			prepare, err := sandbox.NewBunInspectionPreparer(native, intake)
			if err != nil {
				t.Fatal(err)
			}
			backend, err := sandbox.NewLinuxBackendWithExecutor(intake, executor, observer.Observer())
			if err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(root, "project")
			if err := os.Mkdir(target, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(target, "package.json"), []byte(item.manifest), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(target, "user-notes"), []byte("preserve user file\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			destination, err := domain.NewInstallTarget(target)
			if err != nil {
				t.Fatal(err)
			}
			install, err := domain.NewNPMProjectInstallContext(destination)
			if err != nil {
				t.Fatal(err)
			}
			bindings := bunProjectBindings{intake, evidenceRoot, cacheRoot, stateRoot, source, store, static, native, prepare, backend}
			apply := func(intent artifactnpm.BunSelectionIntent, selector string) (domain.ProjectVerifiedSet, string) {
				service, err := newBunProjectService(bindings, intent, selector)
				if err != nil {
					t.Fatal(err)
				}
				approved, err := service.Apply(ctx, install)
				if err != nil || !approved.Valid() {
					t.Fatal("actual guarded Bun application failed", err)
				}
				body, err := os.ReadFile(filepath.Join(target, ".heliopause/manifest.json"))
				if err != nil {
					t.Fatal(err)
				}
				var record struct {
					Operation string            `json:"operation_id"`
					Entries   []json.RawMessage `json:"entries"`
				}
				if json.Unmarshal(body, &record) != nil || record.Operation == "" || len(record.Entries) != len(approved.Inspected().Inspections()) {
					t.Fatal("published actual operation records differ")
				}
				return approved, record.Operation
			}
			first, firstOperation := apply(artifactnpm.BunInstall, "")
			if len(first.Inspected().Inspections()) != item.count {
				t.Fatal("incomplete published set")
			}
			controlBefore, err := os.Lstat(filepath.Join(target, "package.json"))
			if err != nil {
				t.Fatal(err)
			}
			var codeBefore os.FileInfo
			if item.code != "" {
				codeBefore, err = os.Stat(filepath.Join(target, item.code))
				if err != nil {
					t.Fatal(err)
				}
			}
			second, nextOperation := apply(artifactnpm.BunInstall, "")
			if item.count != 0 {
				firstRuns := map[string]bool{}
				for _, i := range first.Inspected().Inspections() {
					firstRuns[i.RunID().String()] = true
				}
				for _, i := range second.Inspected().Inspections() {
					if firstRuns[i.RunID().String()] {
						t.Fatal("no-op reused old inspection run")
					}
				}
			} else if _, err := os.Lstat(filepath.Join(target, "bun.lock")); !os.IsNotExist(err) {
				t.Fatal("empty transaction manufactured lock", err)
			}
			if firstOperation == nextOperation {
				t.Fatal("repeat reused the previous inspection operation")
			}
			controlAfter, err := os.Lstat(filepath.Join(target, "package.json"))
			if err != nil || !os.SameFile(controlBefore, controlAfter) || controlAfter.Mode() != 0o600 {
				t.Fatal("no-op replaced original control", err)
			}
			if item.code != "" {
				after, err := os.Stat(filepath.Join(target, item.code))
				if err != nil || !os.SameFile(codeBefore, after) {
					t.Fatal("no-op rewrote published native payload", err)
				}
			}
			if item.name == "single" {
				updated, _ := apply(artifactnpm.BunAdd, "is-number@7.0.0")
				if len(updated.Inspected().Inspections()) != 1 || updated.Inspected().Snapshot().Dependencies()[0].Identity().Version() != "7.0.0" {
					t.Fatal("named actual add/update did not publish its selected version")
				}
				body, err := os.ReadFile(filepath.Join(target, item.code))
				if err != nil {
					t.Fatal(err)
				}
				var pkg struct {
					Version string `json:"version"`
				}
				if json.Unmarshal(body, &pkg) != nil || pkg.Version != "7.0.0" {
					t.Fatal("Host bytes differ from approved selected package")
				}
				beforeUpdate, err := os.Stat(filepath.Join(target, item.code))
				if err != nil {
					t.Fatal(err)
				}
				refreshed, _ := apply(artifactnpm.BunUpdate, "is-number")
				if len(refreshed.Inspected().Inspections()) != 1 || refreshed.Inspected().Snapshot().Dependencies()[0].Identity().Version() != "7.0.0" {
					t.Fatal("named native update returned another selected version")
				}
				afterUpdate, err := os.Stat(filepath.Join(target, item.code))
				if err != nil || !os.SameFile(beforeUpdate, afterUpdate) {
					t.Fatal("named no-op update rewrote approved payload", err)
				}
				t.Log("bun_named_add_version=7.0.0 native_update_mode=true selected_version_preserved=true update_noop_inode=true")
			}
			if body, err := os.ReadFile(filepath.Join(target, "user-notes")); err != nil || string(body) != "preserve user file\n" {
				t.Fatal("unmanaged user file changed", err)
			}
			states, err := os.ReadDir(stateRoot)
			if err != nil || len(states) != 1 {
				t.Fatal("independent approval publication missing", err)
			}
			t.Logf("bun_public_transaction fixture=%s entries=%d actual_fresh_operation=true Policy=ALLOW target_published=true independent_state=true repeated_native_noop=true original_control_inode=true user_files_preserved=true root_script=NOT_EXECUTED", item.name, item.count)
		}) {
			t.FailNow()
		}
	}
}
