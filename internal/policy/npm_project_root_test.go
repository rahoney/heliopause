package policy

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestNPMProjectRootPolicyKeepsExactChecksAndHostActionBoundary(t *testing.T) {
	for _, fault := range []string{"required Host action", "no Host action", "missing lifecycle", "optional lifecycle", "incomplete", "unknown action", "duplicate action", "unexpected check", "honeytoken"} {
		t.Run(fault, func(t *testing.T) {
			local, _ := domain.NewSourceID("project-local")
			identity, _ := domain.NewResolvedArtifactIdentity(local, "project-fixture", "snapshot", "npm-source")
			digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
			artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:npm-source", 1024, "sha256:"+digest.String())
			if err != nil {
				t.Fatal(err)
			}
			verificationID, _ := domain.NewCheckID("project-build-input-binding")
			profile, _ := domain.CheckProfileDigest("m3-fixture-v1", "")
			verified, _ := domain.NewCheckExecution(verificationID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", profile)
			verification, err := domain.NewVerificationReport(verified, domain.VerificationVerified, []domain.Evidence{evidence(t, "source-binding", verified, artifact)})
			if err != nil {
				t.Fatal(err)
			}
			static := check(t, "npm-project-root-static", domain.ExecutionCompleted, "")
			staticEvidence := []domain.Evidence{evidence(t, "static-structure", static, artifact)}
			kind := domain.EvidenceHostInstallActionRequired
			if fault == "no Host action" {
				kind = domain.EvidenceHostInstallActionNotRequired
			}
			actionID, _ := domain.NewEvidenceID("root-action")
			action, err := domain.NewEvidence(actionID, static.ID(), identity, digest, kind, "controller-owned fixture action")
			if err != nil {
				t.Fatal(err)
			}
			if fault != "unknown action" {
				staticEvidence = append(staticEvidence, action)
			}
			if fault == "duplicate action" {
				duplicateID, _ := domain.NewEvidenceID("root-action-duplicate")
				duplicate, err := domain.NewEvidence(duplicateID, static.ID(), identity, digest, kind, "duplicate fixture")
				if err != nil {
					t.Fatal(err)
				}
				staticEvidence = append(staticEvidence, duplicate)
			}
			staticReport, err := domain.NewInspectionReport(static, nil, staticEvidence)
			if err != nil {
				t.Fatal(err)
			}
			factCheckID, _ := domain.NewCheckID("npm-project-root-observation-facts")
			factCheck, _ := domain.NewCheckExecution(factCheckID, domain.CheckInspection, false, domain.CapabilitySupported, domain.ExecutionCompleted, "", profile)
			factReport, err := domain.NewInspectionReport(factCheck, nil, []domain.Evidence{evidence(t, "accepted-observations", factCheck, artifact)})
			if err != nil {
				t.Fatal(err)
			}
			leaves := []domain.InspectionReport{staticReport, factReport}
			if fault != "missing lifecycle" {
				id := "npm-project-root-lifecycle"
				if fault == "unexpected check" {
					id = "unexpected-check"
				}
				status, limitation := domain.ExecutionCompleted, ""
				if fault == "incomplete" {
					status, limitation = domain.ExecutionIncomplete, "M13_NPM_PROJECT_ROOT_INCOMPLETE"
				}
				checkID, _ := domain.NewCheckID(id)
				dynamic, err := domain.NewCheckExecution(checkID, domain.CheckInspection, fault != "optional lifecycle", domain.CapabilitySupported, status, limitation, profile)
				if err != nil {
					t.Fatal(err)
				}
				observed := evidence(t, "root-observation", dynamic, artifact)
				findings := []domain.Finding{}
				if fault == "honeytoken" {
					finding, err := domain.NewFinding("M3_HONEYTOKEN_ACCESS", []domain.EvidenceID{observed.ID()})
					if err != nil {
						t.Fatal(err)
					}
					findings = append(findings, finding)
				}
				report, err := domain.NewInspectionReport(dynamic, findings, []domain.Evidence{observed})
				if err != nil {
					t.Fatal(err)
				}
				leaves = append(leaves, report)
			}
			inspection, err := domain.NewCompositeInspectionReport(leaves)
			if err != nil {
				t.Fatal(err)
			}
			var references []domain.EvidenceReference
			for _, fact := range append(verification.Evidence(), inspection.Evidence()...) {
				reference, err := domain.NewEvidenceReference(fact.ID(), "fixture:"+fact.ID().String())
				if err != nil {
					t.Fatal(err)
				}
				references = append(references, reference)
			}
			run, _ := domain.ParseRunID("run_aaaaaaaaaaaaaaaaaaaaaaaaaa")
			var requirements []domain.CheckRequirement
			for _, name := range []string{"project-build-input-binding", "npm-project-root-static", "npm-project-root-lifecycle", "npm-project-root-observation-facts"} {
				id, _ := domain.NewCheckID(name)
				kind := domain.CheckInspection
				if name == "project-build-input-binding" {
					kind = domain.CheckVerification
				}
				requirement, e := domain.NewCheckRequirement(id, kind, name != "npm-project-root-observation-facts", profile)
				if e != nil {
					t.Fatal(e)
				}
				requirements = append(requirements, requirement)
			}
			plan, err := domain.NewCheckPlan(run, artifact, requirements)
			if err != nil {
				t.Fatal(err)
			}
			input, err := domain.NewPolicyInput(run, artifact, verification, inspection, references, plan)
			if fault == "missing lifecycle" || fault == "optional lifecycle" || fault == "unexpected check" {
				if err == nil {
					t.Fatal("invalid declared membership accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			decision, err := (NPMProjectRoot{}).Evaluate(input)
			if err != nil {
				t.Fatal(err)
			}
			expected := domain.DecisionManualReview
			if fault == "no Host action" {
				expected = domain.DecisionAllow
			}
			if fault == "honeytoken" {
				expected = domain.DecisionBlock
			}
			if decision.Decision() != expected {
				t.Fatal("root behavior/coverage/Host authority boundary changed", fault, decision.Decision(), expected)
			}
		})
	}
}
