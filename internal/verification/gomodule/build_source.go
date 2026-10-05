package gomodule

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// BuildSourceReader rehashes bounded frozen transport bytes. It confers no
// public source authentication or approval on user project code.
type BuildSourceReader interface {
	VerifyBuildSource(context.Context, domain.AcquiredArtifact) error
}

type BuildSourceVerifier struct{ reader BuildSourceReader }

func NewBuildSourceVerifier(reader BuildSourceReader) (*BuildSourceVerifier, error) {
	if reader == nil {
		return nil, errors.New("project source verification requires a bounded reader")
	}
	return &BuildSourceVerifier{reader}, nil
}

func (v *BuildSourceVerifier) Verify(ctx context.Context, artifact domain.AcquiredArtifact) (domain.VerificationReport, error) {
	if v == nil || v.reader == nil || ctx == nil || artifact.Identity().Source().String() != "project-local" || artifact.Identity().Variant() != "go-source" {
		return domain.VerificationReport{}, errors.New("project source verification request is invalid")
	}
	if err := ctx.Err(); err != nil {
		return domain.VerificationReport{}, err
	}
	if err := v.reader.VerifyBuildSource(ctx, artifact); err != nil {
		return domain.VerificationReport{}, errors.Join(errors.New("project source byte binding could not be verified"), ctx.Err())
	}
	checkID, _ := domain.NewCheckID("project-build-input-binding")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidenceID, _ := domain.NewEvidenceID("project-build-input-binding-result")
	evidence, err := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), "go-build-source-binding", "Frozen local source archive identity and complete consumed bytes rechecked. This verifies local byte binding, not public registry provenance or project safety.")
	if err != nil {
		return domain.VerificationReport{}, err
	}
	return domain.NewVerificationReport(check, domain.VerificationVerified, []domain.Evidence{evidence})
}
