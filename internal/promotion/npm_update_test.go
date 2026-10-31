package promotion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
)

type npmUpdateResolverFixture struct {
	resolution domain.DependencyResolution
	before     func()
	calls      int
}

func TestNPMUpdateRejectsSubstitutedSelectionBeforeInspection(t *testing.T) {
	for _, mode := range []string{"manifest-range", "lock-version", "missing-control", "lock-digest"} {
		t.Run(mode, func(t *testing.T) {
			root, install, bundle, _, resolution := publicNPMUpdateFixture(t)
			controls := resolution.ProjectControls()
			manifestIndex, lockIndex := 0, 0
			for i, control := range controls {
				if control.Name() == "package.json" {
					manifestIndex = i
				} else if control.Name() == "package-lock.json" {
					lockIndex = i
				}
			}
			switch mode {
			case "manifest-range":
				controls[manifestIndex], _ = domain.NewProjectControlFile("package.json", []byte(strings.ReplaceAll(string(controls[manifestIndex].Body()), "^1.0.0", "*")), true)
			case "lock-version":
				controls[lockIndex], _ = domain.NewProjectControlFile("package-lock.json", []byte(strings.ReplaceAll(string(controls[lockIndex].Body()), `"version":"1.0.0"`, `"version":"1.1.0"`)), true)
			case "missing-control":
				controls = controls[:1]
			case "lock-digest":
				digest, _ := domain.NewSHA256Digest(strings.Repeat("e", 64))
				resolution, _ = domain.NewDependencyResolution(resolution.Graph(), resolution.RuntimeIdentity(), digest)
			}
			var err error
			resolution, err = resolution.WithProjectControls(controls)
			if err != nil {
				t.Fatal(err)
			}
			resolver := &npmUpdateResolverFixture{resolution: resolution}
			promoter, _ := newNPMPromotion(filepath.Join(root, "staging"), &fixturePromotionRunner{bundle: bundle}, "linux", "amd64")
			preparation, _ := NewNPMUpdatePreparation(promoter, resolver)
			source, _ := domain.NewSourceID("npm")
			session, err := preparation.BeginInstallUpdate(context.Background(), source, "", install, true)
			if err != nil {
				t.Fatal(err)
			}
			selected, err := session.ResolveDependencies(context.Background(), session.Reference(), install)
			if err == nil || len(selected.Graph().Nodes()) != 0 {
				t.Fatal("substituted selection escaped the guard", err)
			}
			if err := session.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(filepath.Join(install.Target().String(), ".heliopause")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("uninspected selection registered ownership", err)
			}
		})
	}
}

