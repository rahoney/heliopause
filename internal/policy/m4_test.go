package policy

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestM4AggregatesEveryDependencyDecisionFailClosed(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		first     domain.Decision
		second    domain.Decision
		want      domain.Decision
		wantCause string
	}{
		{"all allow", domain.DecisionAllow, domain.DecisionAllow, domain.DecisionAllow, "M4_VERIFIED_SET_COMPLETED"},
		{"manual review", domain.DecisionAllow, domain.DecisionManualReview, domain.DecisionManualReview, "M4_DEPENDENCY_REVIEW_REQUIRED"},
		{"block takes precedence", domain.DecisionManualReview, domain.DecisionBlock, domain.DecisionBlock, "M4_DEPENDENCY_BLOCKED"},
	} {
		t.Run(test.name, func(t *testing.T) {
			set := m4Set(t, test.first, test.second, false)
			decision, err := (M4{}).EvaluateSet(set)
			if err != nil || decision.Decision() != test.want || len(decision.Reasons()) != 1 || decision.Reasons()[0] != test.wantCause {
				t.Fatalf("EvaluateSet() = (%#v, %v)", decision, err)
			}
		})
	}
}

func TestM4RequiresManualReviewForHostInstallAction(t *testing.T) {
	t.Parallel()
	set := m4Set(t, domain.DecisionAllow, domain.DecisionAllow, true)
	decision, err := (M4{}).EvaluateSet(set)
	if err != nil || decision.Decision() != domain.DecisionManualReview || decision.Reasons()[0] != "M4_HOST_LIFECYCLE_UNSUPPORTED" {
		t.Fatalf("EvaluateSet() = (%#v, %v)", decision, err)
	}
}

func m4Set(t *testing.T, firstDecision, secondDecision domain.Decision, hostInstallAction bool) domain.InspectedDependencySet {
	t.Helper()
	source, _ := domain.NewSourceID("npm")
	firstID, _ := domain.NewDependencyNodeID("first")
	secondID, _ := domain.NewDependencyNodeID("second")
	firstIdentity, _ := domain.NewResolvedArtifactIdentity(source, "first", "1.0.0", "tarball")
	secondIdentity, _ := domain.NewResolvedArtifactIdentity(source, "second", "1.0.0", "tarball")
	firstResolved, _ := domain.NewResolvedArtifact(firstIdentity, "registry:npm:first", "sha512-first")
	secondResolved, _ := domain.NewResolvedArtifact(secondIdentity, "registry:npm:second", "sha512-second")
	first, _ := domain.NewLockedDependencyWithHostInstallAction(firstID, domain.DependencyPrimary, firstResolved, hostInstallAction)
	second, _ := domain.NewLockedDependency(secondID, domain.DependencyTransitive, secondResolved)
	edge, _ := domain.NewDependencyEdge(firstID, secondID)
	graph, _ := domain.NewLockedDependencyGraph([]domain.LockedDependency{first, second}, []domain.DependencyEdge{edge})
	build := func(dependency domain.LockedDependency, digestCharacter string, decision domain.Decision) domain.DependencyInspection {
		run, runErr := domain.NewRunID()
		if runErr != nil {
			t.Fatal(runErr)
		}
		digest, _ := domain.NewSHA256Digest(strings.Repeat(digestCharacter, 64))
		artifact, _ := domain.NewAcquiredArtifactWithDeclaredIntegrity(dependency.Artifact().Identity(), digest, "intake:"+run.String()+":tarball", 1, dependency.Artifact().DeclaredIntegrity())
		checkID, _ := domain.NewCheckID("fixture-check")
		check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		evidenceID, _ := domain.NewEvidenceID("fixture-evidence-" + dependency.Node().String())
		reference, _ := domain.NewEvidenceReference(evidenceID, "fixture:"+evidenceID.String())
		policyDecision, policyErr := domain.NewPolicyDecision(decision, "fixture-policy", 1, []string{"FIXTURE_RESULT"})
		if policyErr != nil {
			t.Fatal(policyErr)
		}
		inspection, err := domain.NewDependencyInspection(dependency.Node(), run, artifact, []domain.CheckExecution{check}, []domain.EvidenceReference{reference}, policyDecision)
		if err != nil {
			t.Fatal(err)
		}
		return inspection
	}
	set, err := domain.NewInspectedDependencySet(graph, []domain.DependencyInspection{
		build(first, "a", firstDecision),
		build(second, "b", secondDecision),
	})
	if err != nil {
		t.Fatal(err)
	}
	return set
}
