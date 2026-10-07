//go:build integration

package cargo_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/application"
	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectioncargo "github.com/rahoney/heliopause/internal/inspection/cargo"
	"github.com/rahoney/heliopause/internal/policy"
	verificationcargo "github.com/rahoney/heliopause/internal/verification/cargo"
)

// This uses an empty operation-owned intake and real public source, independent
// official HTTPS index and frozen checksum, normalized static inspection, recorded Evidence and entry Policy.
// It does not qualify project mutation, verified cache or sandbox builds.
func TestCargoPublicCrateAcquisitionIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	root := t.TempDir()
	intake := filepath.Join(root, "intake")
	client, err := artifactcargo.NewPublicClient(intake)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := local.NewStore(filepath.Join(root, "evidence"))
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := verificationcargo.NewIntegrityVerifier(intake)
	if err != nil {
		t.Fatal(err)
	}
	inspector, err := inspectioncargo.NewStaticInspector(intake)
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewInspectService(client, verifier, inspector, evidence, policy.M3{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := artifactcargo.ParseReference("itoa@1.0.17")
	if err != nil {
		t.Fatal(err)
	}
	request, err := application.NewInspectRequest(ref)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Inspect(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	decision, ok := result.PolicyDecision()
	if !ok || decision.Decision() != domain.DecisionAllow || len(result.Checks()) != 2 || len(result.Evidence()) != 2 {
		t.Fatalf("incomplete acquisition/verification/inspection/evidence/policy: status=%s", result.Status())
	}
	for _, check := range result.Checks() {
		if check.Status() != domain.ExecutionCompleted || !check.Required() {
			t.Fatal("incomplete required check")
		}
	}
	identity, _ := result.ResolvedIdentity()
	digest, _ := result.Digest()
	if digest.String() != "92ecc6618181def0457392ccd0ee51198e065e016d1d527a7ac1b6dc7c1f09d2" {
		t.Fatal("actual crate bytes differ from independent qualification pin")
	}
	t.Logf("source=%s version=%s digest=%s checks=%d evidence=%d policy=%s", ref.Source().String(), identity.Version(), digest.String(), len(result.Checks()), len(result.Evidence()), decision.Decision())
}
