package sandbox

import (
	"context"
	"os"
	"testing"
	"time"

	artifactterraform "github.com/rahoney/heliopause/internal/artifact/terraformprovider"
	"github.com/rahoney/heliopause/internal/core/domain"
	verificationterraform "github.com/rahoney/heliopause/internal/verification/terraformprovider"
)

// Required opt-in positive gate. Help stdout/exit alone never proves observer
// completion, trust, Policy ALLOW, provider RPC behavior or installation.
func TestLinuxTerraformProviderProbeIntegration(t *testing.T) {
	if os.Getenv("HELOX_TERRAFORM_PROVIDER_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	for _, fixture := range []struct{ reference, digest string }{
		{"hashicorp/random@3.7.2", "7b8434212eef0f8c83f5a90c6d76feaf850f6502b61b53c329e85b3b281cba34"},
		{"integrations/github@6.6.0", "772edb5890d72b32868f9fdc0a9a1d4f4701d8e7f8acb37a7ac530d053c776e3"},
	} {
		t.Run(fixture.reference, func(t *testing.T) {
			root := t.TempDir()
			if e := os.Chmod(root, 0o700); e != nil {
				t.Fatal(e)
			}
			client, e := artifactterraform.NewPublicClient(root)
			if e != nil {
				t.Fatal(e)
			}
			ref, e := artifactterraform.ParseReference(fixture.reference)
			if e != nil {
				t.Fatal(e)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			resolved, e := client.Resolve(ctx, ref)
			if e != nil {
				t.Fatal(e)
			}
			run, e := domain.NewRunID()
			if e != nil {
				t.Fatal(e)
			}
			acquired, e := client.Acquire(ctx, run, resolved)
			if e != nil {
				t.Fatal(e)
			}
			bundle, e := artifactterraform.ReadIntake(root, acquired)
			if e != nil || bundle.ArchiveDigest() != fixture.digest {
				t.Fatalf("independent archive pin mismatch %v", e)
			}
			verifier, e := verificationterraform.NewIntegrityVerifier(root)
			if e != nil {
				t.Fatal(e)
			}
			report, e := verifier.Verify(ctx, acquired)
			if e != nil || report.Outcome() != domain.VerificationVerified {
				t.Fatalf("signed verification failed: %v", e)
			}
			supervisor := integrationObserverSupervisor(t)
			defer supervisor.Close()
			runner := integrationRunner{t: t}
			elf, e := NewGitHubELFBackend(runner, root, supervisor.Observer(), integrationCapabilityProbe(runner))
			if e != nil {
				t.Fatal(e)
			}
			backend := &TerraformProviderBackend{elf: elf}
			request, e := domain.NewSandboxRequest(acquired)
			if e != nil {
				t.Fatal(e)
			}
			result, e := backend.Execute(ctx, request)
			if e != nil || result.Status() != domain.SandboxCompleted {
				code, _ := result.LimitationCode()
				t.Fatalf("Provider probe result=%s/%s observer_reason=%s error=%v", result.Status(), code, integrationObserverFaultReason(supervisor), e)
			}
			for _, o := range result.Observations() {
				if o.Category() == domain.ObservationNetwork || o.Category() == domain.ObservationResource || o.Category() == domain.ObservationHoneytoken || (o.Category() == domain.ObservationFilesystem && o.Subject() != "filesystem-workspace-access") || (o.Category() == domain.ObservationProcess && (o.Subject() == "process-exec-unexpected" || o.Subject() == "process-unexpected")) {
					t.Fatalf("positive provider has disallowed observation: %s/%s", o.Category(), o.Subject())
				}
			}
			summary, e := result.ObservationSummary()
			if e != nil {
				t.Fatal(e)
			}
			t.Logf("Exact public provider help probe completed: reference=%s archive=%s envelope=%s %s", fixture.reference, bundle.ArchiveDigest(), acquired.Digest(), summary)
		})
	}
}
