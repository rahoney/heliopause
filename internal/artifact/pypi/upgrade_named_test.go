package pypi

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestUpgradeNamedRequirementBindsProfileAndLocalVersionIntent(t *testing.T) {
	for _, name := range []string{"cpu", "cu126", "cu130", "cu132"} {
		profile := mustPyTorchProfile(t, name)
		for _, value := range []string{"Torch", "torch>=2,<3", "torch==2.0", "torch==2.0+" + name, "torch@2.0+" + name} {
			r, err := ParseUpgradeRequirementForSource(value, profile.Source())
			if err != nil || r.Reference().Source() != profile.Source() || r.Project() != "torch" || !r.AcceptsVersion("2.0+"+name) || value != "Torch" && r.AcceptsVersion("3.0+"+name) || r.AcceptsVersion("2.0rc1+"+name) {
				t.Fatal(value, r, err)
			}
		}
		r, err := ParseUpgradeRequirementForSource("torch==2.0rc1", profile.Source())
		if err != nil || !r.AcceptsVersion("2.0rc1+"+name) || r.AcceptsVersion("2.0+"+name) {
			t.Fatal("explicit prerelease intent lost", r, err)
		}
		for _, value := range []string{"numpy", "torch[extra]", "torch!=2", "torch @ https://elsewhere.invalid/a.whl"} {
			if _, err := ParseUpgradeRequirementForSource(value, profile.Source()); err == nil {
				t.Fatal("unsupported named intent accepted", value)
			}
		}
	}
	unknown, _ := domain.NewSourceID("unknown-python")
	if _, err := ParseUpgradeRequirementForSource("torch", unknown); err == nil {
		t.Fatal("unknown profile accepted")
	}
	if _, err := ParseUpgradeRequirement("torch==2.0+cpu"); err == nil {
		t.Fatal("public input now accepts named local versions")
	}
}

func TestUpgradeNamedControlRoundtripPreservesSourceAndPublicWire(t *testing.T) {
	profile := mustPyTorchProfile(t, "cpu")
	r, _ := ParseUpgradeRequirementForSource("torch==2.0+cpu", profile.Source())
	input, err := NewUpgradeInput(r, []InstalledMetadataFile{upgradeMetadataFile("torch", "1.0+cpu", "")}, pythonRuntimeVersionForTest)
	if err != nil {
		t.Fatal(err)
	}
	control, _ := input.Control()
	parsed, err := ParseUpgradeInput([]domain.ProjectControlFile{control}, pythonRuntimeVersionForTest)
	if err != nil || parsed.Requirement().Reference() != r.Reference() || !bytes.Contains(control.Body(), []byte(`"source_id":"pytorch-cpu"`)) {
		t.Fatal("named source missing from immutable input", parsed, err)
	}
	for _, source := range []string{"", "pypi", "unknown-python"} {
		var wire upgradeInputWire
		if err := json.Unmarshal(control.Body(), &wire); err != nil {
			t.Fatal(err)
		}
		wire.Source = source
		body, _ := json.Marshal(wire)
		altered, _ := domain.NewProjectControlFile(UpgradeInputControlName, body, true)
		if _, err := ParseUpgradeInput([]domain.ProjectControlFile{altered}, pythonRuntimeVersionForTest); err == nil {
			t.Fatal("local named input survived source removal/substitution", source)
		}
	}
	public, _ := ParseUpgradeRequirement("root")
	publicInput, _ := NewUpgradeInput(public, nil, pythonRuntimeVersionForTest)
	publicControl, _ := publicInput.Control()
	want := `{"version":1,"requirement":"root","python":"3.14.7","installed":[]}`
	if string(publicControl.Body()) != want {
		t.Fatal("public v1 representation changed", string(publicControl.Body()))
	}
}

func namedUpgradeCandidate(profile SourceProfile, version string, requirements ...string) pipInstall {
	item := upgradeCandidate("torch", version, true, requirements...)
	item.DownloadInfo.URL = profile.IndexURL() + "torch/torch-" + version + "-py3-none-any.whl"
	return item
}

