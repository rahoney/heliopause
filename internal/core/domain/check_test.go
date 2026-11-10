package domain_test

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestCheckExecutionKeepsStatusAxesSeparate(t *testing.T) {
	t.Parallel()

	checkID := mustCheckID(t, "fixture-static")
	tests := []struct {
		name       string
		capability domain.Capability
		status     domain.ExecutionStatus
		limitation string
		valid      bool
	}{
		{name: "completed", capability: domain.CapabilitySupported, status: domain.ExecutionCompleted, valid: true},
		{name: "unsupported", capability: domain.CapabilityUnsupported, status: domain.ExecutionNotExecuted, limitation: "CAPABILITY_UNSUPPORTED", valid: true},
		{name: "unavailable", capability: domain.CapabilitySupported, status: domain.ExecutionUnavailable, limitation: "TOOL_UNAVAILABLE", valid: true},
		{name: "unsupported completed", capability: domain.CapabilityUnsupported, status: domain.ExecutionCompleted},
		{name: "missing limitation", capability: domain.CapabilitySupported, status: domain.ExecutionFailed},
		{name: "completed limitation", capability: domain.CapabilitySupported, status: domain.ExecutionCompleted, limitation: "UNEXPECTED"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewCheckExecution(checkID, domain.CheckInspection, true, test.capability, test.status, test.limitation)
			if (err == nil) != test.valid {
				t.Fatalf("NewCheckExecution() error = %v, valid = %v", err, test.valid)
			}
		})
	}
}

