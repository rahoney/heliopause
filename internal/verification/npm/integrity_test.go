package npm

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestIntegrityVerifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		declared    string
		observed    string
		outcome     domain.VerificationOutcome
		findingCode string
	}{
		{name: "matching SHA-512", declared: sri("a"), observed: sri("a"), outcome: domain.VerificationVerified},
		{name: "mismatch", declared: sri("a"), observed: sri("b"), outcome: domain.VerificationMismatch, findingCode: "M2_DECLARED_INTEGRITY_MISMATCH"},
		{name: "missing declaration", observed: sri("a"), outcome: domain.VerificationMismatch, findingCode: "M2_DECLARED_INTEGRITY_INVALID"},
		{name: "non SHA-512 declaration", declared: "sha256-YQ==", observed: sri("a"), outcome: domain.VerificationMismatch, findingCode: "M2_DECLARED_INTEGRITY_INVALID"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			artifact := testArtifact(t, test.declared, test.observed)
			report, err := (IntegrityVerifier{}).Verify(context.Background(), artifact)
			if err != nil {
				t.Fatalf("Verify() error = %v", err)
			}
			if report.Execution().ID().String() != checkIDValue || !report.Execution().Required() || report.Execution().Status() != domain.ExecutionCompleted || report.Outcome() != test.outcome {
				t.Fatalf("report = %#v", report)
			}
			if findings := report.Findings(); test.findingCode == "" && len(findings) != 0 {
				t.Fatalf("Findings() = %#v", findings)
			} else if test.findingCode != "" && (len(findings) != 1 || findings[0].Code() != test.findingCode) {
				t.Fatalf("Findings() = %#v", findings)
			}
			for _, evidence := range report.Evidence() {
				if (test.declared != "" && strings.Contains(evidence.Summary(), test.declared)) || (test.observed != "" && strings.Contains(evidence.Summary(), test.observed)) {
					t.Fatalf("Evidence leaked raw integrity value: %q", evidence.Summary())
				}
			}
		})
	}
}

func TestIntegrityVerifierPreservesCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (IntegrityVerifier{}).Verify(ctx, testArtifact(t, sri("a"), sri("a")))
	if err != context.Canceled {
		t.Fatalf("Verify() error = %v", err)
	}
}

func testArtifact(t *testing.T, declared, observed string) domain.AcquiredArtifact {
	t.Helper()
	source, err := domain.NewSourceID("npm")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := domain.NewResolvedArtifactIdentity(source, "tiny", "1.2.3", "tarball")
	if err != nil {
		t.Fatal(err)
	}
	digest, err := domain.NewSHA256Digest(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := domain.NewAcquiredArtifactWithIntegrity(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:tarball", 1, declared, observed)
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func sri(character string) string {
	return "sha512-" + base64.StdEncoding.EncodeToString([]byte(strings.Repeat(character, 64)))
}
