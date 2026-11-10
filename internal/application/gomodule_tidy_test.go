package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/policy"
)

type goTidyPipeline struct{ *projectUpdatePipeline }

func (f goTidyPipeline) ResolveProjectLockUpdate(ctx context.Context, install domain.InstallContext, controls []domain.ProjectControlFile, selector string) (domain.ProjectDependencyUpdate, error) {
	if selector != "" {
		return domain.ProjectDependencyUpdate{}, errors.New("tidy invented a selector")
	}
	return f.projectUpdatePipeline.ResolveProjectLockUpdate(ctx, install, controls, "fixture")
}

func TestGoTidyRequiresEveryApprovalBoundary(t *testing.T) {
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
			original, _ := domain.NewProjectControlFile("fixture.control", []byte("original"), true)
			selected, _ := domain.NewProjectControlFile("fixture.control", []byte("selected"), true)
			digest, _ := domain.NewProjectControlDigest(selected.Name(), selected.Digest())
			snapshot, err = domain.NewProjectDependencySnapshot(snapshot.Context(), snapshot.Source(), []domain.ProjectControlDigest{digest}, snapshot.Dependencies(), snapshot.GraphDigest())
			if err != nil {
				t.Fatal(err)
			}
			update, err := domain.NewProjectDependencyUpdateFromSnapshot([]domain.ProjectControlFile{original}, []domain.ProjectControlFile{selected}, snapshot)
			if err != nil {
				t.Fatal(err)
			}
			f := goTidyPipeline{&projectUpdatePipeline{goDownloadPipeline: goDownloadPipeline{snapshot: snapshot, inspector: inspector, fail: failure, controls: []domain.ProjectControlFile{original}}, update: update}}
			service, err := application.NewGoModuleTidyService(f, f, f, f)
			if err != nil {
				t.Fatal(err)
			}
			result, err := service.Tidy(context.Background(), snapshot.Context())
			if failure == "" {
				if err != nil || !result.Valid() || !reflect.DeepEqual(f.calls, want) {
					t.Fatalf("result=%v calls=%v", err, f.calls)
				}
				return
			}
			if !errors.Is(err, errDownloadBoundary) || result.Valid() {
				t.Fatalf("failed boundary returned success or lost cause: %v", err)
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
