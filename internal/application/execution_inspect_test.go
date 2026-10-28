package application_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/policy"
)

func executionResolution(t *testing.T, graph domain.LockedDependencyGraph) domain.DependencyResolution {
	t.Helper()
	lock, _ := domain.NewSHA256Digest(strings.Repeat("c", 64))
	resolution, err := domain.NewDependencyResolution(graph, "original-selected-runtime", lock)
	if err != nil {
		t.Fatal(err)
	}
	return resolution
}

func TestExecutionInspectionAlwaysCreatesFreshWholeSet(t *testing.T) {
	graph := twoNodeGraph(t)
	resolution := executionResolution(t, graph)
	p := newMultiInspectionPorts(t, graph)
	service, err := application.NewExecutionInspectService(p, p, p, p, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	var prior application.ExecutionInspection
	for i := 0; i < 2; i++ {
		result, err := service.InspectResolved(context.Background(), resolution)
		if err != nil || !result.Valid() || !result.Set().Valid() || result.Resolution().LockfileDigest() != resolution.LockfileDigest() || result.Resolution().RuntimeIdentity() != resolution.RuntimeIdentity() || len(result.Set().Inspected().Inspections()) != 2 {
			t.Fatal("fresh complete non-install inspection failed", err)
		}
		inspections := result.Set().Inspected().Inspections()
		if inspections[0].RunID() == inspections[1].RunID() {
			t.Fatal("entries shared a run")
		}
		if prior.Valid() {
			if prior.OperationID() == result.OperationID() {
				t.Fatal("repeated request reused operation")
			}
			for j, entry := range inspections {
				old := prior.Set().Inspected().Inspections()[j]
				if entry.RunID() == old.RunID() || entry.Artifact().ContentHandle() == old.Artifact().ContentHandle() {
					t.Fatal("same resolution reused earlier acquisition/approval")
				}
			}
		}
		prior = result
	}
	want := []string{"acquire:first", "verify:first", "inspect:first", "evidence:first", "acquire:second", "verify:second", "inspect:second", "evidence:second"}
	want = append(want, want...)
	if !reflect.DeepEqual(p.calls, want) {
		t.Fatal("current complete entry lifecycle bypassed", p.calls)
	}
	if (application.ExecutionInspection{}).Valid() {
		t.Fatal("zero result accepted")
	}
}

func TestExecutionInspectionReusesGraphCapabilityAfterAllAcquisition(t *testing.T) {
	graph := twoNodeGraph(t)
	p := &graphInspectionPorts{multiInspectionPorts: newMultiInspectionPorts(t, graph)}
	service, err := application.NewExecutionInspectService(p, p, p, p, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.InspectResolved(context.Background(), executionResolution(t, graph))
	if err != nil || !result.Valid() || p.graphCalls != 1 {
		t.Fatal("graph capability bypassed", err)
	}
	want := []string{"acquire:first", "verify:first", "acquire:second", "verify:second", "graph", "evidence:first", "evidence:second"}
	if !reflect.DeepEqual(p.calls, want) {
		t.Fatal("graph ran before whole acquired/verified coverage", p.calls)
	}

	for _, mode := range []string{"missing", "foreign-handle"} {
		t.Run(mode, func(t *testing.T) {
			base := &graphInspectionPorts{multiInspectionPorts: newMultiInspectionPorts(t, graph), omitReport: mode == "missing"}
			if mode == "missing" {
				s, err := application.NewExecutionInspectService(base, base, base, base, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
				if err != nil {
					t.Fatal(err)
				}
				if result, err := s.InspectResolved(context.Background(), executionResolution(t, graph)); err == nil || result.Valid() || result.Set().Valid() {
					t.Fatal("missing graph report approved")
				}
			} else {
				bad := &badHandleGraphPorts{base}
				s, err := application.NewExecutionInspectService(bad, bad, bad, bad, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
				if err != nil {
					t.Fatal(err)
				}
				if result, err := s.InspectResolved(context.Background(), executionResolution(t, graph)); err == nil || result.Valid() || base.graphCalls != 0 {
					t.Fatal("substituted acquisition reached graph/approval")
				}
			}
		})
	}
}

func TestExecutionInspectionNeverReturnsPartialOrDeniedApproval(t *testing.T) {
	for _, failure := range []string{"verify", "mismatch", "evidence"} {
		t.Run(failure, func(t *testing.T) {
			graph := twoNodeGraph(t)
			p := &projectFailingPorts{newArchiveActionPorts(t, graph), failure}
			s, err := application.NewExecutionInspectService(p, p, p, p, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			result, err := s.InspectResolved(context.Background(), executionResolution(t, graph))
			if err == nil || result.Valid() || result.Set().Valid() || result.OperationID().String() == "" {
				t.Fatal("failed entry approved or operation lost")
			}
			if failure == "mismatch" {
				var denied *application.ExecutionPolicyFailure
				if !errors.As(err, &denied) || denied.Decision().Decision() != domain.DecisionBlock || len(denied.Inspected().Inspections()) != 2 {
					t.Fatal("complete denied facts lost", err)
				}
			}
		})
	}
}

type executionSetPolicy struct {
	err        error
	cancel     context.CancelFunc
	deny       bool
	forceAllow bool
}

func (p executionSetPolicy) EvaluateSet(set domain.InspectedDependencySet) (domain.PolicyDecision, error) {
	if p.cancel != nil {
		p.cancel()
	}
	if p.err != nil {
		return domain.PolicyDecision{}, p.err
	}
	if p.deny {
		return domain.NewPolicyDecision(domain.DecisionManualReview, "fixture-set", 1, []string{"REVIEW"})
	}
	if p.forceAllow {
		return domain.NewPolicyDecision(domain.DecisionAllow, "fixture-set", 1, []string{"ALLOW"})
	}
	return (policy.M4{}).EvaluateSet(set)
}

func TestExecutionInspectionRetainsEntryDenialDespiteSetAllow(t *testing.T) {
	graph := twoNodeGraph(t)
	p := &projectFailingPorts{newArchiveActionPorts(t, graph), "mismatch"}
	s, err := application.NewExecutionInspectService(p, p, p, p, policy.M3{}, executionSetPolicy{forceAllow: true}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	if result, err := s.InspectResolved(context.Background(), executionResolution(t, graph)); err == nil || result.Valid() || result.Set().Valid() {
		t.Fatal("set ALLOW overrode denied dependency")
	}
}

func TestExecutionInspectionPreservesLaterRunIDFailure(t *testing.T) {
	graph := twoNodeGraph(t)
	p := newMultiInspectionPorts(t, graph)
	original := errors.New("second run ID generation failed")
	calls := 0
	generate := func() (domain.RunID, error) {
		calls++
		if calls == 2 {
			return domain.RunID{}, original
		}
		return domain.NewRunID()
	}
	s, err := application.NewExecutionInspectService(p, p, p, p, policy.M1{}, policy.M4{}, domain.NewOperationID, generate)
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.InspectResolved(context.Background(), executionResolution(t, graph))
	if !errors.Is(err, original) || result.Valid() || result.Set().Valid() || result.OperationID().String() == "" || calls != 2 || len(p.calls) != 4 {
		t.Fatal("later ID failure lost cause or approved partial graph", err, p.calls)
	}
}

func TestExecutionInspectionRejectsDerivedGraphBeforeEntryPorts(t *testing.T) {
	source, _ := domain.NewSourceID("pypi")
	primary, _ := domain.NewResolvedArtifactIdentity(source, "first", "1.0.0", "sdist")
	primaryResolved, _ := domain.NewResolvedArtifact(primary, "https://example.test/first-1.0.0.tar.gz", "sha256:"+strings.Repeat("a", 64))
	primaryNode, _ := domain.NewDependencyNodeID("first")
	primaryDependency, _ := domain.NewLockedDependency(primaryNode, domain.DependencyPrimary, primaryResolved)
	graph, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{primaryDependency}, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := newMultiInspectionPorts(t, graph)
	identity, _ := domain.NewResolvedArtifactIdentity(primary.Source(), primary.Name(), primary.Version(), "derived-wheel")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("b", 64))
	resolved, _ := domain.NewResolvedArtifact(identity, "derived:first", "sha256:"+digest.String())
	node, _ := domain.NewDependencyNodeID("first-derived")
	dependency, _ := domain.NewLockedDependency(node, domain.DependencyTransitive, resolved)
	run, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+run.String()+":derived-wheel", 1, resolved.DeclaredIntegrity())
	if err != nil {
		t.Fatal(err)
	}
	binding, err := domain.NewDerivationBinding(digest, []domain.ContentDigest{digest}, "fixture-build", digest)
	if err != nil {
		t.Fatal(err)
	}
	checkID, _ := domain.NewCheckID("fixture-derived-build")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidenceID, _ := domain.NewEvidenceID("fixture-derived-build")
	evidence, _ := domain.NewEvidence(evidenceID, checkID, identity, digest, "fixture-build", "Fixture build completed")
	derived, err := domain.NewDerivedDependency(graph.Primary(), dependency, artifact, binding, check, evidence)
	if err != nil {
		t.Fatal(err)
	}
	extended, err := domain.ExtendLockedDependencyGraph(graph, []domain.DerivedDependency{derived})
	if err != nil {
		t.Fatal(err)
	}
	s, err := application.NewExecutionInspectService(p, p, p, p, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	if result, err := s.InspectResolved(context.Background(), executionResolution(t, extended)); err == nil || result.Valid() || len(p.calls) != 0 {
		t.Fatal("unsupported derived graph reached entry ports")
	}
}

func TestExecutionInspectionPreservesSetFailureAndFinalCancellation(t *testing.T) {
	original := errors.New("original set provider failure")
	for _, mode := range []string{"error", "review", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			graph := twoNodeGraph(t)
			p := newMultiInspectionPorts(t, graph)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			setPolicy := executionSetPolicy{deny: mode == "review"}
			if mode == "error" {
				setPolicy.err = original
			}
			if mode == "cancel" {
				setPolicy.cancel = cancel
			}
			s, err := application.NewExecutionInspectService(p, p, p, p, policy.M1{}, setPolicy, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			result, err := s.InspectResolved(ctx, executionResolution(t, graph))
			if err == nil || result.Valid() || result.Set().Valid() || result.OperationID().String() == "" || result.Resolution().LockfileDigest().String() == "" {
				t.Fatal("failed set approved or original context lost")
			}
			switch mode {
			case "error":
				if !errors.Is(err, original) {
					t.Fatal("original provider cause lost", err)
				}
			case "cancel":
				if !errors.Is(err, context.Canceled) {
					t.Fatal("cancellation lost", err)
				}
			case "review":
				var denied *application.ExecutionPolicyFailure
				if !errors.As(err, &denied) || denied.Decision().Decision() != domain.DecisionManualReview || !denied.Inspected().Valid() {
					t.Fatal("manual review facts lost", err)
				}
			}
		})
	}
}

type executionGraphFailurePorts struct {
	*graphInspectionPorts
	providerErr, recordErr error
	mode                   string
	records, policies      int
}

func (p *executionGraphFailurePorts) InspectGraph(ctx context.Context, graph domain.LockedDependencyGraph, artifacts map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	reports, err := p.graphInspectionPorts.InspectGraph(ctx, graph, artifacts)
	if err != nil {
		return reports, err
	}
	if p.mode == "foreign" {
		nodes := graph.Nodes()
		reports[nodes[1].Node()] = reports[nodes[0].Node()]
	}
	return reports, p.providerErr
}
func (p *executionGraphFailurePorts) Record(ctx context.Context, run domain.RunID, evidence []domain.Evidence) ([]domain.EvidenceReference, error) {
	p.records++
	if p.recordErr != nil {
		return nil, p.recordErr
	}
	return p.multiInspectionPorts.Record(ctx, run, evidence)
}
func (p *executionGraphFailurePorts) Evaluate(input domain.PolicyInput) (domain.PolicyDecision, error) {
	p.policies++
	return (policy.M1{}).Evaluate(input)
}

func TestExecutionGraphFailureRetainsOnlyExactFactsAndOriginalCauses(t *testing.T) {
	provider := errors.New("original graph provider failure")
	store := errors.New("independent evidence store failure")
	for _, mode := range []string{"exact", "missing", "foreign", "store"} {
		t.Run(mode, func(t *testing.T) {
			graph := twoNodeGraph(t)
			p := &executionGraphFailurePorts{graphInspectionPorts: &graphInspectionPorts{multiInspectionPorts: newMultiInspectionPorts(t, graph), omitReport: mode == "missing"}, providerErr: provider, mode: mode}
			if mode == "store" {
				p.recordErr = store
			}
			s, err := application.NewExecutionInspectService(p, p, p, p, p, policy.M4{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			result, err := s.InspectResolved(context.Background(), executionResolution(t, graph))
			if !errors.Is(err, provider) || result.Valid() || result.Set().Valid() || p.policies != 0 {
				t.Fatal("provider failure became approval or original cause lost", err)
			}
			wantRecords := 2
			if mode == "missing" || mode == "foreign" {
				wantRecords = 0
			}
			if p.records != wantRecords {
				t.Fatal("inexact failed graph facts recorded", p.records)
			}
			if mode == "store" && !errors.Is(err, store) {
				t.Fatal("cleanup/store replaced original error", err)
			}
		})
	}
}

func TestExecutionInspectionRejectsInvalidInputBeforeEntryPorts(t *testing.T) {
	graph := twoNodeGraph(t)
	p := newMultiInspectionPorts(t, graph)
	resolution := executionResolution(t, graph)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s, err := application.NewExecutionInspectService(p, p, p, p, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []struct {
		ctx        context.Context
		resolution domain.DependencyResolution
	}{
		{nil, resolution}, {ctx, resolution}, {context.Background(), domain.DependencyResolution{}},
	} {
		if result, err := s.InspectResolved(input.ctx, input.resolution); err == nil || result.Valid() {
			t.Fatal("invalid input accepted")
		}
	}
	if len(p.calls) != 0 {
		t.Fatal("entry ports called before input preflight", p.calls)
	}
	var absent *application.ExecutionInspectService
	if result, err := absent.InspectResolved(context.Background(), resolution); err == nil || result.Valid() {
		t.Fatal("nil service accepted")
	}
	if service, err := application.NewExecutionInspectService(nil, p, p, p, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID); err == nil || service != nil {
		t.Fatal("missing adapter accepted")
	}
	original := errors.New("original operation ID failure")
	for _, mode := range []string{"error", "zero", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			generate := func() (domain.OperationID, error) {
				if mode == "error" {
					return domain.OperationID{}, original
				}
				if mode == "zero" {
					return domain.OperationID{}, nil
				}
				cancel()
				return domain.NewOperationID()
			}
			s, err := application.NewExecutionInspectService(p, p, p, p, policy.M1{}, policy.M4{}, generate, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			if result, err := s.InspectResolved(ctx, resolution); err == nil || result.Valid() || len(p.calls) != 0 || mode == "error" && !errors.Is(err, original) || mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal("ID failure/cancel reached entry ports or lost cause", err)
			}
		})
	}
}