func TestNPMUpdateDoesNotReleaseReplacedGuardOrOverwriteOriginal(t *testing.T) {
	root, install, bundle, _, resolution := publicNPMUpdateFixture(t)
	project := install.Target().String()
	resolver := &npmUpdateResolverFixture{resolution: resolution}
	promoter, _ := newNPMPromotion(filepath.Join(root, "staging"), &fixturePromotionRunner{bundle: bundle}, "linux", "amd64")
	preparation, _ := NewNPMUpdatePreparation(promoter, resolver)
	source, _ := domain.NewSourceID("npm")
	session, err := preparation.BeginInstallUpdate(context.Background(), source, "", install, true)
	if err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(project, ".heliopause-npm-transaction.lock")
	if err := os.Rename(lock, filepath.Join(project, "retained-original-lock")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lock, []byte("another transaction"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := session.ResolveDependencies(context.Background(), session.Reference(), install); err == nil || resolver.calls != 0 {
		t.Fatal("replaced guard allowed selection", err)
	}
	if err := session.Close(); err == nil {
		t.Fatal("replaced guard reported successful cleanup")
	}
	body, err := os.ReadFile(lock)
	if err != nil || string(body) != "another transaction" {
		t.Fatal("foreign transaction lock removed", err)
	}
	if _, err := os.Lstat(filepath.Join(project, ".heliopause")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("uninspected update changed ownership", err)
	}
}

func TestNPMUpdateCommitUsesEarlyInventoryAndRollsBackPartialPublication(t *testing.T) {
	for _, mode := range []string{"drift-between-check-and-backup", "partial-publication"} {
		t.Run(mode, func(t *testing.T) {
			root, install, bundle, _, resolution := publicNPMUpdateFixture(t)
			project := install.Target().String()
			before, err := snapshotNPMTransactionMember(filepath.Join(project, "node_modules"))
			if err != nil {
				t.Fatal(err)
			}
			manifest, _ := os.ReadFile(filepath.Join(project, "package.json"))
			lock, _ := os.ReadFile(filepath.Join(project, "package-lock.json"))
			promoter, _ := newNPMPromotion(filepath.Join(root, "staging"), &fixturePromotionRunner{bundle: bundle}, "linux", "amd64")
			preparation, _ := NewNPMUpdatePreparation(promoter, &npmUpdateResolverFixture{resolution: resolution})
			source, _ := domain.NewSourceID("npm")
			guarded, err := preparation.BeginInstallUpdate(context.Background(), source, "", install, true)
			if err != nil {
				t.Fatal(err)
			}
			session := guarded.(*npmUpdateSession)
			workspace, err := session.plan.privateWorkspace()
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(workspace)
			transaction, err := beginNPMProjectTransaction(session.plan, workspace)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "drift-between-check-and-backup" {
				// Model an external edit after commit's initial whole-project
				// check but before it reaches the node_modules backup member.
				changed := filepath.Join(project, "node_modules/haa-promotion-fixture/index.js")
				if err := os.WriteFile(changed, []byte("concurrent-user-edit"), 0o600); err != nil {
					t.Fatal(err)
				}
				cause := transaction.backupCurrent()
				if cause == nil {
					t.Fatal("fresh backup inventory replaced original-state guard")
				}
				if err := transaction.fail(cause); !errors.Is(err, cause) {
					t.Fatal("backup failure was swallowed", err)
				}
				body, _ := os.ReadFile(changed)
				if string(body) != "concurrent-user-edit" {
					t.Fatal("concurrent user's file overwritten")
				}
			} else {
				// Publish package.json, then fail on the missing lock. This
				// exercises the existing real rollback using an update plan.
				if err := os.Remove(filepath.Join(workspace, "package-lock.json")); err != nil {
					t.Fatal(err)
				}
				if err := transaction.commit(); err == nil {
					t.Fatal("partial publication reported update success")
				}
				if err := before.verify(filepath.Join(project, "node_modules")); err != nil {
					t.Fatal("original payload not restored", err)
				}
			}
			for name, want := range map[string][]byte{"package.json": manifest, "package-lock.json": lock} {
				body, err := os.ReadFile(filepath.Join(project, name))
				if err != nil || string(body) != string(want) {
					t.Fatal("original control not restored", name, err)
				}
			}
			if err := rejectInterruptedNPMTransaction(project); err != nil {
				t.Fatal("confirmed rollback left recovery state", err)
			}
			if err := session.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNPMUpdateRollbackConflictRetainsRecoveryStateAndPrimaryFailure(t *testing.T) {
	root, install, bundle, _, resolution := publicNPMUpdateFixture(t)
	project := install.Target().String()
	original, _ := os.ReadFile(filepath.Join(project, "package.json"))
	promoter, _ := newNPMPromotion(filepath.Join(root, "staging"), &fixturePromotionRunner{bundle: bundle}, "linux", "amd64")
	preparation, _ := NewNPMUpdatePreparation(promoter, &npmUpdateResolverFixture{resolution: resolution})
	source, _ := domain.NewSourceID("npm")
	guarded, err := preparation.BeginInstallUpdate(context.Background(), source, "", install, true)
	if err != nil {
		t.Fatal(err)
	}
	session := guarded.(*npmUpdateSession)
	workspace, err := session.plan.privateWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workspace)
	transaction, err := beginNPMProjectTransaction(session.plan, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if err := transaction.backupCurrent(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "package.json"), []byte("foreign concurrent file"), 0o600); err != nil {
		t.Fatal(err)
	}
	primary := errors.New("injected update commit failure")
	failed := transaction.fail(primary)
	if !errors.Is(failed, primary) || !strings.Contains(failed.Error(), "rollback is incomplete") {
		t.Fatal("rollback uncertainty or primary failure lost", failed)
	}
	foreign, _ := os.ReadFile(filepath.Join(project, "package.json"))
	retained, err := os.ReadFile(filepath.Join(transaction.backup, "package.json"))
	if string(foreign) != "foreign concurrent file" || err != nil || string(retained) != string(original) {
		t.Fatal("foreign file overwritten or recovery original discarded", err)
	}
	if err := rejectInterruptedNPMTransaction(project); err == nil {
		t.Fatal("uncertain rollback permits another update")
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if next, err := preparation.BeginInstallUpdate(context.Background(), source, "", install, true); err == nil || next != nil {
		t.Fatal("recovery state silently adopted", err)
	}
}

func (r *npmUpdateResolverFixture) ResolveDependencyUpdate(_ context.Context, _ domain.ArtifactReference, _ domain.InstallContext, controls []domain.ProjectControlFile) (domain.DependencyResolution, error) {
	r.calls++
	if r.before != nil {
		r.before()
	}
	if len(controls) != 2 {
		return domain.DependencyResolution{}, errors.New("missing original controls")
	}
	return r.resolution, nil
}

type npmUpdateRunnerFixture struct {
	base  fixturePromotionRunner
	after func()
	err   error
}

func (r *npmUpdateRunnerFixture) Run(ctx context.Context, root string, args []string) error {
	if err := r.base.Run(ctx, root, args); err != nil {
		return err
	}
	if r.after != nil {
		r.after()
	}
	return r.err
}

func publicNPMUpdateFixture(t *testing.T) (string, domain.InstallContext, domain.VerifiedBundle, domain.StagedSet, domain.DependencyResolution) {
	t.Helper()
	root := realPromotionRoot(t)
	project := filepath.Join(root, "project")
	if err := os.Mkdir(project, 0o700); err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(project)
	install, _ := domain.NewNPMProjectInstallContext(target)
	old, _ := validStagedNPMFixtureForContext(t, root, install)
	node := old.Set().Inspected().Graph().Nodes()[0]
	manifest := []byte(`{"name":"user-project","scripts":{"test":"do-not-execute"},"dependencies":{"haa-promotion-fixture":"^1.0.0"}}`)
	lock, err := json.Marshal(promotionLock{LockfileVersion: 3, Packages: map[string]promotionPackage{"": {Name: "user-project", Dependencies: map[string]string{"haa-promotion-fixture": "^1.0.0"}}, node.RecordPath(): {Version: "1.0.0", Resolved: node.Artifact().AcquisitionLocator(), Integrity: node.Artifact().DeclaredIntegrity()}}})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := artifactnpm.ProjectRootReference(manifest)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := artifactnpm.ParseProjectControls(ref, manifest, lock)
	if err != nil {
		t.Fatal(err)
	}
	prior := old.Set().Inspected().Inspections()[0]
	inspection, err := domain.NewDependencyInspection(graph.Primary(), prior.RunID(), prior.Artifact(), prior.Checks(), prior.Evidence(), prior.PolicyDecision())
	if err != nil {
		t.Fatal(err)
	}
	inspected, err := domain.NewInspectedDependencySet(graph, []domain.DependencyInspection{inspection})
	if err != nil {
		t.Fatal(err)
	}
	verified, err := domain.NewVerifiedSet(inspected, old.Set().Decision())
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(lock)
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	controls := []domain.ProjectControlFile{}
	for _, item := range []struct {
		name string
		body []byte
	}{{"package.json", manifest}, {"package-lock.json", lock}} {
		control, err := domain.NewProjectControlFile(item.name, item.body, true)
		if err != nil {
			t.Fatal(err)
		}
		controls = append(controls, control)
		if err := os.WriteFile(filepath.Join(project, item.name), item.body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	operation, _ := domain.NewOperationID()
	bundle, err := evidence.BuildVerifiedBundle(evidence.ManifestContext{OperationID: operation, InstallContext: install, ResolverRuntime: "fixture-runtime", LockfileDigest: digest, ProjectControls: controls}, verified)
	if err != nil {
		t.Fatal(err)
	}
	staging, err := NewLocalStaging(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "staging"))
	if err != nil {
		t.Fatal(err)
	}
	staged, err := staging.Stage(context.Background(), bundle)
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := domain.NewDependencyResolution(graph, "fixture-runtime", digest)
	if err != nil {
		t.Fatal(err)
	}
	resolution, err = resolution.WithProjectControls(controls)
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(project, node.RecordPath())
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"package.json": `{"name":"haa-promotion-fixture","version":"1.0.0"}`, "index.js": "old-native-content"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(project, "README.user"), []byte("keep user files"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, install, bundle, staged, resolution
}

func TestNPMUpdateAdoptsOnlyExplicitNativeStateAndRetainsPublicControls(t *testing.T) {
	root, install, bundle, staged, resolution := publicNPMUpdateFixture(t)
	runner := &npmUpdateRunnerFixture{base: fixturePromotionRunner{bundle: bundle}}
	p, _ := newNPMPromotion(filepath.Join(root, "staging"), runner, "linux", "amd64")
	resolver := &npmUpdateResolverFixture{resolution: resolution}
	preparation, _ := NewNPMUpdatePreparation(p, resolver)
	source, _ := domain.NewSourceID("npm")
	if session, err := preparation.BeginInstallUpdate(context.Background(), source, "", install, false); err == nil || session != nil || resolver.calls != 0 {
		t.Fatal("implicit native ownership accepted", err)
	}
	for _, adopt := range []bool{true, false} {
		session, err := preparation.BeginInstallUpdate(context.Background(), source, "haa-promotion-fixture", install, adopt)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := session.ResolveDependencies(context.Background(), session.Reference(), install); err != nil {
			t.Fatal(err)
		}
		if _, err := session.Promote(context.Background(), staged, bundle, install); err != nil {
			t.Fatal(err)
		}
		if err := session.Close(); err != nil {
			t.Fatal(err)
		}
		if err := session.Close(); err != nil {
			t.Fatal("close not idempotent", err)
		}
		for _, control := range resolution.ProjectControls() {
			body, err := os.ReadFile(filepath.Join(install.Target().String(), control.Name()))
			if err != nil || string(body) != string(control.Body()) {
				t.Fatal("public selection replaced by offline recipe", control.Name(), err)
			}
		}
		plan, err := freezeNPMProject(install.Target().String())
		if err != nil || plan.verifyManagedOrEmpty() != nil {
			t.Fatal("committed ownership does not bind public controls", err)
		}
		body, _ := os.ReadFile(filepath.Join(install.Target().String(), "README.user"))
		if string(body) != "keep user files" {
			t.Fatal("unrelated user files replaced")
		}
	}
}

func TestNPMUpdateStopsOnConcurrentOriginalDriftCancellationAndStagedSwap(t *testing.T) {
	for _, mode := range []string{"during-resolve", "during-offline-install", "cancel", "staged-swap", "install-error"} {
		t.Run(mode, func(t *testing.T) {
			root, install, bundle, staged, resolution := publicNPMUpdateFixture(t)
			project := install.Target().String()
			original, _ := os.ReadFile(filepath.Join(project, "package-lock.json"))
			changed := filepath.Join(project, "node_modules/haa-promotion-fixture/index.js")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			resolver := &npmUpdateResolverFixture{resolution: resolution}
			runner := &npmUpdateRunnerFixture{base: fixturePromotionRunner{bundle: bundle}}
			mutate := func() {
				if err := os.WriteFile(changed, []byte("user-concurrent-change"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			switch mode {
			case "during-resolve":
				resolver.before = mutate
			case "during-offline-install":
				runner.after = mutate
			case "cancel":
				runner.after = cancel
			case "install-error":
				runner.err = errors.New("offline install failed")
			case "staged-swap":
				digest := bundle.Set().Inspected().Inspections()[0].Artifact().Digest().String()
				path := filepath.Join(root, "staging", bundle.ManifestID().String(), "artifacts", digest+".tgz")
				// Deliberately break only this disposable fixture's read-only mode
				// to test content re-verification after a storage-integrity fault.
				if err := os.Chmod(path, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("swapped"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			promoter, _ := newNPMPromotion(filepath.Join(root, "staging"), runner, "linux", "amd64")
			preparation, _ := NewNPMUpdatePreparation(promoter, resolver)
			source, _ := domain.NewSourceID("npm")
			session, err := preparation.BeginInstallUpdate(ctx, source, "", install, true)
			if err != nil {
				t.Fatal(err)
			}
			_, err = session.ResolveDependencies(ctx, session.Reference(), install)
			if err == nil {
				_, err = session.Promote(ctx, staged, bundle, install)
			}
			if err == nil {
				t.Fatal("invalid update reported success")
			}
			if err := session.Close(); err != nil {
				t.Fatal(err)
			}
			after, _ := os.ReadFile(filepath.Join(project, "package-lock.json"))
			if string(after) != string(original) {
				t.Fatal("original controls changed on failure")
			}
			if _, err := os.Lstat(filepath.Join(project, ".heliopause")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("failed native update registered ownership", err)
			}
			if _, err := os.Lstat(filepath.Join(project, ".heliopause-npm-transaction.lock")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("guard lock leaked", err)
			}
			if mode == "during-resolve" || mode == "during-offline-install" {
				body, _ := os.ReadFile(changed)
				if string(body) != "user-concurrent-change" {
					t.Fatal("external user change overwritten")
				}
			}
		})
	}
}

func TestNPMUpdateRejectsForeignNamespaceAndUnsupportedOriginalsBeforeSelection(t *testing.T) {
	for _, mode := range []string{"foreign-file", "foreign-package", "foreign-metadata", "legacy", "selector-version", "wrong-package", "oversize"} {
		t.Run(mode, func(t *testing.T) {
			root, install, bundle, _, resolution := publicNPMUpdateFixture(t)
			project := install.Target().String()
			selector := ""
			switch mode {
			case "foreign-file":
				if err := os.WriteFile(filepath.Join(project, "node_modules/user-note"), []byte("do not delete"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "foreign-package":
				directory := filepath.Join(project, "node_modules/foreign")
				if err := os.Mkdir(directory, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(directory, "package.json"), []byte(`{"name":"foreign","version":"1.0.0"}`), 0o600); err != nil {
					t.Fatal(err)
				}
			case "foreign-metadata":
				if err := os.Mkdir(filepath.Join(project, ".heliopause"), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(project, ".heliopause/user-note"), []byte("do not delete"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "legacy":
				if err := os.WriteFile(filepath.Join(project, "package.json"), []byte(`{"dependencies":{"haa-promotion-fixture":"file:.heliopause/artifacts/a.tgz"}}`), 0o600); err != nil {
					t.Fatal(err)
				}
			case "selector-version":
				selector = "haa-promotion-fixture@2.0.0"
			case "wrong-package":
				selector = "another"
			case "oversize":
				if err := os.Truncate(filepath.Join(project, "package.json"), artifactnpm.ProjectControlLimit+1); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := os.ReadFile(filepath.Join(project, "package.json"))
			resolver := &npmUpdateResolverFixture{resolution: resolution}
			promoter, _ := newNPMPromotion(filepath.Join(root, "staging"), &fixturePromotionRunner{bundle: bundle}, "linux", "amd64")
			preparation, _ := NewNPMUpdatePreparation(promoter, resolver)
			source, _ := domain.NewSourceID("npm")
			if session, err := preparation.BeginInstallUpdate(context.Background(), source, selector, install, true); err == nil || session != nil || resolver.calls != 0 {
				t.Fatal("unsupported/foreign update accepted", err)
			}
			after, _ := os.ReadFile(filepath.Join(project, "package.json"))
			if string(before) != string(after) {
				t.Fatal("rejected original state changed")
			}
			if _, err := os.Lstat(filepath.Join(project, ".heliopause-npm-transaction.lock")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("failed preparation leaked guard")
			}
		})
	}
}
