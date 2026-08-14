package domain_test

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestNewInspectedDependencySetRequiresCompleteMatchingGraphCoverage(t *testing.T) {
	t.Parallel()

	graph, first, second := inspectedGraph(t)
	firstInspection := dependencyInspection(t, first, "a")
	secondInspection := dependencyInspection(t, second, "b")
	if _, err := domain.NewInspectedDependencySet(graph, []domain.DependencyInspection{firstInspection}); err == nil {
		t.Fatal("NewInspectedDependencySet() accepted partial graph coverage")
	}
	if _, err := domain.NewInspectedDependencySet(graph, []domain.DependencyInspection{firstInspection, firstInspection}); err == nil {
		t.Fatal("NewInspectedDependencySet() accepted duplicate node coverage")
	}
	set, err := domain.NewInspectedDependencySet(graph, []domain.DependencyInspection{secondInspection, firstInspection})
	if err != nil {
		t.Fatal(err)
	}
	if !set.Valid() || len(set.Inspections()) != 2 || set.Inspections()[0].Node().String() != "first" {
		t.Fatalf("set = %#v", set)
	}
}

func inspectedGraph(t *testing.T) (domain.LockedDependencyGraph, domain.LockedDependency, domain.LockedDependency) {
	t.Helper()
	source, _ := domain.NewSourceID("npm")
	firstID, _ := domain.NewDependencyNodeID("first")
	secondID, _ := domain.NewDependencyNodeID("second")
	firstIdentity, _ := domain.NewResolvedArtifactIdentity(source, "first", "1.0.0", "tarball")
	secondIdentity, _ := domain.NewResolvedArtifactIdentity(source, "second", "2.0.0", "tarball")
	firstResolved, _ := domain.NewResolvedArtifact(firstIdentity, "registry:npm:first@1.0.0", "sha512-first")
	secondResolved, _ := domain.NewResolvedArtifact(secondIdentity, "registry:npm:second@2.0.0", "sha512-second")
	first, _ := domain.NewLockedDependency(firstID, domain.DependencyPrimary, firstResolved)
	second, _ := domain.NewLockedDependency(secondID, domain.DependencyTransitive, secondResolved)
	edge, _ := domain.NewDependencyEdge(firstID, secondID)
	graph, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{first, second}, []domain.DependencyEdge{edge})
	if err != nil {
		t.Fatal(err)
	}
	return graph, first, second
}

func dependencyInspection(t *testing.T, dependency domain.LockedDependency, digestCharacter string) domain.DependencyInspection {
	t.Helper()
	runID, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := domain.NewSHA256Digest(strings.Repeat(digestCharacter, 64))
	artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(dependency.Artifact().Identity(), digest, "intake:"+runID.String()+":tarball", 1, dependency.Artifact().DeclaredIntegrity())
	if err != nil {
		t.Fatal(err)
	}
	checkID, _ := domain.NewCheckID("fixture-check")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidenceID, _ := domain.NewEvidenceID("fixture-evidence-" + dependency.Node().String())
	reference, _ := domain.NewEvidenceReference(evidenceID, "fixture:"+evidenceID.String())
	decision, err := domain.NewPolicyDecision(domain.DecisionAllow, "fixture-policy", 1, []string{"FIXTURE_ALLOW"})
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := domain.NewDependencyInspection(dependency.Node(), runID, artifact, []domain.CheckExecution{check}, []domain.EvidenceReference{reference}, decision)
	if err != nil {
		t.Fatal(err)
	}
	return inspection
}
