package policy

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestProjectSetPolicyUsesCompleteCoverageAndBlockPrecedence(t *testing.T) {
	for _, test := range []struct {
		name                string
		first, second, want domain.Decision
		status              domain.ExecutionStatus
		cause               string
	}{
		{"allow", domain.DecisionAllow, domain.DecisionAllow, domain.DecisionAllow, domain.ExecutionCompleted, "M4_VERIFIED_SET_COMPLETED"},
		{"review", domain.DecisionAllow, domain.DecisionManualReview, domain.DecisionManualReview, domain.ExecutionCompleted, "M4_DEPENDENCY_REVIEW_REQUIRED"},
		{"block", domain.DecisionManualReview, domain.DecisionBlock, domain.DecisionBlock, domain.ExecutionCompleted, "M4_DEPENDENCY_BLOCKED"},
		{"incomplete", domain.DecisionAllow, domain.DecisionAllow, domain.DecisionManualReview, domain.ExecutionIncomplete, "M4_REQUIRED_CHECK_INCOMPLETE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			old := m4Set(t, test.first, test.second, false)
			target, err := domain.NewInstallTarget(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			ctx, err := domain.NewInstallContext(target)
			if err != nil {
				t.Fatal(err)
			}
			hash, err := domain.NewSHA256Digest(strings.Repeat("a", 64))
			if err != nil {
				t.Fatal(err)
			}
			control, err := domain.NewProjectControlDigest("fixture.lock", hash)
			if err != nil {
				t.Fatal(err)
			}
			var dependencies []domain.ResolvedArtifact
			for _, node := range old.Graph().Nodes() {
				dependencies = append(dependencies, node.Artifact())
			}
			snapshot, err := domain.NewProjectDependencySnapshot(ctx, dependencies[0].Identity().Source(), []domain.ProjectControlDigest{control}, dependencies, hash)
			if err != nil {
				t.Fatal(err)
			}
			var entries []domain.DependencyInspection
			for index, i := range old.Inspections() {
				node, err := domain.ProjectDependencyNodeID(i.Artifact().Identity())
				if err != nil {
					t.Fatal(err)
				}
				checkID, err := domain.NewCheckID("fixture-verification")
				if err != nil {
					t.Fatal(err)
				}
				status, limit := domain.ExecutionCompleted, ""
				if index == 1 && test.status != domain.ExecutionCompleted {
					status, limit = test.status, "FIXTURE_INCOMPLETE"
				}
				check, err := domain.NewCheckExecution(checkID, domain.CheckVerification, true, domain.CapabilitySupported, status, limit)
				if err != nil {
					t.Fatal(err)
				}
				checks := append(i.Checks(), check)
				entry, err := domain.NewDependencyInspection(node, i.RunID(), i.Artifact(), checks, i.Evidence(), i.PolicyDecision())
				if err != nil {
					t.Fatal(err)
				}
				entries = append(entries, entry)
			}
			set, err := domain.NewInspectedProjectSet(snapshot, entries)
			if err != nil {
				t.Fatal(err)
			}
			decision, err := (M4{}).EvaluateProjectSet(set)
			if err != nil || decision.Decision() != test.want || len(decision.Reasons()) != 1 || decision.Reasons()[0] != test.cause {
				t.Fatalf("decision=%v err=%v", decision, err)
			}
		})
	}
	if _, err := (M4{}).EvaluateProjectSet(domain.InspectedProjectSet{}); err == nil {
		t.Fatal("empty project set approved")
	}
}
