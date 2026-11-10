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

func TestRequirementsWorkflowPreservesAllBoundariesAndExplicitNoop(t *testing.T) {
	for _, empty := range []bool{false, true} {
		want := []string{"guard", "resolve", "verify-1", "inspect", "verify-2", "stage", "commit", "close"}
		if empty {
			want = []string{"guard", "resolve", "verify-1", "close"}
		}
		for _, failure := range append([]string{""}, want...) {
			t.Run(map[bool]string{true: "empty/", false: "selected/"}[empty]+failure, func(t *testing.T) {
				graph := twoNodeGraphForSource(t, "pypi")
				snapshot := inspectionProjectSnapshot(t, graph)
				install, _ := domain.NewPythonVenvInstallContext(snapshot.Context().Target())
				original, _ := domain.NewProjectControlFile("source.control", []byte("original"), true)
				selected, _ := domain.NewProjectControlFile("source.control", []byte("selected"), true)
				control, _ := domain.NewProjectControlDigest(selected.Name(), selected.Digest())
				var err error
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
				pipeline := uvSyncPipeline{&projectUpdatePipeline{goDownloadPipeline: goDownloadPipeline{snapshot: snapshot, inspector: inspector, fail: failure, controls: []domain.ProjectControlFile{original}}, update: update}}
				service, err := application.NewPythonRequirementsService(pipeline, pipeline, pipeline, pipeline)
				if err != nil {
					t.Fatal(err)
				}
				verified, noop, err := service.Install(context.Background(), install)
				if failure == "" {
					if err != nil || noop != empty || verified.Valid() == empty || !reflect.DeepEqual(pipeline.calls, want) {
						t.Fatal("requirements workflow differs", err, noop, pipeline.calls)
					}
					return
				}
				if !errors.Is(err, errDownloadBoundary) || verified.Valid() || noop {
					t.Fatal("failed boundary returned success or lost cause", err, noop)
				}
				if pipeline.calls[len(pipeline.calls)-1] != "close" && failure != "guard" {
					t.Fatal("failed request did not close its original guard", pipeline.calls)
				}
			})
		}
	}
}
