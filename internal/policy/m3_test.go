package policy

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestM3RequiresCompletedDynamicCheckAndInterpretsDynamicFindings(t *testing.T) {
	tests := []struct {
		name    string
		dynamic domain.CheckExecution
		finding string
		want    domain.Decision
	}{
		{"clean", check(t, "npm-dynamic-lifecycle", domain.ExecutionCompleted, ""), "", domain.DecisionAllow},
		{"unavailable", check(t, "npm-dynamic-lifecycle", domain.ExecutionIncomplete, "M3_DYNAMIC_OBSERVER_FAILED"), "", domain.DecisionManualReview},
		{"network", check(t, "npm-dynamic-lifecycle", domain.ExecutionCompleted, ""), "M3_NETWORK_ATTEMPT", domain.DecisionManualReview},
		{"advisory review", check(t, "npm-dynamic-lifecycle", domain.ExecutionCompleted, ""), "M13_ADVISORY_REVIEW", domain.DecisionManualReview},
		{"advisory high", check(t, "npm-dynamic-lifecycle", domain.ExecutionCompleted, ""), "M13_ADVISORY_HIGH", domain.DecisionBlock},
		{"explicit malware advisory", check(t, "npm-dynamic-lifecycle", domain.ExecutionCompleted, ""), "M13_ADVISORY_MALICIOUS", domain.DecisionBlock},
		{"honeytoken", check(t, "npm-dynamic-lifecycle", domain.ExecutionCompleted, ""), "M3_HONEYTOKEN_ACCESS", domain.DecisionBlock},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := policyInput(t, test.dynamic, test.finding)
			decision, err := (M3{}).Evaluate(input)
			if err != nil || decision.Decision() != test.want {
				t.Fatalf("Evaluate() = (%q, %v), want %q", decision.Decision(), err, test.want)
			}
		})
	}
}

