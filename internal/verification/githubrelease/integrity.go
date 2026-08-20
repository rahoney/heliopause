// Package githubrelease verifies the SHA-256 declared by the exact GitHub
// Releases API response against bytes acquired by the controlled adapter.
package githubrelease

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type IntegrityVerifier struct{}

func (IntegrityVerifier) Verify(ctx context.Context, artifact domain.AcquiredArtifact) (domain.VerificationReport, error) {
	if ctx == nil || ctx.Err() != nil || artifact.Identity().Source().String() != "github-release" {
		return domain.VerificationReport{}, errors.New("GitHub Release integrity verification request is invalid")
	}
	checkID, _ := domain.NewCheckID("github-release-declared-sha256")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	declared, ok := artifact.DeclaredIntegrity()
	value, valid := strings.CutPrefix(declared, "sha256:")
	match := ok && valid && len(value) == 64 && subtle.ConstantTimeCompare([]byte(value), []byte(artifact.Digest().String())) == 1
	evidenceID, _ := domain.NewEvidenceID("github-release-declared-sha256-result")
	evidence, _ := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), "github-release-declared-sha256", "GitHub Releases API SHA-256 was checked against controlled intake.")
	if !match {
		finding, _ := domain.NewFinding("M6_DECLARED_SHA256_MISMATCH", []domain.EvidenceID{evidenceID})
		return domain.NewVerificationReportWithFindings(check, domain.VerificationMismatch, []domain.Finding{finding}, []domain.Evidence{evidence})
	}
	return domain.NewVerificationReport(check, domain.VerificationVerified, []domain.Evidence{evidence})
}
