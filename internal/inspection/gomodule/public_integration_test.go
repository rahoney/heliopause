//go:build integration

package gomodule_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/application"
	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectiongo "github.com/rahoney/heliopause/internal/inspection/gomodule"
	"github.com/rahoney/heliopause/internal/policy"
	verificationgo "github.com/rahoney/heliopause/internal/verification/gomodule"
)

// This uses an empty operation-owned intake and real public source, signed
// SumDB/proofs, normalized static inspection, recorded Evidence and entry Policy.
// It does not qualify project mutation, verified cache or sandbox builds.
func TestGoPublicModuleAcquisitionIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	root := t.TempDir()
	intake := filepath.Join(root, "intake")
	client, err := artifactgo.NewPublicClient(intake)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := local.NewStore(filepath.Join(root, "evidence"))
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := verificationgo.NewIntegrityVerifier(intake)
	if err != nil {
		t.Fatal(err)
	}
	inspector, err := inspectiongo.NewStaticInspector(intake)
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewInspectService(client, verifier, inspector, evidence, policy.M3{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := artifactgo.ParseReference("github.com/spf13/pflag@v1.0.9")
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
	t.Logf("source=%s version=%s digest=%s checks=%d evidence=%d policy=%s", ref.Source().String(), identity.Version(), digest.String(), len(result.Checks()), len(result.Evidence()), decision.Decision())
}
