package pypi

import (
	"encoding/json"
	"strings"
	"testing"
)

func upgradeCandidate(name, version string, primary bool, requirements ...string) pipInstall {
	var item pipInstall
	item.Metadata.Name, item.Metadata.Version = name, version
	item.Metadata.RequiresDist = requirements
	item.DownloadInfo.URL = "https://files.pythonhosted.org/packages/" + name + "-" + version + "-py3-none-any.whl"
	item.DownloadInfo.ArchiveInfo.Hashes = map[string]string{"sha256": strings.Repeat("b", 64)}
	item.Requested = primary
	return item
}

func upgradeReport(t *testing.T, items ...pipInstall) []byte {
	t.Helper()
	if items == nil {
		items = []pipInstall{}
	}
	var report pipReport
	report.Version, report.PipVersion, report.Install = "1", pipRuntimeVersionForTest, items
	report.Environment = pipEnvironment{"cpython", pythonRuntimeVersionForTest, pythonRuntimeVersionForTest, "x86_64", "linux"}
	body, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestUpgradeProposalPreservesOnlyIfNeededNoopTransitiveAndMissingRoot(t *testing.T) {
	for _, mode := range []string{"new-root-version", "noop", "transitive-only", "missing-root"} {
		t.Run(mode, func(t *testing.T) {
			requirement, _ := ParseUpgradeRequirement("root")
			rootVersion, dependencyVersion := "1.0", "1.0"
			if mode == "noop" || mode == "transitive-only" {
				rootVersion = "2.0"
			}
			if mode == "noop" {
				dependencyVersion = "2.0"
			}
			files := []InstalledMetadataFile{upgradeMetadataFile("dependency", dependencyVersion, ""), upgradeMetadataFile("retained", "1.0", ""), upgradeMetadataFile("other", "1.0", "Requires-Dist: dependency<3\n")}
			if mode != "missing-root" {
				files = append(files, upgradeMetadataFile("root", rootVersion, "Requires-Dist: dependency>=1\nRequires-Dist: retained>=1\n"))
			}
			input, err := NewUpgradeInput(requirement, files, pythonRuntimeVersionForTest)
			if err != nil {
				t.Fatal(err)
			}
			root := upgradeCandidate("root", "2.0", true, "dependency>=2", "retained>=1")
			dependency := upgradeCandidate("dependency", "2.0", false)
			delta := []pipInstall{root, dependency}
			if mode == "noop" {
				delta = nil
			} else if mode == "transitive-only" {
				delta = []pipInstall{dependency}
			}
			proposal, err := input.ParseUpgradeDelta(upgradeReport(t, delta...), pipRuntimeVersionForTest)
			if err != nil || proposal.Reference().Locator() != "root@2.0" {
				t.Fatal(proposal, err)
			}
			for _, constraint := range []string{"root==2.0\n", "dependency==2.0\n", "retained==1.0\n", "other==1.0\n", "dependency<3\n"} {
				if !strings.Contains(string(proposal.Constraints()), constraint) {
					t.Fatal("retained environment constraint missing", constraint)
				}
			}
			final, err := ParseInstallationReport(proposal.Reference(), upgradeReport(t, root, dependency, upgradeCandidate("retained", "1.0", false)), pipRuntimeVersionForTest, pythonRuntimeVersionForTest)
			if err != nil || proposal.ValidateFinal(final) != nil {
				t.Fatal("complete existing report parser could not be reused", err)
			}
			graph, err := BuildLockedGraph(proposal.Reference(), final.Candidates())
			if err != nil || len(graph.Nodes()) != 3 || len(graph.Edges()) != 2 {
				t.Fatal("unrelated installed package became a fabricated graph node/root", graph, err)
			}
			copy := proposal.Constraints()
			copy[0] = 'X'
			if proposal.Constraints()[0] == 'X' {
				t.Fatal("caller modified frozen constraints")
			}
		})
	}
}

func TestUpgradeProposalDropsSupersededRequirementsAndRetainsActualOtherMetadata(t *testing.T) {
	requirement, _ := ParseUpgradeRequirement("root>=1,<3")
	input, _ := NewUpgradeInput(requirement, []InstalledMetadataFile{upgradeMetadataFile("root", "1.0", "Requires-Dist: dependency<2\n"), upgradeMetadataFile("dependency", "1.0", ""), upgradeMetadataFile("other", "1.0", "Requires-Dist: dependency<2.5\n")}, pythonRuntimeVersionForTest)
	proposal, err := input.ParseUpgradeDelta(upgradeReport(t, upgradeCandidate("root", "2.0", true, "dependency>=2"), upgradeCandidate("dependency", "2.0", false)), pipRuntimeVersionForTest)
	if err != nil || strings.Contains(string(proposal.Constraints()), "dependency<2\n") || !strings.Contains(string(proposal.Constraints()), "dependency<2.5\n") {
		t.Fatal("superseded/retained requirements confused", string(proposal.Constraints()), err)
	}
}

func TestUpgradeProposalRejectsChangedCandidateSubstitutionAndUnexpectedSelection(t *testing.T) {
	requirement, _ := ParseUpgradeRequirement("root")
	input, _ := NewUpgradeInput(requirement, []InstalledMetadataFile{upgradeMetadataFile("root", "1.0", ""), upgradeMetadataFile("dependency", "1.0", ""), upgradeMetadataFile("retained", "1.0", "")}, pythonRuntimeVersionForTest)
	root, dependency := upgradeCandidate("root", "2.0", true, "dependency>=2", "retained>=1"), upgradeCandidate("dependency", "2.0", false)
	proposal, err := input.ParseUpgradeDelta(upgradeReport(t, root, dependency), pipRuntimeVersionForTest)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"digest", "source-url", "missing-change", "retained-version", "unexpected-package", "metadata-edges"} {
		t.Run(mode, func(t *testing.T) {
			items := []pipInstall{upgradeCandidate("root", "2.0", true, "dependency>=2", "retained>=1"), upgradeCandidate("dependency", "2.0", false), upgradeCandidate("retained", "1.0", false)}
			switch mode {
			case "digest":
				items[0].DownloadInfo.ArchiveInfo.Hashes["sha256"] = strings.Repeat("c", 64)
			case "source-url":
				items[0].DownloadInfo.URL = "https://files.pythonhosted.org/packages/another/root-2.0-py3-none-any.whl"
			case "missing-change":
				items = []pipInstall{items[0], items[2]}
			case "retained-version":
				items[2] = upgradeCandidate("retained", "2.0", false)
			case "unexpected-package":
				items = append(items, upgradeCandidate("unexpected", "1.0", false))
			case "metadata-edges":
				items[0].Metadata.RequiresDist = []string{"dependency>=1", "retained>=1"}
			}
			final, err := ParseInstallationReport(proposal.Reference(), upgradeReport(t, items...), pipRuntimeVersionForTest, pythonRuntimeVersionForTest)
			if err == nil && proposal.ValidateFinal(final) == nil {
				t.Fatal("changed or substituted upgrade selection accepted")
			}
		})
	}
}

