package releaseprovenance

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
	githubverification "github.com/rahoney/heliopause/internal/verification/githubrelease"
)

// ArtifactVerifier binds controlled bytes to an official release subject. The
// declaration and proof may be untrusted; only Verify can establish provenance.
// A successful report still grants no inspection or installation authority.
type ArtifactVerifier struct {
	expected Identity
	bundles  [][]byte
}

func NewArtifactVerifier(expected Identity, bundles [][]byte) (*ArtifactVerifier, error) {
	if !validIdentity(expected) || len(bundles) == 0 || len(bundles) > 16 {
		return nil, errors.New("release authenticity configuration is invalid")
	}
	owned := make([][]byte, len(bundles))
	for i, body := range bundles {
		if len(body) == 0 || len(body) > maxBundle {
			return nil, errors.New("release authenticity proof is invalid or excessive")
		}
		owned[i] = append([]byte(nil), body...)
	}
	return &ArtifactVerifier{expected, owned}, nil
}

func (v *ArtifactVerifier) Verify(ctx context.Context, artifact domain.AcquiredArtifact) (domain.VerificationReport, error) {
	if ctx == nil || v == nil {
		return domain.VerificationReport{}, errors.New("release authenticity request is invalid")
	}
	if err := ctx.Err(); err != nil {
		return domain.VerificationReport{}, err
	}
	integrity, err := (githubverification.IntegrityVerifier{}).Verify(ctx, artifact)
	if err != nil {
		return domain.VerificationReport{}, err
	}
	matched := false
	if integrity.Outcome() == domain.VerificationVerified && artifact.Identity().Name() == "rahoney-heliopause" && artifact.Identity().Version() == v.expected.Version {
		for _, body := range v.bundles {
			_, err := Verify(ctx, body, v.expected, []Subject{{Name: artifact.Identity().Variant(), SHA256: artifact.Digest().String()}})
			if ctx.Err() != nil {
				return domain.VerificationReport{}, ctx.Err()
			}
			if err == nil {
				matched = true
				break
			}
		}
	}
	checkID, _ := domain.NewCheckID("release-authenticity")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidenceID, _ := domain.NewEvidenceID("release-authenticity-result")
	summary := "Declared SHA-256 and official release provenance did not both match the exact controlled subject."
	if matched {
		summary = "Declared SHA-256 and official release provenance matched the exact controlled subject and release source."
	}
	evidence, err := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), "release-authenticity", summary)
	if err != nil {
		return domain.VerificationReport{}, err
	}
	if matched {
		return domain.NewVerificationReport(check, domain.VerificationVerified, []domain.Evidence{evidence})
	}
	finding, _ := domain.NewFinding("SELF_UPDATE_AUTHENTICITY_MISMATCH", []domain.EvidenceID{evidenceID})
	return domain.NewVerificationReportWithFindings(check, domain.VerificationMismatch, []domain.Finding{finding}, []domain.Evidence{evidence})
}
