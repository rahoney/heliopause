package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/policy"
)

func TestInstallOrdersAllowManifestStagingAndPromotion(t *testing.T) {
	t.Parallel()
	request, inspection := installWorkflowFixture(t)
	calls := []string{}
	manifest := &manifestPort{calls: &calls}
	staging := &stagingPort{calls: &calls}
	promotion := &promotionPort{calls: &calls}
	service, err := application.NewInstallService(inspection, manifest, staging, promotion)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := service.Install(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 3 || calls[0] != "manifest" || calls[1] != "staging" || calls[2] != "promotion" || outcome.Promoted().Target() != request.Context().Target() {
		t.Fatalf("calls=%v outcome=%#v", calls, outcome)
	}
}

func TestInstallPreservesAllowAndStopsAfterStagingFailure(t *testing.T) {
	t.Parallel()
	request, inspection := installWorkflowFixture(t)
	calls := []string{}
	want := errors.New("staging failed")
	service, _ := application.NewInstallService(inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls, err: want}, &promotionPort{calls: &calls})
	outcome, err := service.Install(context.Background(), request)
	if !errors.Is(err, want) || outcome.Inspection().Decision().Decision() != domain.DecisionAllow || outcome.Bundle().ManifestID().String() == "" || outcome.Promoted().Target().String() != "" {
		t.Fatalf("Install() outcome=%#v error=%v", outcome, err)
	}
	if len(calls) != 2 || calls[1] != "staging" {
		t.Fatalf("calls after staging failure = %v", calls)
	}
}

func installWorkflowFixture(t *testing.T) (application.InstallRequest, *application.InstallInspectService) {
	t.Helper()
	graph := twoNodeGraph(t)
	ports := newMultiInspectionPorts(t, graph)
	source, _ := domain.NewSourceID("npm")
	reference, _ := domain.NewArtifactReference(source, "first")
	target, _ := domain.NewInstallTarget("/tmp/heliopause-promoted-target")
	installContext, _ := domain.NewInstallContext(target)
	request, _ := application.NewInstallRequest(reference, installContext)
	inspection, err := application.NewInstallInspectService(&lockedResolver{graph: graph}, ports, ports, ports, ports, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	return request, inspection
}

type manifestPort struct{ calls *[]string }

func (p *manifestPort) Build(ctx context.Context, operationID domain.OperationID, installContext domain.InstallContext, resolution domain.DependencyResolution, set domain.VerifiedSet) (domain.VerifiedBundle, error) {
	*p.calls = append(*p.calls, "manifest")
	return (evidence.Generator{}).Build(ctx, operationID, installContext, resolution, set)
}

type stagingPort struct {
	calls *[]string
	err   error
}

func (p *stagingPort) Stage(_ context.Context, bundle domain.VerifiedBundle) (domain.StagedSet, error) {
	*p.calls = append(*p.calls, "staging")
	if p.err != nil {
		return domain.StagedSet{}, p.err
	}
	return domain.NewStagedSet(bundle.ManifestID(), "staging:"+bundle.ManifestID().String())
}

type promotionPort struct{ calls *[]string }

func (p *promotionPort) Promote(_ context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, installContext domain.InstallContext) (domain.PromotedInstall, error) {
	*p.calls = append(*p.calls, "promotion")
	if staged.ManifestID() != bundle.ManifestID() {
		return domain.PromotedInstall{}, errors.New("fixture binding mismatch")
	}
	return domain.NewPromotedInstall(bundle.ManifestID(), installContext.Target())
}
