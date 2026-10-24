package promotion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type nativeOriginalPorts struct {
	root, archive      string
	resolved           domain.ResolvedArtifact
	resolves, acquires int
	corrupt            bool
	before             func()
}

func TestNativePyPIEnrollmentRejectsOverlappingCacheScopesBeforeMutation(t *testing.T) {
	for _, mode := range []string{"target-intake", "target-staging", "nested-intake"} {
		t.Run(mode, func(t *testing.T) {
			_, plan, install := managedPyPIUpgradeFixture(t)
			if err := os.Chmod(plan.root, 0o750); err != nil {
				t.Fatal(err)
			}
			root := realPromotionRoot(t)
			intakeRoot, stagingRoot := filepath.Join(root, "intake"), filepath.Join(root, "staging")
			if mode == "target-intake" {
				intakeRoot = plan.root
			}
			if mode == "target-staging" {
				stagingRoot = plan.root
			}
			if mode == "nested-intake" {
				intakeRoot = filepath.Join(plan.root, "uncreated-cache")
			}
			original := &nativeOriginalPorts{root: intakeRoot}
			update := &pypiUpgradeSelectionFixture{}
			promoter, _ := newPyPIPromotion(stagingRoot, &fixturePyPIUpgradeRunner{t: t}, "linux", "amd64")
			base, _ := NewPyPIUpdatePreparation(promoter, update)
			preparation, err := base.WithNativeEnrollment(original, original, intakeRoot)
			if err != nil {
				t.Fatal(err)
			}
			source, _ := domain.NewSourceID("pypi")
			session, err := preparation.BeginInstallUpdate(context.Background(), source, "fixture", install, true)
			if err == nil || session != nil || ports.InstallUpdateFailureCode(err) != "UPDATE_ORIGINAL_STATE_UNSUPPORTED" || original.resolves != 0 || original.acquires != 0 || update.calls != 0 {
				t.Fatal("overlapping cache escaped original-state preflight", err)
			}
			if info, err := os.Stat(plan.root); err != nil || info.Mode().Perm() != 0o750 {
				t.Fatal("unsupported scope changed original root permissions", err)
			}
			if _, err := os.Lstat(filepath.Join(plan.root, ".heliopause-pypi-transaction.lock")); !os.IsNotExist(err) {
				t.Fatal("unsupported scope created a guard", err)
			}
		})
	}
}

func (p *nativeOriginalPorts) ResolveDependencies(_ context.Context, reference domain.ArtifactReference, _ domain.InstallContext) (domain.DependencyResolution, error) {
	p.resolves++
	if reference.Source() != p.resolved.Identity().Source() || reference.Locator() != p.resolved.Identity().Name()+"@"+p.resolved.Identity().Version() {
		return domain.DependencyResolution{}, errors.New("unexpected original footprint reference")
	}
	if p.before != nil {
		p.before()
	}
	nodeID, _ := domain.NewDependencyNodeID("original-native-wheel")
	node, _ := domain.NewLockedDependencyWithRecordPath(nodeID, domain.DependencyPrimary, p.resolved, "haa_promotion_fixture-1.0.0-py3-none-any.whl", false)
	graph, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{node}, nil)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	lock, _ := domain.NewSHA256Digest(strings.Repeat("d", 64))
	python := sandbox.PinnedPythonRuntime()
	return domain.NewDependencyResolution(graph, "python:"+python.PythonVersion+";pip:"+python.PipVersion+";target:"+python.InterpreterTag+"/"+python.ABITag+"/"+python.PlatformTag+";source:pypi", lock)
}

func (*nativeOriginalPorts) Resolve(context.Context, domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	return domain.ResolvedArtifact{}, errors.New("use the isolated resolver")
}
func (p *nativeOriginalPorts) Acquire(_ context.Context, run domain.RunID, resolved domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	p.acquires++
	body, err := os.ReadFile(p.archive)
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	if p.corrupt {
		body = []byte("substituted old wheel")
	}
	directory := filepath.Join(p.root, run.String())
	if err := os.Mkdir(directory, 0o700); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	if err := os.WriteFile(filepath.Join(directory, "wheel.whl"), body, 0o600); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	if err := os.WriteFile(filepath.Join(directory, "filename"), []byte("haa_promotion_fixture-1.0.0-py3-none-any.whl"), 0o600); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	digest, _ := domain.NewSHA256Digest(strings.TrimPrefix(resolved.DeclaredIntegrity(), "sha256:"))
	return domain.NewAcquiredArtifactWithDeclaredIntegrity(resolved.Identity(), digest, "intake:"+run.String()+":wheel", uint64(len(body)), resolved.DeclaredIntegrity())
}

type nativeUpdateRunner struct {
	fixturePyPIUpgradeRunner
	metadataCalls int
	target        string
}

