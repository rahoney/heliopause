package pypi

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestUVRunRequirementsBoundCanonicalIndependentRoots(t *testing.T) {
	primary, _ := ParseUVRunRequirement("root[Feature]@1.0.0")
	a, _ := ParseUVRunRequirement("second@1.0.0")
	b, _ := ParseUVRunRequirement("first")
	input := []UVRunRequirement{a, b}
	request, err := NewUVRunRequirements(primary, input)
	if err != nil || !request.Valid() || strings.Join(request.Requirements(), ";") != "root[feature]==1.0.0;first;second==1.0.0" {
		t.Fatal("joint canonical request unavailable", err)
	}
	input[0] = UVRunRequirement{}
	copyRoots := request.Supplemental()
	copyRoots[0] = UVRunRequirement{}
	copyText := request.Requirements()
	copyText[0] = "mutated"
	if !request.Supplemental()[0].Valid() || request.Primary().Base().Project() != "root" {
		t.Fatal("mutable root input")
	}
	for _, roots := range [][]UVRunRequirement{nil, {a, a}, {primary}, {UVRunRequirement{}}, make([]UVRunRequirement, domain.MaximumExecutionArguments)} {
		if r, e := NewUVRunRequirements(primary, roots); e == nil || r.Valid() {
			t.Fatal("missing/duplicate/unbounded supplemental request accepted")
		}
	}
}

func TestUVRunSelectionBindsRealRootGraphsWithoutInventedEdges(t *testing.T) {
	for _, row := range []struct {
		name, rootRequires, childRequires, childRequest string
		edges                                           int
	}{
		{"independent", "", "", "child==1.0.0", 0},
		{"shared dependency", "child>=1", "", "child==1.0.0", 1},
		{"supplemental imports primary", "", "haa-tool-fixture>=1", "child==1.0.0", 1},
		{"supplemental extras", "", "haa-tool-fixture>=1; extra == 'feature'\nProvides-Extra: feature", "child[feature]==1.0.0", 1},
	} {
		t.Run(row.name, func(t *testing.T) {
			lock, inputs, pages := toolTestSelection(t, toolTestWheel(t, "haa-tool-fixture", row.rootRequires, toolTestEntrypoints), toolTestWheel(t, "child", row.childRequires, ""))
			primary, _ := ParseUVRunRequirement("haa-tool-fixture==1.0.0")
			child, _ := ParseUVRunRequirement(row.childRequest)
			request, _ := NewUVRunRequirements(primary, []UVRunRequirement{child})
			selection, err := lock.BindUVRunSelection(request, inputs, pages, WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits())
			if err != nil || !selection.Valid() || len(selection.RootGraphs()) != 2 || len(selection.Nodes()) != 2 || len(selection.Edges()) != row.edges {
				t.Fatal("joint whole selection lost real roots or edges", err)
			}
			if row.edges == 0 {
				if _, err := lock.BindPipxQueryWheels(inputs, pages, WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits()); err == nil {
					t.Fatal("old single-root graph accepted disconnected selection")
				}
			}
		})
	}
}

func TestUVRunSelectionRejectsPartialForeignOrUnrequestedSource(t *testing.T) {
	for _, fault := range []string{"missing wheel", "undeclared wheel", "absent root", "wrong version", "unknown extra", "unsafe startup", "foreign source", "zero request"} {
		t.Run(fault, func(t *testing.T) {
			var extra []wheelTestEntry
			if fault == "unsafe startup" {
				extra = []wheelTestEntry{{name: "startup.pth", body: []byte("import os\n")}}
			}
			lock, inputs, pages := toolTestSelection(t, toolTestWheel(t, "haa-tool-fixture", "", toolTestEntrypoints), toolTestWheel(t, "child", "", "", extra...))
			primary, _ := ParseUVRunRequirement("haa-tool-fixture==1.0.0")
			text := "child==1.0.0"
			switch fault {
			case "missing wheel":
				inputs = inputs[:1]
			case "undeclared wheel", "absent root":
				text = "missing==1.0.0"
			case "wrong version":
				text = "child==2.0.0"
			case "unknown extra":
				text = "child[unknown]==1.0.0"
			case "foreign source":
				pages[1].files[0].yanked = true
			}
			child, _ := ParseUVRunRequirement(text)
			request, _ := NewUVRunRequirements(primary, []UVRunRequirement{child})
			if fault == "zero request" {
				request = UVRunRequirements{}
			}
			if s, err := lock.BindUVRunSelection(request, inputs, pages, WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits()); err == nil || s.Valid() {
				t.Fatal("partial, conflicting or unsafe joint source accepted")
			}
		})
	}
}
