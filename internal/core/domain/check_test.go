package domain_test

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestCheckExecutionKeepsStatusAxesSeparate(t *testing.T) {
	t.Parallel()

	checkID := mustCheckID(t, "fixture-static")
	tests := []struct {
		name       string
		capability domain.Capability
		status     domain.ExecutionStatus
		limitation string
		valid      bool
	}{
		{name: "completed", capability: domain.CapabilitySupported, status: domain.ExecutionCompleted, valid: true},
		{name: "unsupported", capability: domain.CapabilityUnsupported, status: domain.ExecutionNotExecuted, limitation: "CAPABILITY_UNSUPPORTED", valid: true},
		{name: "unavailable", capability: domain.CapabilitySupported, status: domain.ExecutionUnavailable, limitation: "TOOL_UNAVAILABLE", valid: true},
		{name: "unsupported completed", capability: domain.CapabilityUnsupported, status: domain.ExecutionCompleted},
		{name: "missing limitation", capability: domain.CapabilitySupported, status: domain.ExecutionFailed},
		{name: "completed limitation", capability: domain.CapabilitySupported, status: domain.ExecutionCompleted, limitation: "UNEXPECTED"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewCheckExecution(checkID, domain.CheckInspection, true, test.capability, test.status, test.limitation)
			if (err == nil) != test.valid {
				t.Fatalf("NewCheckExecution() error = %v, valid = %v", err, test.valid)
			}
		})
	}
}

func TestReportsRequireInterpretableBoundEvidence(t *testing.T) {
	t.Parallel()

	identity, digest := evidenceSubject(t)
	verificationID := mustCheckID(t, "fixture-integrity")
	verificationExecution, err := domain.NewCheckExecution(verificationID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	if err != nil {
		t.Fatal(err)
	}
	verificationEvidence := mustEvidence(t, "verification-evidence", verificationID, identity, digest)
	if _, err := domain.NewVerificationReport(verificationExecution, domain.VerificationVerified, []domain.Evidence{verificationEvidence}); err != nil {
		t.Fatalf("NewVerificationReport() error = %v", err)
	}
	if _, err := domain.NewVerificationReport(verificationExecution, "", []domain.Evidence{verificationEvidence}); err == nil {
		t.Fatal("completed verification without outcome accepted")
	}
	if _, err := domain.NewVerificationReport(verificationExecution, domain.VerificationVerified, nil); err == nil {
		t.Fatal("completed verification without Evidence accepted")
	}

	inspectionID := mustCheckID(t, "fixture-static")
	inspectionExecution, err := domain.NewCheckExecution(inspectionID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	if err != nil {
		t.Fatal(err)
	}
	inspectionEvidence := mustEvidence(t, "inspection-evidence", inspectionID, identity, digest)
	finding, err := domain.NewFinding("M1_BLOCK_FINDING", []domain.EvidenceID{inspectionEvidence.ID()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := domain.NewInspectionReport(inspectionExecution, []domain.Finding{finding}, []domain.Evidence{inspectionEvidence}); err != nil {
		t.Fatalf("NewInspectionReport() error = %v", err)
	}
	foreignEvidence := mustEvidence(t, "foreign-evidence", inspectionID, identity, digest)
	foreignFinding, err := domain.NewFinding("M1_BLOCK_FINDING", []domain.EvidenceID{foreignEvidence.ID()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := domain.NewInspectionReport(inspectionExecution, []domain.Finding{foreignFinding}, []domain.Evidence{inspectionEvidence}); err == nil {
		t.Fatal("Finding with foreign Evidence accepted")
	}
}

func TestEvidenceRejectsObviousSensitiveOrUnboundedSummary(t *testing.T) {
	t.Parallel()

	identity, digest := evidenceSubject(t)
	checkID := mustCheckID(t, "fixture-static")
	id, err := domain.NewEvidenceID("sensitive-evidence")
	if err != nil {
		t.Fatal(err)
	}
	for _, summary := range []string{
		"token=secret-value",
		"read /Users/alice/project/file",
		strings.Repeat("x", 1025),
		"line one\nline two",
	} {
		if _, err := domain.NewEvidence(id, checkID, identity, digest, "static-inspection", summary); err == nil {
			t.Fatalf("NewEvidence(%q) error = nil", summary)
		}
	}
}

func mustCheckID(t *testing.T, value string) domain.CheckID {
	t.Helper()
	id, err := domain.NewCheckID(value)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func evidenceSubject(t *testing.T) (domain.ResolvedArtifactIdentity, domain.ContentDigest) {
	t.Helper()
	source := mustSource(t, "fixture")
	identity := mustIdentity(t, source, "safe", "1.0.0", "default")
	digest, err := domain.NewSHA256Digest(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	return identity, digest
}

func mustEvidence(t *testing.T, idValue string, checkID domain.CheckID, identity domain.ResolvedArtifactIdentity, digest domain.ContentDigest) domain.Evidence {
	t.Helper()
	id, err := domain.NewEvidenceID(idValue)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := domain.NewEvidence(id, checkID, identity, digest, "normalized-fact", "Synthetic normalized evidence for the exact fixture subject.")
	if err != nil {
		t.Fatal(err)
	}
	return evidence
}