func (r *nativeUpdateRunner) Run(ctx context.Context, project string, args []string) error {
	for _, arg := range args {
		if arg == nativePythonRuntimeMetadataScript {
			r.metadataCalls++
			joined := strings.Join(args, " ")
			if !strings.Contains(joined, "--network none --read-only") || !strings.Contains(joined, sandbox.PinnedPythonRuntime().ImageReference+" -I -c ") || strings.Contains(joined, r.target) {
				r.t.Fatal("native runtime query exposed original packages or lost its fixed scope", joined)
			}
			return os.WriteFile(filepath.Join(project, "runtime.json"), []byte(`{"version":"3.14.7","magic":"2b0e0d0a","cache_tag":"cpython-314"}`), 0o600)
		}
	}
	return r.fixturePyPIUpgradeRunner.Run(ctx, project, args)
}

func nativeVerifiedBundle(t *testing.T, seed domain.VerifiedBundle, resolution domain.DependencyResolution, install domain.InstallContext) domain.VerifiedBundle {
	t.Helper()
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
	bundle, err := evidence.BuildVerifiedBundle(evidence.ManifestContext{OperationID: op, InstallContext: install, ResolverRuntime: resolution.RuntimeIdentity(), LockfileDigest: resolution.LockfileDigest(), ProjectControls: resolution.ProjectControls()}, set)
	if err != nil {
		t.Fatal(err)
	}
	return bundle
}

