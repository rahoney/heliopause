package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/policy"
)

// Synthetic port producers test orchestration and failure authority. Public
// source/cache authentication is covered by the separate actual integrations.
func TestGoDownloadRequiresEveryApprovalBoundary(t *testing.T) {
	want := []string{"guard", "resolve", "verify-1", "inspect", "verify-2", "stage", "commit", "close"}
	for _, failure := range append([]string{""}, want...) {
		t.Run("failure-"+failure, func(t *testing.T) {
			graph := twoNodeGraphForSource(t, "go-proxy")
			snapshot := inspectionProjectSnapshot(t, graph)
			producer := newMultiInspectionPorts(t, graph)
			inspector, err := application.NewProjectInspectService(producer, producer, producer, producer, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			f := &goDownloadPipeline{snapshot: snapshot, inspector: inspector, fail: failure}
			control, err := domain.NewProjectControlFile("fixture.control", []byte("original"), true)
			if err != nil {
				t.Fatal(err)
			}
			digest, _ := domain.NewProjectControlDigest(control.Name(), control.Digest())
			f.snapshot, err = domain.NewProjectDependencySnapshot(snapshot.Context(), snapshot.Source(), []domain.ProjectControlDigest{digest}, snapshot.Dependencies(), snapshot.GraphDigest())
			if err != nil {
				t.Fatal(err)
			}
			f.controls = []domain.ProjectControlFile{control}
			service, err := application.NewGoModuleProjectResolutionService(f, f, f, f)
			if err != nil {
				t.Fatal(err)
			}
			result, err := service.Resolve(context.Background(), f.snapshot.Context())
			if failure == "" {
				if err != nil || !result.Valid() || !reflect.DeepEqual(f.calls, want) {
					t.Fatalf("download=%v calls=%v", err, f.calls)
				}
				return
			}
			if !errors.Is(err, errDownloadBoundary) || result.Valid() {
				t.Fatalf("failed boundary returned a successful snapshot: %v", err)
			}
			var expected []string
			for _, step := range want {
				expected = append(expected, step)
				if step == failure {
					break
				}
			}
			if failure != "guard" && failure != "close" {
				expected = append(expected, "close")
			}
			if !reflect.DeepEqual(f.calls, expected) {
				t.Fatalf("continued after %s: %v", failure, f.calls)
			}
		})
	}
}

var errDownloadBoundary = errors.New("fixture download boundary failure")

type goDownloadPipeline struct {
	snapshot  domain.ProjectDependencySnapshot
	controls  []domain.ProjectControlFile
	inspector *application.ProjectInspectService
	calls     []string
	fail      string
	verifies  int
}

func (f *goDownloadPipeline) step(name string) error {
	f.calls = append(f.calls, name)
	if f.fail == name {
		return errDownloadBoundary
	}
	return nil
}
func (f *goDownloadPipeline) Begin(context.Context, domain.InstallContext) (ports.ProjectMutationGuard, error) {
	if err := f.step("guard"); err != nil {
		return nil, err
	}
	return f, nil
}
func (f *goDownloadPipeline) ResolveProjectDependencies(context.Context, domain.InstallContext) (domain.ProjectDependencySnapshot, error) {
	return f.snapshot, f.step("resolve")
}
func (f *goDownloadPipeline) Controls() []domain.ProjectControlFile { return f.controls }
func (f *goDownloadPipeline) VerifyUnchanged(context.Context) error {
	f.verifies++
	if f.verifies == 1 {
		return f.step("verify-1")
	}
	return f.step("verify-2")
}
func (f *goDownloadPipeline) InspectProject(ctx context.Context, snapshot domain.ProjectDependencySnapshot) (domain.ProjectVerifiedSet, error) {
	if err := f.step("inspect"); err != nil {
		return domain.ProjectVerifiedSet{}, err
	}
	return f.inspector.InspectProject(ctx, snapshot)
}
func (f *goDownloadPipeline) StageProject(_ context.Context, set domain.ProjectVerifiedSet) (domain.StagedProjectSet, error) {
	if err := f.step("stage"); err != nil {
		return domain.StagedProjectSet{}, err
	}
	id, err := domain.NewRunID()
	if err != nil {
		return domain.StagedProjectSet{}, err
	}
	return domain.NewStagedProjectSet(set, "project-cache:"+id.String(), set.Inspected().Snapshot().GraphDigest())
}
func (f *goDownloadPipeline) CommitSnapshot(_ context.Context, snapshot domain.ProjectDependencySnapshot, staged domain.StagedProjectSet) error {
	if !staged.Valid() || snapshot.GraphDigest() != f.snapshot.GraphDigest() {
		return errors.New("fixture received mismatched approval")
	}
	return f.step("commit")
}
func (*goDownloadPipeline) Commit(context.Context, domain.ProjectDependencyUpdate, domain.StagedProjectSet) error {
	return errors.New("download must not fabricate a primary update")
}
func (f *goDownloadPipeline) Close() error { return f.step("close") }
