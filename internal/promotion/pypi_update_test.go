package promotion

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type pypiUpgradeSelectionFixture struct{ calls int }

func (r *pypiUpgradeSelectionFixture) ResolveDependencyUpdate(context.Context, domain.ArtifactReference, domain.InstallContext, []domain.ProjectControlFile) (domain.DependencyResolution, error) {
	r.calls++
	return domain.DependencyResolution{}, errors.New("controlled selection failure")
}

type reportedPyPIUpgradeFixture struct {
	t        *testing.T
	artifact domain.ResolvedArtifact
	mutate   func(domain.DependencyResolution) domain.DependencyResolution
}

func (r *reportedPyPIUpgradeFixture) ResolveDependencyUpdate(_ context.Context, reference domain.ArtifactReference, _ domain.InstallContext, controls []domain.ProjectControlFile) (domain.DependencyResolution, error) {
	r.t.Helper()
	item := map[string]any{"requested": true, "download_info": map[string]any{"url": r.artifact.AcquisitionLocator(), "archive_info": map[string]any{"hashes": map[string]string{"sha256": strings.TrimPrefix(r.artifact.DeclaredIntegrity(), "sha256:")}}}, "metadata": map[string]any{"name": r.artifact.Identity().Name(), "version": r.artifact.Identity().Version()}}
	python := sandbox.PinnedPythonRuntime()
	body, err := json.Marshal(map[string]any{"version": "1", "pip_version": python.PipVersion, "environment": map[string]string{"implementation_name": "cpython", "implementation_version": python.PythonVersion, "python_full_version": python.PythonVersion, "platform_machine": "x86_64", "sys_platform": "linux"}, "install": []any{item}})
	if err != nil {
		r.t.Fatal(err)
	}
	report, err := artifactpypi.ParseInstallationReport(reference, body, python.PipVersion, python.PythonVersion)
	if err != nil {
		r.t.Fatal(err)
	}
	graph, err := artifactpypi.BuildLockedGraph(reference, report.Candidates())
	if err != nil {
		r.t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	resolution, err := domain.NewDependencyResolution(graph, "python:"+python.PythonVersion+";pip:"+python.PipVersion+";target:"+python.InterpreterTag+"/"+python.ABITag+"/"+python.PlatformTag+";source:pypi", digest)
	if err != nil {
		r.t.Fatal(err)
	}
	for _, name := range []string{"pypi-update-delta.json", "pypi-update-report.json"} {
		control, err := domain.NewProjectControlFile(name, body, true)
		if err != nil {
			r.t.Fatal(err)
		}
		controls = append(controls, control)
	}
	resolution, err = resolution.WithProjectControls(controls)
	if err != nil {
		r.t.Fatal(err)
	}
	if r.mutate != nil {
		resolution = r.mutate(resolution)
	}
	return resolution, nil
}

// Only the repository's own synthetic wheel is extracted by this fake offline
// runner; actual pip/observer qualification is deliberately separate.
type fixturePyPIUpgradeRunner struct {
	t      *testing.T
	before func()
	calls  int
}

func (r *fixturePyPIUpgradeRunner) Run(_ context.Context, project string, _ []string) error {
	r.calls++
	archive, err := zip.OpenReader(filepath.Join(project, "wheels", "haa_promotion_fixture-1.0.0-py3-none-any.whl"))
	if err != nil {
		return err
	}
	defer archive.Close()
	for _, file := range archive.File {
		if strings.Contains(file.Name, "..") || strings.HasPrefix(file.Name, "/") {
			r.t.Fatal("invalid synthetic wheel path")
		}
		path := filepath.Join(project, "site", file.Name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		input, err := file.Open()
		if err != nil {
			return err
		}
		body, err := io.ReadAll(input)
		closeErr := input.Close()
		if err != nil || closeErr != nil {
			return errors.Join(err, closeErr)
		}
		if err := os.WriteFile(path, body, 0o600); err != nil {
			return err
		}
	}
	if r.before != nil {
		r.before()
	}
	return nil
}

func TestPyPIUpgradeUsesSameEarlyTransactionThroughExactOutputAndRollback(t *testing.T) {
	for _, mode := range []string{"upgrade", "partial-publication", "precommit-drift", "precommit-cancel", "manifest-substitution", "staged-tamper"} {
		t.Run(mode, func(t *testing.T) {
			root, plan := makePypiVenvFixture(t)
			oldOutput := venvDestinations(t, map[string]string{"site/haa_promotion_fixture-0.9.0.dist-info/METADATA": "Name: haa-promotion-fixture\nVersion: 0.9.0\n\n", "site/haa_promotion_fixture/__init__.py": "old payload", "site/obsolete.py": "obsolete"}, "haa-promotion-fixture")
			for i := range oldOutput {
				oldOutput[i].Version = "0.9.0"
			}
			if err := plan.commit("", oldOutput); err != nil {
				t.Fatal(err)
			}
			runVenvFixture(t, plan, map[string]string{"site/other-1.0.dist-info/METADATA": "Name: other\nVersion: 1.0\n\n", "site/other.py": "retained unrelated"}, "other")
			before, _, err := plan.readState()
			if err != nil {
				t.Fatal(err)
			}
			evidenceRoot := realPromotionRoot(t)
			seed, _ := validStagedPyPIFixture(t, evidenceRoot)
			resolver := &reportedPyPIUpgradeFixture{t: t, artifact: seed.Set().Inspected().Graph().Nodes()[0].Artifact()}
			runner := &fixturePyPIUpgradeRunner{t: t}
			promoter, _ := newPyPIPromotion(filepath.Join(evidenceRoot, "staging"), runner, "linux", "amd64")
			preparation, _ := NewPyPIUpdatePreparation(promoter, resolver)
			source, _ := domain.NewSourceID("pypi")
			target, _ := domain.NewInstallTarget(root)
			install, _ := domain.NewPythonVenvInstallContext(target)
			guarded, err := preparation.BeginInstallUpdate(context.Background(), source, "haa-promotion-fixture", install, false)
			if err != nil {
				t.Fatal(err)
			}
			session := guarded.(*pypiUpdateSession)
			resolution, err := session.ResolveDependencies(context.Background(), session.Reference(), install)
			if err != nil {
				t.Fatal(err)
			}
			original := seed.Set().Inspected().Inspections()[0]
			inspection, err := domain.NewDependencyInspection(resolution.Graph().Primary(), original.RunID(), original.Artifact(), original.Checks(), original.Evidence(), original.PolicyDecision())
			if err != nil {
				t.Fatal(err)
			}
			inspected, err := domain.NewInspectedDependencySet(resolution.Graph(), []domain.DependencyInspection{inspection})
			if err != nil {
				t.Fatal(err)
			}
			set, err := domain.NewVerifiedSet(inspected, seed.Set().Decision())
			if err != nil {
				t.Fatal(err)
			}
			op, _ := domain.NewOperationID()
			controls := resolution.ProjectControls()
			if mode == "manifest-substitution" {
				controls = controls[:1]
			}
			bundle, err := evidence.BuildVerifiedBundle(evidence.ManifestContext{OperationID: op, InstallContext: install, ResolverRuntime: resolution.RuntimeIdentity(), LockfileDigest: resolution.LockfileDigest(), ProjectControls: controls}, set)
			if err != nil {
				t.Fatal(err)
			}
			staging, _ := NewLocalStaging(filepath.Join(evidenceRoot, "intake"), filepath.Join(evidenceRoot, "evidence"), filepath.Join(evidenceRoot, "staging"))
			staged, err := staging.Stage(context.Background(), bundle)
			if err != nil {
				t.Fatal(err)
			}
			cause := errors.New("controlled publication failure")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "partial-publication" {
				session.transaction.fault = func(step string) error {
					if step == "after-publish:site" {
						return cause
					}
					return nil
				}
			}
			if mode == "precommit-cancel" {
				runner.before = cancel
			}
			if mode == "precommit-drift" {
				runner.before = func() {
					if err := os.WriteFile(filepath.Join(plan.site, "haa_promotion_fixture/__init__.py"), []byte("user edit"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			}
			if mode == "staged-tamper" {
				path := filepath.Join(evidenceRoot, "staging", bundle.ManifestID().String(), "artifacts", original.Artifact().Digest().String()+".whl")
				if err := os.Chmod(path, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("substituted"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			promoted, err := session.Promote(ctx, staged, bundle, install)
			if (mode == "upgrade") != (err == nil) || mode == "upgrade" && promoted.ManifestID() != bundle.ManifestID() {
				t.Fatal("publication result", mode, err)
			}
			if mode == "partial-publication" && !errors.Is(err, cause) || mode == "precommit-cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal("primary cause lost", err)
			}
			if (mode == "manifest-substitution" || mode == "staged-tamper") && runner.calls != 0 {
				t.Fatal("unbound selection reached private install")
			}
			if err := session.Close(); err != nil {
				t.Fatal(err)
			}
			body, readErr := os.ReadFile(filepath.Join(plan.site, "haa_promotion_fixture/__init__.py"))
			want := "old payload"
			if mode == "upgrade" {
				want = "value = 42\n"
			}
			if mode == "precommit-drift" {
				want = "user edit"
			}
			if readErr != nil || string(body) != want {
				t.Fatal("original/promotion content", string(body), readErr)
			}
			retained, err := os.ReadFile(filepath.Join(plan.site, "other.py"))
			if err != nil || string(retained) != "retained unrelated" {
				t.Fatal("unrelated output changed", err)
			}
			if mode != "upgrade" && mode != "precommit-drift" {
				if err := plan.verifyState(before); err != nil {
					t.Fatal("rollback failed to restore original", err)
				}
			}
		})
	}
}

func managedPyPIUpgradeFixture(t *testing.T) (string, pypiVenvPlan, domain.InstallContext) {
	t.Helper()
	root, plan := makePypiVenvFixture(t)
	runVenvFixture(t, plan, map[string]string{"site/fixture-1.0.dist-info/METADATA": "Name: fixture\nVersion: 1.0\n\n", "site/fixture.py": "old payload"}, "fixture")
	target, _ := domain.NewInstallTarget(root)
	install, _ := domain.NewPythonVenvInstallContext(target)
	return root, plan, install
}

func TestPyPIUpgradeRecognizesEmptyRetiredMetadataDirectoryAndGuardsItsContents(t *testing.T) {
	root, plan, install := managedPyPIUpgradeFixture(t)
	retired := filepath.Join(plan.site, "fixture-0.9.dist-info")
	if err := os.Mkdir(retired, 0o700); err != nil {
		t.Fatal(err)
	}
	resolver := &pypiUpgradeSelectionFixture{}
	promoter, _ := newPyPIPromotion(filepath.Join(root, "staging"), &venvGuardCheckingRunner{}, "linux", "amd64")
	preparation, _ := NewPyPIUpdatePreparation(promoter, resolver)
	source, _ := domain.NewSourceID("pypi")
	guarded, err := preparation.BeginInstallUpdate(context.Background(), source, "fixture", install, false)
	if err != nil {
		t.Fatal("empty directory retained by the existing transaction was treated as an installed distribution", err)
	}
	session := guarded.(*pypiUpdateSession)
	if len(session.input.MetadataFiles()) != 1 {
		t.Fatal("retired empty directory became an installed distribution")
	}
	path := filepath.Join(retired, "user-file")
	if err := os.WriteFile(path, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := session.ResolveDependencies(context.Background(), session.Reference(), install); err == nil || resolver.calls != 0 {
		t.Fatal("empty directory gained unguarded contents", err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := preparation.BeginInstallUpdate(context.Background(), source, "fixture", install, false); err == nil {
		t.Fatal("nonempty dist-info with missing METADATA was silently ignored")
	}
	if body, err := os.ReadFile(path); err != nil || string(body) != "preserve" {
		t.Fatal("unmanaged contents overwritten", err)
	}
}

func TestPyPIUpgradeRecognizesRetiredDirectoryOnlyTreeAndGuardsEveryDirectory(t *testing.T) {
	for _, mode := range []string{"unchanged", "new-file", "new-directory", "removed-directory", "replaced-directory", "symlink", "nonempty-at-start"} {
		t.Run(mode, func(t *testing.T) {
			root, plan, install := managedPyPIUpgradeFixture(t)
			retired := filepath.Join(plan.site, "fixture-0.9.dist-info")
			child := filepath.Join(retired, "licenses", "third_party")
			if err := os.MkdirAll(child, 0o700); err != nil {
				t.Fatal(err)
			}
			userFile := filepath.Join(child, "user-data")
			if mode == "nonempty-at-start" {
				if err := os.WriteFile(userFile, []byte("preserve"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			resolver := &pypiUpgradeSelectionFixture{}
			promoter, _ := newPyPIPromotion(filepath.Join(root, "staging"), &venvGuardCheckingRunner{}, "linux", "amd64")
			preparation, _ := NewPyPIUpdatePreparation(promoter, resolver)
			source, _ := domain.NewSourceID("pypi")
			session, err := preparation.BeginInstallUpdate(context.Background(), source, "fixture", install, false)
			if mode == "nonempty-at-start" {
				if err == nil || session != nil || resolver.calls != 0 {
					t.Fatal("nonempty metadata tree silently omitted", err)
				}
				body, err := os.ReadFile(userFile)
				if err != nil || string(body) != "preserve" {
					t.Fatal("user data changed", err)
				}
				return
			}
			if err != nil {
				t.Fatal("directory-only retired metadata treated as an installed distribution", err)
			}
			if len(session.(*pypiUpdateSession).input.MetadataFiles()) != 1 {
				t.Fatal("retired tree became a distribution")
			}
			switch mode {
			case "new-file":
				if err := os.WriteFile(userFile, []byte("preserve"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "new-directory":
				if err := os.Mkdir(filepath.Join(child, "new"), 0o700); err != nil {
					t.Fatal(err)
				}
			case "removed-directory":
				if err := os.Remove(child); err != nil {
					t.Fatal(err)
				}
			case "replaced-directory", "symlink":
				// Keep the old inode alive so replacement cannot reuse it.
				old, err := os.Open(child)
				if err != nil {
					t.Fatal(err)
				}
				defer old.Close()
				if err := os.Remove(child); err != nil {
					t.Fatal(err)
				}
				if mode == "symlink" {
					if err := os.Symlink(realPromotionRoot(t), child); err != nil {
						t.Fatal(err)
					}
				} else if err := os.Mkdir(child, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			_, err = session.ResolveDependencies(context.Background(), session.Reference(), install)
			if err == nil || (mode == "unchanged") != (resolver.calls == 1) {
				t.Fatal("retired subtree drift was not guarded before selection", err, resolver.calls)
			}
			if err := session.Close(); err != nil {
				t.Fatal(err)
			}
			if mode == "new-file" {
				body, err := os.ReadFile(userFile)
				if err != nil || string(body) != "preserve" {
					t.Fatal("guard mutated user data", err)
				}
			}
		})
	}
}

func TestPyPIUpgradeGuardFreezesMetadataAndOwnershipBeforeSelection(t *testing.T) {
	for _, mode := range []string{"selection-failure", "metadata-drift", "payload-drift", "new-distribution", "direct-source", "replaced-lock", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			root, plan, install := managedPyPIUpgradeFixture(t)
			resolver := &pypiUpgradeSelectionFixture{}
			promoter, _ := newPyPIPromotion(filepath.Join(root, "staging"), &venvGuardCheckingRunner{}, "linux", "amd64")
			preparation, _ := NewPyPIUpdatePreparation(promoter, resolver)
			source, _ := domain.NewSourceID("pypi")
			session, err := preparation.BeginInstallUpdate(context.Background(), source, "fixture", install, false)
			if err != nil {
				t.Fatal(err)
			}
			if tx, err := beginPyPIVenvTransaction(plan); err == nil {
				_ = tx.close()
				t.Fatal("second transaction entered during selection")
			}
			var changedPath, changedBody string
			switch mode {
			case "metadata-drift":
				changedPath = filepath.Join(plan.site, "fixture-1.0.dist-info/METADATA")
				changedBody = "Name: fixture\nVersion: 1.0\nRequires-Dist: attacker>=1\n\n"
			case "payload-drift":
				changedPath = filepath.Join(plan.site, "fixture.py")
				changedBody = "concurrent user edit"
			case "direct-source":
				changedPath = filepath.Join(plan.site, "fixture-1.0.dist-info/direct_url.json")
				changedBody = `{"url":"file:///unmanaged"}`
			case "new-distribution":
				if err := os.Mkdir(filepath.Join(plan.site, "other-1.0.dist-info"), 0o700); err != nil {
					t.Fatal(err)
				}
			case "replaced-lock":
				changedPath = filepath.Join(root, ".heliopause-pypi-transaction.lock")
				changedBody = "another transaction"
				if err := os.Rename(changedPath, filepath.Join(root, "retained-original-lock")); err != nil {
					t.Fatal(err)
				}
			}
			if changedPath != "" {
				if err := os.WriteFile(changedPath, []byte(changedBody), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			resolution, err := session.ResolveDependencies(ctx, session.Reference(), install)
			if err == nil || len(resolution.Graph().Nodes()) != 0 || (mode == "selection-failure") != (resolver.calls == 1) || mode == "cancelled" && !errors.Is(err, context.Canceled) {
				t.Fatal("invalid original state reached selection", err, resolver.calls)
			}
			closeErr := session.Close()
			if (mode == "replaced-lock") != (closeErr != nil) {
				t.Fatal("guard close result", closeErr)
			}
			if changedPath != "" {
				body, err := os.ReadFile(changedPath)
				if err != nil || string(body) != changedBody {
					t.Fatal("concurrent original was overwritten", err)
				}
			}
		})
	}
}

func TestPyPIUpgradeNativeAndUnsupportedInventoryRemainUnchanged(t *testing.T) {
	for _, mode := range []string{"native", "native-adopt", "legacy", "direct-source", "oversize", "symlink"} {
		t.Run(mode, func(t *testing.T) {
			root, plan, install := managedPyPIUpgradeFixture(t)
			switch mode {
			case "native", "native-adopt":
				if err := os.Remove(filepath.Join(root, pypiVenvMetadata)); err != nil {
					t.Fatal(err)
				}
			case "legacy":
				if err := os.Mkdir(filepath.Join(plan.site, "legacy.egg-info"), 0o700); err != nil {
					t.Fatal(err)
				}
			case "direct-source":
				if err := os.WriteFile(filepath.Join(plan.site, "fixture-1.0.dist-info/direct_url.json"), []byte("{}"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "oversize", "symlink":
				dir := filepath.Join(plan.site, "native-1.0.dist-info")
				if err := os.Mkdir(dir, 0o700); err != nil {
					t.Fatal(err)
				}
				if mode == "oversize" {
					if err := os.WriteFile(filepath.Join(dir, "METADATA"), []byte(strings.Repeat("a", (1<<20)+1)), 0o600); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := os.Symlink(filepath.Join(plan.site, "fixture-1.0.dist-info/METADATA"), filepath.Join(dir, "METADATA")); err != nil {
						t.Fatal(err)
					}
				}
			}
			resolver := &pypiUpgradeSelectionFixture{}
			promoter, _ := newPyPIPromotion(filepath.Join(root, "staging"), &venvGuardCheckingRunner{}, "linux", "amd64")
			preparation, _ := NewPyPIUpdatePreparation(promoter, resolver)
			source, _ := domain.NewSourceID("pypi")
			session, err := preparation.BeginInstallUpdate(context.Background(), source, "fixture", install, mode == "native-adopt")
			if err == nil || session != nil || resolver.calls != 0 {
				t.Fatal("unproven inventory admitted", err)
			}
			if mode == "native" && ports.InstallUpdateFailureCode(err) != "UPDATE_ADOPTION_REQUIRED" || mode == "native-adopt" && ports.InstallUpdateFailureCode(err) != "UPDATE_ORIGINAL_STATE_UNSUPPORTED" {
				t.Fatal("native ownership explanation lost", err)
			}
			body, err := os.ReadFile(filepath.Join(plan.site, "fixture.py"))
			if err != nil || string(body) != "old payload" {
				t.Fatal("failed preparation changed payload", err)
			}
			if _, err := os.Lstat(filepath.Join(root, ".heliopause-pypi-transaction.lock")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("failed preparation leaked guard", err)
			}
		})
	}
}