func policyInput(t *testing.T, dynamic domain.CheckExecution, code string, sources ...string) domain.PolicyInput {
	t.Helper()
	sourceName := "npm"
	if len(sources) == 1 {
		sourceName = sources[0]
	}
	source, _ := domain.NewSourceID(sourceName)
	identity, _ := domain.NewResolvedArtifactIdentity(source, "tiny", "1.2.3", "tarball")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	artifact, _ := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:tarball", 1)
	verificationID, _ := domain.NewCheckID("npm-integrity")
	profile, _ := domain.CheckProfileDigest("m3-fixture-v1", "")
	verificationCheck, _ := domain.NewCheckExecution(verificationID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", profile)
	verificationEvidence := evidence(t, "integrity-evidence", verificationCheck, artifact)
	verification, err := domain.NewVerificationReport(verificationCheck, domain.VerificationVerified, []domain.Evidence{verificationEvidence})
	if err != nil {
		t.Fatal(err)
	}
	staticCheck := check(t, "npm-static-archive", domain.ExecutionCompleted, "")
	staticEvidence := evidence(t, "static-evidence", staticCheck, artifact)
	dynamicEvidence := evidence(t, "dynamic-evidence", dynamic, artifact)
	findings := []domain.Finding{}
	if code != "" {
		finding, _ := domain.NewFinding(code, []domain.EvidenceID{dynamicEvidence.ID()})
		findings = []domain.Finding{finding}
	}
	dynamicReport, err := domain.NewInspectionReport(dynamic, findings, func() []domain.Evidence {
		if dynamic.Status() == domain.ExecutionCompleted {
			return []domain.Evidence{dynamicEvidence}
		}
		return nil
	}())
	if err != nil {
		t.Fatal(err)
	}
	staticReport, _ := domain.NewInspectionReport(staticCheck, nil, []domain.Evidence{staticEvidence})
	inspection, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{staticReport, dynamicReport})
	if err != nil {
		t.Fatal(err)
	}
	references := []domain.EvidenceReference{}
	for _, item := range append(verification.Evidence(), inspection.Evidence()...) {
		reference, _ := domain.NewEvidenceReference(item.ID(), "fixture:"+item.ID().String())
		references = append(references, reference)
	}
	runID, _ := domain.ParseRunID("run_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	requirements := []domain.CheckRequirement{}
	for index, name := range []string{"npm-integrity", "npm-static-archive", "npm-dynamic-lifecycle"} {
		id, _ := domain.NewCheckID(name)
		kind := domain.CheckInspection
		if index == 0 {
			kind = domain.CheckVerification
		}
		requirement, err := domain.NewCheckRequirement(id, kind, true, profile)
		if err != nil {
			t.Fatal(err)
		}
		requirements = append(requirements, requirement)
	}
	plan, err := domain.NewCheckPlan(runID, artifact, requirements)
	if err != nil {
		t.Fatal(err)
	}
	input, err := domain.NewPolicyInput(runID, artifact, verification, inspection, references, plan)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func check(t *testing.T, value string, status domain.ExecutionStatus, limitation string) domain.CheckExecution {
	t.Helper()
	id, _ := domain.NewCheckID(value)
	profile, _ := domain.CheckProfileDigest("m3-fixture-v1", "")
	execution, err := domain.NewCheckExecution(id, domain.CheckInspection, true, domain.CapabilitySupported, status, limitation, profile)
	if err != nil {
		t.Fatal(err)
	}
	return execution
}
func evidence(t *testing.T, value string, check domain.CheckExecution, artifact domain.AcquiredArtifact) domain.Evidence {
	t.Helper()
	id, _ := domain.NewEvidenceID(value)
	item, err := domain.NewEvidence(id, check.ID(), artifact.Identity(), artifact.Digest(), "fixture", "completed")
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func TestM3UsesStrongestFindingRegardlessOfReportOrder(t *testing.T) {
	for _, codes := range [][2]string{
		{"M3_NETWORK_ATTEMPT", "M3_HONEYTOKEN_ACCESS"},
		{"M3_UNEXPECTED_PROCESS", "M3_FILESYSTEM_VIOLATION"},
		{"M3_NETWORK_ATTEMPT", "UNKNOWN_SECURITY_FINDING"},
		{"M13_ADVISORY_REVIEW", "M13_ADVISORY_HIGH"},
		{"M13_ADVISORY_REVIEW", "M13_ADVISORY_MALICIOUS"},
		{"M13_ADVISORY_REVIEW", "M3_HONEYTOKEN_ACCESS"},
	} {
		for _, reverse := range []bool{false, true} {
			ordered := codes
			if reverse {
				ordered[0], ordered[1] = ordered[1], ordered[0]
			}
			t.Run(strings.Join(ordered[:], "-"), func(t *testing.T) {
				input := policyInput(t, check(t, "npm-dynamic-lifecycle", domain.ExecutionCompleted, ""), "")
				evidence := input.Inspection().Evidence()
				findings := []domain.Finding{}
				for _, code := range ordered {
					finding, err := domain.NewFinding(code, []domain.EvidenceID{evidence[1].ID()})
					if err != nil {
						t.Fatal(err)
					}
					findings = append(findings, finding)
				}
				executions := input.Inspection().Executions()
				static, err := domain.NewInspectionReport(executions[0], nil, evidence[:1])
				if err != nil {
					t.Fatal(err)
				}
				dynamic, err := domain.NewInspectionReport(executions[1], findings, evidence[1:])
				if err != nil {
					t.Fatal(err)
				}
				inspection, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{static, dynamic})
				if err != nil {
					t.Fatal(err)
				}
				selected, err := domain.NewPolicyInput(input.RunID(), input.Artifact(), input.Verification(), inspection, input.Evidence())
				if err != nil {
					t.Fatal(err)
				}
				decision, err := (M3{}).Evaluate(selected)
				if err != nil || decision.Decision() != domain.DecisionBlock {
					t.Fatalf("later block was hidden by finding order: decision=%s error=%v", decision.Decision(), err)
				}
			})
		}
	}
}

func TestM3RefusesLegacyUndeclaredCheckCoverage(t *testing.T) {
	input := policyInput(t, check(t, "npm-dynamic-lifecycle", domain.ExecutionCompleted, ""), "")
	legacy, err := domain.NewPolicyInput(input.RunID(), input.Artifact(), input.Verification(), input.Inspection(), input.Evidence())
	if err != nil {
		t.Fatal(err)
	}
	decision, err := (M3{}).Evaluate(legacy)
	if err != nil || decision.Decision() != domain.DecisionManualReview {
		t.Fatal("undeclared reports automatically allowed", err)
	}
}

func TestM6PreservesBlockPriorityAndRequiresDeclaredCoverage(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		input := policyInput(t, check(t, "npm-dynamic-lifecycle", domain.ExecutionCompleted, ""), "", "github-release")
		evidence := input.Inspection().Evidence()
		executions := input.Inspection().Executions()
		codes := []string{"M6_FORMAT_UNSUPPORTED", "M3_HONEYTOKEN_ACCESS"}
		if reverse {
			codes[0], codes[1] = codes[1], codes[0]
		}
		var findings []domain.Finding
		for _, code := range codes {
			finding, err := domain.NewFinding(code, []domain.EvidenceID{evidence[1].ID()})
			if err != nil {
				t.Fatal(err)
			}
			findings = append(findings, finding)
		}
		static, err := domain.NewInspectionReport(executions[0], nil, evidence[:1])
		if err != nil {
			t.Fatal(err)
		}
		dynamic, err := domain.NewInspectionReport(executions[1], findings, evidence[1:])
		if err != nil {
			t.Fatal(err)
		}
		inspection, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{static, dynamic})
		if err != nil {
			t.Fatal(err)
		}
		selected, err := domain.NewPolicyInput(input.RunID(), input.Artifact(), input.Verification(), inspection, input.Evidence(), input.CheckPlan())
		if err != nil {
			t.Fatal(err)
		}
		decision, err := (M6{}).Evaluate(selected)
		if err != nil || decision.Decision() != domain.DecisionBlock {
			t.Fatal("M6 format review hid later block", decision, err)
		}
		legacy, err := domain.NewPolicyInput(input.RunID(), input.Artifact(), input.Verification(), input.Inspection(), input.Evidence())
		if err != nil {
			t.Fatal(err)
		}
		decision, err = (M6{}).Evaluate(legacy)
		if err != nil || decision.Decision() != domain.DecisionManualReview {
			t.Fatal("M6 accepted undeclared legacy coverage", decision, err)
		}
	}
}

