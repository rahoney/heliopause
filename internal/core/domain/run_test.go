package domain_test

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestInspectionRunCompletedLifecycle(t *testing.T) {
	t.Parallel()

	run, artifact := newRunFixture(t)
	if run.Lifecycle() != domain.RunCreated {
		t.Fatalf("lifecycle = %q", run.Lifecycle())
	}
	if err := run.Activate(); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	if err := run.BindAcquiredArtifact(artifact); err != nil {
		t.Fatalf("BindAcquiredArtifact() error = %v", err)
	}
	policyDecision := mustPolicyDecision(t, domain.DecisionAllow, "M1_REQUIRED_CHECKS_COMPLETED")
	if err := run.FinalizeCompleted(policyDecision); err != nil {
		t.Fatalf("FinalizeCompleted() error = %v", err)
	}

	if outcome, ok := run.Outcome(); !ok || outcome != domain.RunCompleted {
		t.Fatalf("Outcome() = %q, %v", outcome, ok)
	}
	if decision, ok := run.PolicyDecision(); !ok || decision.Decision() != domain.DecisionAllow || decision.PolicyID() != "m1-fake-inspect" || decision.Version() != 1 {
		t.Fatalf("PolicyDecision() = %#v, %v", decision, ok)
	}
	if _, ok := run.FailureCode(); ok {
		t.Fatal("FailureCode() unexpectedly present")
	}
	if err := run.Activate(); err == nil {
		t.Fatal("finalized Run activated again")
	}
}

func TestInspectionRunFailedLifecycleHasNoPolicy(t *testing.T) {
	t.Parallel()

	run, _ := newRunFixture(t)
	if err := run.Activate(); err != nil {
		t.Fatal(err)
	}
	if err := run.FinalizeFailed("ARTIFACT_ACQUIRE_FAILED"); err != nil {
		t.Fatalf("FinalizeFailed() error = %v", err)
	}
	if outcome, ok := run.Outcome(); !ok || outcome != domain.RunFailed {
		t.Fatalf("Outcome() = %q, %v", outcome, ok)
	}
	if _, ok := run.PolicyDecision(); ok {
		t.Fatal("PolicyDecision() unexpectedly present")
	}
	if code, ok := run.FailureCode(); !ok || code != "ARTIFACT_ACQUIRE_FAILED" {
		t.Fatalf("FailureCode() = %q, %v", code, ok)
	}
}

func TestInspectionRunRejectsInvalidTransitionsAndSubjects(t *testing.T) {
	t.Parallel()

	run, artifact := newRunFixture(t)
	if err := run.BindAcquiredArtifact(artifact); err == nil {
		t.Fatal("artifact bound before activation")
	}
	if err := run.FinalizeCompleted(mustPolicyDecision(t, domain.DecisionAllow, "M1_REQUIRED_CHECKS_COMPLETED")); err == nil {
		t.Fatal("CREATED Run finalized")
	}
	if err := run.Activate(); err != nil {
		t.Fatal(err)
	}
	if err := run.FinalizeCompleted(mustPolicyDecision(t, domain.DecisionAllow, "M1_REQUIRED_CHECKS_COMPLETED")); err == nil {
		t.Fatal("Run without acquired artifact completed")
	}

	otherSource := mustSource(t, "other")
	otherIdentity := mustIdentity(t, otherSource, "safe", "1.0.0", "default")
	digest, err := domain.NewSHA256Digest(strings.Repeat("b", 64))
	if err != nil {
		t.Fatal(err)
	}
	otherArtifact, err := domain.NewAcquiredArtifact(otherIdentity, digest, "fixture-content:other", 42)
	if err != nil {
		t.Fatal(err)
	}
	if err := run.BindAcquiredArtifact(otherArtifact); err == nil {
		t.Fatal("mismatched acquired identity bound")
	}
	if err := run.BindAcquiredArtifact(artifact); err != nil {
		t.Fatal(err)
	}
	if err := run.BindAcquiredArtifact(artifact); err == nil {
		t.Fatal("second acquired artifact bound")
	}
	if err := run.FinalizeCompleted(domain.PolicyDecision{}); err == nil {
		t.Fatal("zero Policy Decision accepted")
	}
	if err := run.FinalizeFailed("lowercase"); err == nil {
		t.Fatal("invalid operational failure code accepted")
	}
}

func TestPolicyDecisionOwnsOrderedReasons(t *testing.T) {
	t.Parallel()

	reasons := []string{"M1_BLOCK_FINDING", "SECOND_REASON"}
	decision, err := domain.NewPolicyDecision(domain.DecisionBlock, "m1-fake-inspect", 1, reasons)
	if err != nil {
		t.Fatalf("NewPolicyDecision() error = %v", err)
	}
	reasons[0] = "MUTATED"
	returned := decision.Reasons()
	returned[1] = "MUTATED"
	if got := decision.Reasons(); got[0] != "M1_BLOCK_FINDING" || got[1] != "SECOND_REASON" {
		t.Fatalf("Reasons() = %v", got)
	}

	invalid := []struct {
		name     string
		decision domain.Decision
		policyID string
		version  uint64
		reasons  []string
	}{
		{name: "decision", decision: "UNKNOWN", policyID: "m1-fake-inspect", version: 1, reasons: []string{"REASON"}},
		{name: "policy ID", decision: domain.DecisionAllow, policyID: "M1", version: 1, reasons: []string{"REASON"}},
		{name: "version", decision: domain.DecisionAllow, policyID: "m1", version: 0, reasons: []string{"REASON"}},
		{name: "reasons", decision: domain.DecisionAllow, policyID: "m1", version: 1},
		{name: "reason format", decision: domain.DecisionAllow, policyID: "m1", version: 1, reasons: []string{"lowercase"}},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := domain.NewPolicyDecision(test.decision, test.policyID, test.version, test.reasons); err == nil {
				t.Fatal("NewPolicyDecision() error = nil")
			}
		})
	}
}

func mustPolicyDecision(t *testing.T, decision domain.Decision, reason string) domain.PolicyDecision {
	t.Helper()
	result, err := domain.NewPolicyDecision(decision, "m1-fake-inspect", 1, []string{reason})
	if err != nil {
		t.Fatalf("NewPolicyDecision() error = %v", err)
	}
	return result
}

func newRunFixture(t *testing.T) (*domain.InspectionRun, domain.AcquiredArtifact) {
	t.Helper()
	operationID, err := domain.ParseOperationID("op_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	runID, err := domain.ParseRunID("run_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	source := mustSource(t, "fixture")
	reference, err := domain.NewArtifactReference(source, "safe@latest")
	if err != nil {
		t.Fatal(err)
	}
	identity := mustIdentity(t, source, "safe", "1.0.0", "default")
	digest, err := domain.NewSHA256Digest(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "fixture-content:safe", 42)
	if err != nil {
		t.Fatal(err)
	}
	run, err := domain.NewInspectionRun(runID, operationID, reference, identity)
	if err != nil {
		t.Fatal(err)
	}
	return run, artifact
}
