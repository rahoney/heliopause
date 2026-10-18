package pypi

import (
	"strings"
	"testing"
)

func TestUpgradeNodeSelectionRequiresFreshExactCandidateAfterChangedAndEmptyDelta(t *testing.T) {
	profile := mustPyTorchProfile(t, "cpu")
	r, _ := ParseUpgradeRequirementForSource("torch>=1,<3", profile.Source())
	input, err := NewUpgradeInput(r, []InstalledMetadataFile{upgradeMetadataFile("torch", "1.0+cpu", ""), upgradeMetadataFile("dependency", "1.0", "")}, pythonRuntimeVersionForTest)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"root-changed", "root-noop", "dependency-changed", "dependency-noop"} {
		t.Run(mode, func(t *testing.T) {
			requirement, source := "torch>=1,<3", profile.Source()
			version := "2.0+cpu"
			item := namedUpgradeCandidate(profile, version)
			if strings.HasPrefix(mode, "dependency") {
				requirement, source = "dependency>=1", PublicPyPIProfile().Source()
				version = "2.0"
				item = upgradeCandidate("dependency", version, true)
			}
			delta := upgradeReport(t, item)
			if strings.HasSuffix(mode, "noop") {
				delta = upgradeReport(t)
				version = "1.0+cpu"
				item = namedUpgradeCandidate(profile, version)
				if strings.HasPrefix(mode, "dependency") {
					version = "1.0"
					item = upgradeCandidate("dependency", version, true)
				}
			}
			selection, err := input.ParseSourcePinnedNodeSelection(requirement, source, nil, delta, pipRuntimeVersionForTest)
			if err != nil {
				t.Fatal(err)
			}
			project, _ := RequestedProject(selection.Reference())
			selectedVersion, _, _ := RequestedVersion(selection.Reference())
			if selectedVersion != version {
				t.Fatal("installed empty-delta version lost", selection.Reference())
			}
			nodeProfile, _ := ProfileForSource(source)
			fresh, err := parseReportCandidate(item, nodeProfile, pythonRuntimeVersionForTest, nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := selection.ValidateFreshCandidate(fresh); err != nil {
				t.Fatal(err)
			}
			wrong := fresh
			wrong.version = "9.0"
			if selection.ValidateFreshCandidate(wrong) == nil {
				t.Fatal("another exact version accepted", project)
			}
			if selection.ValidateFreshCandidate(fresh.WithPrimary(false)) == nil {
				t.Fatal("single-node report role was lost")
			}
			changedBytes := fresh
			changedBytes.sha256 = strings.Repeat("c", 64)
			err = selection.ValidateFreshCandidate(changedBytes)
			if strings.HasSuffix(mode, "changed") && err == nil {
				t.Fatal("changed candidate bytes drifted between selections")
			}
			if strings.HasSuffix(mode, "noop") && err != nil {
				t.Fatal("empty delta reused an old digest instead of requiring a fresh artifact", err)
			}
		})
	}
}

func TestUpgradeNodeSelectionKeepsVersionChoiceSeparateFromSupersededRootExtras(t *testing.T) {
	profile := mustPyTorchProfile(t, "cu132")
	r, _ := ParseUpgradeRequirementForSource("torch", profile.Source())
	input, err := NewUpgradeInput(r, []InstalledMetadataFile{upgradeMetadataFile("torch", "2.14.0+cu132", "Requires-Dist: cuda-toolkit[cublas,cudart]==13.2; platform_system == \"Linux\"\n"), upgradeMetadataFile("cuda-toolkit", "13.2", "")}, pythonRuntimeVersionForTest)
	if err != nil {
		t.Fatal(err)
	}
	request := `cuda-toolkit[cudart]==13.2; platform_system == "Linux"`
	selection, err := input.ParseSourcePinnedNodeSelection(request, PublicPyPIProfile().Source(), []string{"cudart"}, upgradeReport(t), pipRuntimeVersionForTest)
	if err != nil {
		t.Fatal("one-node version choice incorrectly applies old whole-root extras", err)
	}
	fresh, err := parseReportCandidate(upgradeCandidate("cuda-toolkit", "13.2", true, `nvidia-cudart>=13.2; extra == "cudart"`), PublicPyPIProfile(), pythonRuntimeVersionForTest, []string{"cudart"})
	if err != nil || selection.ValidateFreshCandidate(fresh) != nil {
		t.Fatal("fresh node extra context rejected", err)
	}
	fresh.requestedExtras = nil
	if selection.ValidateFreshCandidate(fresh) == nil {
		t.Fatal("fresh no-op candidate dropped its actual extra context")
	}
}

func TestUpgradeNodeSelectionRejectsSourceIntentRuntimeCardinalityAndMissingInstalledState(t *testing.T) {
	profile := mustPyTorchProfile(t, "cpu")
	r, _ := ParseUpgradeRequirementForSource("torch>=1,<3", profile.Source())
	input, _ := NewUpgradeInput(r, []InstalledMetadataFile{upgradeMetadataFile("torch", "1.0+cpu", "")}, pythonRuntimeVersionForTest)
	root := namedUpgradeCandidate(profile, "2.0+cpu")
	for _, row := range []struct {
		requirement string
		source      SourceProfile
		extras      []string
		body        []byte
	}{
		{"torch>=2,<3", profile, nil, upgradeReport(t, root)},
		{"torch>=1,<3", PublicPyPIProfile(), nil, upgradeReport(t, root)},
		{"dependency>=1", profile, nil, upgradeReport(t, upgradeCandidate("dependency", "2.0", true))},
		{"dependency>=1", PublicPyPIProfile(), nil, upgradeReport(t)},
		{"dependency>=1", PublicPyPIProfile(), nil, upgradeReport(t, upgradeCandidate("another", "2.0", true))},
		{"torch>=1,<3", profile, nil, upgradeReport(t, root, upgradeCandidate("dependency", "2.0", false))},
		{"torch>=1,<3", profile, nil, []byte(strings.Replace(string(upgradeReport(t, root)), `"sys_platform":"linux"`, `"sys_platform":"win32"`, 1))},
		{"cuda-toolkit[cudart]>=13; platform_system == \"Linux\"", PublicPyPIProfile(), []string{"cublas"}, upgradeReport(t)},
	} {
		if _, err := input.ParseSourcePinnedNodeSelection(row.requirement, row.source.Source(), row.extras, row.body, pipRuntimeVersionForTest); err == nil {
			t.Fatal("unsupported node selection accepted", row.requirement, row.source.Name())
		}
	}
	if (UpgradeNodeSelection{}).ValidateFreshCandidate(Candidate{}) == nil {
		t.Fatal("missing node selection accepted")
	}
}
