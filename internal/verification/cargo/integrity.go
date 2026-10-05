// Package cargo verifies exact crate bytes against independently retrieved
// official sparse-index checksums and the frozen Cargo.lock declaration.
package cargo

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type registryLookup interface {
	LookupChecksum(context.Context, string, string) (string, error)
}

type IntegrityVerifier struct {
	intakeRoot string
	registry   registryLookup
}

func NewIntegrityVerifier(intakeRoot string) (*IntegrityVerifier, error) {
	if !filepath.IsAbs(intakeRoot) || filepath.Clean(intakeRoot) != intakeRoot || intakeRoot == "/" {
		return nil, errors.New("cargo verifier intake root is invalid")
	}
	client, err := artifactcargo.NewPublicClient(intakeRoot)
	if err != nil {
		return nil, err
	}
	return &IntegrityVerifier{intakeRoot, client}, nil
}

func (v *IntegrityVerifier) Verify(ctx context.Context, artifact domain.AcquiredArtifact) (domain.VerificationReport, error) {
	if v == nil || v.registry == nil || ctx == nil || artifact.Identity().Source() != artifactcargo.Source() {
		return domain.VerificationReport{}, errors.New("cargo verification request is invalid")
	}
	if err := ctx.Err(); err != nil {
		return domain.VerificationReport{}, err
	}
	if _, err := artifactcargo.ReadIntake(v.intakeRoot, artifact); err != nil {
		return domain.VerificationReport{}, err
	}
	declared, present := artifact.DeclaredIntegrity()
	checksum, err := artifactcargo.ParseIntegrity(declared)
	if !present || err != nil || checksum != artifact.Digest().String() {
		return verificationReport(artifact, false, "M12_CARGO_LOCK_CHECKSUM_MISMATCH")
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	indexed, err := v.registry.LookupChecksum(lookupCtx, artifact.Identity().Name(), artifact.Identity().Version())
	if err != nil {
		return domain.VerificationReport{}, err
	}
	if indexed != checksum {
		return verificationReport(artifact, false, "M12_CARGO_INDEX_CHECKSUM_MISMATCH")
	}
	return verificationReport(artifact, true, "")
}

func verificationReport(artifact domain.AcquiredArtifact, verified bool, code string) (domain.VerificationReport, error) {
	checkID, _ := domain.NewCheckID("cargo-registry-integrity")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidenceID, _ := domain.NewEvidenceID("cargo-registry-integrity-result")
	summary := "Exact crate bytes compared with frozen lock and independently retrieved official HTTPS sparse-index checksum."
	if !verified {
		summary = "Exact crate checksum comparison found a frozen lock or official sparse-index mismatch."
	}
	evidence, err := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), "cargo-registry-integrity", summary)
	if err != nil {
		return domain.VerificationReport{}, err
	}
	if verified {
		return domain.NewVerificationReport(check, domain.VerificationVerified, []domain.Evidence{evidence})
	}
	finding, err := domain.NewFinding(code, []domain.EvidenceID{evidenceID})
	if err != nil {
		return domain.VerificationReport{}, err
	}
	return domain.NewVerificationReportWithFindings(check, domain.VerificationMismatch, []domain.Finding{finding}, []domain.Evidence{evidence})
}
