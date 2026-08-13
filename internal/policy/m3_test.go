package policy

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestM3RequiresCompletedDynamicCheckAndInterpretsDynamicFindings(t *testing.T) {
	tests := []struct {
		name    string
		dynamic domain.CheckExecution
		finding string
		want    domain.Decision
	}{
		{"clean", check(t, "npm-dynamic-lifecycle", domain.ExecutionCompleted, ""), "", domain.DecisionAllow},
		{"unavailable", check(t, "npm-dynamic-lifecycle", domain.ExecutionIncomplete, "M3_DYNAMIC_OBSERVER_FAILED"), "", domain.DecisionManualReview},
		{"network", check(t, "npm-dynamic-lifecycle", domain.ExecutionCompleted, ""), "M3_NETWORK_ATTEMPT", domain.DecisionManualReview},
		{"honeytoken", check(t, "npm-dynamic-lifecycle", domain.ExecutionCompleted, ""), "M3_HONEYTOKEN_ACCESS", domain.DecisionBlock},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := policyInput(t, test.dynamic, test.finding)
			decision, err := (M3{}).Evaluate(input)
			if err != nil || decision.Decision() != test.want {
				t.Fatalf("Evaluate() = (%q, %v), want %q", decision.Decision(), err, test.want)
			}
		})
	}
}

func policyInput(t *testing.T, dynamic domain.CheckExecution, code string) domain.PolicyInput {
	t.Helper()
	source, _ := domain.NewSourceID("npm")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "tiny", "1.2.3", "tarball")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	artifact, _ := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:tarball", 1)
	verificationID, _ := domain.NewCheckID("npm-integrity")
	verificationCheck, _ := domain.NewCheckExecution(verificationID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	verificationEvidence := evidence(t, "integrity-evidence", verificationCheck, artifact)
	verification, err := domain.NewVerificationReport(verificationCheck, domain.VerificationVerified, []domain.Evidence{verificationEvidence})
	if err != nil {
		t.Fatal(err)
	}
	staticCheck := check(t, "npm-static-archive", domain.ExecutionCompleted, "")
	staticEvidence := evidence(t, "static-evidence", staticCheck, artifact)
	dynamicEvidence := evidence(t, "dynamic-evidence", dynamic, artifact)
	findings := []domain.Finding{}
	if code != "" {
		finding, _ := domain.NewFinding(code, []domain.EvidenceID{dynamicEvidence.ID()})
		findings = []domain.Finding{finding}
	}
	dynamicReport, err := domain.NewInspectionReport(dynamic, findings, func() []domain.Evidence {
		if dynamic.Status() == domain.ExecutionCompleted {
			return []domain.Evidence{dynamicEvidence}
		}
		return nil
	}())
	if err != nil {
		t.Fatal(err)
	}
	staticReport, _ := domain.NewInspectionReport(staticCheck, nil, []domain.Evidence{staticEvidence})
	inspection, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{staticReport, dynamicReport})
	if err != nil {
		t.Fatal(err)
	}
	references := []domain.EvidenceReference{}
	for _, item := range append(verification.Evidence(), inspection.Evidence()...) {
		reference, _ := domain.NewEvidenceReference(item.ID(), "fixture:"+item.ID().String())
		references = append(references, reference)
	}
	runID, _ := domain.ParseRunID("run_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	input, err := domain.NewPolicyInput(runID, artifact, verification, inspection, references)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func check(t *testing.T, value string, status domain.ExecutionStatus, limitation string) domain.CheckExecution {
	t.Helper()
	id, _ := domain.NewCheckID(value)
	execution, err := domain.NewCheckExecution(id, domain.CheckInspection, true, domain.CapabilitySupported, status, limitation)
	if err != nil {
		t.Fatal(err)
	}
	return execution
}
func evidence(t *testing.T, value string, check domain.CheckExecution, artifact domain.AcquiredArtifact) domain.Evidence {
	t.Helper()
	id, _ := domain.NewEvidenceID(value)
	item, err := domain.NewEvidence(id, check.ID(), artifact.Identity(), artifact.Digest(), "fixture", "completed")
	if err != nil {
		t.Fatal(err)
	}
	return item
}