func TestNativePyPIEnrollmentIsExplicitAndPublishesOnlyAfterNewSetApproval(t *testing.T) {
	for _, mode := range []string{"adopt", "no-adopt", "proof-only", "old-intake-tamper", "claimed-user-file", "source-drift", "partial-publication", "recovery-required"} {
		t.Run(mode, func(t *testing.T) {
			fixtureMode := "cache"
			if mode == "claimed-user-file" {
				fixtureMode = "user-file"
			}
			plan, initial, _, _, oldDigest, oldArchive := nativeWheelOwnershipFixture(t, fixtureMode)
			if err := initial.close(); err != nil {
				t.Fatal(err)
			}
			metadataPath := filepath.Join(plan.site, "haa_promotion_fixture-1.0.0.dist-info/METADATA")
			metadataBefore, err := os.ReadFile(metadataPath)
			if err != nil {
				t.Fatal(err)
			}
			recordPath := filepath.Join(plan.site, "haa_promotion_fixture-1.0.0.dist-info/RECORD")
			recordBefore, err := os.ReadFile(recordPath)
			if err != nil {
				t.Fatal(err)
			}
			cachePath := filepath.Join(plan.site, "haa_promotion_fixture/__pycache__/__init__.cpython-314.pyc")
			cacheBefore, _ := os.ReadFile(cachePath)
			root := realPromotionRoot(t)
			seed, _ := validStagedPyPIFixture(t, root)
			source, _ := domain.NewSourceID("pypi")
			identity, _ := domain.NewResolvedArtifactIdentity(source, "haa-promotion-fixture", "1.0.0", "wheel")
			resolved, _ := domain.NewResolvedArtifact(identity, "https://files.pythonhosted.org/packages/haa_promotion_fixture-1.0.0-py3-none-any.whl", "sha256:"+oldDigest)
			original := &nativeOriginalPorts{root: filepath.Join(root, "intake"), archive: oldArchive, resolved: resolved, corrupt: mode == "old-intake-tamper"}
			if mode == "source-drift" {
				original.before = func() {
					if err := os.WriteFile(filepath.Join(plan.site, "haa_promotion_fixture/__init__.py"), []byte("user edit"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			}
			runner := &nativeUpdateRunner{fixturePyPIUpgradeRunner: fixturePyPIUpgradeRunner{t: t}, target: plan.root}
			promoter, _ := newPyPIPromotion(filepath.Join(root, "staging"), runner, "linux", "amd64")
			update := &reportedPyPIUpgradeFixture{t: t, artifact: seed.Set().Inspected().Graph().Nodes()[0].Artifact()}
			base, _ := NewPyPIUpdatePreparation(promoter, update)
			preparation, err := base.WithNativeEnrollment(original, original, original.root)
			if err != nil {
				t.Fatal(err)
			}
			if base.native != nil {
				t.Fatal("constructor copy mutated the existing preparation")
			}
			target, _ := domain.NewInstallTarget(plan.root)
			install, _ := domain.NewPythonVenvInstallContext(target)
			guarded, err := preparation.BeginInstallUpdate(context.Background(), source, "haa-promotion-fixture", install, mode != "no-adopt")
			if mode == "no-adopt" {
				if err == nil || guarded != nil || original.resolves != 0 || original.acquires != 0 || runner.metadataCalls != 0 {
					t.Fatal("native environment was silently enrolled", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			session := guarded.(*pypiUpdateSession)
			if len(session.transaction.current.Files) != 0 {
				t.Fatal("untrusted native claims became owners before independent proof")
			}
			resolution, err := session.ResolveDependencies(context.Background(), session.Reference(), install)
			if mode == "old-intake-tamper" || mode == "claimed-user-file" || mode == "source-drift" {
				if err == nil || len(resolution.Graph().Nodes()) != 0 || len(session.transaction.current.Files) != 0 || runner.calls != 0 {
					t.Fatal("unproven originals reached inspection/publication", err)
				}
				if err := session.Close(); err != nil {
					t.Fatal(err)
				}
				if _, err := os.Lstat(filepath.Join(plan.root, pypiVenvMetadata)); !os.IsNotExist(err) {
					t.Fatal("failed proof wrote enrollment", err)
				}
				if mode == "claimed-user-file" {
					if body, err := os.ReadFile(filepath.Join(plan.site, "user-notes.txt")); err != nil || string(body) != "preserve unrelated user data" {
						t.Fatal("forged RECORD removed user data", err)
					}
				}
				if mode == "source-drift" {
					if body, err := os.ReadFile(filepath.Join(plan.site, "haa_promotion_fixture/__init__.py")); err != nil || string(body) != "user edit" {
						t.Fatal("concurrent original edit lost", err)
					}
				}
				return
			}
			if err != nil || len(resolution.ProjectControls()) != 4 || original.resolves != 1 || original.acquires != 1 || runner.metadataCalls != 1 {
				t.Fatal("independent canonical footprint/runtime binding lost", err)
			}
			if _, err := os.Lstat(filepath.Join(plan.root, pypiVenvMetadata)); !os.IsNotExist(err) {
				t.Fatal("old ownership proof was persisted before new approval", err)
			}
			if mode == "proof-only" {
				if _, err := session.Promote(context.Background(), domain.StagedSet{}, domain.VerifiedBundle{}, install); err == nil || runner.calls != 0 {
					t.Fatal("old footprint substituted for new inspection")
				}
				if err := session.Close(); err != nil {
					t.Fatal(err)
				}
				return
			}
			bundle := nativeVerifiedBundle(t, seed, resolution, install)
			staging, _ := NewLocalStaging(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "staging"))
			staged, err := staging.Stage(context.Background(), bundle)
			if err != nil {
				t.Fatal(err)
			}
			cause := errors.New("controlled native publication failure")
			if mode == "partial-publication" || mode == "recovery-required" {
				session.transaction.fault = func(step string) error {
					if step == "after-publish:site" {
						return cause
					}
					if step == "rollback" && mode == "recovery-required" {
						return errors.New("controlled native rollback failure")
					}
					return nil
				}
			}
			_, err = session.Promote(context.Background(), staged, bundle, install)
			if (mode == "adopt") != (err == nil) || mode != "adopt" && !errors.Is(err, cause) {
				t.Fatal("native publication/primary result", mode, err)
			}
			if err := session.Close(); err != nil {
				t.Fatal(err)
			}
			if mode == "recovery-required" {
				if !session.transaction.recovery || !strings.Contains(err.Error(), "rollback incomplete") {
					t.Fatal("native recovery uncertainty hidden", err)
				}
				if _, err := preparation.BeginInstallUpdate(context.Background(), source, "haa-promotion-fixture", install, true); err == nil {
					t.Fatal("uncertain native commit was silently retried")
				}
				return
			}
			if mode == "partial-publication" {
				metadata, _ := os.ReadFile(metadataPath)
				record, _ := os.ReadFile(recordPath)
				cache, _ := os.ReadFile(cachePath)
				if string(metadata) != string(metadataBefore) || string(record) != string(recordBefore) || string(cache) != string(cacheBefore) {
					t.Fatal("native rollback did not restore exact original payload/record/cache")
				}
				if _, err := os.Lstat(filepath.Join(plan.root, pypiVenvMetadata)); !os.IsNotExist(err) {
					t.Fatal("rollback invented a prior HAA ledger", err)
				}
				return
			}
			state, present, err := plan.readState()
			if err != nil || !present || plan.verifyState(state) != nil {
				t.Fatal("native enrollment did not publish validated ownership", err)
			}
			if _, err := os.Lstat(cachePath); !os.IsNotExist(err) {
				t.Fatal("old unapproved bytecode survived new exact output", err)
			}
			if _, err := os.Lstat(filepath.Join(plan.root, ".heliopause-pypi-transaction.lock")); !os.IsNotExist(err) {
				t.Fatal("successful native enrollment leaked a guard", err)
			}
		})
	}
}
