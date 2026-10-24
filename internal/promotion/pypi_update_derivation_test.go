package promotion

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestPyPIUpgradePreservesResolverGraphAndAuthenticatedDerivedWheel(t *testing.T) {
	for _, mode := range []string{"valid", "source-digest", "derived-identity", "source-locator"} {
		t.Run(mode, func(t *testing.T) {
			source, _ := domain.NewSourceID("pypi")
			sourceIdentity, _ := domain.NewResolvedArtifactIdentity(source, "example", "1.0", "sdist")
			sourceDigest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
			derivedDigest, _ := domain.NewSHA256Digest(strings.Repeat("b", 64))
			inputDigest, _ := domain.NewSHA256Digest(strings.Repeat("c", 64))
			configDigest, _ := domain.NewSHA256Digest(strings.Repeat("d", 64))
			resolved, _ := domain.NewResolvedArtifact(sourceIdentity, "https://files.pythonhosted.org/example.tar.gz", "sha256:"+sourceDigest.String())
			nodeID, _ := domain.NewDependencyNodeID("example")
			node, _ := domain.NewLockedDependency(nodeID, domain.DependencyPrimary, resolved)
			original, _ := domain.NewLockedDependencyGraph([]domain.LockedDependency{node}, nil)
			selected := original
			if mode == "source-locator" {
				changed, _ := domain.NewResolvedArtifact(sourceIdentity, "https://files.pythonhosted.org/substituted.tar.gz", "sha256:"+sourceDigest.String())
				node, _ = domain.NewLockedDependency(nodeID, domain.DependencyPrimary, changed)
				selected, _ = domain.NewLockedDependencyGraph([]domain.LockedDependency{node}, nil)
			}
			name := "example"
			if mode == "derived-identity" {
				name = "replacement"
			}
			derivedIdentity, _ := domain.NewResolvedArtifactIdentity(source, name, "1.0", "derived-wheel")
			derivedResolved, _ := domain.NewResolvedArtifact(derivedIdentity, "derived:example", "sha256:"+derivedDigest.String())
			derivedID, _ := domain.NewDependencyNodeID("example-derived")
			derivedNode, _ := domain.NewLockedDependency(derivedID, domain.DependencyTransitive, derivedResolved)
			sourceArtifact, _ := domain.NewAcquiredArtifactWithDeclaredIntegrity(sourceIdentity, sourceDigest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:sdist", 1, "sha256:"+sourceDigest.String())
			derivedArtifact, _ := domain.NewAcquiredArtifactWithDeclaredIntegrity(derivedIdentity, derivedDigest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:derived-wheel", 1, "sha256:"+derivedDigest.String())
			boundSource := sourceDigest
			if mode == "source-digest" {
				boundSource = inputDigest
			}
			// The build input is deliberately not a resolver graph node.
			binding, _ := domain.NewDerivationBinding(boundSource, []domain.ContentDigest{inputDigest}, "pep517-gvisor", configDigest)
			checkID, _ := domain.NewCheckID("pypi-pep517-build")
			check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
			evidenceID, _ := domain.NewEvidenceID("pypi-pep517-build-result")
			evidence, _ := domain.NewEvidence(evidenceID, checkID, derivedIdentity, derivedDigest, "pypi-pep517-build", "Trusted build completed.")
			derived, err := domain.NewDerivedDependency(nodeID, derivedNode, derivedArtifact, binding, check, evidence)
			if err != nil {
				t.Fatal(err)
			}
			graph, err := domain.ExtendLockedDependencyGraph(selected, []domain.DerivedDependency{derived})
			if err != nil {
				t.Fatal(err)
			}
			runID, _ := domain.NewRunID()
			reference, _ := domain.NewEvidenceReference(evidenceID, "evidence:"+runID.String()+":"+evidenceID.String())
			decision, _ := domain.NewPolicyDecision(domain.DecisionAllow, "fixture-policy", 1, []string{"ALLOW"})
			first, err := domain.NewDependencyInspection(nodeID, runID, sourceArtifact, []domain.CheckExecution{check}, []domain.EvidenceReference{reference}, decision)
			if err != nil {
				t.Fatal(err)
			}
			second, err := domain.NewDependencyInspection(derivedID, runID, derivedArtifact, []domain.CheckExecution{check}, []domain.EvidenceReference{reference}, decision)
			if err != nil {
				t.Fatal(err)
			}
			inspected, err := domain.NewInspectedDependencySet(graph, []domain.DependencyInspection{first, second})
			if err != nil {
				t.Fatal(err)
			}
			if pypiUpgradeGraphExtends(original, inspected) != (mode == "valid") {
				t.Fatal("derived binding or original resolver selection changed", mode)
			}
		})
	}
}
