package terraformprovider

import (
	"strings"
	"testing"

	artifactterraform "github.com/rahoney/heliopause/internal/artifact/terraformprovider"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/policy"
)

func TestProviderProbePreservesSuspiciousFactsAndRequiresCompleteTerminal(t *testing.T) {
	identity, e := domain.NewResolvedArtifactIdentity(artifactterraform.Source(), "hashicorp_random", "3.7.2", "linux/amd64")
	if e != nil {
		t.Fatal(e)
	}
	digest, e := domain.NewSHA256Digest(strings.Repeat("a", 64))
	if e != nil {
		t.Fatal(e)
	}
	artifact, e := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:linux/amd64", 1)
	if e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		name          string
		status        domain.SandboxStatus
		limit         string
		terminal      bool
		category      domain.ObservationCategory
		subject, code string
		want          domain.ExecutionStatus
	}{
		{"help-zero", domain.SandboxCompleted, "", true, "", "", "", domain.ExecutionCompleted},
		{"missing-terminal", domain.SandboxCompleted, "", false, "", "", "", domain.ExecutionIncomplete},
		{"network", domain.SandboxCompleted, "", true, domain.ObservationNetwork, "network-attempt", "M3_NETWORK_ATTEMPT", domain.ExecutionCompleted},
		{"outside-filesystem", domain.SandboxCompleted, "", true, domain.ObservationFilesystem, "filesystem-outside-workspace", "M3_FILESYSTEM_VIOLATION", domain.ExecutionCompleted},
		{"unexpected-process", domain.SandboxCompleted, "", true, domain.ObservationProcess, "process-exec-unexpected", "M3_UNEXPECTED_PROCESS", domain.ExecutionCompleted},
		{"honeytoken", domain.SandboxCompleted, "", true, domain.ObservationHoneytoken, "honeytoken-access", "M3_HONEYTOKEN_ACCESS", domain.ExecutionCompleted},
		{"resource", domain.SandboxCompleted, "", true, domain.ObservationResource, "resource-limit", "", domain.ExecutionIncomplete},
		{"observer-failure-with-network", domain.SandboxIncomplete, "M3_DYNAMIC_OBSERVER_FAILED", false, domain.ObservationNetwork, "network-attempt", "M3_NETWORK_ATTEMPT", domain.ExecutionIncomplete},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var observations []domain.SandboxObservation
			add := func(category domain.ObservationCategory, subject string) {
				o, e := domain.NewSandboxObservation(category, subject)
				if e != nil {
					t.Fatal(e)
				}
				observations = append(observations, o)
			}
			if tc.terminal {
				add(domain.ObservationProcess, "terraform-help-exit-0")
				add(domain.ObservationProcess, "lifecycle-completed")
			}
			if tc.category != "" {
				add(tc.category, tc.subject)
			}
			session, e := domain.NewSandboxSessionID()
			if e != nil {
				t.Fatal(e)
			}
			result, e := domain.NewSandboxResult(session, tc.status, tc.limit, observations)
			if e != nil {
				t.Fatal(e)
			}
			reports, e := providerProbeReports(artifact, result)
			if e != nil {
				t.Fatal(e)
			}
			report, e := domain.NewCompositeInspectionReport(reports)
			if e != nil {
				t.Fatal(e)
			}
			required := false
			for _, check := range report.Executions() {
				if check.ID().String() == "terraform-provider-dynamic" {
					required = true
					if !check.Required() || check.Status() != tc.want {
						t.Fatalf("required check=%v", check)
					}
				}
			}
			if !required {
				t.Fatal("required probe coverage lost")
			}
			signedID, _ := domain.NewCheckID("fixture-signed-source")
			signedCheck, _ := domain.NewCheckExecution(signedID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
			signedEvidenceID, _ := domain.NewEvidenceID("fixture-signed-evidence")
			signedEvidence, e := domain.NewEvidence(signedEvidenceID, signedID, artifact.Identity(), artifact.Digest(), "fixture", "Synthetic verification for Policy composition only.")
			if e != nil {
				t.Fatal(e)
			}
			verification, e := domain.NewVerificationReport(signedCheck, domain.VerificationVerified, []domain.Evidence{signedEvidence})
			if e != nil {
				t.Fatal(e)
			}
			var refs []domain.EvidenceReference
			for _, ev := range append(verification.Evidence(), report.Evidence()...) {
				ref, e := domain.NewEvidenceReference(ev.ID(), "fixture:"+ev.ID().String())
				if e != nil {
					t.Fatal(e)
				}
				refs = append(refs, ref)
			}
			run, _ := domain.NewRunID()
			input, e := domain.NewPolicyInput(run, artifact, verification, report, refs)
			if e != nil {
				t.Fatal(e)
			}
			decision, e := (policy.M3{}).Evaluate(input)
			if e != nil {
				t.Fatal(e)
			}
			wantPolicy := domain.DecisionAllow
			if tc.want != domain.ExecutionCompleted || tc.code == "M3_NETWORK_ATTEMPT" || tc.code == "M3_UNEXPECTED_PROCESS" {
				wantPolicy = domain.DecisionManualReview
			}
			if tc.code == "M3_FILESYSTEM_VIOLATION" || tc.code == "M3_HONEYTOKEN_ACCESS" {
				wantPolicy = domain.DecisionBlock
			}
			if decision.Decision() != wantPolicy {
				t.Fatalf("unexpected Policy %s", decision.Decision())
			}
			if len(report.Evidence()) != 1 {
				t.Fatal("observed facts lost on incomplete outcome")
			}
			if tc.code != "" {
				if len(report.Findings()) != 1 || report.Findings()[0].Code() != tc.code {
					t.Fatalf("finding lost: %v", report.Findings())
				}
			} else if len(report.Findings()) != 0 {
				t.Fatal("unexpected finding")
			}
		})
	}
}
