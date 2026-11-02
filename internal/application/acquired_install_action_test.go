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
func (p acquiredActionFixture) Record(_ context.Context, _ domain.RunID, facts []domain.Evidence, checks ...domain.CheckExecution) ([]domain.EvidenceReference, error) {
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
	for _, mode := range []string{"complete", "duplicate-action", "derived", "undeclared-derived", "foreign-derived-fact"} {
		duplicate := mode == "duplicate-action"
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
		verify, _ := domain.NewCheckExecution(verifyID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", acquiredActionProfile())
		verifyEvidenceID, _ := domain.NewEvidenceID("fixture-verification")
		verifyFact, _ := domain.NewEvidence(verifyEvidenceID, verifyID, identity, digest, "verification", "verification completed")
		verification, err := domain.NewVerificationReport(verify, domain.VerificationVerified, []domain.Evidence{verifyFact})
		if err != nil {
			t.Fatal(err)
		}
		checkID, _ := domain.NewCheckID("npm-static-archive")
		check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", acquiredActionProfile())
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
		var extraChecks []domain.CheckExecution
		var requirements []domain.CheckRequirement
		if mode == "derived" || mode == "undeclared-derived" || mode == "foreign-derived-fact" {
			id, _ := domain.NewCheckID("fixture-derived-build")
			check, e := domain.NewCheckExecution(id, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", acquiredActionProfile())
			if e != nil {
				t.Fatal(e)
			}
			factID, _ := domain.NewEvidenceID("fixture-derived-build")
			factCheck := id
			if mode == "foreign-derived-fact" {
				factCheck, _ = domain.NewCheckID("foreign-derived-check")
			}
			fact, e := domain.NewEvidence(factID, factCheck, identity, digest, "fixture-derived-build", "Completed independently declared controlled build.")
			if e != nil {
				t.Fatal(e)
			}
			extraChecks = []domain.CheckExecution{check}
			extra = []domain.Evidence{fact}
			if mode != "undeclared-derived" {
				requirement, e := domain.NewCheckRequirement(id, domain.CheckInspection, true, acquiredActionProfile())
				if e != nil {
					t.Fatal(e)
				}
				requirements = append(requirements, requirement)
			}
		}
		record, err := s.inspectAcquiredDependency(context.Background(), operation, dependency, artifact, extraChecks, extra, requirements...)
		if duplicate || mode == "undeclared-derived" || mode == "foreign-derived-fact" {
			if err == nil {
				t.Fatal("conflicting derived fact accepted")
			}
			continue
		}
		if mode == "derived" && (len(record.Checks()) != 3 || len(record.CheckPlan().Requirements()) != 3 || len(record.Evidence()) != 3) {
			t.Fatal("derived check/evidence/declaration lost")
		}
		retained, known := record.HostInstallActionEvidence()
		if err != nil || !known || retained != fact || record.PolicyDecision().Decision() != domain.DecisionAllow {
			t.Fatalf("acquired fact or inspect-only policy lost: %v %v", record, err)
		}
	}
}

func acquiredActionProfile() domain.ContentDigest {
	profile, _ := domain.CheckProfileDigest("acquired-action-fixture-v1", "")
	return profile
}
func (p acquiredActionFixture) ExpectedChecks(_ context.Context, _ domain.AcquiredArtifact, kind domain.CheckKind) ([]domain.CheckRequirement, error) {
	name := "fixture-verification"
	if kind == domain.CheckInspection {
		name = "npm-static-archive"
	}
	id, _ := domain.NewCheckID(name)
	requirement, err := domain.NewCheckRequirement(id, kind, true, acquiredActionProfile())
	return []domain.CheckRequirement{requirement}, err
}