func TestReportsRequireInterpretableBoundEvidence(t *testing.T) {
	t.Parallel()

	identity, digest := evidenceSubject(t)
	verificationID := mustCheckID(t, "fixture-integrity")
	verificationExecution, err := domain.NewCheckExecution(verificationID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	if err != nil {
		t.Fatal(err)
	}
	verificationEvidence := mustEvidence(t, "verification-evidence", verificationID, identity, digest)
	if _, err := domain.NewVerificationReport(verificationExecution, domain.VerificationVerified, []domain.Evidence{verificationEvidence}); err != nil {
		t.Fatalf("NewVerificationReport() error = %v", err)
	}
	if _, err := domain.NewVerificationReport(verificationExecution, "", []domain.Evidence{verificationEvidence}); err == nil {
		t.Fatal("completed verification without outcome accepted")
	}
	if _, err := domain.NewVerificationReport(verificationExecution, domain.VerificationVerified, nil); err == nil {
		t.Fatal("completed verification without Evidence accepted")
	}

	inspectionID := mustCheckID(t, "fixture-static")
	inspectionExecution, err := domain.NewCheckExecution(inspectionID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	if err != nil {
		t.Fatal(err)
	}
	inspectionEvidence := mustEvidence(t, "inspection-evidence", inspectionID, identity, digest)
	finding, err := domain.NewFinding("M1_BLOCK_FINDING", []domain.EvidenceID{inspectionEvidence.ID()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := domain.NewInspectionReport(inspectionExecution, []domain.Finding{finding}, []domain.Evidence{inspectionEvidence}); err != nil {
		t.Fatalf("NewInspectionReport() error = %v", err)
	}
	foreignEvidence := mustEvidence(t, "foreign-evidence", inspectionID, identity, digest)
	foreignFinding, err := domain.NewFinding("M1_BLOCK_FINDING", []domain.EvidenceID{foreignEvidence.ID()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := domain.NewInspectionReport(inspectionExecution, []domain.Finding{foreignFinding}, []domain.Evidence{inspectionEvidence}); err == nil {
		t.Fatal("Finding with foreign Evidence accepted")
	}
}

func TestEvidenceRejectsObviousSensitiveOrUnboundedSummary(t *testing.T) {
	t.Parallel()

	identity, digest := evidenceSubject(t)
	checkID := mustCheckID(t, "fixture-static")
	id, err := domain.NewEvidenceID("sensitive-evidence")
	if err != nil {
		t.Fatal(err)
	}
	for _, summary := range []string{
		"token=secret-value",
		"read /Users/alice/project/file",
		strings.Repeat("x", 1025),
		"line one\nline two",
	} {
		if _, err := domain.NewEvidence(id, checkID, identity, digest, "static-inspection", summary); err == nil {
			t.Fatalf("NewEvidence(%q) error = nil", summary)
		}
	}
}

func TestInspectionDiagnosticVocabularyIsBounded(t *testing.T) {
	t.Parallel()

	valid := []struct {
		cause domain.InspectionDiagnosticCause
		stage domain.InspectionDiagnosticStage
	}{
		{domain.InspectionDiagnosticStaticFinding, ""},
		{domain.InspectionDiagnosticStaticIncomplete, ""},
		{domain.InspectionDiagnosticSdistDynamicUnavailable, ""},
		{domain.InspectionDiagnosticResourceExceeded, domain.InspectionDiagnosticStageGraphUncompressed},
		{domain.InspectionDiagnosticInvalidWheel, domain.InspectionDiagnosticStageMetadataIdentity},
	}
	for _, test := range valid {
		if _, err := domain.NewInspectionDiagnostic(test.cause, test.stage); err != nil {
			t.Fatalf("NewInspectionDiagnostic(%q, %q) error = %v", test.cause, test.stage, err)
		}
	}
	invalid := []struct {
		cause domain.InspectionDiagnosticCause
		stage domain.InspectionDiagnosticStage
	}{
		{"RAW_PARSER_ERROR", ""},
		{domain.InspectionDiagnosticInvalidWheel, "RAW_ERROR"},
		{domain.InspectionDiagnosticResourceExceeded, domain.InspectionDiagnosticStageMetadataIdentity},
		{domain.InspectionDiagnosticStaticFinding, domain.InspectionDiagnosticStageOther},
	}
	for _, test := range invalid {
		if _, err := domain.NewInspectionDiagnostic(test.cause, test.stage); err == nil {
			t.Fatalf("NewInspectionDiagnostic(%q, %q) accepted", test.cause, test.stage)
		}
	}
}

func mustCheckID(t *testing.T, value string) domain.CheckID {
	t.Helper()
	id, err := domain.NewCheckID(value)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func evidenceSubject(t *testing.T) (domain.ResolvedArtifactIdentity, domain.ContentDigest) {
	t.Helper()
	source := mustSource(t, "fixture")
	identity := mustIdentity(t, source, "safe", "1.0.0", "default")
	digest, err := domain.NewSHA256Digest(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	return identity, digest
}

func mustEvidence(t *testing.T, idValue string, checkID domain.CheckID, identity domain.ResolvedArtifactIdentity, digest domain.ContentDigest) domain.Evidence {
	t.Helper()
	id, err := domain.NewEvidenceID(idValue)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := domain.NewEvidence(id, checkID, identity, digest, "normalized-fact", "Synthetic normalized evidence for the exact fixture subject.")
	if err != nil {
		t.Fatal(err)
	}
	return evidence
}

func TestCompositeInspectionCoverageBoundaries(t *testing.T) {
	identity, digest := evidenceSubject(t)
	report := func(name, limitation, code string) domain.InspectionReport {
		t.Helper()
		id := mustCheckID(t, name)
		status := domain.ExecutionCompleted
		if limitation != "" {
			status = domain.ExecutionIncomplete
		}
		execution, err := domain.NewCheckExecution(id, domain.CheckInspection, true, domain.CapabilitySupported, status, limitation)
		if err != nil {
			t.Fatal(err)
		}
		var evidence []domain.Evidence
		var findings []domain.Finding
		if status == domain.ExecutionCompleted {
			item := mustEvidence(t, name+"-evidence", id, identity, digest)
			evidence = []domain.Evidence{item}
			if code != "" {
				finding, err := domain.NewFinding(code, []domain.EvidenceID{item.ID()})
				if err != nil {
					t.Fatal(err)
				}
				findings = []domain.Finding{finding}
			}
		}
		result, err := domain.NewInspectionReport(execution, findings, evidence)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	first := report("first", "", "")
	second := report("second", "", "")
	third := report("third", "", "")
	t.Run("duplicate check", func(t *testing.T) {
		if _, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{first, first}); err == nil {
			t.Fatal("duplicate execution accepted")
		}
	})
	t.Run("nested checks", func(t *testing.T) {
		inner, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{first, second})
		if err != nil {
			t.Fatal(err)
		}
		outer, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{inner, third})
		if err != nil || len(outer.Executions()) != 3 || len(outer.Evidence()) != 3 {
			t.Fatalf("nested required check lost: checks=%d error=%v", len(outer.Executions()), err)
		}
	})
	t.Run("partial retains confirmed finding", func(t *testing.T) {
		incomplete := report("not-finished", "INSPECTION_INCOMPLETE", "")
		blocked := report("confirmed", "", "M3_HONEYTOKEN_ACCESS")
		inner, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{incomplete, blocked})
		if err != nil {
			t.Fatal(err)
		}
		outer, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{inner, third})
		if err != nil || len(outer.Executions()) != 3 || len(outer.Findings()) != 1 {
			t.Fatalf("confirmed violation lost next to incomplete check: %v", err)
		}
	})
}

func TestCheckPlanFreezesExactRunSubjectAndProfileMembership(t *testing.T) {
	identity, digest := evidenceSubject(t)
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "fixture:subject", 1)
	if err != nil {
		t.Fatal(err)
	}
	run, err := domain.ParseRunID("run_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	profile, _ := domain.NewSHA256Digest(strings.Repeat("b", 64))
	foreignProfile, _ := domain.NewSHA256Digest(strings.Repeat("c", 64))
	ids := []domain.CheckID{mustCheckID(t, "planned-integrity"), mustCheckID(t, "planned-static"), mustCheckID(t, "planned-dynamic")}
	requirements := []domain.CheckRequirement{}
	checks := []domain.CheckExecution{}
	for index, id := range ids {
		kind := domain.CheckInspection
		if index == 0 {
			kind = domain.CheckVerification
		}
		requirement, err := domain.NewCheckRequirement(id, kind, true, profile)
		if err != nil {
			t.Fatal(err)
		}
		requirements = append(requirements, requirement)
		execution, err := domain.NewCheckExecution(id, kind, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", profile)
		if err != nil {
			t.Fatal(err)
		}
		checks = append(checks, execution)
	}
	plan, err := domain.NewCheckPlan(run, artifact, requirements)
	if err != nil {
		t.Fatal(err)
	}
	optionalID, _ := domain.NewCheckID("optional-observation")
	optional, _ := domain.NewCheckRequirement(optionalID, domain.CheckInspection, false, profile)
	optionalPlan, err := domain.NewCheckPlan(run, artifact, append(append([]domain.CheckRequirement(nil), requirements...), optional))
	if err != nil {
		t.Fatal(err)
	}
	if err := optionalPlan.Validate(run, artifact, checks); err != nil {
		t.Fatal("absent declared optional fact became required", err)
	}

	requirements[0] = domain.CheckRequirement{}
	copied := plan.Requirements()
	copied[1] = domain.CheckRequirement{}
	if err := plan.Validate(run, artifact, checks); err != nil {
		t.Fatal("declaration mutated", err)
	}
	for _, kind := range []string{"missing", "duplicate", "extra", "weakened-required", "wrong-kind", "wrong-profile", "unbound-legacy"} {
		t.Run(kind, func(t *testing.T) {
			actual := append([]domain.CheckExecution(nil), checks...)
			switch kind {
			case "missing":
				actual = actual[:2]
			case "duplicate":
				actual[2] = actual[1]
			case "extra":
				actual = append(actual, actual[0])
			case "weakened-required":
				actual[2], err = domain.NewCheckExecution(ids[2], domain.CheckInspection, false, domain.CapabilitySupported, domain.ExecutionCompleted, "", profile)
			case "wrong-kind":
				actual[2], err = domain.NewCheckExecution(ids[2], domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", profile)
			case "wrong-profile":
				actual[2], err = domain.NewCheckExecution(ids[2], domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", foreignProfile)
			case "unbound-legacy":
				actual[2], err = domain.NewCheckExecution(ids[2], domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := plan.Validate(run, artifact, actual); err == nil {
				t.Fatal("inexact membership accepted")
			}
		})
	}
	foreignRun, _ := domain.ParseRunID("run_bbbbbbbbbbbbbbbbbbbbbbbbbb")
	if err := plan.Validate(foreignRun, artifact, checks); err == nil {
		t.Fatal("foreign Run accepted")
	}
	foreignArtifact, _ := domain.NewAcquiredArtifact(identity, foreignProfile, "fixture:foreign", 1)
	if err := plan.Validate(run, foreignArtifact, checks); err == nil {
		t.Fatal("foreign digest accepted")
	}
	incomplete := append([]domain.CheckExecution(nil), checks...)
	incomplete[2], err = domain.NewCheckExecution(ids[2], domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionIncomplete, "OBSERVER_INCOMPLETE", profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Validate(run, artifact, incomplete); err != nil {
		t.Fatal("membership must preserve failed actual status for Policy", err)
	}
	// PolicyInput must enforce the declared set using the same reports and evidence.
	verificationEvidence := mustEvidence(t, "planned-integrity-evidence", ids[0], identity, digest)
	verification, err := domain.NewVerificationReport(checks[0], domain.VerificationVerified, []domain.Evidence{verificationEvidence})
	if err != nil {
		t.Fatal(err)
	}
	var reports []domain.InspectionReport
	evidence := []domain.Evidence{verificationEvidence}
	for index := 1; index < len(checks); index++ {
		item := mustEvidence(t, ids[index].String()+"-evidence", ids[index], identity, digest)
		evidence = append(evidence, item)
		report, err := domain.NewInspectionReport(checks[index], nil, []domain.Evidence{item})
		if err != nil {
			t.Fatal(err)
		}
		reports = append(reports, report)
	}
	inspection, err := domain.NewCompositeInspectionReport(reports)
	if err != nil {
		t.Fatal(err)
	}
	var references []domain.EvidenceReference
	for _, item := range evidence {
		reference, err := domain.NewEvidenceReference(item.ID(), "fixture:"+item.ID().String())
		if err != nil {
			t.Fatal(err)
		}
		references = append(references, reference)
	}
	input, err := domain.NewPolicyInput(run, artifact, verification, inspection, references, plan)
	if err != nil || !input.CheckPlan().Valid() {
		t.Fatal("frozen plan not carried to Policy", err)
	}
	partial, err := domain.NewInspectionReport(checks[1], nil, evidence[1:2])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := domain.NewPolicyInput(run, artifact, verification, partial, references[:2], plan); err == nil {
		t.Fatal("missing dynamic check reached Policy input")
	}
	if _, err := domain.NewCheckPlan(run, artifact, append(plan.Requirements(), plan.Requirements()[0])); err == nil {
		t.Fatal("duplicate declaration accepted")
	}
}
