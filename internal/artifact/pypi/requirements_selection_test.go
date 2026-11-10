package pypi

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func requirementsSelectionFixture(t *testing.T, client string, original ...InstalledMetadataFile) (RequirementsInput, RequirementsSelection, []UVWheelInput, []SimpleProject) {
	t.Helper()
	base, inputs, pages := uvTestSelection(t, "child>=1.0")
	source, err := NewRequirementsInput(nil, nil, []string{"parent==1.0"})
	if err != nil {
		t.Fatal(err)
	}
	source, err = source.WithInstallation(client, false, original)
	if err != nil {
		t.Fatal(err)
	}
	control, _ := source.Control()
	parser, _ := source.ParserControl()
	text := sha256.Sum256([]byte("parent==1.0"))
	native, _ := json.Marshal(map[string]any{"version": 1, "input_sha256": control.Digest().String(), "parser_sha256": parser.Digest().String(), "python": runtimeidentity.PythonVersion, "pip": runtimeidentity.PipVersion, "require_hashes": false, "clauses": []any{map[string]any{"active": true, "source": "@positional", "line": 1, "text_sha256": hex.EncodeToString(text[:]), "requirement": "parent==1.0", "project": "parent", "extras": []string{}, "constraint": false, "hashes": map[string][]string{}}}})
	var delta, selected []byte
	var census []UVInstalledWheelControls
	if client == "uv" {
		delta = []byte(`{"schema":{"version":"preview"},"dry_run":true,"changes":[{"name":"parent","version":"1.0","action":"installed"},{"name":"child","version":"1.0","action":"installed"}]}`)
		selected = uvProjectBody(base.Packages())
		census = uvControlFixture()
	} else {
		items := []pipInstall{}
		for _, project := range base.Packages() {
			item := upgradeCandidate(project.Name, project.Version, project.Name == "parent")
			item.DownloadInfo.URL = project.Wheels[0].URL
			item.DownloadInfo.ArchiveInfo.Hashes["sha256"] = project.Wheels[0].SHA256
			items = append(items, item)
		}
		selected = requirementsReport(t, items...)
		delta = selected
	}
	selection, err := source.NewSelection(native, delta, selected, census, WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"})
	if err != nil {
		t.Fatal(err)
	}
	return source, selection, inputs, pages
}

func TestRequirementsSelectionBindsWholeArchivesAndReplaysExactMetadata(t *testing.T) {
	for _, client := range []string{"pip", "uv"} {
		t.Run(client, func(t *testing.T) {
			source, selection, inputs, pages := requirementsSelectionFixture(t, client)
			target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
			calls := 0
			validate := func(body []byte) ([]byte, error) {
				calls++
				sum := sha256.Sum256(body)
				return json.Marshal(requirementsMetadataGraph{Version: 1, Input: hex.EncodeToString(sum[:]), Python: runtimeidentity.PythonVersion, Pip: runtimeidentity.PipVersion, Roots: []string{"parent"}, Edges: [][]string{{"parent", "child"}}})
			}
			if _, err := selection.Control(); err == nil {
				t.Fatal("native selection became bound before whole archive checks")
			}
			bound, err := selection.BindWheels(inputs, pages, target, DefaultWheelLimits(), validate)
			if err != nil || calls != 1 {
				t.Fatal("whole selection binding failed", err)
			}
			control, err := bound.Control()
			if err != nil {
				t.Fatal(err)
			}
			sourceControl, _ := source.Control()
			snapshot, err := bound.Snapshot(uvProjectContext(t, "/tmp/requirements-bound"), []domain.ProjectControlFile{sourceControl, control}, target)
			if err != nil || len(snapshot.Dependencies()) != 2 {
				t.Fatal("whole snapshot lost transitive dependency", err)
			}
			replayed, err := ParseRequirementsSelection(source, control, target)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := replayed.Snapshot(snapshot.Context(), []domain.ProjectControlFile{sourceControl, control}, target); err == nil {
				t.Fatal("saved projection skipped archive readback")
			}
			nodes, edges, err := replayed.ProjectRecords(snapshot.Dependencies(), inputs, target, DefaultWheelLimits())
			if err != nil || len(nodes) != 2 || len(edges) != 1 {
				t.Fatal("record replay differs", err)
			}
			changed := append([]UVWheelInput(nil), inputs...)
			changed[0].Reader = bytes.NewReader(bytes.Repeat([]byte("x"), int(changed[0].Size)))
			if _, _, err := replayed.ProjectRecords(snapshot.Dependencies(), changed, target, DefaultWheelLimits()); err == nil {
				t.Fatal("changed archive reused approved metadata")
			}
			if _, err := selection.BindWheels(changed, pages, target, DefaultWheelLimits(), validate); err == nil || calls != 1 {
				t.Fatal("native metadata consumed unauthenticated archive", err)
			}
			if _, err := selection.BindWheels(inputs, nil, target, DefaultWheelLimits(), validate); err == nil || calls != 1 {
				t.Fatal("independent source coverage was skipped", err)
			}
		})
	}
}

func TestRequirementsDeclaredHashesBindTheSameCompleteArchiveBytes(t *testing.T) {
	_, selection, inputs, _ := requirementsSelectionFixture(t, "pip")
	for _, algorithm := range []string{"sha256", "sha384", "sha512"} {
		t.Run(algorithm, func(t *testing.T) {
			clauses := []RequirementsClause{}
			for _, input := range inputs {
				body := make([]byte, input.Size)
				if _, err := input.Reader.ReadAt(body, 0); err != nil {
					t.Fatal(err)
				}
				var project string
				// Match by URL because native order and acquisition order may differ.
				for _, wheel := range selection.wheels {
					if wheel.URL == input.URL {
						project, _, _, _, _, _ = ParseWheelFilename(wheel.Filename)
					}
				}
				var digest string
				switch algorithm {
				case "sha256":
					sum := sha256.Sum256(body)
					digest = hex.EncodeToString(sum[:])
				case "sha384":
					sum := sha512.Sum384(body)
					digest = hex.EncodeToString(sum[:])
				case "sha512":
					sum := sha512.Sum512(body)
					digest = hex.EncodeToString(sum[:])
				}
				clauses = append(clauses, RequirementsClause{Active: true, Project: project, Hashes: map[string][]string{algorithm: {digest}}})
			}
			native := selection.native
			native.clauses = clauses
			if err := verifyRequirementsHashes(native, selection.wheels, inputs); err != nil {
				t.Fatal("correct whole archive hash rejected", err)
			}
			value := native.clauses[0].Hashes[algorithm][0]
			native.clauses[0].Hashes[algorithm][0] = "0" + value[1:]
			if value[0] == '0' {
				native.clauses[0].Hashes[algorithm][0] = "1" + value[1:]
			}
			if err := verifyRequirementsHashes(native, selection.wheels, inputs); err == nil {
				t.Fatal("wrong declared hash accepted")
			}
		})
	}
}

func TestRequirementsSourceBindingPreservesGeneralPythonClausesWithoutChangingLegacyComparison(t *testing.T) {
	const python = ">=2.7, !=3.0.*, !=3.1.*, !=3.2.*, !=3.3.*"
	source := PublicPyPIProfile().Source()
	candidate := Candidate{project: "pysocks", source: source, version: "1.7.1", filename: "PySocks-1.7.1-py3-none-any.whl", url: "https://files.pythonhosted.org/packages/PySocks-1.7.1-py3-none-any.whl", sha256: strings.Repeat("a", 64), requiresPython: python}
	report := InstallationReport{candidates: []Candidate{candidate}}
	for _, kind := range []string{"exact", "different-python", "whitespace-difference", "different-hash", "different-url", "yanked", "duplicate", "absent"} {
		t.Run(kind, func(t *testing.T) {
			file := SimpleFile{filename: candidate.filename, url: candidate.url, sha256: candidate.sha256, requiresPython: python, requiresPythonPresent: true}
			switch kind {
			case "different-python":
				file.requiresPython = ">=3.8"
			case "whitespace-difference":
				file.requiresPython = strings.ReplaceAll(python, " ", "")
			case "different-hash":
				file.sha256 = strings.Repeat("b", 64)
			case "different-url":
				file.url += "changed"
			case "yanked":
				file.yanked = true
			}
			page := SimpleProject{project: "pysocks", source: source, files: []SimpleFile{file}}
			if kind == "duplicate" {
				page.files = append(page.files, file)
			}
			if kind == "absent" {
				page.files = nil
			}
			if _, err := crossCheckReport(report, []SimpleProject{page}, true); (err == nil) != (kind == "exact" || kind == "whitespace-difference") {
				t.Fatal("requirements source equality or identity differs", kind, err)
			}
			if _, err := CrossCheckReport(report, []SimpleProject{page}); err == nil {
				t.Fatal("legacy comparison widened", kind)
			}
		})
	}
}

func TestRequirementsMetadataBindsGeneralOriginalObligationsOutsideTheSelectedGraph(t *testing.T) {
	const requirement = `child>=1 ; os_name == "posix"`
	_, selection, inputs, pages := requirementsSelectionFixture(t, "pip", InstalledMetadataFile{"retained-1.0.dist-info", []byte("Metadata-Version: 2.1\nName: retained\nVersion: 1.0\nRequires-Dist: " + requirement + "\n\n")})
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	bound, err := selection.BindWheels(inputs, pages, target, DefaultWheelLimits(), func(body []byte) ([]byte, error) {
		var payload struct {
			Originals []requirementsProjectMetadata `json:"originals"`
			Wheels    []requirementsProjectMetadata `json:"wheels"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, err
		}
		if len(payload.Originals) != 1 || payload.Originals[0].Name != "retained" || payload.Originals[0].SHA256 != "" || len(payload.Originals[0].Requirements) != 1 || payload.Originals[0].Requirements[0] != requirement || len(payload.Wheels) != 2 {
			t.Fatal("original metadata was omitted, reinterpreted or made a selected artifact")
		}
		sum := sha256.Sum256(body)
		return json.Marshal(requirementsMetadataGraph{Version: 1, Input: hex.EncodeToString(sum[:]), Python: runtimeidentity.PythonVersion, Pip: runtimeidentity.PipVersion, Roots: []string{"parent"}, Edges: [][]string{{"parent", "child"}}})
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bound.Wheels()) != 2 {
		t.Fatal("unrelated retained package became a new inspection root")
	}
}
