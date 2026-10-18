package application_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/policy"
)

type updatePreparationFixture struct {
	session *updateSessionFixture
	calls   *[]string
	err     error
}

func (p updatePreparationFixture) BeginInstallUpdate(_ context.Context, _ domain.SourceID, _ string, _ domain.InstallContext, _ bool) (ports.InstallUpdateSession, error) {
	*p.calls = append(*p.calls, "begin")
	return p.session, p.err
}

type updateSessionFixture struct {
	reference domain.ArtifactReference
	resolver  ports.DependencyResolver
	promoter  ports.Promotion
	calls     *[]string
	closeErr  error
}

func (s *updateSessionFixture) Reference() domain.ArtifactReference { return s.reference }
func (s *updateSessionFixture) ResolveDependencies(ctx context.Context, ref domain.ArtifactReference, install domain.InstallContext) (domain.DependencyResolution, error) {
	*s.calls = append(*s.calls, "resolve")
	return s.resolver.ResolveDependencies(ctx, ref, install)
}
func (s *updateSessionFixture) Promote(ctx context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, install domain.InstallContext) (domain.PromotedInstall, error) {
	return s.promoter.Promote(ctx, staged, bundle, install)
}
func (s *updateSessionFixture) Close() error { *s.calls = append(*s.calls, "close"); return s.closeErr }

func TestInstallUpdateGuardsBeforeSelectionAndReusesPipelineWithoutRebindingBase(t *testing.T) {
	old, inspection := installWorkflowFixture(t)
	calls := []string{}
	base, err := application.NewInstallService(inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls}, &promotionPort{calls: &calls})
	if err != nil {
		t.Fatal(err)
	}
	session := &updateSessionFixture{reference: old.Reference(), resolver: &lockedResolver{graph: twoNodeGraph(t)}, promoter: &promotionPort{calls: &calls}, calls: &calls}
	updater, err := application.NewInstallUpdateService(base, updatePreparationFixture{session: session, calls: &calls})
	if err != nil {
		t.Fatal(err)
	}
	install, _ := domain.NewNPMProjectInstallContext(old.Context().Target())
	request, err := application.NewInstallUpdateRequest(old.Source(), "", install, false)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := updater.Install(context.Background(), request)
	if err != nil || outcome.Promoted().Target() != install.Target() || outcome.Inspection().Request().Operation() != domain.OperationUpdate {
		t.Fatal(outcome, err)
	}
	if !reflect.DeepEqual(calls, []string{"begin", "resolve", "manifest", "staging", "promotion", "close"}) {
		t.Fatal(calls)
	}
	calls = nil
	if _, err := base.Install(context.Background(), old); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls, []string{"manifest", "staging", "promotion"}) || old.Operation() != domain.OperationInstall {
		t.Fatal("base installer mutated by update", calls)
	}
}

func TestInstallUpdateRejectsBeforePublicationAndPreservesPrimaryAndCloseFailure(t *testing.T) {
	for _, decision := range []domain.Decision{domain.DecisionBlock, domain.DecisionManualReview} {
		old, inspection := installWorkflowFixtureWithPolicy(t, fixedSetPolicy{decision: decision})
		calls := []string{}
		base, _ := application.NewInstallService(inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls}, &promotionPort{calls: &calls})
		session := &updateSessionFixture{reference: old.Reference(), resolver: &lockedResolver{graph: twoNodeGraph(t)}, promoter: &promotionPort{calls: &calls}, calls: &calls}
		updater, _ := application.NewInstallUpdateService(base, updatePreparationFixture{session: session, calls: &calls})
		install, _ := domain.NewNPMProjectInstallContext(old.Context().Target())
		request, _ := application.NewInstallUpdateRequest(old.Source(), "first", install, false)
		outcome, err := updater.Install(context.Background(), request)
		if err != nil || outcome.Inspection().Decision().Decision() != decision || !reflect.DeepEqual(calls, []string{"begin", "resolve", "close"}) {
			t.Fatal(calls, outcome, err)
		}
	}
	old, inspection := installWorkflowFixture(t)
	calls := []string{}
	primary, cleanup := errors.New("staging failed"), errors.New("guard close failed")
	base, _ := application.NewInstallService(inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls, err: primary}, &promotionPort{calls: &calls})
	session := &updateSessionFixture{reference: old.Reference(), resolver: &lockedResolver{graph: twoNodeGraph(t)}, promoter: &promotionPort{calls: &calls}, calls: &calls, closeErr: cleanup}
	updater, _ := application.NewInstallUpdateService(base, updatePreparationFixture{session: session, calls: &calls})
	install, _ := domain.NewNPMProjectInstallContext(old.Context().Target())
	request, _ := application.NewInstallUpdateRequest(old.Source(), "first", install, false)
	outcome, err := updater.Install(context.Background(), request)
	if !errors.Is(err, primary) || !errors.Is(err, cleanup) || outcome.FailureCode() != "STAGING_FAILED" || outcome.Inspection().Decision().Decision() != domain.DecisionAllow || !reflect.DeepEqual(calls, []string{"begin", "resolve", "manifest", "staging", "close"}) {
		t.Fatal(outcome, calls, err)
	}
}

