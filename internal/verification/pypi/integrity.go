// Package pypi verifies the public PyPI SHA-256 selected by the isolated resolver.
package pypi

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type IntegrityVerifier struct{}

func (IntegrityVerifier) Verify(ctx context.Context, artifact domain.AcquiredArtifact) (domain.VerificationReport, error) {
	if ctx == nil || ctx.Err() != nil || artifact.Identity().Source().String() != "pypi" {
		return domain.VerificationReport{}, errors.New("PyPI integrity verification request is invalid")
	}
	checkID, _ := domain.NewCheckID("pypi-declared-sha256")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	declared, ok := artifact.DeclaredIntegrity()
	valid := ok && strings.HasPrefix(declared, "sha256:") && len(strings.TrimPrefix(declared, "sha256:")) == 64
	match := valid && subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(declared, "sha256:")), []byte(artifact.Digest().String())) == 1
	evidenceID, _ := domain.NewEvidenceID("pypi-declared-sha256-result")
	evidence, _ := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), "pypi-declared-sha256", "PyPI selected SHA-256 was checked against controlled intake.")
	if !match {
		finding, _ := domain.NewFinding("M5_DECLARED_SHA256_MISMATCH", []domain.EvidenceID{evidenceID})
		return domain.NewVerificationReportWithFindings(check, domain.VerificationMismatch, []domain.Finding{finding}, []domain.Evidence{evidence})
	}
	return domain.NewVerificationReport(check, domain.VerificationVerified, []domain.Evidence{evidence})
}
