package pypi

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func sourcePinnedControlsFixture(t *testing.T, mode string) (UpgradeInput, []SourcePinnedUpgradeRecord, []SourcePinnedUpgradeRecord) {
	t.Helper()
	profile := mustPyTorchProfile(t, "cpu")
	r, _ := ParseUpgradeRequirementForSource("torch>=1,<3", profile.Source())
	oldRoot, newDep, dependencyRequest := "1.0+cpu", "2.0", "dependency>=2"
	if mode == "retained" {
		newDep, dependencyRequest = "1.0", "dependency>=1"
	}
	if mode == "transitive-only" {
		oldRoot = "2.0+cpu"
	}
	files := []InstalledMetadataFile{upgradeMetadataFile("torch", oldRoot, "Requires-Dist: "+dependencyRequest+"\n"), upgradeMetadataFile("dependency", "1.0", ""), upgradeMetadataFile("retained", "9.0", "Requires-Dist: dependency<3\n")}
	input, err := NewUpgradeInput(r, files, pythonRuntimeVersionForTest)
	if err != nil {
		t.Fatal(err)
	}
	root := namedUpgradeCandidate(profile, "2.0+cpu", dependencyRequest)
	dep := upgradeCandidate("dependency", newDep, true)
	rootDelta := upgradeReport(t, root)
	if mode == "transitive-only" {
		rootDelta = upgradeReport(t)
	}
	depDelta := upgradeReport(t, dep)
	if mode == "retained" {
		depDelta = upgradeReport(t)
	}
	first := []SourcePinnedUpgradeRecord{{SourceID: profile.Source().String(), Requirement: r.Requirement(), SelectionReport: rootDelta, CandidateReport: upgradeReport(t, root)}, {SourceID: PublicPyPIProfile().Source().String(), Requirement: dependencyRequest, SelectionReport: depDelta, CandidateReport: upgradeReport(t, dep)}}
	final := []SourcePinnedUpgradeRecord{{SourceID: profile.Source().String(), Requirement: "torch==2.0+cpu", CandidateReport: upgradeReport(t, root)}, {SourceID: PublicPyPIProfile().Source().String(), Requirement: "dependency==" + newDep, CandidateReport: upgradeReport(t, dep)}}
	return input, first, final
}

func TestUpgradeSourceControlsBindCompleteFreshSelectionAndRetainedState(t *testing.T) {
	for _, mode := range []string{"changed", "retained", "transitive-only"} {
		t.Run(mode, func(t *testing.T) {
			input, first, final := sourcePinnedControlsFixture(t, mode)
			selection, err := input.BuildSourcePinnedUpgrade(first, final, pipRuntimeVersionForTest)
			if err != nil || len(selection.Graph().Nodes()) != 2 || len(selection.Controls()) != 4 || selection.Digest().String() == "" {
				t.Fatal(selection, err)
			}
			parsed, err := input.ParseSourcePinnedUpgrade(selection.Controls(), pipRuntimeVersionForTest)
			if err != nil || parsed.Digest() != selection.Digest() || len(parsed.Graph().Nodes()) != 2 {
				t.Fatal("frozen source selection roundtrip failed", err)
			}
			for _, control := range parsed.Controls() {
				if control.Name() == SourcePinnedUpgradeConstraintsControlName && !bytes.Contains(control.Body(), []byte("dependency<3\n")) {
					t.Fatal("actual retained constraint lost")
				}
			}
			for _, node := range parsed.Graph().Nodes() {
				if node.Artifact().Identity().Name() == "retained" {
					t.Fatal("unrelated installed package became a graph root")
				}
			}
		})
	}
}

