package pypi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// Synthetic pinned-parser DATA for binding tests, not a PEP semantic oracle.
func uvProjectProjectionFixture(t *testing.T, source RequirementsInput) []byte {
	t.Helper()
	control, _ := source.Control()
	parser, _ := source.ParserControl()
	sources, err := source.nativeClauseSources()
	if err != nil {
		t.Fatal(err)
	}
	var wire requirementsInputWire
	if err := json.Unmarshal(source.body, &wire); err != nil {
		t.Fatal(err)
	}
	clauses := []any{}
	for _, item := range sources {
		document, err := ParseRequirementsFile(wire.Files[item.file])
		if err != nil {
			t.Fatal(err)
		}
		text := ""
		for _, line := range document.Lines() {
			if line.Number == item.line {
				text = line.Text
			}
		}
		clauses = append(clauses, map[string]any{"active": true, "source": item.file, "line": item.line, "text_sha256": item.hash, "requirement": text, "project": item.project, "extras": []string{}, "constraint": item.constraint, "hashes": map[string][]string{}})
	}
	body, err := json.Marshal(map[string]any{"version": 1, "input_sha256": control.Digest().String(), "parser_sha256": parser.Digest().String(), "python": runtimeidentity.PythonVersion, "pip": runtimeidentity.PipVersion, "require_hashes": false, "clauses": clauses})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestUVProjectTargetProofBindsArchivesStaticRecordsAndReplay(t *testing.T) {
	legacy, inputs, pages := uvProjectExportBindingFixture(t)
	manifest := append(legacy.native.input.Body(), []byte("\n[dependency-groups]\ndev=[]\n")...)
	input, err := ParseUVProjectInput(manifest, nil, false, UVProjectSync, "")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := ParseUVNativeProjectLock(input, legacy.native.body)
	if err != nil {
		t.Fatal(err)
	}
	export, err := lock.ParseTargetExport(legacy.body)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := export.Controls(); err == nil {
		t.Fatal("portable export accepted before target projection")
	}
	source, err := export.ProjectionInput()
	if err != nil {
		t.Fatal(err)
	}
	export, err = export.ProjectTarget(uvProjectProjectionFixture(t, source))
	if err != nil {
		t.Fatal(err)
	}
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	calls := 0
	validator := func(payload []byte) ([]byte, error) {
		calls++
		sum := sha256.Sum256(payload)
		return json.Marshal(requirementsMetadataGraph{Version: 1, Input: hex.EncodeToString(sum[:]), Python: runtimeidentity.PythonVersion, Pip: runtimeidentity.PipVersion, Roots: []string{"parent"}, Edges: [][]string{{"parent", "child"}}})
	}
	if _, err := export.IntentControl(); err == nil {
		t.Fatal("unvalidated metadata became an intent proof")
	}
	bound, err := export.BindTargetMetadata(inputs, pages, target, DefaultWheelLimits(), validator)
	if err != nil || calls != 1 {
		t.Fatal("whole metadata binding failed", err)
	}
	intent, err := bound.IntentControl()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseUVProjectControls([]domain.ProjectControlFile{mustUVControl(t, UVProjectInputControl, manifest), mustUVControl(t, UVNativeLockControl, legacy.native.body), intent}); err == nil {
		t.Fatal("selected metadata proof accepted as original intent")
	}
	replayed, err := export.RestoreTarget(intent)
	if err != nil {
		t.Fatal(err)
	}
	controls, err := replayed.Controls()
	if err != nil {
		t.Fatal(err)
	}
	controls = append(controls, intent)
	snapshot, err := replayed.BindProject(uvProjectContext(t, "/tmp/uv-target-proof"), controls, inputs, pages, target, DefaultWheelLimits())
	if err != nil || len(snapshot.Dependencies()) != 2 {
		t.Fatal("replayed whole snapshot rejected", err)
	}
	nodes, edges, err := replayed.ProjectRecords(snapshot.Dependencies(), inputs, target, DefaultWheelLimits())
	if err != nil || len(nodes) != 2 || len(edges) != 1 {
		t.Fatal("cached metadata graph lost closure", err)
	}
	changed := append([]UVWheelInput{}, inputs...)
	changed[0].Reader = bytes.NewReader(bytes.Repeat([]byte("x"), int(changed[0].Size)))
	if _, err := bound.BindProject(snapshot.Context(), controls, changed, pages, target, DefaultWheelLimits()); err == nil {
		t.Fatal("bound metadata bypassed changed archive validation")
	}
	if _, _, err := replayed.ProjectRecords(snapshot.Dependencies(), changed, target, DefaultWheelLimits()); err == nil {
		t.Fatal("cache replay skipped archive readback")
	}
	if _, err := export.BindTargetMetadata(inputs, nil, target, DefaultWheelLimits(), validator); err == nil || calls != 1 {
		t.Fatal("metadata validator ran without independent source coverage")
	}
	var inspections []WheelInspection
	for _, item := range replayed.Packages() {
		for _, archive := range inputs {
			if archive.URL == item.Wheels[0].URL {
				wheel, err := InspectWheel(archive.Reader, archive.Size, item.Wheels[0].Filename, item.Wheels[0].SHA256, target, DefaultWheelLimits())
				if err != nil {
					t.Fatal(err)
				}
				inspections = append(inspections, wheel)
			}
		}
	}
	if _, edges, err := replayed.ProjectRecordsFromInspections(snapshot.Dependencies(), inspections); err != nil || len(edges) != 1 {
		t.Fatal("static graph differs from cached metadata", err)
	}
	inspections[0].RequiresDist = append(inspections[0].RequiresDist, "foreign==1.0")
	if _, _, err := replayed.ProjectRecordsFromInspections(snapshot.Dependencies(), inspections); err == nil {
		t.Fatal("changed metadata reused saved graph")
	}
}

func mustUVControl(t *testing.T, name string, body []byte) domain.ProjectControlFile {
	t.Helper()
	control, err := domain.NewProjectControlFile(name, body, true)
	if err != nil {
		t.Fatal(err)
	}
	return control
}

func TestUVNativeConditionalGraphRetainsAggregateDownloadBudget(t *testing.T) {
	input, body, _ := uvNativeProjectFixture(t)
	input, err := ParseUVProjectInput(append(input.Body(), []byte("\n[dependency-groups]\ndev=[]\n")...), nil, false, UVProjectSync, "")
	if err != nil {
		t.Fatal(err)
	}
	var wire uvNativeProjectLockWire
	if err := toml.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	for i := range wire.Packages {
		for j := range wire.Packages[i].Wheels {
			wire.Packages[i].Wheels[j].Size = PublicPyPIProfile().ResourcePolicy().MaxArtifactCompressed()
		}
	}
	changed, err := toml.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseUVNativeProjectLock(input, changed); err == nil {
		t.Fatal("per-variant validation bypassed the aggregate graph budget")
	}
}

func TestUVProjectFrozenUsesCurrentDefaultGroupsAndLockedVersions(t *testing.T) {
	input, body, _ := uvNativeProjectFixture(t)
	manifest := append(input.Body(), []byte("\n[dependency-groups]\ndev=['child==1.0']\nlint=['parent>=1.0']\n[tool.uv]\npackage=false\ndefault-groups=['dev']\n")...)
	var wire uvNativeProjectLockWire
	if err := toml.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	for i := range wire.Packages {
		if wire.Packages[i].Name == input.Project() {
			wire.Packages[i].DefaultGroups = []string{"lint"}
			wire.Packages[i].Groups = map[string][]uvNativeDependency{"dev": {{Name: "child"}}, "lint": {{Name: "parent"}}}
			wire.Packages[i].Metadata.RequiresDev = map[string][]uvNativeRequirement{"dev": {{Name: "child", Specifier: "==1.0"}}, "lint": {{Name: "parent", Specifier: ">=1.0"}}}
		}
	}
	body, err := toml.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	input, err = ParseUVProjectInput(manifest, body, true, UVProjectSync, "", UVProjectSelection{Frozen: true})
	if err != nil {
		t.Fatal(err)
	}
	lock, err := ParseUVNativeProjectLock(input, body)
	if err != nil {
		t.Fatal(err)
	}
	roots, err := lock.frozenRequirements()
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 2 || roots[0] != "parent==1.0" || roots[1] != "child==1.0" {
		t.Fatal("frozen used old default groups or current version constraints", roots)
	}
}

func TestUVProjectPortableMarkerCannotInjectRequirementLines(t *testing.T) {
	legacy, _, _ := uvProjectExportBindingFixture(t)
	input, err := ParseUVProjectInput(append(legacy.native.input.Body(), []byte("\n[dependency-groups]\ndev=[]\n")...), nil, false, UVProjectSync, "")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := ParseUVNativeProjectLock(input, legacy.native.body)
	if err != nil {
		t.Fatal(err)
	}
	var wire uvProjectExportWire
	if err := toml.Unmarshal(legacy.body, &wire); err != nil {
		t.Fatal(err)
	}
	(*wire.Packages)[0].Marker = "sys_platform == 'linux'\nforeign==1.0"
	body, err := toml.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	export, err := lock.ParseTargetExport(body)
	if err == nil {
		_, err = export.ProjectionInput()
	}
	if err == nil {
		t.Fatal("portable marker introduced another requirements clause")
	}
}
