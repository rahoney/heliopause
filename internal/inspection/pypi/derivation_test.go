package pypi

import (
	"context"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestDeriverRejectsNonAllowSdistBeforeBuild(t *testing.T) {
	t.Parallel()
	source, _ := domain.NewSourceID("pypi")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "example", "1.0", "sdist")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	artifact, _ := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:sdist", 1, "sha256:"+digest.String())
	node, _ := domain.NewDependencyNodeID("example")
	run, _ := domain.ParseRunID("run_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	checkID, _ := domain.NewCheckID("pypi-static-archive")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidenceID, _ := domain.NewEvidenceID("pypi-static-result")
	reference, _ := domain.NewEvidenceReference(evidenceID, "fixture:pypi-static-result")
	decision, _ := domain.NewPolicyDecision(domain.DecisionManualReview, "m5-pypi-pip", 1, []string{"M5_REQUIRED_CHECK_INCOMPLETE"})
	inspection, _ := domain.NewDependencyInspection(node, run, artifact, []domain.CheckExecution{check}, []domain.EvidenceReference{reference}, decision)

	if _, err := (&Deriver{}).Derive(context.Background(), []domain.DependencyInspection{inspection}); err == nil {
		t.Fatal("Derive accepted a source that is not a complete ALLOW inspection")
	}
}
