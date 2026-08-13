// Package npm verifies integrity values supplied by the public npm registry.
package npm

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	checkIDValue    = "npm-declared-integrity"
	evidenceIDValue = "npm-declared-integrity-result"
)

// IntegrityVerifier compares public npm's declared SHA-512 SRI to the acquired byte stream.
type IntegrityVerifier struct{}

// Verify reports invalid declarations and mismatches as completed, required verification results.
func (IntegrityVerifier) Verify(ctx context.Context, artifact domain.AcquiredArtifact) (domain.VerificationReport, error) {
	if ctx == nil {
		return domain.VerificationReport{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return domain.VerificationReport{}, err
	}
	if artifact.Identity().Source().String() != "npm" || artifact.Digest().String() == "" {
		return domain.VerificationReport{}, errors.New("npm integrity verification requires an acquired npm Artifact")
	}

	checkID, err := domain.NewCheckID(checkIDValue)
	if err != nil {
		return domain.VerificationReport{}, err
	}
	execution, err := domain.NewCheckExecution(checkID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	if err != nil {
		return domain.VerificationReport{}, err
	}

	declared, declaredOK := artifact.DeclaredIntegrity()
	observed, observedOK := artifact.ObservedIntegrity()
	declaredDigest, validDeclared := decodeSHA512SRI(declared)
	observedDigest, validObserved := decodeSHA512SRI(observed)
	if !declaredOK || !observedOK || !validDeclared || !validObserved {
		return mismatchReport(execution, artifact, "M2_DECLARED_INTEGRITY_INVALID", "npm declared SHA-512 integrity is missing or invalid.")
	}
	if subtle.ConstantTimeCompare(declaredDigest, observedDigest) != 1 {
		return mismatchReport(execution, artifact, "M2_DECLARED_INTEGRITY_MISMATCH", "npm declared SHA-512 integrity did not match acquired content.")
	}
	evidence, err := newEvidence(checkID, artifact, "npm declared SHA-512 integrity matched acquired content.")
	if err != nil {
		return domain.VerificationReport{}, err
	}
	return domain.NewVerificationReport(execution, domain.VerificationVerified, []domain.Evidence{evidence})
}

func mismatchReport(execution domain.CheckExecution, artifact domain.AcquiredArtifact, code, summary string) (domain.VerificationReport, error) {
	evidence, err := newEvidence(execution.ID(), artifact, summary)
	if err != nil {
		return domain.VerificationReport{}, err
	}
	finding, err := domain.NewFinding(code, []domain.EvidenceID{evidence.ID()})
	if err != nil {
		return domain.VerificationReport{}, err
	}
	return domain.NewVerificationReportWithFindings(execution, domain.VerificationMismatch, []domain.Finding{finding}, []domain.Evidence{evidence})
}

func newEvidence(checkID domain.CheckID, artifact domain.AcquiredArtifact, summary string) (domain.Evidence, error) {
	evidenceID, err := domain.NewEvidenceID(evidenceIDValue)
	if err != nil {
		return domain.Evidence{}, err
	}
	return domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), "npm-declared-integrity", summary)
}

func decodeSHA512SRI(value string) ([]byte, bool) {
	algorithm, encoded, found := strings.Cut(value, "-")
	if !found || algorithm != "sha512" || encoded == "" || strings.Contains(encoded, "-") {
		return nil, false
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(decoded) != 64 {
		return nil, false
	}
	return decoded, true
}
