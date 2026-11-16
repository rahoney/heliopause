package pypi

import (
	"bytes"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// These are frozen parser/metadata outputs, never inspection or Policy authority.
// They live in the existing intent control; portable export bytes stay native.
type uvProjectTargetWire struct {
	Projection []byte `json:"projection"`
	Graph      []byte `json:"graph"`
}

func (e UVProjectExport) NeedsMetadataProjection() bool { return e.native.rich }

func (l UVNativeProjectLock) parseScopedExport(body []byte) (UVProjectExport, error) {
	invalid := errors.New("UV portable selection differs from its universal lock")
	var wire uvProjectExportWire
	if err := toml.NewDecoder(bytes.NewReader(body)).DisallowUnknownFields().Decode(&wire); err != nil {
		return UVProjectExport{}, err
	}
	if wire.Version != "1.0" || wire.Creator != "uv" || !uvProjectPythonMatches(l.python, wire.Python) || wire.Packages == nil || len(*wire.Packages) > len(l.packages) {
		return UVProjectExport{}, invalid
	}
	native := map[string]UVLockedPackage{}
	for _, item := range l.packages {
		native[item.Name+"@"+item.Version] = item
	}
	seen := map[string]bool{}
	view := make([]map[string]any, 0, len(*wire.Packages))
	for _, item := range *wire.Packages {
		key := item.Name + "@" + item.Version
		original, ok := native[key]
		if !ok || seen[key] || !uvPublicProjectIndex(item.Index) || (item.Marker != "" && !validUVProjectText([]byte(item.Marker), false)) {
			return UVProjectExport{}, invalid
		}
		seen[key] = true
		source, hasSource := l.sources[key]
		if hasSource != (item.Sdist != nil) {
			return UVProjectExport{}, invalid
		}
		if item.Sdist != nil && (item.Sdist.Name != "" || item.Sdist.URL != source.URL || item.Sdist.Size != source.Size || len(item.Sdist.Hashes) != 1 || item.Sdist.Hashes["sha256"] != strings.TrimPrefix(source.Hash, "sha256:")) {
			return UVProjectExport{}, invalid
		}
		allowed := map[UVLockedWheel]bool{}
		for _, wheel := range original.Wheels {
			allowed[wheel] = true
		}
		wheels := make([]map[string]any, 0, len(item.Wheels))
		for _, wheel := range item.Wheels {
			location := wheel.URL[strings.LastIndex(wheel.URL, "/")+1:]
			if !validUVExportWheelName(wheel.Name, location, item.Name) || !allowed[UVLockedWheel{URL: wheel.URL, Filename: location, SHA256: wheel.Hashes["sha256"], Size: wheel.Size}] || len(wheel.Hashes) != 1 {
				return UVProjectExport{}, invalid
			}
			wheels = append(wheels, map[string]any{"url": wheel.URL, "size": wheel.Size, "hashes": wheel.Hashes})
		}
		view = append(view, map[string]any{"name": item.Name, "version": item.Version, "wheels": wheels})
	}
	validation, err := toml.Marshal(map[string]any{"lock-version": "1.0", "created-by": "uv", "requires-python": wire.Python, "packages": view})
	if err != nil {
		return UVProjectExport{}, err
	}
	if _, err := parseUVLockedPackages(validation, runtimeidentity.PythonVersion, "", "", false, true, true); err != nil {
		return UVProjectExport{}, err
	}
	return UVProjectExport{body: bytes.Clone(body), native: l, portable: append([]uvProjectExportPackage{}, (*wire.Packages)...)}, nil
}

// ProjectionInput uses the existing fixed PEP parser for both requested roots
// and portable package markers. The latter are constraints, never extra roots.
func (e UVProjectExport) ProjectionInput() (RequirementsInput, error) {
	if !e.native.rich || len(e.body) == 0 {
		return RequirementsInput{}, errors.New("UV target projection is unavailable")
	}
	roots, err := e.native.input.SelectedRequirements()
	if e.native.input.selection.Frozen {
		roots, err = e.native.frozenRequirements()
	}
	if err != nil {
		return RequirementsInput{}, err
	}
	for _, requirement := range roots {
		if _, err := uvProjectPublicRequirementName(requirement); err != nil {
			return RequirementsInput{}, err
		}
	}
	selected := make([]string, 0, len(e.portable))
	for _, item := range e.portable {
		text := item.Name + "==" + item.Version
		if item.Marker != "" {
			text += "; " + item.Marker
		}
		if name, err := uvProjectPublicRequirementName(text); err != nil || name != item.Name {
			return RequirementsInput{}, errors.New("UV portable requirement is not one public clause")
		}
		selected = append(selected, text)
	}
	return NewRequirementsInput([]RequirementsSource{{Name: "roots.txt"}, {Name: "portable.txt", Constraint: true}}, map[string][]byte{"roots.txt": []byte(strings.Join(roots, "\n")), "portable.txt": []byte(strings.Join(selected, "\n"))}, nil)
}

func (l UVNativeProjectLock) frozenRequirements() ([]string, error) {
	input := l.input
	invalid := errors.New("UV frozen scope is absent from the existing lock")
	byName := map[string][]string{}
	for _, item := range l.packages {
		byName[item.Name] = append(byName[item.Name], item.Version)
	}
	convert := func(edges []uvNativeDependency) ([]string, error) {
		result := make([]string, 0, len(edges))
		for _, edge := range edges {
			version := edge.Version
			if version == "" && len(byName[edge.Name]) == 1 {
				version = byName[edge.Name][0]
			}
			if version == "" {
				return nil, invalid
			}
			text := edge.Name
			if len(edge.Extra) > 0 {
				text += "[" + strings.Join(edge.Extra, ",") + "]"
			}
			text += "==" + version
			if edge.Marker != "" {
				text += "; " + edge.Marker
			}
			result = append(result, text)
		}
		return result, nil
	}
	var err error
	input.rawRequirements, err = convert(l.root.Dependencies)
	if err != nil {
		return nil, err
	}
	input.optional = map[string][]string{}
	input.groups = map[string]uvProjectGroup{}
	for name, edges := range l.root.Optional {
		values, err := convert(edges)
		if err != nil {
			return nil, err
		}
		input.optional[name] = values
	}
	for name, edges := range l.root.Groups {
		values, err := convert(edges)
		if err != nil {
			return nil, err
		}
		input.groups[name] = uvProjectGroup{requirements: values}
	}
	if l.root.Metadata != nil {
		for name, requirements := range l.root.Metadata.RequiresDev {
			if len(requirements) == 0 {
				if _, present := input.groups[name]; !present {
					input.groups[name] = uvProjectGroup{}
				}
			}
		}
	}
	// Native frozen keeps locked versions/edges while group selection and
	// configured defaults belong to the current request/manifest.
	for _, names := range [][]string{input.defaultGroups, input.selection.Groups, input.selection.OnlyGroups, input.selection.NoGroups} {
		for _, name := range names {
			if _, ok := input.groups[name]; !ok {
				return nil, invalid
			}
		}
	}
	for _, name := range input.selection.Extras {
		if _, ok := input.optional[name]; !ok {
			return nil, invalid
		}
	}
	return input.SelectedRequirements()
}

func (e UVProjectExport) ProjectTarget(body []byte) (UVProjectExport, error) {
	source, err := e.ProjectionInput()
	if err != nil {
		return UVProjectExport{}, err
	}
	native, err := source.ParseNativeInput(body)
	if err != nil {
		return UVProjectExport{}, err
	}
	active := map[string]string{}
	for _, clause := range native.Clauses() {
		if !clause.Active || !clause.Constraint {
			continue
		}
		requirement, err := ParseUpgradeRequirement(clause.Requirement)
		if err != nil || requirement.exact == "" || active[clause.Project] != "" {
			return UVProjectExport{}, errors.New("UV target variants are ambiguous")
		}
		active[clause.Project] = requirement.exact
	}
	packages := make([]map[string]any, 0, len(active))
	for _, item := range e.portable {
		if active[item.Name] != item.Version {
			continue
		}
		wheels := make([]map[string]any, 0, len(item.Wheels))
		for _, wheel := range item.Wheels {
			wheels = append(wheels, map[string]any{"url": wheel.URL, "size": wheel.Size, "hashes": wheel.Hashes})
		}
		packages = append(packages, map[string]any{"name": item.Name, "version": item.Version, "wheels": wheels})
	}
	view, err := toml.Marshal(map[string]any{"lock-version": "1.0", "created-by": "uv", "requires-python": e.native.python, "packages": packages})
	if err != nil {
		return UVProjectExport{}, err
	}
	selection, err := ParseUVProjectLock(view, runtimeidentity.PythonVersion)
	if err != nil {
		return UVProjectExport{}, err
	}
	e.selection, e.projection, e.target = selection, native, &uvProjectTargetWire{Projection: bytes.Clone(body)}
	return e, nil
}

func (e UVProjectExport) targetMetadataPayload(metadata []requirementsProjectMetadata) ([]byte, error) {
	source, err := e.ProjectionInput()
	if err != nil {
		return nil, err
	}
	control, err := source.Control()
	if err != nil {
		return nil, err
	}
	constraints := make([]string, 0, len(e.selection.packages))
	for _, item := range e.selection.packages {
		constraints = append(constraints, item.Name+"=="+item.Version)
	}
	sort.Strings(constraints)
	return requirementsMetadataPayload(control.Digest().String(), e.projection, []byte(strings.Join(constraints, "\n")), metadata, []requirementsProjectMetadata{})
}

func (e UVProjectExport) bindTargetMetadata(inputs []UVWheelInput, pages []SimpleProject, target WheelTarget, limits WheelLimits, validate func([]byte) ([]byte, error)) ([]Candidate, error) {
	if e.target == nil || !e.projection.Valid() {
		return nil, errors.New("UV native target metadata projection is missing")
	}
	metadata, candidates, err := inspectRequirementsMetadata(e.selection.packages, inputs, target, limits)
	if err != nil {
		return nil, err
	}
	if pages != nil {
		if len(pages) != len(candidates) {
			return nil, errors.New("UV target independent source coverage is incomplete")
		}
		if len(candidates) > 0 {
			if _, err := crossCheckReport(InstallationReport{candidates: candidates}, pages, true); err != nil {
				return nil, err
			}
		}
	}
	payload, err := e.targetMetadataPayload(metadata)
	if err != nil {
		return nil, err
	}
	graph := e.target.Graph
	if validate != nil {
		graph, err = validate(payload)
		if err != nil {
			return nil, err
		}
	}
	return bindRequirementsMetadataGraph(e.projection, payload, graph, candidates)
}

func (e UVProjectExport) BindTargetMetadata(inputs []UVWheelInput, pages []SimpleProject, target WheelTarget, limits WheelLimits, validate func([]byte) ([]byte, error)) (UVProjectExport, error) {
	if validate == nil || len(pages) != len(e.selection.packages) {
		return UVProjectExport{}, errors.New("UV fixed metadata validator is missing")
	}
	if !e.projection.HasActiveRoots() && len(e.selection.packages) == 0 {
		// The shared archive reader intentionally opens no session for no wheels.
		// Its native root projection already proves the explicit empty selection.
		validate = emptyRequirementsMetadataGraph
	}
	var graph []byte
	_, err := e.bindTargetMetadata(inputs, pages, target, limits, func(payload []byte) ([]byte, error) { var err error; graph, err = validate(payload); return graph, err })
	if err != nil {
		return UVProjectExport{}, err
	}
	e.target = &uvProjectTargetWire{Projection: bytes.Clone(e.target.Projection), Graph: bytes.Clone(graph)}
	return e, nil
}

func (e UVProjectExport) IntentControl() (domain.ProjectControlFile, error) {
	controls, err := e.native.input.Controls()
	if err != nil {
		return domain.ProjectControlFile{}, err
	}
	var wire uvProjectIntentWire
	if json.Unmarshal(controls[2].Body(), &wire) != nil {
		return domain.ProjectControlFile{}, errors.New("UV original intent is invalid")
	}
	if e.native.rich {
		if e.target == nil || len(e.target.Graph) == 0 {
			return domain.ProjectControlFile{}, errors.New("UV target metadata proof is missing")
		}
		wire.Target = e.target
	}
	body, err := json.Marshal(wire)
	if err != nil || len(body) > maxReportBytes {
		return domain.ProjectControlFile{}, errors.New("UV target proof exceeds bounds")
	}
	return domain.NewProjectControlFile(UVProjectIntentControl, body, true)
}

func (e UVProjectExport) RestoreTarget(intent domain.ProjectControlFile) (UVProjectExport, error) {
	invalid := errors.New("UV target proof differs from its frozen intent")
	if !intent.Present() || intent.Name() != UVProjectIntentControl || len(intent.Body()) > maxReportBytes || validateNativeJSONKeys(intent.Body(), 6) != nil {
		return UVProjectExport{}, invalid
	}
	var wire uvProjectIntentWire
	decoder := json.NewDecoder(bytes.NewReader(intent.Body()))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&wire) != nil || ensureSingleJSONValue(decoder) != nil || wire.Target == nil || len(wire.Target.Graph) == 0 {
		return UVProjectExport{}, invalid
	}
	result, err := e.ProjectTarget(wire.Target.Projection)
	if err != nil {
		return UVProjectExport{}, err
	}
	result.target = &uvProjectTargetWire{Projection: bytes.Clone(wire.Target.Projection), Graph: bytes.Clone(wire.Target.Graph)}
	canonical, err := result.IntentControl()
	if err != nil || !bytes.Equal(canonical.Body(), intent.Body()) {
		return UVProjectExport{}, invalid
	}
	return result, nil
}