func TestM6KeepsExplicitBinaryAdvisoryLimitAndExistingRequiredChecks(t *testing.T) {
	for _, tc := range []struct {
		name     string
		required bool
		dynamic  domain.ExecutionStatus
		code     string
		want     domain.Decision
	}{
		{"bounded normal binary", false, domain.ExecutionCompleted, "", domain.DecisionAllow},
		{"required mapping unavailable", true, domain.ExecutionCompleted, "", domain.DecisionManualReview},
		{"required dynamic incomplete", false, domain.ExecutionIncomplete, "", domain.DecisionManualReview},
		{"existing blocker preserved", false, domain.ExecutionCompleted, "M3_HONEYTOKEN_ACCESS", domain.DecisionBlock},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limitation := ""
			if tc.dynamic != domain.ExecutionCompleted {
				limitation = "DYNAMIC_INCOMPLETE"
			}
			input := policyInput(t, check(t, "npm-dynamic-lifecycle", tc.dynamic, limitation), tc.code, "github-release")
			id, _ := domain.NewCheckID("dependency-advisory")
			profile, _ := domain.CheckProfileDigest("binary-advisory-test", "fixed source applicability")
			execution, err := domain.NewCheckExecution(id, domain.CheckInspection, tc.required, domain.CapabilityUnsupported, domain.ExecutionNotExecuted, "M13_ADVISORY_BINARY_MAPPING_UNSUPPORTED", profile)
			if err != nil {
				t.Fatal(err)
			}
			limit, err := domain.NewInspectionReport(execution, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			inspection, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{input.Inspection(), limit})
			if err != nil {
				t.Fatal(err)
			}
			requirement, err := domain.NewCheckRequirement(id, domain.CheckInspection, tc.required, profile)
			if err != nil {
				t.Fatal(err)
			}
			plan, err := domain.NewCheckPlan(input.RunID(), input.Artifact(), append(input.CheckPlan().Requirements(), requirement))
			if err != nil {
				t.Fatal(err)
			}
			selected, err := domain.NewPolicyInput(input.RunID(), input.Artifact(), input.Verification(), inspection, input.Evidence(), plan)
			if err != nil {
				t.Fatal(err)
			}
			decision, err := (M6{}).Evaluate(selected)
			if err != nil || decision.Decision() != tc.want {
				t.Fatal("binary limit changed required/blocking semantics", decision, err)
			}
		})
	}
}
