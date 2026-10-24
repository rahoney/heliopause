package application

import (
	"context"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/policy"
)

type acquiredActionFixture struct {
	verification domain.VerificationReport
	inspection   domain.InspectionReport
}

func (p acquiredActionFixture) Verify(context.Context, domain.AcquiredArtifact) (domain.VerificationReport, error) {
	return p.verification, nil
}
func (p acquiredActionFixture) Inspect(context.Context, domain.AcquiredArtifact) (domain.InspectionReport, error) {
	return p.inspection, nil
}
func (p acquiredActionFixture) Record(_ context.Context, _ domain.RunID, facts []domain.Evidence) ([]domain.EvidenceReference, error) {
	var refs []domain.EvidenceReference
	for _, fact := range facts {
		ref, err := domain.NewEvidenceReference(fact.ID(), "fixture:"+fact.ID().String())
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func TestAcquiredDependencyRetainsArchiveActionAndRejectsConflicts(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		run, err := domain.NewRunID()
		if err != nil {
			t.Fatal(err)
		}
		operation, err := domain.NewOperationID()
		if err != nil {
			t.Fatal(err)
		}
		source, _ := domain.NewSourceID("npm")
		identity, _ := domain.NewResolvedArtifactIdentity(source, "fixture", "1.0.0", "tarball")
		digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
		resolved, _ := domain.NewResolvedArtifact(identity, "registry:npm:fixture", "sha512-fixture")
		artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+run.String()+":tarball", 1, resolved.DeclaredIntegrity())
		if err != nil {
			t.Fatal(err)
		}
		node, _ := domain.NewDependencyNodeID("fixture")
		dependency, err := domain.NewLockedDependency(node, domain.DependencyTransitive, resolved)
		if err != nil {
			t.Fatal(err)
		}
		verifyID, _ := domain.NewCheckID("fixture-verification")
		verify, _ := domain.NewCheckExecution(verifyID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		verifyEvidenceID, _ := domain.NewEvidenceID("fixture-verification")
		verifyFact, _ := domain.NewEvidence(verifyEvidenceID, verifyID, identity, digest, "verification", "verification completed")
		verification, err := domain.NewVerificationReport(verify, domain.VerificationVerified, []domain.Evidence{verifyFact})
		if err != nil {
			t.Fatal(err)
		}
		checkID, _ := domain.NewCheckID("npm-static-archive")
		check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		factID, _ := domain.NewEvidenceID("npm-host-install-action")
		fact, err := domain.NewEvidence(factID, checkID, identity, digest, domain.EvidenceHostInstallActionRequired, "Host installation action declared")
		if err != nil {
			t.Fatal(err)
		}
		inspection, err := domain.NewInspectionReport(check, nil, []domain.Evidence{fact})
		if err != nil {
			t.Fatal(err)
		}
		p := acquiredActionFixture{verification: verification, inspection: inspection}
		s := &InstallInspectService{verification: p, inspection: p, evidence: p, entryPolicy: policy.M3{}, newRunID: func() (domain.RunID, error) { return run, nil }}
		var extra []domain.Evidence
		if duplicate {
			otherID, _ := domain.NewEvidenceID("conflicting-install-action")
			other, err := domain.NewEvidence(otherID, checkID, identity, digest, domain.EvidenceHostInstallActionNotRequired, "conflicting declaration")
			if err != nil {
				t.Fatal(err)
			}
			extra = []domain.Evidence{other}
		}
		record, err := s.inspectAcquiredDependency(context.Background(), operation, dependency, artifact, nil, extra)
		if duplicate {
			if err == nil {
				t.Fatal("conflicting derived fact accepted")
			}
			continue
		}
		retained, known := record.HostInstallActionEvidence()
		if err != nil || !known || retained != fact || record.PolicyDecision().Decision() != domain.DecisionAllow {
			t.Fatalf("acquired fact or inspect-only policy lost: %v %v", record, err)
		}
	}
}