func (e UVProjectExport) SelectResolvedWheels(artifacts []domain.ResolvedArtifact) ([]UVLockedWheel, error) {
	return e.selection.SelectResolvedWheels(artifacts)
}

func (e UVProjectExport) ProjectRecords(artifacts []domain.ResolvedArtifact, inputs []UVWheelInput, target WheelTarget, limits WheelLimits) ([]domain.LockedDependency, []domain.DependencyEdge, error) {
	if !e.native.rich {
		return e.selection.ProjectRecords(artifacts, inputs, target, limits)
	}
	if _, err := e.SelectResolvedWheels(artifacts); err != nil {
		return nil, nil, err
	}
	candidates, err := e.bindTargetMetadata(inputs, nil, target, limits, nil)
	if err != nil {
		return nil, nil, err
	}
	return projectRecordsFromCandidates(artifacts, candidates)
}

func (e UVProjectExport) ProjectRecordsFromInspections(artifacts []domain.ResolvedArtifact, wheels []WheelInspection) ([]domain.LockedDependency, []domain.DependencyEdge, error) {
	if !e.native.rich {
		return ProjectRecordsFromWheels(artifacts, wheels, runtimeidentity.PythonVersion)
	}
	selected, err := e.SelectResolvedWheels(artifacts)
	if err != nil {
		return nil, nil, err
	}
	metadata, candidates, err := requirementsMetadataFromInspections(selected, wheels)
	if err != nil {
		return nil, nil, err
	}
	payload, err := e.targetMetadataPayload(metadata)
	if err != nil {
		return nil, nil, err
	}
	candidates, err = bindRequirementsMetadataGraph(e.projection, payload, e.target.Graph, candidates)
	if err != nil {
		return nil, nil, err
	}
	return projectRecordsFromCandidates(artifacts, candidates)
}
