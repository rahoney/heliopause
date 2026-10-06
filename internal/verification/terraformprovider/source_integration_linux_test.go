package terraformprovider_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	artifactterraform "github.com/rahoney/heliopause/internal/artifact/terraformprovider"
	"github.com/rahoney/heliopause/internal/core/domain"
	verifier "github.com/rahoney/heliopause/internal/verification/terraformprovider"
)

// Real public acquisition and source verification only. This does not execute
// a provider, assert Policy ALLOW, or substitute for init transaction gates.
func TestPublicTerraformProviderSourceIntegration(t *testing.T) {
	if os.Getenv("HELOX_RUN_TERRAFORM_SOURCE_INTEGRATION") != "1" {
		t.Skip("explicit public provider source qualification required")
	}
	for _, fixture := range []struct {
		reference, digest string
		tier              verifier.TrustTier
	}{
		{"hashicorp/random@3.7.2", "7b8434212eef0f8c83f5a90c6d76feaf850f6502b61b53c329e85b3b281cba34", verifier.TrustOfficial},
		{"integrations/github@6.6.0", "772edb5890d72b32868f9fdc0a9a1d4f4701d8e7f8acb37a7ac530d053c776e3", verifier.TrustPartner},
	} {
		t.Run(fixture.reference, func(t *testing.T) {
			root := t.TempDir()
			if parent := os.Getenv("HELOX_TERRAFORM_TEST_INTAKE_ROOT"); parent != "" {
				info, err := os.Lstat(parent)
				if err != nil || !filepath.IsAbs(parent) || filepath.Clean(parent) != parent || !info.IsDir() || info.Mode().Perm() != 0o700 || info.Mode()&os.ModeSymlink != 0 {
					t.Fatal("retained qualification parent is not private and canonical")
				}
				root, err = os.MkdirTemp(parent, "provider-source-")
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Chmod(root, 0o700); err != nil {
				t.Fatal(err)
			}
			client, err := artifactterraform.NewPublicClient(root)
			if err != nil {
				t.Fatal(err)
			}
			reference, err := artifactterraform.ParseReference(fixture.reference)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			resolved, err := client.Resolve(ctx, reference)
			if err != nil {
				t.Fatal(err)
			}
			run, err := domain.NewRunID()
			if err != nil {
				t.Fatal(err)
			}
			acquired, err := client.Acquire(ctx, run, resolved)
			if err != nil {
				t.Fatal(err)
			}
			bundle, err := artifactterraform.ReadIntake(root, acquired)
			if err != nil {
				t.Fatal(err)
			}
			if bundle.ArchiveDigest() != fixture.digest {
				t.Fatal("public archive differs from independent frozen fixture pin")
			}
			provider, err := bundle.Registry().Parse(reference, artifactterraform.Platform{OS: "linux", Arch: "amd64"})
			if err != nil {
				t.Fatal(err)
			}
			binding, err := verifier.VerifySignedChecksums(provider, bundle.Checksums(), bundle.Signature(), time.Now().UTC())
			if err != nil || binding.Trust != fixture.tier {
				t.Fatalf("public trust binding=%+v %v", binding, err)
			}
			integrity, err := verifier.NewIntegrityVerifier(root)
			if err != nil {
				t.Fatal(err)
			}
			report, err := integrity.Verify(ctx, acquired)
			if err != nil || report.Outcome() != domain.VerificationVerified || report.Execution().Status() != domain.ExecutionCompleted {
				t.Fatalf("public normalized verification=%s %v", report.Outcome(), err)
			}
			t.Logf("Exact public provider verified; archive=%s; envelope=%s; signer=%s; trust=%s", bundle.ArchiveDigest(), acquired.Digest(), binding.Fingerprint, binding.Trust)
			t.Logf("Qualification intake directory=%s; Run=%s", root, run)
		})
	}
}
