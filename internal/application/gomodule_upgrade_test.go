package application_test

import (
	"context"
	"errors"
	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/policy"
	"reflect"
	"testing"
)

func (f *projectUpdatePipeline) ResolveProjectDependencyUpgrade(ctx context.Context, install domain.InstallContext, controls []domain.ProjectControlFile, selector string) (domain.ProjectDependencyUpdate, error) {
	if selector != "first" || !reflect.DeepEqual(controls, f.controls) {
		return domain.ProjectDependencyUpdate{}, errors.New("named upgrade lost selector or original controls")
	}
	return f.update, f.step("resolve")
}

func TestGoNamedUpgradeRequiresEveryApprovalBoundary(t *testing.T) {
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
			control, _ := domain.NewProjectControlDigest(selected.Name(), selected.Digest())
			snapshot, err = domain.NewProjectDependencySnapshot(snapshot.Context(), snapshot.Source(), []domain.ProjectControlDigest{control}, snapshot.Dependencies(), snapshot.GraphDigest())
			if err != nil {
				t.Fatal(err)
			}
			resolution, err := domain.NewDependencyResolution(graph, "synthetic:named-upgrade", snapshot.GraphDigest())
			if err != nil {
				t.Fatal(err)
			}
			update, err := domain.NewProjectDependencyUpdate([]domain.ProjectControlFile{original}, []domain.ProjectControlFile{selected}, snapshot, resolution)
			if err != nil {
				t.Fatal(err)
			}
			f := &projectUpdatePipeline{goDownloadPipeline: goDownloadPipeline{snapshot: snapshot, inspector: inspector, fail: failure, controls: []domain.ProjectControlFile{original}}, update: update}
			service, err := application.NewGoModuleUpgradeService(f, f, f, f)
			if err != nil {
				t.Fatal(err)
			}
			result, err := service.Upgrade(context.Background(), snapshot.Context(), "first")
			if failure == "" {
				if err != nil || !result.Valid() || result.Inspected().Snapshot().GraphDigest() != snapshot.GraphDigest() || !reflect.DeepEqual(f.calls, want) {
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
			if failure == "verify-2" || failure == "stage" || failure == "commit" || failure == "close" {
				var facts *application.ProjectUpdateFailure
				if !errors.As(err, &facts) || !facts.Inspected().Valid() || facts.Decision().Decision() != domain.DecisionAllow {
					t.Fatalf("fresh inspection facts lost: %v", err)
				}
			}
		})
	}
}

func TestGoNamedUpgradeInspectsEveryEntryAndRefusesNewUnsafeSet(t *testing.T) {
	for _, name := range []string{"first", "second"} {
		for _, kind := range []string{"block", "review", "incomplete", "integrity"} {
			t.Run(name+"/"+kind, func(t *testing.T) {
				graph := twoNodeGraphForSource(t, "go-proxy")
				snapshot := inspectionProjectSnapshot(t, graph)
				producer := &cargoUpdateAdversePorts{newMultiInspectionPorts(t, graph), name, kind}
				inspector, err := application.NewProjectInspectService(producer, producer, producer, producer, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
				if err != nil {
					t.Fatal(err)
				}
				original, _ := domain.NewProjectControlFile("fixture.control", []byte("original"), true)
				control, _ := domain.NewProjectControlDigest(original.Name(), original.Digest())
				snapshot, err = domain.NewProjectDependencySnapshot(snapshot.Context(), snapshot.Source(), []domain.ProjectControlDigest{control}, snapshot.Dependencies(), snapshot.GraphDigest())
				if err != nil {
					t.Fatal(err)
				}
				resolution, err := domain.NewDependencyResolution(graph, "synthetic:named-upgrade", snapshot.GraphDigest())
				if err != nil {
					t.Fatal(err)
				}
				update, err := domain.NewProjectDependencyUpdate([]domain.ProjectControlFile{original}, []domain.ProjectControlFile{original}, snapshot, resolution)
				if err != nil {
					t.Fatal(err)
				}
				f := &projectUpdatePipeline{goDownloadPipeline: goDownloadPipeline{snapshot: snapshot, inspector: inspector, controls: []domain.ProjectControlFile{original}}, update: update}
				service, err := application.NewGoModuleUpgradeService(f, f, f, f)
				if err != nil {
					t.Fatal(err)
				}
				result, err := service.Upgrade(context.Background(), snapshot.Context(), "first")
				var denied *application.ProjectPolicyFailure
				if err == nil || result.Valid() || !errors.As(err, &denied) || len(denied.Inspected().Inspections()) != 2 {
					t.Fatalf("unsafe complete selection returned approval or lost facts: %v", err)
				}
				wantDecision := domain.DecisionBlock
				if kind == "review" || kind == "incomplete" {
					wantDecision = domain.DecisionManualReview
				}
				if denied.Decision().Decision() != wantDecision {
					t.Fatal(denied.Decision())
				}
				if !reflect.DeepEqual(f.calls, []string{"guard", "resolve", "verify-1", "inspect", "close"}) {
					t.Fatalf("unsafe update reached cache/publication: %v", f.calls)
				}
				if len(producer.calls) != 8 {
					t.Fatalf("selected entries were not independently inspected: %v", producer.calls)
				}
			})
		}
	}
}
