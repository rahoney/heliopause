package domain_test

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestVerifiedSetRequiresCompleteAllowDecision(t *testing.T) {
	t.Parallel()
	graph, first, second := inspectedGraph(t)
	inspected, err := domain.NewInspectedDependencySet(graph, []domain.DependencyInspection{dependencyInspection(t, first, "a"), dependencyInspection(t, second, "b")})
	if err != nil {
		t.Fatal(err)
	}
	block, _ := domain.NewPolicyDecision(domain.DecisionBlock, "m4-set-policy", 1, []string{"BLOCK"})
	if _, err := domain.NewVerifiedSet(inspected, block); err == nil {
		t.Fatal("NewVerifiedSet() accepted set-level BLOCK")
	}
	allow, _ := domain.NewPolicyDecision(domain.DecisionAllow, "m4-set-policy", 1, []string{"ALLOW"})
	set, err := domain.NewVerifiedSet(inspected, allow)
	if err != nil || !set.Valid() {
		t.Fatalf("NewVerifiedSet() = %#v, %v", set, err)
	}
}

func TestVerifiedBundleCopiesDocuments(t *testing.T) {
	t.Parallel()
	graph, first, second := inspectedGraph(t)
	inspected, _ := domain.NewInspectedDependencySet(graph, []domain.DependencyInspection{dependencyInspection(t, first, "a"), dependencyInspection(t, second, "b")})
	allow, _ := domain.NewPolicyDecision(domain.DecisionAllow, "m4-set-policy", 1, []string{"ALLOW"})
	set, _ := domain.NewVerifiedSet(inspected, allow)
	digest, _ := domain.NewSHA256Digest(strings.Repeat("c", 64))
	manifest, sbom := []byte(`{"manifest":true}`), []byte(`{"sbom":true}`)
	bundle, err := domain.NewVerifiedBundle(digest, set, manifest, sbom)
	if err != nil {
		t.Fatal(err)
	}
	manifest[0], sbom[0] = 'x', 'x'
	if bundle.ManifestDocument()[0] == 'x' || bundle.SBOMDocument()[0] == 'x' {
		t.Fatal("NewVerifiedBundle() retained caller-owned document bytes")
	}
}
