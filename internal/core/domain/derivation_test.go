package domain_test

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestExtendLockedDependencyGraphRetainsDerivedRecipeBinding(t *testing.T) {
	t.Parallel()
	source, _ := domain.NewSourceID("pypi")
	sourceIdentity, _ := domain.NewResolvedArtifactIdentity(source, "example", "1.0", "sdist")
	sourceResolved, _ := domain.NewResolvedArtifact(sourceIdentity, "https://files.pythonhosted.org/example.tar.gz", "sha256:"+strings.Repeat("a", 64))
	sourceNode, _ := domain.NewDependencyNodeID("example")
	sourceDependency, _ := domain.NewLockedDependency(sourceNode, domain.DependencyPrimary, sourceResolved)
	graph, _ := domain.NewLockedDependencyGraph([]domain.LockedDependency{sourceDependency}, nil)

	derivedIdentity, _ := domain.NewResolvedArtifactIdentity(source, "example", "1.0", "derived-wheel")
	derivedDigest, _ := domain.NewSHA256Digest(strings.Repeat("b", 64))
	derivedResolved, _ := domain.NewResolvedArtifact(derivedIdentity, "derived:example", "sha256:"+derivedDigest.String())
	derivedNode, _ := domain.NewDependencyNodeID("example-derived")
	derivedDependency, _ := domain.NewLockedDependency(derivedNode, domain.DependencyTransitive, derivedResolved)
	artifact, _ := domain.NewAcquiredArtifactWithDeclaredIntegrity(derivedIdentity, derivedDigest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:derived-wheel", 1, "sha256:"+derivedDigest.String())
	sourceDigest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	inputDigest, _ := domain.NewSHA256Digest(strings.Repeat("c", 64))
	configDigest, _ := domain.NewSHA256Digest(strings.Repeat("d", 64))
	binding, _ := domain.NewDerivationBinding(sourceDigest, []domain.ContentDigest{inputDigest}, "pep517-gvisor", configDigest)
	checkID, _ := domain.NewCheckID("pypi-pep517-build")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidenceID, _ := domain.NewEvidenceID("pypi-pep517-build-result")
	evidence, _ := domain.NewEvidence(evidenceID, checkID, derivedIdentity, derivedDigest, "pypi-pep517-build", "Trusted build completed.")
	derived, err := domain.NewDerivedDependency(sourceNode, derivedDependency, artifact, binding, check, evidence)
	if err != nil {
		t.Fatal(err)
	}
	extended, err := domain.ExtendLockedDependencyGraph(graph, []domain.DerivedDependency{derived})
	if err != nil {
		t.Fatal(err)
	}
	if len(extended.Derivations()) != 1 || extended.Derivations()[0].Binding().ConfigDigest() != configDigest {
		t.Fatalf("derived binding = %#v", extended.Derivations())
	}
}
