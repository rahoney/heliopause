package githubrelease

import (
	"context"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestIntegrityVerifierRejectsMismatch(t *testing.T) {
	t.Parallel()
	source, _ := domain.NewSourceID("github-release")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "owner-repo", "v1", "asset")
	digest, _ := domain.NewSHA256Digest("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	artifact, _ := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:run:github-release", 1, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	report, err := (IntegrityVerifier{}).Verify(context.Background(), artifact)
	if err != nil || report.Outcome() != domain.VerificationMismatch {
		t.Fatalf("Verify() = %#v, %v", report, err)
	}
}
