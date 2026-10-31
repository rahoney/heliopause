package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/policy"
)

type contentUpdateFixture struct {
	*updateSessionFixture
	ports.Artifact
	ports.Verification
	ports.Inspection
}

var _ ports.InstallUpdateContentSession = (*contentUpdateFixture)(nil)

type selectedUpdatePreparation map[string]ports.InstallUpdateSession

func (p selectedUpdatePreparation) BeginInstallUpdate(_ context.Context, _ domain.SourceID, selector string, _ domain.InstallContext, _ bool) (ports.InstallUpdateSession, error) {
	session := p[selector]
	if session == nil {
		return nil, errors.New("unknown test selection")
	}
	return session, nil
}

type concurrentUpdateEvidence struct{}

func (concurrentUpdateEvidence) Record(_ context.Context, run domain.RunID, items []domain.Evidence) ([]domain.EvidenceReference, error) {
	refs := make([]domain.EvidenceReference, 0, len(items))
	for _, item := range items {
		ref, err := domain.NewEvidenceReference(item.ID(), "fixture:"+run.String()+":"+item.ID().String())
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

type concurrentUpdateStaging struct{}

func (concurrentUpdateStaging) Stage(_ context.Context, bundle domain.VerifiedBundle) (domain.StagedSet, error) {
	return domain.NewStagedSet(bundle.ManifestID(), "staging:"+bundle.ManifestID().String())
}

type overlappingUpdateResolver struct {
	ports.DependencyResolver
	ready   chan<- struct{}
	release <-chan struct{}
}

func (r overlappingUpdateResolver) ResolveDependencies(ctx context.Context, ref domain.ArtifactReference, install domain.InstallContext) (domain.DependencyResolution, error) {
	select {
	case r.ready <- struct{}{}:
	case <-ctx.Done():
		return domain.DependencyResolution{}, ctx.Err()
	}
	select {
	case <-r.release:
		return r.DependencyResolver.ResolveDependencies(ctx, ref, install)
	case <-ctx.Done():
		return domain.DependencyResolution{}, ctx.Err()
	}
}

func versionedUpdateGraph(t *testing.T, version string) domain.LockedDependencyGraph {
	t.Helper()
	old := twoNodeGraph(t)
	nodes := old.Nodes()
	for i, node := range nodes {
		id := node.Artifact().Identity()
		identity, err := domain.NewResolvedArtifactIdentity(id.Source(), id.Name(), version, id.Variant())
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := domain.NewResolvedArtifact(identity, strings.ReplaceAll(node.Artifact().AcquisitionLocator(), "1.0.0", version), node.Artifact().DeclaredIntegrity())
		if err != nil {
			t.Fatal(err)
		}
		nodes[i], err = domain.NewLockedDependency(node.Node(), node.Role(), artifact)
		if err != nil {
			t.Fatal(err)
		}
	}
	graph, err := domain.NewLockedDependencyGraph(nodes, old.Edges())
	if err != nil {
		t.Fatal(err)
	}
	return graph
}

func TestInstallUpdateContentIsPerRequestUnderConcurrentUpdates(t *testing.T) {
	old, _ := installWorkflowFixture(t)
	oldGraph := twoNodeGraph(t)
	oldPorts := newMultiInspectionPorts(t, oldGraph)
	inspection, err := application.NewInstallInspectService(&lockedResolver{graph: oldGraph}, oldPorts, oldPorts, oldPorts, concurrentUpdateEvidence{}, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	baseCalls := []string{}
	base, err := application.NewInstallService(inspection, evidence.Generator{}, concurrentUpdateStaging{}, &promotionPort{calls: &baseCalls})
	if err != nil {
		t.Fatal(err)
	}
	ready, release := make(chan struct{}, 2), make(chan struct{})
	sessions := selectedUpdatePreparation{}
	content := map[string]*multiInspectionPorts{}
	calls := map[string]*[]string{}
	for _, version := range []string{"2.0.0", "3.0.0"} {
		graph := versionedUpdateGraph(t, version)
		fresh := newMultiInspectionPorts(t, graph)
		log := []string{}
		sessions[version] = &contentUpdateFixture{
			updateSessionFixture: &updateSessionFixture{reference: old.Reference(), resolver: overlappingUpdateResolver{&lockedResolver{graph: graph}, ready, release}, promoter: &promotionPort{calls: &log}, calls: &log},
			Artifact:             fresh, Verification: fresh, Inspection: fresh,
		}
		content[version], calls[version] = fresh, &log
	}
	updater, err := application.NewInstallUpdateService(base, sessions)
	if err != nil {
		t.Fatal(err)
	}
	install, _ := domain.NewNPMProjectInstallContext(old.Context().Target())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	type result struct {
		version string
		outcome application.InstallOutcome
		err     error
	}
	results := make(chan result, 2)
	for _, version := range []string{"2.0.0", "3.0.0"} {
		request, err := application.NewInstallUpdateRequest(old.Source(), version, install, false)
		if err != nil {
			t.Fatal(err)
		}
		go func() { outcome, err := updater.Install(ctx, request); results <- result{version, outcome, err} }()
	}
	for range 2 {
		select {
		case <-ready:
		case <-ctx.Done():
			t.Fatal("updates did not overlap", ctx.Err())
		}
	}
	close(release)
	operations := map[domain.OperationID]bool{}
	for range 2 {
		r := <-results
		if r.err != nil || r.outcome.Promoted().Target() != install.Target() {
			t.Fatal(r)
		}
		op := r.outcome.Inspection().OperationID()
		if operations[op] {
			t.Fatal("shared operation ID")
		}
		operations[op] = true
		for _, item := range r.outcome.Inspection().Set().Inspections() {
			if item.Artifact().Identity().Version() != r.version {
				t.Fatal("cross-request content", r.version, item.Artifact().Identity())
			}
		}
		for _, name := range []string{"first", "second"} {
			for _, phase := range []string{"acquire:", "verify:", "inspect:"} {
				if !containsCall(content[r.version].calls, phase+name) {
					t.Fatal("session port skipped", r.version, phase, name)
				}
			}
		}
		if strings.Join(*calls[r.version], ",") != "resolve,promotion,close" {
			t.Fatal(*calls[r.version])
		}
	}
	if len(oldPorts.calls) != 0 || len(baseCalls) != 0 {
		t.Fatal("base ports used by an update", oldPorts.calls, baseCalls)
	}
	if _, err := base.Install(context.Background(), old); err != nil {
		t.Fatal(err)
	}
	if !containsCall(oldPorts.calls, "acquire:first") || strings.Join(baseCalls, ",") != "promotion" {
		t.Fatal("base bindings changed", oldPorts.calls, baseCalls)
	}
}

func TestInstallUpdateContentStillRequiresExistingPolicyBeforePublication(t *testing.T) {
	for _, mode := range []string{"inspection-incomplete", "blocked"} {
		t.Run(mode, func(t *testing.T) {
			old, inspection := installWorkflowFixture(t)
			calls := []string{}
			base, _ := application.NewInstallService(inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls}, &promotionPort{calls: &calls})
			graph := versionedUpdateGraph(t, "2.0.0")
			fresh := &updateFindingPorts{multiInspectionPorts: newMultiInspectionPorts(t, graph), badName: "second", mode: mode, findingCode: "M1_BLOCK_FINDING"}
			session := &contentUpdateFixture{&updateSessionFixture{reference: old.Reference(), resolver: &lockedResolver{graph: graph}, promoter: &promotionPort{calls: &calls}, calls: &calls}, fresh, fresh, fresh}
			updater, _ := application.NewInstallUpdateService(base, updatePreparationFixture{session: session, calls: &calls})
			install, _ := domain.NewNPMProjectInstallContext(old.Context().Target())
			request, _ := application.NewInstallUpdateRequest(old.Source(), "2.0.0", install, false)
			outcome, err := updater.Install(context.Background(), request)
			want := domain.DecisionBlock
			if mode == "inspection-incomplete" {
				want = domain.DecisionManualReview
			}
			if err != nil || outcome.Inspection().Decision().Decision() != want || strings.Join(calls, ",") != "begin,resolve,close" {
				t.Fatal(outcome, calls, err)
			}
			if !containsCall(fresh.calls, "inspect:second") {
				t.Fatal("new session content was not inspected", fresh.calls)
			}
		})
	}
}
