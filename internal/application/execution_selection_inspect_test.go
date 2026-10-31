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

type selectionInspectionPorts struct {
	*multiInspectionPorts
	selectionCalls int
	omitReport     bool
}

func (p *selectionInspectionPorts) InspectExecutionSelection(ctx context.Context, selection domain.ExecutionDependencySelection, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	p.selectionCalls++
	reports := map[domain.DependencyNodeID]domain.InspectionReport{}
	for _, node := range selection.Nodes() {
		report, err := p.Inspect(ctx, acquired[node.Node()])
		if err != nil {
			return nil, err
		}
		reports[node.Node()] = report
	}
	if p.omitReport {
		delete(reports, selection.Primary())
	}
	return reports, nil
}

type selectionInspectionPolicy struct {
	executionSetPolicy
	decision domain.Decision
	calls    int
	entries  int
}

func (p *selectionInspectionPolicy) EvaluateExecutionSelection(set domain.InspectedExecutionSelection) (domain.PolicyDecision, error) {
	p.calls++
	p.entries = len(set.Inspections())
	return domain.NewPolicyDecision(p.decision, "fixture-selection", 1, []string{"FIXTURE"})
}

func selectionInspectionFixture(t *testing.T) (domain.ExecutionSelectionResolution, *selectionInspectionPorts) {
	t.Helper()
	facts := twoNodeGraph(t)
	var roots []domain.LockedDependencyGraph
	for _, node := range facts.Nodes() {
		root, _ := domain.NewLockedDependencyWithRecordPath(node.Node(), domain.DependencyPrimary, node.Artifact(), node.RecordPath(), node.HostInstallAction())
		graph, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{root}, nil)
		if err != nil {
			t.Fatal(err)
		}
		roots = append(roots, graph)
	}
	selection, err := domain.NewExecutionDependencySelection(roots[0], roots[1:])
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := domain.NewSHA256Digest(strings.Repeat("c", 64))
	resolution, err := domain.NewExecutionSelectionResolution(selection, "actual-selection-runtime", digest)
	if err != nil {
		t.Fatal(err)
	}
	return resolution, &selectionInspectionPorts{multiInspectionPorts: newMultiInspectionPorts(t, facts)}
}

func TestExecutionSelectionInspectionUsesOneFreshWholeEntryLifecycle(t *testing.T) {
	resolution, p := selectionInspectionFixture(t)
	whole := &selectionInspectionPolicy{decision: domain.DecisionAllow}
	service, err := application.NewExecutionInspectService(p, p, p, p, policy.M1{}, whole, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for range 2 {
		result, err := service.InspectSelection(context.Background(), resolution)
		if err != nil || !result.Valid() || result.Resolution().LockfileDigest() != resolution.LockfileDigest() || seen[result.OperationID().String()] || len(result.Set().Inspected().Inspections()) != 2 {
			t.Fatal("whole fresh root selection missing", err)
		}
		seen[result.OperationID().String()] = true
		for _, entry := range result.Set().Inspected().Inspections() {
			if seen[entry.RunID().String()] || seen[entry.Artifact().ContentHandle()] {
				t.Fatal("prior acquisition or approval reused")
			}
			seen[entry.RunID().String()], seen[entry.Artifact().ContentHandle()] = true, true
		}
	}
	want := []string{"acquire:first", "verify:first", "acquire:second", "verify:second", "inspect:first", "inspect:second", "evidence:first", "evidence:second"}
	want = append(want, want...)
	if p.selectionCalls != 2 || whole.calls != 2 || whole.entries != 2 || !reflect.DeepEqual(p.calls, want) {
		t.Fatal("inspection preceded whole acquisition or shared entries were repeated", p.calls)
	}
}

func TestExecutionSelectionInspectionRefusesPartialAndLegacyAuthority(t *testing.T) {
	for _, mode := range []string{"missing report", "whole review", "legacy inspector", "legacy policy", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			resolution, p := selectionInspectionFixture(t)
			whole := &selectionInspectionPolicy{decision: domain.DecisionAllow}
			if mode == "whole review" {
				whole.decision = domain.DecisionManualReview
			}
			p.omitReport = mode == "missing report"
			service, err := application.NewExecutionInspectService(p, p, p, p, policy.M1{}, whole, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "legacy inspector" {
				service, err = application.NewExecutionInspectService(p, p, p.multiInspectionPorts, p, policy.M1{}, whole, domain.NewOperationID, domain.NewRunID)
			}
			if mode == "legacy policy" {
				service, err = application.NewExecutionInspectService(p, p, p, p, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
			}
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			cause := errors.New("original selection cancellation")
			if mode == "canceled" {
				var cancel context.CancelCauseFunc
				ctx, cancel = context.WithCancelCause(ctx)
				cancel(cause)
			}
			result, err := service.InspectSelection(ctx, resolution)
			if err == nil || result.Valid() || result.Set().Valid() {
				t.Fatal("partial or denied selection became approval")
			}
			if mode == "whole review" {
				var denied *application.ExecutionSelectionPolicyFailure
				if !errors.As(err, &denied) || !denied.Inspected().Valid() || denied.Decision().Decision() != domain.DecisionManualReview {
					t.Fatal("completed whole denial facts were lost", err)
				}
			}
			if mode == "legacy inspector" || mode == "legacy policy" || mode == "canceled" {
				if len(p.calls) != 0 || p.selectionCalls != 0 {
					t.Fatal("unsupported authority reached entry ports")
				}
			}
			if mode == "canceled" && !errors.Is(err, cause) {
				t.Fatal("original cancellation cause lost")
			}
		})
	}
}
