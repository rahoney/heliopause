package pypi

import (
	"slices"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestUpgradeCUDARetainedExtraIntentCanBeRepresentedWithoutLoss(t *testing.T) {
	profile := mustPyTorchProfile(t, "cu132")
	requirement, err := ParseUpgradeRequirementForSource("torch", profile.Source())
	if err != nil {
		t.Fatal(err)
	}
	files := []InstalledMetadataFile{
		upgradeMetadataFile("torch", "2.14.0+cu132", "Requires-Dist: cuda-toolkit[cublas,cudart]==13.2; platform_system == \"Linux\"\n"),
		upgradeMetadataFile("cuda-toolkit", "13.2", "Requires-Dist: nvidia-cublas>=13.2; extra == \"cublas\"\nRequires-Dist: nvidia-cudart>=13.2; extra == \"cudart\"\n"),
		upgradeMetadataFile("nvidia-cublas", "13.2", ""),
		upgradeMetadataFile("nvidia-cudart", "13.2", ""),
	}
	if _, err := NewUpgradeInput(requirement, files, pythonRuntimeVersionForTest); err != nil {
		t.Fatal("supported retained CUDA extra intent has no lossless update representation", err)
	}
}

func TestUpgradeCUDARetainsTypedExtrasAndConditionalConstraints(t *testing.T) {
	for _, mode := range []string{"unchanged", "changed-parent", "changed-toolkit"} {
		t.Run(mode, func(t *testing.T) {
			profile := mustPyTorchProfile(t, "cu132")
			r, _ := ParseUpgradeRequirementForSource("torch", profile.Source())
			files := []InstalledMetadataFile{
				upgradeMetadataFile("torch", "2.14.0+cu132", "Requires-Dist: cuda-toolkit[cublas,cudart]==13.2; platform_system == \"Linux\"\n"),
				upgradeMetadataFile("cuda-toolkit", "13.2", "Requires-Dist: nvidia-cublas>=13.2; extra == \"cublas\"\nRequires-Dist: nvidia-cudart>=13.2; extra == \"cudart\"\n"),
				upgradeMetadataFile("nvidia-cublas", "13.2", ""), upgradeMetadataFile("nvidia-cudart", "13.2", "")}
			input, err := NewUpgradeInput(r, files, pythonRuntimeVersionForTest)
			if err != nil {
				t.Fatal(err)
			}
			rootVersion, toolkitVersion := "2.14.0+cu132", "13.2"
			extras := []string{"cublas", "cudart"}
			if mode != "unchanged" {
				rootVersion = "2.15.0+cu132"
				extras = []string{"cudart"}
			}
			if mode == "changed-toolkit" {
				toolkitVersion = "13.3"
			}
			root, err := parseReportCandidate(namedUpgradeCandidate(profile, rootVersion, `cuda-toolkit[`+strings.Join(extras, ",")+`]==`+toolkitVersion+`; platform_system == "Linux"`), profile, pythonRuntimeVersionForTest, nil)
			if err != nil {
				t.Fatal(err)
			}
			wanted, err := input.RequiredUpgradeExtras("cuda-toolkit", []Candidate{root})
			if err != nil || !slices.Equal(wanted, extras) {
				t.Fatal("retained/superseded extra intent lost", wanted, err)
			}
			wanted[0] = "mutated"
			again, _ := input.RequiredUpgradeExtras("cuda-toolkit", []Candidate{root})
			if !slices.Equal(again, extras) {
				t.Fatal("extra obligations are mutable")
			}
			toolkit, err := parseReportCandidate(upgradeCandidate("cuda-toolkit", toolkitVersion, false, `nvidia-cublas>=`+toolkitVersion+`; extra == "cublas"`, `nvidia-cudart>=`+toolkitVersion+`; extra == "cudart"`), profile, pythonRuntimeVersionForTest, extras)
			if err != nil {
				t.Fatal(err)
			}
			candidates := []Candidate{root, toolkit}
			for _, extra := range extras {
				component, err := parseReportCandidate(upgradeCandidate("nvidia-"+extra, toolkitVersion, false), profile, pythonRuntimeVersionForTest, nil)
				if err != nil {
					t.Fatal(err)
				}
				candidates = append(candidates, component)
			}
			proposal, err := input.ProposeSourcePinnedUpgrade(candidates)
			if err != nil {
				t.Fatal(err)
			}
			if err := proposal.ValidateSourcePinnedFinal(candidates); err != nil {
				t.Fatal(err)
			}
			constraints := string(proposal.Constraints())
			if strings.ContainsAny(constraints, "[];") {
				t.Fatal("extra/marker passed as pip constraint", constraints)
			}
			if mode == "unchanged" && (!strings.Contains(constraints, "nvidia-cublas>=13.2") || !strings.Contains(constraints, "nvidia-cudart>=13.2")) {
				t.Fatal("retained conditional requirements lost", constraints)
			}
			if mode != "unchanged" && strings.Contains(constraints, "nvidia-cublas>=13.2") {
				t.Fatal("superseded extra request remained", constraints)
			}
			if mode == "changed-toolkit" && strings.Contains(constraints, "nvidia-cudart>=13.2") {
				t.Fatal("superseded toolkit metadata remained", constraints)
			}
			wrong := append([]Candidate(nil), candidates...)
			wrong[1].requestedExtras = nil
			if proposal.ValidateSourcePinnedFinal(wrong) == nil {
				t.Fatal("final selection lost required extra context")
			}
			if _, err := input.ProposeSourcePinnedUpgrade(candidates[:len(candidates)-1]); err == nil {
				t.Fatal("declared component edge lost its candidate")
			}
			control, _ := input.Control()
			roundtrip, err := ParseUpgradeInput([]domain.ProjectControlFile{control}, pythonRuntimeVersionForTest)
			if err != nil {
				t.Fatal(err)
			}
			repeated, err := roundtrip.ProposeSourcePinnedUpgrade(candidates)
			if err != nil || string(repeated.Constraints()) != constraints {
				t.Fatal("typed extras lost in original-control roundtrip", err)
			}
		})
	}
}

func TestUpgradePublicSelectionPreservesUnrelatedRetainedCUDAExtras(t *testing.T) {
	r, _ := ParseUpgradeRequirement("requests")
	files := []InstalledMetadataFile{upgradeMetadataFile("requests", "1.0", ""), upgradeMetadataFile("torch", "2.14.0+cu132", "Requires-Dist: cuda-toolkit[cudart]==13.2; platform_system == \"Linux\"\n"), upgradeMetadataFile("cuda-toolkit", "13.2", "Requires-Dist: nvidia-cudart>=13.2; extra == \"cudart\"\n"), upgradeMetadataFile("nvidia-cudart", "13.2", "")}
	input, err := NewUpgradeInput(r, files, pythonRuntimeVersionForTest)
	if err != nil {
		t.Fatal(err)
	}
	root := upgradeCandidate("requests", "2.0", true)
	proposal, err := input.ParseUpgradeDelta(upgradeReport(t, root), pipRuntimeVersionForTest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(proposal.Constraints()), "nvidia-cudart>=13.2") {
		t.Fatal("retained CUDA obligation disappeared")
	}
	final, err := ParseInstallationReport(proposal.Reference(), upgradeReport(t, root), pipRuntimeVersionForTest, pythonRuntimeVersionForTest)
	if err != nil || proposal.ValidateFinal(final) != nil || len(final.Candidates()) != 1 {
		t.Fatal("retained CUDA became an unrelated graph root", err)
	}
}
