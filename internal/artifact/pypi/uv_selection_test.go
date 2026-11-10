package pypi

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type uvChangeFixture struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Action  string `json:"action"`
}

func uvDeltaFixture(t *testing.T, changes ...uvChangeFixture) []byte {
	t.Helper()
	if changes == nil {
		changes = []uvChangeFixture{}
	}
	body, err := json.Marshal(struct {
		Schema struct {
			Version string `json:"version"`
		} `json:"schema"`
		Changes []uvChangeFixture `json:"changes"`
		DryRun  bool              `json:"dry_run"`
	}{struct {
		Version string `json:"version"`
	}{"preview"}, changes, true})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func uvSelectionFixture(t *testing.T, selector string) UpgradeInput {
	t.Helper()
	requirement, err := ParseUpgradeRequirement(selector)
	if err != nil {
		t.Fatal(err)
	}
	input, err := NewUpgradeInput(requirement, []InstalledMetadataFile{upgradeMetadataFile("root", "1.0", "Requires-Dist: dependency>=1\n"), upgradeMetadataFile("dependency", "1.0", ""), upgradeMetadataFile("other", "1.0", "Requires-Dist: dependency<3\n")}, pythonRuntimeVersionForTest)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func TestUVSelectionBindsNativeReplacementRetentionAndNoop(t *testing.T) {
	for _, mode := range []string{"replace-root", "replace-closure", "noop", "new-root"} {
		t.Run(mode, func(t *testing.T) {
			input := uvSelectionFixture(t, "root>=1,<3")
			rootVersion, depVersion := "2.0", "1.0"
			changes := []uvChangeFixture{{"root", "1.0", "uninstalled"}, {"root", "2.0", "installed"}}
			if mode == "replace-closure" {
				depVersion = "2.0"
				changes = append(changes, uvChangeFixture{"dependency", "1.0", "uninstalled"}, uvChangeFixture{"dependency", "2.0", "installed"})
			}
			if mode == "noop" {
				rootVersion = "1.0"
				changes = nil
			}
			if mode == "new-root" {
				requirement, _ := ParseUpgradeRequirement("root")
				input, _ = NewUpgradeInput(requirement, input.MetadataFiles()[0:2], pythonRuntimeVersionForTest)
				changes = []uvChangeFixture{{"root", "2.0", "installed"}}
			}
			body := uvDeltaFixture(t, changes...)
			selection, err := input.ParseUVSelection(body)
			if err != nil {
				t.Fatal(err)
			}
			if selection.Reference().Locator() != "root@"+rootVersion {
				t.Fatal("root selection differs", selection.Reference())
			}
			for _, want := range []string{"dependency==" + depVersion + "\n", "other==1.0\n", "dependency<3\n"} {
				if !bytes.Contains(selection.Constraints(), []byte(want)) {
					t.Fatal("retained constraint missing", want)
				}
			}
			report, err := ParseInstallationReport(selection.Reference(), upgradeReport(t, upgradeCandidate("root", rootVersion, true, "dependency>=1"), upgradeCandidate("dependency", depVersion, false)), pipRuntimeVersionForTest, pythonRuntimeVersionForTest)
			if err != nil {
				t.Fatal(err)
			}
			graph, err := BuildLockedGraph(selection.Reference(), report.Candidates())
			if err != nil || selection.ValidateGraph(graph) != nil || len(graph.Nodes()) != 2 {
				t.Fatal("independent graph binding failed or unrelated original became root", err, selection.ValidateGraph(graph))
			}
			body[0] = 'X'
			copy := selection.Constraints()
			copy[0] = 'X'
			delta := selection.Delta()
			delta[0] = 'X'
			if selection.Delta()[0] == 'X' || selection.Constraints()[0] == 'X' {
				t.Fatal("caller modified frozen selection")
			}
		})
	}
}

func TestUVSelectionRejectsAmbiguousOrInconsistentNativeDelta(t *testing.T) {
	input := uvSelectionFixture(t, "root>=1,<3")
	good := uvDeltaFixture(t, uvChangeFixture{"root", "1.0", "uninstalled"}, uvChangeFixture{"root", "2.0", "installed"})
	bad := [][]byte{
		nil, []byte(`null`), []byte(`{}`), append(append([]byte(nil), good...), []byte(` {}`)...),
		bytes.Replace(good, []byte(`"preview"`), []byte(`"next"`), 1), bytes.Replace(good, []byte(`"dry_run":true`), []byte(`"dry_run":false`), 1),
		bytes.Replace(good, []byte(`"dry_run":true`), []byte(`"dry_run":true,"dry_run":true`), 1),
		bytes.Replace(good, []byte(`"name":"root"`), []byte(`"name":"root","name":"root"`), 1),
		bytes.Replace(good, []byte(`"version":"preview"`), []byte(`"version":"preview","extra":true`), 1),
		uvDeltaFixture(t, uvChangeFixture{"root", "2.0", "installed"}),
		uvDeltaFixture(t, uvChangeFixture{"root", "9.0", "uninstalled"}, uvChangeFixture{"root", "2.0", "installed"}),
		uvDeltaFixture(t, uvChangeFixture{"root", "1.0", "uninstalled"}),
		uvDeltaFixture(t, uvChangeFixture{"missing", "1.0", "uninstalled"}, uvChangeFixture{"missing", "2.0", "installed"}),
		uvDeltaFixture(t, uvChangeFixture{"root", "1.0", "uninstalled"}, uvChangeFixture{"root", "3.0", "installed"}),
		uvDeltaFixture(t, uvChangeFixture{"root", "1.0", "uninstalled"}, uvChangeFixture{"root", "2.0rc1", "installed"}),
		uvDeltaFixture(t, uvChangeFixture{"root", "1.0", "uninstalled"}, uvChangeFixture{"root", "2.0", "installed"}, uvChangeFixture{"root", "2.0", "installed"}),
		uvDeltaFixture(t, uvChangeFixture{"root", "1.0", "removed"}),
		bytes.Repeat([]byte("x"), maxReportBytes+1),
	}
	for i, body := range bad {
		if _, err := input.ParseUVSelection(body); err == nil {
			t.Fatalf("invalid native delta accepted: %d %s", i, body)
		}
	}
	if _, err := (UpgradeInput{}).ParseUVSelection(good); err == nil {
		t.Fatal("missing original input accepted")
	}
	if _, err := input.ParseUVSelection([]byte(`{"dry_run":true,"changes":[],"schema":{"version":"preview"}}`)); err != nil {
		t.Fatal("valid reordered native JSON rejected", err)
	}
}

func TestUVSelectionGraphCannotSubstituteVersionsOmitChangesOrBreakRetainedState(t *testing.T) {
	for _, mode := range []string{"wrong-version", "missing-change", "retained-conflict", "zero"} {
		input := uvSelectionFixture(t, "root")
		body := uvDeltaFixture(t, uvChangeFixture{"root", "1.0", "uninstalled"}, uvChangeFixture{"root", "2.0", "installed"})
		if mode == "missing-change" {
			body = uvDeltaFixture(t, uvChangeFixture{"unrelated", "1.0", "installed"}, uvChangeFixture{"root", "1.0", "uninstalled"}, uvChangeFixture{"root", "2.0", "installed"})
		}
		if mode == "retained-conflict" {
			body = uvDeltaFixture(t, uvChangeFixture{"dependency", "1.0", "uninstalled"}, uvChangeFixture{"dependency", "3.0", "installed"}, uvChangeFixture{"root", "1.0", "uninstalled"}, uvChangeFixture{"root", "2.0", "installed"})
		}
		selection, err := input.ParseUVSelection(body)
		if err != nil {
			t.Fatal(err)
		}
		depVersion := "1.0"
		if mode == "wrong-version" {
			depVersion = "2.0"
		}
		if mode == "retained-conflict" {
			depVersion = "3.0"
		}
		report, _ := ParseInstallationReport(selection.Reference(), upgradeReport(t, upgradeCandidate("root", "2.0", true, "dependency>=1"), upgradeCandidate("dependency", depVersion, false)), pipRuntimeVersionForTest, pythonRuntimeVersionForTest)
		graph, err := BuildLockedGraph(selection.Reference(), report.Candidates())
		if err != nil {
			t.Fatal(err)
		}
		if mode == "zero" {
			graph = domain.LockedDependencyGraph{}
		}
		if err := selection.ValidateGraph(graph); err == nil {
			t.Fatal("invalid graph accepted", mode)
		}
	}
}

func TestUVSelectionRequestBindsExactOriginalIntentAndRuntime(t *testing.T) {
	input := uvSelectionFixture(t, "root")
	original, _ := input.Control()
	for _, intent := range []UVSelectionIntent{UVInstallSelection, UVUpgradeSelection, UVUpgradePackageSelection} {
		control, err := NewUVSelectionIntentControl(intent)
		if err != nil {
			t.Fatal(err)
		}
		parsed, got, err := ParseUVSelectionRequest([]domain.ProjectControlFile{control, original}, input.Python())
		if err != nil || got != intent || parsed.Requirement().Requirement() != "root" {
			t.Fatal("intent/input binding failed", err)
		}
		for _, mode := range []string{"duplicate", "missing", "altered", "absent", "runtime"} {
			controls := []domain.ProjectControlFile{original, control}
			python := input.Python()
			switch mode {
			case "duplicate":
				controls[1] = original
			case "missing":
				controls = controls[:1]
			case "altered":
				controls[1], _ = domain.NewProjectControlFile(control.Name(), append([]byte(" "), control.Body()...), true)
			case "absent":
				controls[1], _ = domain.NewProjectControlFile(control.Name(), nil, false)
			case "runtime":
				python = "3.14.99"
			}
			if _, _, err := ParseUVSelectionRequest(controls, python); err == nil {
				t.Fatal("ambiguous frozen request accepted", mode)
			}
		}
	}
	if _, err := NewUVSelectionIntentControl(UVSelectionIntent(strings.Repeat("x", 100))); err == nil {
		t.Fatal("arbitrary intent accepted")
	}
}
