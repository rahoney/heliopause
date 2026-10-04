package promotion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestGoProjectGuardCommitsOneFrozenApprovedSelection(t *testing.T) {
	p, guard, update, staged := approvedGoFixture(t)
	if _, err := p.Begin(context.Background(), update.Snapshot().Context()); err == nil {
		t.Fatal("concurrent transaction entered")
	}
	if err := guard.Commit(context.Background(), update, staged); err != nil {
		t.Fatal(err)
	}
	for _, control := range update.SelectedControls() {
		body, err := os.ReadFile(filepath.Join(guard.plan.root, control.Name()))
		if err != nil || string(body) != string(control.Body()) {
			t.Fatalf("selected %s was not committed: %v", control.Name(), err)
		}
	}
	if err := guard.Close(); err != nil {
		t.Fatal(err)
	}
	retained, err := p.Begin(context.Background(), update.Snapshot().Context())
	if err != nil {
		t.Fatalf("trusted retained approval: %v", err)
	}
	if err := retained.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(p.cache.evidenceRoot); err != nil {
		t.Fatal(err)
	}
	if g, err := p.Begin(context.Background(), update.Snapshot().Context()); err == nil || g != nil {
		t.Fatal("retained approval outlived required Evidence")
	}
}

func TestGoProjectGuardRejectsControlDrift(t *testing.T) {
	for _, name := range []string{"content", "presence", "inode", "cache", "evidence"} {
		t.Run(name, func(t *testing.T) {
			p, guard, update, staged := approvedGoFixture(t)
			defer guard.Close()
			project := guard.plan.root
			original := update.OriginalControls()[0].Body()
			switch name {
			case "content":
				if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.com/changed\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "presence":
				if err := os.WriteFile(filepath.Join(project, "go.sum"), nil, 0o600); err != nil {
					t.Fatal(err)
				}
			case "inode":
				if err := os.WriteFile(filepath.Join(project, "replacement"), original, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(filepath.Join(project, "replacement"), filepath.Join(project, "go.mod")); err != nil {
					t.Fatal(err)
				}
			case "cache":
				dir, err := p.cache.OpenProjectCache(context.Background(), staged)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "poison.go"), []byte("poison"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "evidence":
				if err := os.RemoveAll(p.cache.evidenceRoot); err != nil {
					t.Fatal(err)
				}
			}
			if err := guard.Commit(context.Background(), update, staged); err == nil {
				t.Fatal("invalid frozen transaction committed")
			}
			if _, err := os.Lstat(filepath.Join(p.stateRoot, guard.stateName)); !os.IsNotExist(err) {
				t.Fatal("failed transaction published retained authority")
			}
			if name != "content" {
				body, err := os.ReadFile(filepath.Join(project, "go.mod"))
				if err != nil || string(body) != string(original) {
					t.Fatal("failed transaction changed module controls")
				}
			}
		})
	}
}

func TestGoRawProjectMarkerCannotAuthorizeMutation(t *testing.T) {
	project := writeManagedGoProject(t)
	root := canonicalGoTestRoot(t)
	cache, err := newGoCacheForTest(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := NewGoProjectPromotion(cache, filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(project)
	install, _ := domain.NewInstallContext(target)
	if guard, err := p.Begin(context.Background(), install); err == nil || guard != nil {
		t.Fatal("raw matching checksum marker restored approval")
	}
	if _, err := os.Lstat(filepath.Join(project, ".heliopause-go-transaction.lock")); !os.IsNotExist(err) {
		t.Fatal("failed guard leaked its lock")
	}
}

func TestGoProjectGuardRejectsForeignApproval(t *testing.T) {
	_, guard, update, _ := approvedGoFixture(t)
	defer guard.Close()
	_, foreign, _, staged := approvedGoFixture(t)
	defer foreign.Close()
	if err := guard.Commit(context.Background(), update, staged); err == nil {
		t.Fatal("another project's cache approval authorized publication")
	}
	if err := guard.VerifyUnchanged(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestGoProjectGuardDoesNotRemoveForeignLock(t *testing.T) {
	_, guard, _, _ := approvedGoFixture(t)
	path := guard.guard.path
	if err := os.WriteFile(path+".replacement", []byte("foreign owner"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path+".replacement", path); err != nil {
		t.Fatal(err)
	}
	if err := guard.VerifyUnchanged(context.Background()); err == nil {
		t.Fatal("replaced transaction lock accepted")
	}
	if err := guard.Close(); err == nil {
		t.Fatal("foreign lock treated as owned cleanup")
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != "foreign owner" {
		t.Fatalf("cleanup removed foreign lock: %v", err)
	}
}

func TestGoProjectGuardRejectsUnreconciledJournal(t *testing.T) {
	p, guard, update, _ := approvedGoFixture(t)
	if err := guard.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(guard.plan.root, ".heliopause-go-commit-interrupted"), 0o700); err != nil {
		t.Fatal(err)
	}
	if next, err := p.Begin(context.Background(), update.Snapshot().Context()); err == nil || next != nil {
		t.Fatal("unreconciled journal allowed a new selection")
	}
	if _, err := os.Lstat(guard.guard.path); !os.IsNotExist(err) {
		t.Fatal("failed adoption retained its lock")
	}
}

func TestGoTransactionRejectsChangedApprovedControlBytes(t *testing.T) {
	_, guard, update, _ := approvedGoFixture(t)
	defer guard.Close()
	workspace, err := guard.plan.privateWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workspace)
	for _, file := range update.SelectedControls() {
		body := file.Body()
		if file.Name() == "go.mod" {
			body = append(body, []byte("// changed after selection\n")...)
		}
		if err := os.WriteFile(filepath.Join(workspace, file.Name()), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if tx, err := beginGoProjectTransaction(guard.plan, workspace, update.SelectedControls()...); err == nil || tx != nil {
		t.Fatal("changed private bytes were published as the approved selection")
	}
	if err := guard.VerifyUnchanged(context.Background()); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(guard.plan.root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".heliopause-go-commit-") {
			t.Fatal("rejected selection left a recovery journal")
		}
	}
}

func TestGoTransactionRollsBackOnApprovalFailure(t *testing.T) {
	root := writeManagedGoProject(t)
	plan, err := freezeGoProject(root)
	if err != nil {
		t.Fatal(err)
	}
	plan.authorized = true // Internal journal fixture; raw markers are not approval.
	before := map[string][]byte{}
	for _, name := range []string{"go.mod", "go.sum", goTransactionMetadata} {
		before[name], err = os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
	}
	workspace, err := plan.privateWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/app\nrequire example.com/new v1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tx, err := beginGoProjectTransaction(plan, workspace)
	if err != nil {
		t.Fatal(err)
	}
	fault := errors.New("fixture approval publication failure")
	tx.publishApproval = func() error { return fault }
	if err := tx.commit(); !errors.Is(err, fault) {
		t.Fatalf("primary failure lost: %v", err)
	}
	for name, want := range before {
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || string(got) != string(want) {
			t.Fatalf("rollback changed %s: %v", name, err)
		}
	}
	if _, err := os.Lstat(tx.backup); !os.IsNotExist(err) {
		t.Fatal("completed rollback left a backup")
	}
}

// Synthetic ALLOW producer fixture; this tests transaction binding, not public
// source/SumDB authentication or observed build qualification.
func approvedGoFixture(t *testing.T) (*GoProjectPromotion, *approvedGoProjectGuard, domain.ProjectDependencyUpdate, domain.StagedProjectSet) {
	t.Helper()
	ctx := context.Background()
	root := canonicalGoTestRoot(t)
	project := filepath.Join(root, "project")
	if err := os.Mkdir(project, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.com/app\ngo 1.26.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cache, err := newGoCacheForTest(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := NewGoProjectPromotion(cache, filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(project)
	install, _ := domain.NewInstallContext(target)
	guard, err := p.Begin(ctx, install)
	if err != nil {
		t.Fatal(err)
	}
	set, _ := goCacheFixture(t, cache.intakeRoot, project, "example.com/module", map[string]string{"go.mod": "module example.com/module\ngo 1.26.0\n", "module.go": "package module\n"})
	pair := strings.Split(set.Inspected().Snapshot().Dependencies()[0].DeclaredIntegrity(), ";")
	zipSum, modSum := strings.TrimPrefix(pair[0], "h1="), strings.TrimPrefix(pair[1], "go.mod=")
	mod := []byte("module example.com/app\ngo 1.26.0\nrequire example.com/module v1.0.0\n")
	sum := []byte("example.com/module v1.0.0 " + zipSum + "\nexample.com/module v1.0.0/go.mod " + modSum + "\n")
	graphBody := []byte("example.com/app example.com/module@v1.0.0\nexample.com/app go@1.26.0\ngo@1.26.0 toolchain@go1.26.0\n")
	records := []artifactgo.DownloadRecord{{Path: "example.com/module", Version: "v1.0.0", GoMod: "/fixture/mod", Zip: "/fixture/zip", Sum: zipSum, GoModSum: modSum}}
	snapshot, err := artifactgo.BuildProjectSnapshot(install, records, graphBody, mod, sum)
	if err != nil {
		t.Fatal(err)
	}
	ref, _ := artifactgo.ParseReference("example.com/module@v1.0.0")
	graph, err := artifactgo.BuildLockedGraph(ref, records, graphBody, mod)
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := domain.NewDependencyResolution(graph, "fixture", snapshot.GraphDigest())
	if err != nil {
		t.Fatal(err)
	}
	modFile, _ := domain.NewProjectControlFile("go.mod", mod, true)
	sumFile, _ := domain.NewProjectControlFile("go.sum", sum, true)
	update, err := domain.NewProjectDependencyUpdate(guard.Controls(), []domain.ProjectControlFile{modFile, sumFile}, snapshot, resolution)
	if err != nil {
		t.Fatal(err)
	}
	inspected, err := domain.NewInspectedProjectSet(snapshot, set.Inspected().Inspections())
	if err != nil {
		t.Fatal(err)
	}
	verified, err := domain.NewProjectVerifiedSet(inspected, set.Decision())
	if err != nil {
		t.Fatal(err)
	}
	staged, err := cache.StageProject(ctx, verified)
	if err != nil {
		t.Fatal(err)
	}
	return p, guard, update, staged
}