func TestUpgradeNamedProposalPreservesMixedSourcesAndFreshNoopClosure(t *testing.T) {
	for _, mode := range []string{"changed", "noop", "transitive-only"} {
		t.Run(mode, func(t *testing.T) {
			profile := mustPyTorchProfile(t, "cpu")
			r, _ := ParseUpgradeRequirementForSource("torch>=1,<3", profile.Source())
			oldRoot, oldDep := "1.0+cpu", "1.0"
			if mode != "changed" {
				oldRoot = "2.0+cpu"
			}
			if mode == "noop" {
				oldDep = "2.0"
			}
			oldRequirements := "Requires-Dist: dependency>=2\n"
			if mode == "changed" {
				oldRequirements = "Requires-Dist: dependency<2\n"
			}
			files := []InstalledMetadataFile{upgradeMetadataFile("torch", oldRoot, oldRequirements), upgradeMetadataFile("dependency", oldDep, ""), upgradeMetadataFile("retained", "1.0", "Requires-Dist: dependency<3\n")}
			input, err := NewUpgradeInput(r, files, pythonRuntimeVersionForTest)
			if err != nil {
				t.Fatal(err)
			}
			root := namedUpgradeCandidate(profile, "2.0+cpu", "dependency>=2")
			dep := upgradeCandidate("dependency", "2.0", false)
			delta := []pipInstall{root, dep}
			if mode == "noop" {
				delta = nil
			}
			if mode == "transitive-only" {
				delta = []pipInstall{dep}
			}
			proposal, err := input.ParseUpgradeDelta(upgradeReport(t, delta...), pipRuntimeVersionForTest)
			if err != nil || proposal.Reference().Source() != profile.Source() || proposal.Reference().Locator() != "torch@2.0+cpu" {
				t.Fatal(proposal, err)
			}
			if !strings.Contains(string(proposal.Constraints()), "dependency<3\n") || mode == "changed" && strings.Contains(string(proposal.Constraints()), "dependency<2\n") {
				t.Fatal("retained/superseded constraints lost", string(proposal.Constraints()))
			}
			final, err := ParseInstallationReport(proposal.Reference(), upgradeReport(t, root, dep), pipRuntimeVersionForTest, pythonRuntimeVersionForTest)
			if err != nil || proposal.ValidateFinal(final) != nil {
				t.Fatal("complete mixed-source closure rejected", err)
			}
			complete, err := input.ProposeSourcePinnedUpgrade(final.Candidates())
			if err != nil || complete.ValidateSourcePinnedFinal(final.Candidates()) != nil || !bytes.Equal(complete.Constraints(), proposal.Constraints()) {
				t.Fatal("source-pinned traversal lost selection or retained constraints", err)
			}
			changedBytes := dep
			changedBytes.DownloadInfo.ArchiveInfo.Hashes = map[string]string{"sha256": strings.Repeat("c", 64)}
			substituted, err := ParseInstallationReport(complete.Reference(), upgradeReport(t, root, changedBytes), pipRuntimeVersionForTest, pythonRuntimeVersionForTest)
			if err != nil {
				t.Fatal(err)
			}
			if complete.ValidateSourcePinnedFinal(substituted.Candidates()) == nil {
				t.Fatal("same-version candidate changed between source-pinned traversals")
			}
			if _, err := input.ProposeSourcePinnedUpgrade([]Candidate{final.Candidates()[0]}); err == nil {
				t.Fatal("incomplete source-pinned first traversal accepted")
			}
			graph, err := BuildLockedGraph(r.Reference(), final.Candidates())
			if err != nil || len(graph.Nodes()) != 2 {
				t.Fatal(graph, err)
			}
			for _, node := range graph.Nodes() {
				if node.Artifact().Identity().Name() == "dependency" && node.Artifact().Identity().Source() != PublicPyPIProfile().Source() {
					t.Fatal("root source copied onto ordinary dependency")
				}
			}
			wrong := root
			wrong.DownloadInfo.URL = strings.Replace(root.DownloadInfo.URL, "/cpu/", "/cu126/", 1)
			if _, err := input.ParseUpgradeDelta(upgradeReport(t, wrong), pipRuntimeVersionForTest); err == nil {
				t.Fatal("another named profile accepted")
			}
		})
	}
}
