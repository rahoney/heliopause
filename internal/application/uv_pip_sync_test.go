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

type uvSyncPipeline struct{ *projectUpdatePipeline }

func (f uvSyncPipeline) ResolveProjectLockUpdate(_ context.Context, _ domain.InstallContext, controls []domain.ProjectControlFile, selector string) (domain.ProjectDependencyUpdate, error) {
	if selector != "" || !reflect.DeepEqual(controls, f.controls) {
		return domain.ProjectDependencyUpdate{}, errors.New("sync lost complete selection intent or original controls")
	}
	return f.update, f.step("resolve")
}

func TestUVSyncRequiresEveryApprovalBoundaryAndPreservesFailureFacts(t *testing.T) {
	want := []string{"guard", "resolve", "verify-1", "inspect", "verify-2", "stage", "commit", "close"}
	for _, empty := range []bool{false, true} {
		for _, failure := range append([]string{""}, want...) {
			t.Run(failure+map[bool]string{true: "/empty", false: "/complete"}[empty], func(t *testing.T) {
				graph := twoNodeGraphForSource(t, "pypi")
				snapshot := inspectionProjectSnapshot(t, graph)
				install, err := domain.NewPythonVenvInstallContext(snapshot.Context().Target())
				if err != nil {
					t.Fatal(err)
				}
				original, _ := domain.NewProjectControlFile("source.control", []byte("original"), true)
				selected, _ := domain.NewProjectControlFile("source.control", []byte("selected"), true)
				control, _ := domain.NewProjectControlDigest(selected.Name(), selected.Digest())
				if empty {
					snapshot, err = domain.NewDependencyFreeProjectSnapshot(install, snapshot.Source(), []domain.ProjectControlDigest{control}, snapshot.GraphDigest())
				} else {
					snapshot, err = domain.NewProjectDependencySnapshot(install, snapshot.Source(), []domain.ProjectControlDigest{control}, snapshot.Dependencies(), snapshot.GraphDigest())
				}
				if err != nil {
					t.Fatal(err)
				}
				ports := newMultiInspectionPorts(t, graph)
				inspector, err := application.NewProjectInspectService(ports, ports, ports, ports, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
				if err != nil {
					t.Fatal(err)
				}
				update, err := domain.NewProjectDependencyUpdateFromSnapshot([]domain.ProjectControlFile{original}, []domain.ProjectControlFile{selected}, snapshot)
				if err != nil {
					t.Fatal(err)
				}
				f := uvSyncPipeline{&projectUpdatePipeline{goDownloadPipeline: goDownloadPipeline{snapshot: snapshot, inspector: inspector, fail: failure, controls: []domain.ProjectControlFile{original}}, update: update}}
				service, err := application.NewUVPipSyncService(f, f, f, f)
				if err != nil {
					t.Fatal(err)
				}
				result, err := service.Sync(context.Background(), install)
				if failure == "" {
					if err != nil || !result.Valid() || result.Inspected().Snapshot().DependencyFree() != empty || !reflect.DeepEqual(f.calls, want) {
						t.Fatal("complete workflow failed", err, f.calls)
					}
					return
				}
				if !errors.Is(err, errDownloadBoundary) || result.Valid() {
					t.Fatal("failed boundary returned success or lost cause", err)
				}
				expected := []string{}
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
					t.Fatal("workflow crossed failed boundary", f.calls, expected)
				}
				if failure == "verify-2" || failure == "stage" || failure == "commit" || failure == "close" {
					var facts *application.ProjectUpdateFailure
					if !errors.As(err, &facts) || !facts.Inspected().Valid() {
						t.Fatal("lost fresh facts after publication failure", err)
					}
				}
			})
		}
	}
}
