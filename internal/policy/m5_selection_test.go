package policy

import (
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestM5ExecutionSelectionPreservesWholeSetRules(t *testing.T) {
	for _, row := range []struct {
		name, variant, reason string
		entry, want           domain.Decision
		status                domain.ExecutionStatus
	}{
		{"allow", "wheel", "M5_VERIFIED_SET_COMPLETED", domain.DecisionAllow, domain.DecisionAllow, domain.ExecutionCompleted},
		{"supplemental review", "wheel", "M5_DISTRIBUTION_REVIEW_REQUIRED", domain.DecisionManualReview, domain.DecisionManualReview, domain.ExecutionCompleted},
		{"supplemental block", "wheel", "M5_DISTRIBUTION_BLOCKED", domain.DecisionBlock, domain.DecisionBlock, domain.ExecutionCompleted},
		{"supplemental required incomplete", "wheel", "M5_REQUIRED_CHECK_INCOMPLETE", domain.DecisionAllow, domain.DecisionManualReview, domain.ExecutionIncomplete},
		{"unsupported distribution", "tarball", "M5_NON_WHEEL_PROMOTION_UNAVAILABLE", domain.DecisionAllow, domain.DecisionManualReview, domain.ExecutionCompleted},
	} {
		t.Run(row.name, func(t *testing.T) {
			legacy, selection := m5SelectionFacts(t, row.variant, row.entry, row.status)
			first, err := (M5{}).EvaluateSet(legacy)
			if err != nil {
				t.Fatal(err)
			}
			whole, err := (M5{}).EvaluateExecutionSelection(selection)
			if err != nil || whole.Decision() != row.want || whole.PolicyID() != "m5-pypi-pip" || len(whole.Reasons()) != 1 || whole.Reasons()[0] != row.reason || first.Decision() != whole.Decision() || first.Reasons()[0] != whole.Reasons()[0] {
				t.Fatal("whole supplemental-root rules differ from existing M5", whole, err)
			}
		})
	}
	if _, err := (M5{}).EvaluateExecutionSelection(domain.InspectedExecutionSelection{}); err == nil {
		t.Fatal("uninspected roots became a Policy input")
	}
}

func m5SelectionFacts(t *testing.T, variant string, supplementalDecision domain.Decision, supplementalStatus domain.ExecutionStatus) (domain.InspectedDependencySet, domain.InspectedExecutionSelection) {
	t.Helper()
	var nodes []domain.LockedDependency
	var roots []domain.LockedDependencyGraph
	var inspections []domain.DependencyInspection
	for index, name := range []string{"first", "second"} {
		id, _ := domain.NewDependencyNodeID(name)
		identity, _ := domain.NewResolvedArtifactIdentity(artifactpypi.PublicPyPIProfile().Source(), name, "1.0.0", variant)
		digest, _ := domain.NewSHA256Digest(strings.Repeat(string(rune('a'+index)), 64))
		resolved, _ := domain.NewResolvedArtifact(identity, "https://files.pythonhosted.org/packages/"+name+".whl", "sha256:"+digest.String())
		root, err := domain.NewLockedDependency(id, domain.DependencyPrimary, resolved)
		if err != nil {
			t.Fatal(err)
		}
		graph, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{root}, nil)
		if err != nil {
			t.Fatal(err)
		}
		roots = append(roots, graph)
		role := domain.DependencyPrimary
		entryDecision, status := domain.DecisionAllow, domain.ExecutionCompleted
		if index == 1 {
			role, entryDecision, status = domain.DependencyTransitive, supplementalDecision, supplementalStatus
		}
		node, _ := domain.NewLockedDependency(id, role, resolved)
		nodes = append(nodes, node)
		run, err := domain.NewRunID()
		if err != nil {
			t.Fatal(err)
		}
		acquired, _ := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+run.String()+":"+variant, 1, resolved.DeclaredIntegrity())
		verificationID, _ := domain.NewCheckID("verification")
		inspectionID, _ := domain.NewCheckID("inspection")
		verification, _ := domain.NewCheckExecution(verificationID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		failure := ""
		if status != domain.ExecutionCompleted {
			failure = "FIXTURE_INCOMPLETE"
		}
		inspection, err := domain.NewCheckExecution(inspectionID, domain.CheckInspection, true, domain.CapabilitySupported, status, failure)
		if err != nil {
			t.Fatal(err)
		}
		evidenceID, _ := domain.NewEvidenceID("fixture-" + name)
		reference, _ := domain.NewEvidenceReference(evidenceID, "fixture:"+name)
		decision, _ := domain.NewPolicyDecision(entryDecision, "fixture", 1, []string{"FIXTURE"})
		entry, err := domain.NewDependencyInspection(id, run, acquired, []domain.CheckExecution{verification, inspection}, []domain.EvidenceReference{reference}, decision)
		if err != nil {
			t.Fatal(err)
		}
		inspections = append(inspections, entry)
	}
	edge, _ := domain.NewDependencyEdge(nodes[0].Node(), nodes[1].Node())
	graph, err := domain.NewLockedDependencyGraph(nodes, []domain.DependencyEdge{edge})
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := domain.NewInspectedDependencySet(graph, inspections)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := domain.NewExecutionDependencySelection(roots[0], roots[1:])
	if err != nil {
		t.Fatal(err)
	}
	whole, err := domain.NewInspectedExecutionSelection(selected, inspections)
	if err != nil {
		t.Fatal(err)
	}
	return legacy, whole
}
