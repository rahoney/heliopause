package policy

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestProjectSetPolicyRequiresArchiveInstallActionFact(t *testing.T) {
	set := projectSetFromGraph(t, m4Set(t, domain.DecisionAllow, domain.DecisionAllow, false))
	decision, err := (M4{}).EvaluateProjectSet(set)
	if err != nil || decision.Decision() != domain.DecisionManualReview || decision.Reasons()[0] != "M4_HOST_LIFECYCLE_UNCONFIRMED" {
		t.Fatalf("unknown archive install action must require review: decision=%v err=%v", decision, err)
	}
}

func TestProjectSetPolicyPreservesHostActionAndBlockPrecedence(t *testing.T) {
	for _, test := range []struct {
		name, first, second, check, cause string
		block                             bool
		want                              domain.Decision
	}{
		{"all-known", domain.EvidenceHostInstallActionNotRequired, domain.EvidenceHostInstallActionNotRequired, "npm-static-archive", "M4_VERIFIED_SET_COMPLETED", false, domain.DecisionAllow},
		{"required", domain.EvidenceHostInstallActionRequired, domain.EvidenceHostInstallActionNotRequired, "npm-static-archive", "M4_HOST_LIFECYCLE_UNSUPPORTED", false, domain.DecisionManualReview},
		{"missing-second", domain.EvidenceHostInstallActionNotRequired, "", "npm-static-archive", "M4_HOST_LIFECYCLE_UNCONFIRMED", false, domain.DecisionManualReview},
		{"unknown-before-required", "", domain.EvidenceHostInstallActionRequired, "npm-static-archive", "M4_HOST_LIFECYCLE_UNSUPPORTED", false, domain.DecisionManualReview},
		{"wrong-check", domain.EvidenceHostInstallActionNotRequired, domain.EvidenceHostInstallActionNotRequired, "unrelated-inspection", "M4_HOST_LIFECYCLE_UNCONFIRMED", false, domain.DecisionManualReview},
		{"block-before-required", domain.EvidenceHostInstallActionRequired, "", "npm-static-archive", "M4_DEPENDENCY_BLOCKED", true, domain.DecisionBlock},
	} {
		t.Run(test.name, func(t *testing.T) {
			secondDecision := domain.DecisionAllow
			if test.block {
				secondDecision = domain.DecisionBlock
			}
			set := projectSetFromGraph(t, m4Set(t, domain.DecisionAllow, secondDecision, false))
			entries := set.Inspections()
			for index, kind := range []string{test.first, test.second} {
				if kind != "" {
					entries[index] = projectEntryWithArchiveAction(t, entries[index], kind, test.check)
				}
			}
			set, err := domain.NewInspectedProjectSet(set.Snapshot(), entries)
			if err != nil {
				t.Fatal(err)
			}
			decision, err := (M4{}).EvaluateProjectSet(set)
			if err != nil || decision.Decision() != test.want || decision.Reasons()[0] != test.cause {
				t.Fatalf("decision=%v err=%v", decision, err)
			}
		})
	}
}

func projectEntryWithArchiveAction(t *testing.T, entry domain.DependencyInspection, kind, checkName string) domain.DependencyInspection {
	t.Helper()
	id, err := domain.NewEvidenceID("npm-host-install-action")
	if err != nil {
		t.Fatal(err)
	}
	checkID, err := domain.NewCheckID(checkName)
	if err != nil {
		t.Fatal(err)
	}
	check, err := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	if err != nil {
		t.Fatal(err)
	}
	fact, err := domain.NewEvidence(id, checkID, entry.Artifact().Identity(), entry.Artifact().Digest(), kind, "display text does not decide Host execution")
	if err != nil {
		t.Fatal(err)
	}
	reference, err := domain.NewEvidenceReference(id, "fixture:"+id.String())
	if err != nil {
		t.Fatal(err)
	}
	entry, err = domain.NewDependencyInspection(entry.Node(), entry.RunID(), entry.Artifact(), append(entry.Checks(), check), append(entry.Evidence(), reference), entry.PolicyDecision())
	if err != nil {
		t.Fatal(err)
	}
	entry, err = entry.WithHostInstallActionEvidence([]domain.Evidence{fact})
	if err != nil {
		t.Fatal(err)
	}
	return entry
}

func projectSetFromGraph(t *testing.T, old domain.InspectedDependencySet) domain.InspectedProjectSet {
	t.Helper()
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
	for _, inspection := range old.Inspections() {
		node, err := domain.ProjectDependencyNodeID(inspection.Artifact().Identity())
		if err != nil {
			t.Fatal(err)
		}
		checkID, err := domain.NewCheckID("fixture-verification")
		if err != nil {
			t.Fatal(err)
		}
		verification, err := domain.NewCheckExecution(checkID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		if err != nil {
			t.Fatal(err)
		}
		checks := append(inspection.Checks(), verification)
		entry, err := domain.NewDependencyInspection(node, inspection.RunID(), inspection.Artifact(), checks, inspection.Evidence(), inspection.PolicyDecision())
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, entry)
	}
	set, err := domain.NewInspectedProjectSet(snapshot, entries)
	if err != nil {
		t.Fatal(err)
	}
	return set
}
