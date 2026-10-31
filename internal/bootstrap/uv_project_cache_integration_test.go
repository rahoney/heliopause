//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/hosttool"
	inspectionpypi "github.com/rahoney/heliopause/internal/inspection/pypi"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationpypi "github.com/rahoney/heliopause/internal/verification/pypi"
)

type uvProjectEvidenceAvailability struct {
	store *local.Store
	lost  bool
}

func (r *uvProjectEvidenceAvailability) ReadReference(ctx context.Context, run domain.RunID, ref domain.EvidenceReference, identity domain.ResolvedArtifactIdentity, digest domain.ContentDigest) (domain.Evidence, domain.ContentDigest, error) {
	if r.lost {
		return domain.Evidence{}, domain.ContentDigest{}, errors.New("test: recorded Evidence unavailable")
	}
	return r.store.ReadReference(ctx, run, ref, identity, digest)
}

// Actual selection, complete wheel inspection and recorded Policy approval.
// Data-only staging does not publish a Host project or an installed environment.
func TestUVPublicProjectApprovedCacheIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_PROJECT_CACHE_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user UV project cache qualification")
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
	native, err := sandbox.NewLinuxUVResolverWithExecutorAndPolicy(executor, observer.Observer(), policyService)
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := sandbox.NewUVProjectResolver(native)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Error(err)
		}
	}()
	for _, item := range []struct {
		name, dependencies string
		count              int
	}{
		{"two roots", `["idna==3.10","packaging==25.0"]`, 2},
		{"explicit empty", `[]`, 0},
	} {
		if !t.Run(item.name, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			intakeRoot, evidenceRoot, cacheRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache")
			target, err := domain.NewInstallTarget(filepath.Join(root, "host-venv"))
			if err != nil {
				t.Fatal(err)
			}
			install, err := domain.NewPythonVenvInstallContext(target)
			if err != nil {
				t.Fatal(err)
			}
			manifest := []byte(fmt.Sprintf("[project]\nname='haa-cache-project'\nversion='0.1.0'\nrequires-python='==%s'\ndependencies=%s\n", sandbox.PinnedPythonRuntime().PythonVersion, item.dependencies))
			input, err := artifactpypi.ParseUVProjectInput(manifest, nil, false, artifactpypi.UVProjectSync, "")
			if err != nil {
				t.Fatal(err)
			}
			original, err := sandbox.NewUVProjectOriginalControls(input)
			if err != nil {
				t.Fatal(err)
			}
			update, err := resolver.ResolveProjectLockUpdate(ctx, install, original, "")
			if err != nil {
				t.Fatal(err)
			}
			intake, err := artifactpypi.NewPublicIntake(intakeRoot)
			if err != nil {
				t.Fatal(err)
			}
			runtime := sandbox.PinnedPythonRuntime()
			static, err := inspectionpypi.NewStaticInspector(intakeRoot, artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag})
			if err != nil {
				t.Fatal(err)
			}
			backend, err := sandbox.NewLinuxPyPIDynamicBackendWithExecutor(intakeRoot, executor, observer.Observer(), newObservationResourceAdapter())
			if err != nil {
				t.Fatal(err)
			}
			dynamic, err := inspectionpypi.NewDynamicInspector(backend)
			if err != nil {
				t.Fatal(err)
			}
			inspector, err := inspectionpypi.NewCompositeInspector(static, dynamic)
			if err != nil {
				t.Fatal(err)
			}
			store, err := local.NewStore(evidenceRoot)
			if err != nil {
				t.Fatal(err)
			}
			operation, err := domain.NewOperationID()
			if err != nil {
				t.Fatal(err)
			}
			quarantine, err := application.NewProjectInspectService(intake, verificationpypi.IntegrityVerifier{}, inspector, store, policy.M3{}, policy.M4{}, func() (domain.OperationID, error) { return operation, nil }, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			approved, err := quarantine.InspectProject(ctx, update.Snapshot())
			if err != nil || !approved.Valid() || len(approved.Inspected().Inspections()) != item.count {
				t.Fatal("complete actual approval", err)
			}
			for _, entry := range approved.Inspected().Inspections() {
				completed := 0
				for _, check := range entry.Checks() {
					if check.Required() {
						if check.Status() != domain.ExecutionCompleted {
							t.Fatal("required check incomplete")
						}
						completed++
					}
				}
				if completed < 3 || len(entry.Evidence()) < 3 {
					t.Fatal("whole wheel coverage missing")
				}
			}
			reader := &uvProjectEvidenceAvailability{store: store}
			base, err := promotion.NewPythonProjectCache(intakeRoot, evidenceRoot, cacheRoot, reader, evidence.Generator{}, operation)
			if err != nil {
				t.Fatal(err)
			}
			cache, err := promotion.NewUVProjectCache(base, update)
			if err != nil {
				t.Fatal(err)
			}
			if staged, err := cache.StageProject(ctx, domain.ProjectVerifiedSet{}); err == nil || staged.Valid() {
				t.Fatal("selection without approval accepted")
			}
			staged, err := cache.StageProject(ctx, approved)
			if err != nil {
				t.Fatal(err)
			}
			data, err := cache.OpenProjectCache(ctx, staged)
			if err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(filepath.Join(data, "manifest.json"))
			if err != nil {
				t.Fatal(err)
			}
			var doc struct {
				Operation string            `json:"operation_id"`
				Manifest  string            `json:"manifest_id"`
				Primary   json.RawMessage   `json:"primary"`
				Controls  []json.RawMessage `json:"project_controls"`
				Entries   []json.RawMessage `json:"entries"`
			}
			if json.Unmarshal(body, &doc) != nil || doc.Operation != operation.String() || doc.Manifest == "" || len(doc.Primary) != 0 || len(doc.Controls) != 6 || len(doc.Entries) != item.count {
				t.Fatal("actual native complete Manifest lost identity", string(body))
			}
			sbom, err := os.ReadFile(filepath.Join(data, "sbom.cdx.json"))
			if err != nil || !json.Valid(sbom) || !bytes.Contains(sbom, []byte(doc.Manifest)) {
				t.Fatal("actual SBOM lost binding", err)
			}
			if item.count > 0 {
				reader.lost = true
				if _, err := cache.OpenProjectCache(ctx, staged); err == nil {
					t.Fatal("lost current Evidence accepted on readback")
				}
				if s, err := cache.StageProject(ctx, approved); err == nil || s.Valid() {
					t.Fatal("lost current Evidence accepted on stage")
				}
				reader.lost = false
				if _, err := cache.OpenProjectCache(ctx, staged); err != nil {
					t.Fatal("restored actual reader rejected", err)
				}
			}
			if _, err := os.Lstat(install.Target().String()); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("data cache published Host target", err)
			}
			path := filepath.Join(data, "manifest.json")
			if err := os.Chmod(path, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, append(body, ' '), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(path, 0o444); err != nil {
				t.Fatal(err)
			}
			if _, err := cache.OpenProjectCache(ctx, staged); err == nil {
				t.Fatal("changed sealed cache accepted")
			}
			t.Logf("actual_uv_project_cache=PASS entries=%d current Evidence/Policy/native provenance/Manifest/SBOM; no Host install", item.count)
		}) {
			t.FailNow()
		}
	}
}
