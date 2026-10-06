package terraformprovider

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	artifactterraform "github.com/rahoney/heliopause/internal/artifact/terraformprovider"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type IntegrityVerifier struct{ intakeRoot string }

func NewIntegrityVerifier(intakeRoot string) (*IntegrityVerifier, error) {
	if !filepath.IsAbs(intakeRoot) || filepath.Clean(intakeRoot) != intakeRoot || intakeRoot == "/" {
		return nil, errors.New("terraform verifier intake root is invalid")
	}
	return &IntegrityVerifier{intakeRoot}, nil
}

func (v *IntegrityVerifier) Verify(ctx context.Context, artifact domain.AcquiredArtifact) (domain.VerificationReport, error) {
	if v == nil || ctx == nil || artifact.Identity().Source() != artifactterraform.Source() {
		return domain.VerificationReport{}, errors.New("terraform verification request is invalid")
	}
	if err := ctx.Err(); err != nil {
		return domain.VerificationReport{}, err
	}
	bundle, err := artifactterraform.ReadIntake(v.intakeRoot, artifact)
	if err != nil {
		return domain.VerificationReport{}, err
	}
	declared, present := artifact.DeclaredIntegrity()
	registry, archive, err := artifactterraform.ParseIntegrity(declared)
	if !present || err != nil || registry != bundle.RegistryDigest() || archive != bundle.ArchiveDigest() {
		return signedReport(artifact, domain.VerificationMismatch, domain.ExecutionCompleted, "", SignatureBinding{})
	}
	reference, err := artifactterraform.IdentityReference(artifact.Identity())
	if err != nil {
		return domain.VerificationReport{}, err
	}
	provider, err := bundle.Registry().Parse(reference, artifactterraform.Platform{OS: "linux", Arch: "amd64"})
	if err != nil || provider.SHA256 != archive {
		return signedReport(artifact, domain.VerificationMismatch, domain.ExecutionCompleted, "", SignatureBinding{})
	}
	binding, err := VerifySignedChecksums(provider, bundle.Checksums(), bundle.Signature(), time.Now().UTC())
	if err != nil {
		return signedReport(artifact, domain.VerificationMismatch, domain.ExecutionCompleted, "", SignatureBinding{})
	}
	if err := ctx.Err(); err != nil {
		return domain.VerificationReport{}, err
	}
	if binding.Trust == TrustCommunity {
		return signedReport(artifact, "", domain.ExecutionIncomplete, "M12_TERRAFORM_SIGNER_REVIEW_REQUIRED", binding)
	}
	return signedReport(artifact, domain.VerificationVerified, domain.ExecutionCompleted, "", binding)
}

func signedReport(artifact domain.AcquiredArtifact, status domain.VerificationOutcome, execution domain.ExecutionStatus, limitation string, binding SignatureBinding) (domain.VerificationReport, error) {
	checkID, _ := domain.NewCheckID("terraform-signed-package")
	check, err := domain.NewCheckExecution(checkID, domain.CheckVerification, true, domain.CapabilitySupported, execution, limitation)
	if err != nil {
		return domain.VerificationReport{}, err
	}
	evidenceID, _ := domain.NewEvidenceID("terraform-signed-package-result")
	summary := "Exact frozen registry, signed checksum and provider archive comparison failed."
	if binding.Fingerprint != "" {
		summary = "Verified checksum signature; primary=" + binding.Fingerprint + "; signing=" + binding.SigningFingerprint + "; trust=" + string(binding.Trust) + "; archive_sha256=" + binding.ChecksumSHA256 + "; registry, archive and envelope digests are distinct."
	}
	evidence, err := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), "terraform-signed-package", summary)
	if err != nil {
		return domain.VerificationReport{}, err
	}
	if status != domain.VerificationMismatch {
		return domain.NewVerificationReport(check, status, []domain.Evidence{evidence})
	}
	finding, err := domain.NewFinding("M12_TERRAFORM_SIGNED_PACKAGE_MISMATCH", []domain.EvidenceID{evidenceID})
	if err != nil {
		return domain.VerificationReport{}, err
	}
	return domain.NewVerificationReportWithFindings(check, status, []domain.Finding{finding}, []domain.Evidence{evidence})
}
