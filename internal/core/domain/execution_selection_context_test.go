package domain

import (
	"strings"
	"testing"
)

func selectionContextFixture(t *testing.T) ExecutionSelectionContextData {
	t.Helper()
	selection, entries := executionSelectionApprovalFixture(t)
	inspected, err := NewInspectedExecutionSelection(selection, entries)
	if err != nil {
		t.Fatal(err)
	}
	allow, _ := NewPolicyDecision(DecisionAllow, "fixture", 1, []string{"ALLOW"})
	approved, err := NewVerifiedExecutionSelection(inspected, allow)
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := NewSHA256Digest(strings.Repeat("d", 64))
	resolution, err := NewExecutionSelectionResolution(selection, "resolver", digest)
	if err != nil {
		t.Fatal(err)
	}
	op, _ := NewOperationID()
	var primary AcquiredArtifact
	for _, entry := range entries {
		if entry.Node() == selection.Primary() {
			primary = entry.Artifact()
		}
	}
	return ExecutionSelectionContextData{OperationID: op, Resolution: resolution, Set: approved, EntryArtifact: primary, Command: "console", EntryPath: "pkg.dist-info/entry_points.txt", EntryDigest: digest, EntrySize: 10, RuntimeIdentity: "runtime", Arguments: []string{"private", "", "a b"}}
}

func TestExecutionSelectionContextBindsWholeRootsPrimaryAndPrivateArguments(t *testing.T) {
	data := selectionContextFixture(t)
	current, err := NewExecutionSelectionContext(data)
	if err != nil || !current.Valid() {
		t.Fatal("complete current context rejected", err)
	}
	data.Arguments[0] = "changed"
	copyArgs := current.Arguments()
	copyArgs[0] = "also changed"
	if current.Arguments()[0] != "private" {
		t.Fatal("mutable argv")
	}
	data.Arguments = []string{"different"}
	other, err := NewExecutionSelectionContext(data)
	if err != nil || other.PolicyFingerprint() != current.PolicyFingerprint() || other.SourceFingerprint() != current.SourceFingerprint() {
		t.Fatal("private argv entered source or Policy identity", err)
	}
	for _, fault := range []string{"resolution", "approval", "foreign roots", "supplemental console", "member", "command", "runtime", "argv"} {
		t.Run(fault, func(t *testing.T) {
			bad := data
			switch fault {
			case "resolution":
				bad.Resolution = ExecutionSelectionResolution{}
			case "approval":
				bad.Set = VerifiedExecutionSelection{}
			case "foreign roots":
				roots := data.Resolution.Selection().RootGraphs()
				selection, e := NewExecutionDependencySelection(roots[1], roots[:1])
				if e != nil {
					t.Fatal(e)
				}
				bad.Resolution, _ = NewExecutionSelectionResolution(selection, "resolver", data.Resolution.LockfileDigest())
			case "supplemental console":
				for _, entry := range data.Set.Inspected().Inspections() {
					if entry.Node() != data.Resolution.Selection().Primary() {
						bad.EntryArtifact = entry.Artifact()
					}
				}
			case "member":
				bad.EntryPath = "../escape"
			case "command":
				bad.Command = "bad/command"
			case "runtime":
				bad.RuntimeIdentity = ""
			case "argv":
				bad.Arguments = []string{"bad\x00argument"}
			}
			if c, e := NewExecutionSelectionContext(bad); e == nil || c.Valid() {
				t.Fatal("substituted or incomplete context accepted")
			}
		})
	}
}