func TestInstallUpdatePreparationFailureIsStartedUpdateWithoutFabricatedGraph(t *testing.T) {
	old, inspection := installWorkflowFixture(t)
	calls := []string{}
	base, _ := application.NewInstallService(inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls}, &promotionPort{calls: &calls})
	want := errors.New("original state unavailable")
	updater, _ := application.NewInstallUpdateService(base, updatePreparationFixture{calls: &calls, err: want})
	install, _ := domain.NewNPMProjectInstallContext(old.Context().Target())
	request, _ := application.NewInstallUpdateRequest(old.Source(), "", install, false)
	outcome, err := updater.Install(context.Background(), request)
	if !errors.Is(err, want) || outcome.Inspection().OperationID().String() == "" || outcome.Inspection().Request().Operation() != domain.OperationUpdate || outcome.Inspection().Request().Reference().Locator() != "*" || outcome.Inspection().Set().Valid() || !reflect.DeepEqual(calls, []string{"begin"}) {
		t.Fatal(outcome, calls, err)
	}
}

func TestBaseInstallerCannotAcceptAnUpdateWithoutOriginalStateSession(t *testing.T) {
	old, inspection := installWorkflowFixture(t)
	calls := []string{}
	base, _ := application.NewInstallService(inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls}, &promotionPort{calls: &calls})
	install, _ := domain.NewNPMProjectInstallContext(old.Context().Target())
	request, _ := application.NewInstallUpdateRequest(old.Source(), "first", install, false)
	if _, err := base.Install(context.Background(), request); err == nil || len(calls) != 0 {
		t.Fatal("update bypassed early guard", calls, err)
	}
}

type updateFindingPorts struct {
	*multiInspectionPorts
	badName string
	mode    string
}

func (p *updateFindingPorts) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	report, err := p.multiInspectionPorts.Inspect(ctx, artifact)
	if err != nil || artifact.Identity().Name() != p.badName {
		return report, err
	}
	check := report.Execution()
	if p.mode == "inspection-incomplete" {
		check, _ = domain.NewCheckExecution(check.ID(), check.Kind(), true, domain.CapabilitySupported, domain.ExecutionIncomplete, "CONTROLLED_UPDATE_FIXTURE")
		return domain.NewInspectionReport(check, nil, report.Evidence())
	}
	finding, _ := domain.NewFinding("M3_HONEYTOKEN_ACCESS", []domain.EvidenceID{report.Evidence()[0].ID()})
	return domain.NewInspectionReport(check, []domain.Finding{finding}, report.Evidence())
}

func (p *updateFindingPorts) Verify(ctx context.Context, artifact domain.AcquiredArtifact) (domain.VerificationReport, error) {
	report, err := p.multiInspectionPorts.Verify(ctx, artifact)
	if err != nil || artifact.Identity().Name() != p.badName || p.mode != "integrity-mismatch" {
		return report, err
	}
	return domain.NewVerificationReport(report.Execution(), domain.VerificationMismatch, report.Evidence())
}

func TestInstallUpdateInspectsNewPrimaryAndTransitiveBytesBeforePolicy(t *testing.T) {
	for _, mode := range []string{"primary-v2", "transitive-only-v2", "inspection-incomplete", "integrity-mismatch"} {
		t.Run(mode, func(t *testing.T) {
			old := twoNodeGraph(t)
			badName := "first"
			if mode == "transitive-only-v2" {
				badName = "second"
			}
			nodes := old.Nodes()
			for i, node := range nodes {
				identity := node.Artifact().Identity()
				if identity.Name() != badName {
					continue
				}
				identity, _ = domain.NewResolvedArtifactIdentity(identity.Source(), identity.Name(), "2.0.0", identity.Variant())
				artifact, _ := domain.NewResolvedArtifact(identity, strings.ReplaceAll(node.Artifact().AcquisitionLocator(), "1.0.0", "2.0.0"), "sha512-new-version")
				nodes[i], _ = domain.NewLockedDependency(node.Node(), node.Role(), artifact)
			}
			graph, err := domain.NewLockedDependencyGraph(nodes, old.Edges())
			if err != nil {
				t.Fatal(err)
			}
			fixtures := &updateFindingPorts{multiInspectionPorts: newMultiInspectionPorts(t, graph), badName: badName, mode: mode}
			inspection, err := application.NewInstallInspectService(&lockedResolver{graph: graph}, fixtures, fixtures, fixtures, fixtures, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			calls := []string{}
			base, _ := application.NewInstallService(inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls}, &promotionPort{calls: &calls})
			source, _ := domain.NewSourceID("npm")
			reference, _ := domain.NewArtifactReference(source, "first")
			target, _ := domain.NewInstallTarget("/tmp/guarded-npm-update")
			install, _ := domain.NewNPMProjectInstallContext(target)
			session := &updateSessionFixture{reference: reference, resolver: &lockedResolver{graph: graph}, promoter: &promotionPort{calls: &calls}, calls: &calls}
			updater, _ := application.NewInstallUpdateService(base, updatePreparationFixture{session: session, calls: &calls})
			request, _ := application.NewInstallUpdateRequest(source, "", install, false)
			outcome, err := updater.Install(context.Background(), request)
			want := domain.DecisionBlock
			if mode == "inspection-incomplete" {
				want = domain.DecisionManualReview
			}
			if err != nil || outcome.Inspection().Decision().Decision() != want || !reflect.DeepEqual(calls, []string{"begin", "resolve", "close"}) {
				t.Fatal("new unsafe/incomplete selection reached publication", outcome, calls, err)
			}
			for _, name := range []string{"first", "second"} {
				for _, phase := range []string{"acquire:", "verify:", "inspect:", "evidence:"} {
					if !containsCall(fixtures.calls, phase+name) {
						t.Fatal("selected dependency skipped", phase, name, fixtures.calls)
					}
				}
			}
			for _, item := range outcome.Inspection().Set().Inspections() {
				if item.Artifact().Identity().Name() == badName && item.Artifact().Identity().Version() != "2.0.0" {
					t.Fatal("stale version evidence reused")
				}
			}
		})
	}
}