func TestUpgradeDeltaRejectsMissingArraysWrongRuntimeAndUnrequestedRoots(t *testing.T) {
	requirement, _ := ParseUpgradeRequirement("root")
	input, _ := NewUpgradeInput(requirement, []InstalledMetadataFile{upgradeMetadataFile("root", "1.0", "")}, pythonRuntimeVersionForTest)
	body := upgradeReport(t)
	for _, invalid := range [][]byte{
		[]byte(strings.Replace(string(body), `"install":[]`, `"install":null`, 1)),
		[]byte(strings.Replace(string(body), `"pip_version":"26.2.1"`, `"pip_version":"26.1"`, 1)),
		[]byte(strings.Replace(string(body), `"sys_platform":"linux"`, `"sys_platform":"win32"`, 1)),
		upgradeReport(t, upgradeCandidate("another", "1.0", true)),
		upgradeReport(t, upgradeCandidate("root", "1.0", false)),
		upgradeReport(t, upgradeCandidate("root", "1.0", true), upgradeCandidate("root", "1.0", true)),
	} {
		if _, err := input.ParseUpgradeDelta(invalid, pipRuntimeVersionForTest); err == nil {
			t.Fatal("incomplete or ambiguous delta accepted", string(invalid))
		}
	}
	empty, _ := NewUpgradeInput(requirement, nil, pythonRuntimeVersionForTest)
	if _, err := empty.ParseUpgradeDelta(body, pipRuntimeVersionForTest); err == nil {
		t.Fatal("missing requested root fabricated from an empty delta")
	}
}
