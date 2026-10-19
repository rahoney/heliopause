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

type terraformUpgradePipeline struct{ *projectUpdatePipeline }

func (f *terraformUpgradePipeline) ResolveProjectLockUpdate(_ context.Context, _ domain.InstallContext, controls []domain.ProjectControlFile, selector string) (domain.ProjectDependencyUpdate, error) {
	if selector != "" || !reflect.DeepEqual(controls, f.controls) {
		return domain.ProjectDependencyUpdate{}, errors.New("terraform whole-project selection lost frozen controls")
	}
	return f.update, f.step("resolve")
}

func terraformUpgradeWorkflowFixture(t *testing.T, fail, name, kind string) (*application.TerraformUpgradeService, *terraformUpgradePipeline, *cargoUpdateAdversePorts) {
	t.Helper()
	graph := twoNodeGraphForSource(t, "terraform-registry")
	snapshot := inspectionProjectSnapshot(t, graph)
	producer := &cargoUpdateAdversePorts{newMultiInspectionPorts(t, graph), name, kind}
	inspector, err := application.NewProjectInspectService(producer, producer, producer, producer, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	control, _ := domain.NewProjectControlFile("fixture.control", []byte("unchanged"), true)
	digest, _ := domain.NewProjectControlDigest(control.Name(), control.Digest())
	snapshot, err = domain.NewProjectDependencySnapshot(snapshot.Context(), snapshot.Source(), []domain.ProjectControlDigest{digest}, snapshot.Dependencies(), snapshot.GraphDigest())
	if err != nil {
		t.Fatal(err)
	}
	update, err := domain.NewProjectDependencyUpdateFromSnapshot([]domain.ProjectControlFile{control}, []domain.ProjectControlFile{control}, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	f := &terraformUpgradePipeline{&projectUpdatePipeline{goDownloadPipeline: goDownloadPipeline{snapshot: snapshot, inspector: inspector, fail: fail, controls: []domain.ProjectControlFile{control}}, update: update}}
	service, err := application.NewTerraformUpgradeService(f, f, f, f)
	if err != nil {
		t.Fatal(err)
	}
	return service, f, producer
}

func TestTerraformUpgradeRequiresAllApprovalAndCloseBoundaries(t *testing.T) {
	want := []string{"guard", "resolve", "verify-1", "inspect", "verify-2", "stage", "commit", "close"}
	for _, failure := range append([]string{""}, want...) {
		t.Run("failure-"+failure, func(t *testing.T) {
			service, f, _ := terraformUpgradeWorkflowFixture(t, failure, "", "")
			set, err := service.Upgrade(context.Background(), f.snapshot.Context())
			if failure == "" {
				if err != nil || !set.Valid() || !reflect.DeepEqual(f.calls, want) {
					t.Fatal(err, f.calls)
				}
				return
			}
			if !errors.Is(err, errDownloadBoundary) || set.Valid() {
				t.Fatal("failed boundary reported completion or lost cause", err)
			}
			if failure == "verify-2" || failure == "stage" || failure == "commit" || failure == "close" {
				var facts *application.ProjectUpdateFailure
				if !errors.As(err, &facts) || !facts.Inspected().Valid() {
					t.Fatal("fresh facts lost", err)
				}
			}
		})
	}
}

func TestTerraformUpgradeInspectsEachIndependentRootAndRefusesUnsafeSet(t *testing.T) {
	for _, name := range []string{"first", "second"} {
		for _, kind := range []string{"block", "review", "incomplete", "integrity"} {
			t.Run(name+"/"+kind, func(t *testing.T) {
				service, f, producer := terraformUpgradeWorkflowFixture(t, "", name, kind)
				set, err := service.Upgrade(context.Background(), f.snapshot.Context())
				var denied *application.ProjectPolicyFailure
				if err == nil || set.Valid() || !errors.As(err, &denied) || len(denied.Inspected().Inspections()) != 2 {
					t.Fatal("unsafe set approved or facts lost", err)
				}
				want := domain.DecisionBlock
				if kind == "review" || kind == "incomplete" {
					want = domain.DecisionManualReview
				}
				if denied.Decision().Decision() != want {
					t.Fatal(denied.Decision())
				}
				if !reflect.DeepEqual(f.calls, []string{"guard", "resolve", "verify-1", "inspect", "close"}) || len(producer.calls) != 8 {
					t.Fatal("unsafe set reached staging/publication or skipped a root", f.calls, producer.calls)
				}
			})
		}
	}
}

func TestTerraformUpgradeRejectsInventedRequestedPrimary(t *testing.T) {
	service, f, _ := terraformUpgradeWorkflowFixture(t, "", "", "")
	graph := twoNodeGraphForSource(t, "terraform-registry")
	resolution, err := domain.NewDependencyResolution(graph, "synthetic:requested", f.snapshot.GraphDigest())
	if err != nil {
		t.Fatal(err)
	}
	f.update, err = domain.NewProjectDependencyUpdate(f.update.OriginalControls(), f.update.SelectedControls(), f.snapshot, resolution)
	if err != nil {
		t.Fatal(err)
	}
	set, err := service.Upgrade(context.Background(), f.snapshot.Context())
	if err == nil || set.Valid() || !reflect.DeepEqual(f.calls, []string{"guard", "resolve", "close"}) {
		t.Fatal("arbitrary requested primary accepted", err, f.calls)
	}
}
