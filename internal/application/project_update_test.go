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

func TestProjectUpdateRequiresEveryApprovalBoundary(t *testing.T) {
	want := []string{"guard", "resolve", "verify-1", "inspect", "verify-2", "stage", "commit", "close"}
	for _, source := range []string{"go-proxy", "crates-io"} {
		for _, failure := range append([]string{""}, want...) {
			t.Run(source+"/failure-"+failure, func(t *testing.T) {
				graph := twoNodeGraphForSource(t, source)
				snapshot := inspectionProjectSnapshot(t, graph)
				producer := newMultiInspectionPorts(t, graph)
				inspector, err := application.NewProjectInspectService(producer, producer, producer, producer, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
				if err != nil {
					t.Fatal(err)
				}
				original, _ := domain.NewProjectControlFile("fixture.control", []byte("original"), true)
				selected, _ := domain.NewProjectControlFile("fixture.control", []byte("selected"), true)
				control, _ := domain.NewProjectControlDigest(selected.Name(), selected.Digest())
				snapshot, err = domain.NewProjectDependencySnapshot(snapshot.Context(), snapshot.Source(), []domain.ProjectControlDigest{control}, snapshot.Dependencies(), snapshot.GraphDigest())
				if err != nil {
					t.Fatal(err)
				}
				resolution, err := domain.NewDependencyResolution(graph, "synthetic:single-selection", snapshot.GraphDigest())
				if err != nil {
					t.Fatal(err)
				}
				update, err := domain.NewProjectDependencyUpdate([]domain.ProjectControlFile{original}, []domain.ProjectControlFile{selected}, snapshot, resolution)
				if err != nil {
					t.Fatal(err)
				}
				f := &projectUpdatePipeline{goDownloadPipeline: goDownloadPipeline{snapshot: snapshot, inspector: inspector, fail: failure, controls: []domain.ProjectControlFile{original}}, update: update}
				reference, _ := domain.NewArtifactReference(snapshot.Source(), "fixture")
				var run func(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error)
				if source == "go-proxy" {
					s, err := application.NewGoModuleGetService(f, f, f, f)
					if err != nil {
						t.Fatal(err)
					}
					run = s.Get
				} else {
					s, err := application.NewCargoAddService(f, f, f, f)
					if err != nil {
						t.Fatal(err)
					}
					run = s.Resolve
				}
				result, err := run(context.Background(), reference, snapshot.Context())
				if failure == "" {
					if err != nil || !reflect.DeepEqual(result, resolution) || !reflect.DeepEqual(f.calls, want) {
						t.Fatalf("update=%v calls=%v", err, f.calls)
					}
					return
				}
				if !errors.Is(err, errDownloadBoundary) || result.LockfileDigest().String() != "" {
					t.Fatalf("failed boundary returned selected success: %v", err)
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
					t.Fatalf("calls=%v want=%v", f.calls, expected)
				}
			})
		}
	}
}

type projectUpdatePipeline struct {
	goDownloadPipeline
	update domain.ProjectDependencyUpdate
}

func (f *projectUpdatePipeline) Begin(context.Context, domain.InstallContext) (ports.ProjectMutationGuard, error) {
	if err := f.step("guard"); err != nil {
		return nil, err
	}
	return f, nil
}
func (f *projectUpdatePipeline) ResolveProjectDependencyUpdate(_ context.Context, _ domain.ArtifactReference, _ domain.InstallContext, controls []domain.ProjectControlFile) (domain.ProjectDependencyUpdate, error) {
	if !reflect.DeepEqual(controls, f.controls) {
		return domain.ProjectDependencyUpdate{}, errors.New("selection did not receive original guard controls")
	}
	return f.update, f.step("resolve")
}
func (f *projectUpdatePipeline) Commit(_ context.Context, update domain.ProjectDependencyUpdate, staged domain.StagedProjectSet) error {
	if !reflect.DeepEqual(update, f.update) || !staged.Valid() || staged.Set().Inspected().Snapshot().GraphDigest() != update.Snapshot().GraphDigest() {
		return errors.New("commit substituted frozen selection")
	}
	return f.step("commit")
}
