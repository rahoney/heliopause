package domain

import (
	"strings"
	"testing"
)

func TestProjectApprovalRequiresExactIndependentCompleteChecks(t *testing.T) {
	snapshot, entries := projectApprovalFixture(t)
	decision, _ := NewPolicyDecision(DecisionAllow, "fixture", 1, []string{"ALLOW"})
	good, err := NewInspectedProjectSet(snapshot, entries)
	if err != nil {
		t.Fatal(err)
	}
	approved, err := NewProjectVerifiedSet(good, decision)
	if err != nil || !approved.Valid() {
		t.Fatalf("approval: %v", err)
	}
	copyEntries := good.Inspections()
	copyEntries[0] = DependencyInspection{}
	if !good.Valid() || good.Inspections()[0].node.value == "" {
		t.Fatal("mutable inspection alias")
	}
	for _, name := range []string{"partial", "duplicate", "same-run", "foreign-run", "identity", "integrity", "missing-verifier", "duplicate-check"} {
		t.Run(name, func(t *testing.T) {
			changed := append([]DependencyInspection(nil), entries...)
			switch name {
			case "partial":
				changed = changed[:1]
			case "duplicate":
				changed[1] = changed[0]
			case "same-run":
				changed[1].runID = changed[0].runID
			case "foreign-run":
				changed[1].artifact.handle = changed[0].artifact.handle
			case "identity":
				changed[1].artifact.identity = changed[0].artifact.identity
			case "integrity":
				changed[1].artifact.declaredIntegrity = "forged"
			case "missing-verifier":
				changed[1].checks = changed[1].checks[1:]
			case "duplicate-check":
				changed[1].checks = append(changed[1].Checks(), changed[1].checks[0])
			}
			if _, err := NewInspectedProjectSet(snapshot, changed); err == nil {
				t.Fatal("invalid project coverage accepted")
			}
		})
	}
	for _, name := range []string{"block", "review", "incomplete", "unsupported"} {
		t.Run(name, func(t *testing.T) {
			changed := append([]DependencyInspection(nil), entries...)
			changed[1].checks = append([]CheckExecution(nil), changed[1].checks...)
			switch name {
			case "block":
				changed[1].decision, _ = NewPolicyDecision(DecisionBlock, "fixture", 1, []string{"BLOCK"})
			case "review":
				changed[1].decision, _ = NewPolicyDecision(DecisionManualReview, "fixture", 1, []string{"REVIEW"})
			case "incomplete":
				changed[1].checks[0], err = NewCheckExecution(changed[1].checks[0].id, CheckVerification, true, CapabilitySupported, ExecutionIncomplete, "FIXTURE_UNAVAILABLE")
			case "unsupported":
				changed[1].checks[0], err = NewCheckExecution(changed[1].checks[0].id, CheckVerification, true, CapabilityUnsupported, ExecutionNotExecuted, "FIXTURE_UNSUPPORTED")
			}
			if err != nil {
				t.Fatal(err)
			}
			set, err := NewInspectedProjectSet(snapshot, changed)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := NewProjectVerifiedSet(set, decision); err == nil {
				t.Fatal("unapproved or incomplete project accepted")
			}
		})
	}
}

func projectApprovalFixture(t *testing.T) (ProjectDependencySnapshot, []DependencyInspection) {
	t.Helper()
	source, _ := NewSourceID("go-proxy")
	target, _ := NewInstallTarget(t.TempDir())
	ctx, _ := NewInstallContext(target)
	hash, _ := NewSHA256Digest(strings.Repeat("a", 64))
	control, _ := NewProjectControlDigest("go.mod", hash)
	var artifacts []ResolvedArtifact
	var inspections []DependencyInspection
	for _, name := range []string{"example.com/first", "example.com/second"} {
		identity, _ := NewResolvedArtifactIdentity(source, name, "v1.0.0", "module")
		resolved, _ := NewResolvedArtifact(identity, "https://proxy.golang.org/"+name+"/@v/v1.0.0.zip", "fixture-integrity")
		artifacts = append(artifacts, resolved)
		node, err := ProjectDependencyNodeID(identity)
		if err != nil {
			t.Fatal(err)
		}
		run, err := NewRunID()
		if err != nil {
			t.Fatal(err)
		}
		acquired, err := NewAcquiredArtifactWithDeclaredIntegrity(identity, hash, "intake:"+run.String()+":module", 1, resolved.DeclaredIntegrity())
		if err != nil {
			t.Fatal(err)
		}
		vID, _ := NewCheckID("verification")
		iID, _ := NewCheckID("inspection")
		v, _ := NewCheckExecution(vID, CheckVerification, true, CapabilitySupported, ExecutionCompleted, "")
		i, _ := NewCheckExecution(iID, CheckInspection, true, CapabilitySupported, ExecutionCompleted, "")
		eID, err := NewEvidenceID("fixture-" + node.String()[:24])
		if err != nil {
			t.Fatal(err)
		}
		e, _ := NewEvidenceReference(eID, "fixture:"+eID.String())
		decision, _ := NewPolicyDecision(DecisionAllow, "fixture", 1, []string{"ALLOW"})
		inspection, err := NewDependencyInspection(node, run, acquired, []CheckExecution{v, i}, []EvidenceReference{e}, decision)
		if err != nil {
			t.Fatal(err)
		}
		inspections = append(inspections, inspection)
	}
	snapshot, err := NewProjectDependencySnapshot(ctx, source, []ProjectControlDigest{control}, artifacts, hash)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot, inspections
}