func TestUpgradeSourceControlsRejectOmittedChangedOrSubstitutedReports(t *testing.T) {
	for _, mode := range []string{"first-source", "root-intent", "hidden-first-node", "missing-first", "missing-final", "final-selects-again", "final-source", "final-version", "final-bytes", "eager-compatible"} {
		t.Run(mode, func(t *testing.T) {
			fixture := "changed"
			if mode == "eager-compatible" {
				fixture = "retained"
			}
			input, first, final := sourcePinnedControlsFixture(t, fixture)
			switch mode {
			case "first-source":
				first[0].SourceID = "pypi"
			case "root-intent":
				first[0].Requirement = "torch>=0,<3"
			case "hidden-first-node":
				first[0].SelectionReport = upgradeReport(t, namedUpgradeCandidate(mustPyTorchProfile(t, "cpu"), "2.0+cpu"), upgradeCandidate("unexpected", "1.0", false))
			case "missing-first":
				first = first[:1]
			case "missing-final":
				final = final[:1]
			case "final-selects-again":
				final[1].SelectionReport = upgradeReport(t)
			case "final-source":
				final[1].SourceID = mustPyTorchProfile(t, "cpu").Source().String()
			case "final-version":
				final[1].Requirement = "dependency==9.0"
			case "final-bytes":
				final[1].CandidateReport = []byte(strings.ReplaceAll(string(final[1].CandidateReport), strings.Repeat("b", 64), strings.Repeat("c", 64)))
			case "eager-compatible":
				dep := upgradeCandidate("dependency", "2.0", true)
				first[1].Requirement = "dependency>=2"
				first[1].SelectionReport = upgradeReport(t, dep)
				first[1].CandidateReport = upgradeReport(t, dep)
				final[1].Requirement = "dependency==2.0"
				final[1].CandidateReport = upgradeReport(t, dep)
			}
			if _, err := input.BuildSourcePinnedUpgrade(first, final, pipRuntimeVersionForTest); err == nil {
				t.Fatal("source selection substitution accepted", mode)
			}
		})
	}
}

func TestUpgradeSourceControlParserRejectsAlteredConstraintInputAndNoncanonicalWire(t *testing.T) {
	input, first, final := sourcePinnedControlsFixture(t, "changed")
	selection, err := input.BuildSourcePinnedUpgrade(first, final, pipRuntimeVersionForTest)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"constraints", "input", "duplicate", "unknown-json", "trailing-json", "missing-controls"} {
		controls := selection.Controls()
		switch mode {
		case "constraints":
			controls[3], _ = domain.NewProjectControlFile(SourcePinnedUpgradeConstraintsControlName, []byte("dependency==9.0\n"), true)
		case "input":
			controls[0], _ = domain.NewProjectControlFile(UpgradeInputControlName, []byte("{}"), true)
		case "duplicate":
			controls[3] = controls[0]
		case "unknown-json":
			controls[1], _ = domain.NewProjectControlFile(SourcePinnedUpgradeSelectionControlName, bytes.Replace(controls[1].Body(), []byte(`{"version":1`), []byte(`{"unknown":true,"version":1`), 1), true)
		case "trailing-json":
			controls[1], _ = domain.NewProjectControlFile(SourcePinnedUpgradeSelectionControlName, append(controls[1].Body(), []byte(" {}")...), true)
		case "missing-controls":
			controls = controls[:3]
		}
		if _, err := input.ParseSourcePinnedUpgrade(controls, pipRuntimeVersionForTest); err == nil {
			t.Fatal("altered source control accepted", mode)
		}
	}
	if _, err := sourcePinnedUpgradeControl(SourcePinnedUpgradeSelectionControlName, make([]SourcePinnedUpgradeRecord, 65)); err == nil {
		t.Fatal("report count bound bypassed")
	}
	large := first[0]
	large.CandidateReport = bytes.Repeat([]byte("x"), 3<<20)
	if _, err := sourcePinnedUpgradeControl(SourcePinnedUpgradeSelectionControlName, []SourcePinnedUpgradeRecord{large, large}); err == nil {
		t.Fatal("base64 representation escaped the control bound")
	}
}
