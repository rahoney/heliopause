package domain

import "testing"

func executionSelectionApprovalFixture(t *testing.T) (ExecutionDependencySelection, []DependencyInspection) {
	t.Helper()
	// Reuse existing current-inspection facts, without using a project snapshot
	// or install target as the new selection's runtime or approval input.
	snapshot, inspections := projectApprovalFixture(t)
	var roots []LockedDependencyGraph
	for _, artifact := range snapshot.Dependencies() {
		id, err := ProjectDependencyNodeID(artifact.Identity())
		if err != nil {
			t.Fatal(err)
		}
		node, err := NewLockedDependency(id, DependencyPrimary, artifact)
		if err != nil {
			t.Fatal(err)
		}
		graph, err := NewLockedDependencyGraph([]LockedDependency{node}, nil)
		if err != nil {
			t.Fatal(err)
		}
		roots = append(roots, graph)
	}
	selection, err := NewExecutionDependencySelection(roots[0], roots[1:])
	if err != nil {
		t.Fatal(err)
	}
	return selection, inspections
}

func TestExecutionSelectionApprovalRequiresWholeCurrentRoots(t *testing.T) {
	selection, entries := executionSelectionApprovalFixture(t)
	inspected, err := NewInspectedExecutionSelection(selection, entries)
	if err != nil || !inspected.Valid() || len(inspected.Inspections()) != 2 {
		t.Fatal("complete independent-root inspection unavailable", err)
	}
	allow, _ := NewPolicyDecision(DecisionAllow, "fixture", 1, []string{"ALLOW"})
	approved, err := NewVerifiedExecutionSelection(inspected, allow)
	if err != nil || !approved.Valid() || len(approved.Inspected().Selection().RootGraphs()) != 2 {
		t.Fatal("whole current selection approval missing", err)
	}
	entries[0] = DependencyInspection{}
	owned := inspected.Inspections()
	owned[0] = DependencyInspection{}
	if !approved.Valid() || inspected.Inspections()[0].Node().String() == "" {
		t.Fatal("caller mutated approved whole-set facts")
	}
	if _, err := NewVerifiedExecutionSelection(InspectedExecutionSelection{}, allow); err == nil {
		t.Fatal("absent current inspections became approval")
	}
	block, _ := NewPolicyDecision(DecisionBlock, "fixture", 1, []string{"BLOCK"})
	if _, err := NewVerifiedExecutionSelection(inspected, block); err == nil {
		t.Fatal("whole BLOCK became approval")
	}
}

func TestExecutionSelectionApprovalRejectsCrossRootSubstitution(t *testing.T) {
	for _, fault := range []string{"missing supplemental", "duplicate node", "reused run", "foreign handle", "foreign subject", "entry blocked"} {
		t.Run(fault, func(t *testing.T) {
			selection, entries := executionSelectionApprovalFixture(t)
			switch fault {
			case "missing supplemental":
				entries = entries[:1]
			case "duplicate node":
				entries[1] = entries[0]
			case "reused run":
				entries[1].runID = entries[0].runID
				entries[1].artifact.handle = "intake:" + entries[1].runID.String() + ":module"
			case "foreign handle":
				entries[1].artifact.handle = entries[0].artifact.handle
			case "foreign subject":
				entries[1].artifact.identity = entries[0].artifact.identity
			case "entry blocked":
				entries[1].decision, _ = NewPolicyDecision(DecisionBlock, "fixture", 1, []string{"BLOCK"})
			}
			inspected, err := NewInspectedExecutionSelection(selection, entries)
			if fault != "entry blocked" {
				if err == nil || inspected.Valid() {
					t.Fatal("partial or substituted whole selection accepted")
				}
				return
			}
			if err != nil {
				t.Fatal("completed BLOCK facts lost", err)
			}
			allow, _ := NewPolicyDecision(DecisionAllow, "fixture", 1, []string{"ALLOW"})
			if _, err := NewVerifiedExecutionSelection(inspected, allow); err == nil {
				t.Fatal("unapproved supplemental root bypassed whole-set entry checks")
			}
		})
	}
}
