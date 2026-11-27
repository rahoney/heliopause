package pypi

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash"
	"io"
	"sort"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const RequirementsSelectionControlName = "python-requirements-selection.json"

func ParseRequirementsUpdate(update domain.ProjectDependencyUpdate, target WheelTarget) (RequirementsSelection, error) {
	invalid := errors.New("requirements update differs from its single frozen source")
	if !update.Valid() || update.HasRequestedResolution() || len(update.OriginalControls()) != 2 || len(update.SelectedControls()) != 2 {
		return RequirementsSelection{}, invalid
	}
	var sourceControl, selectedControl domain.ProjectControlFile
	for _, control := range update.SelectedControls() {
		switch control.Name() {
		case RequirementsInputControlName:
			sourceControl = control
		case RequirementsSelectionControlName:
			selectedControl = control
		default:
			return RequirementsSelection{}, invalid
		}
	}
	for _, control := range update.OriginalControls() {
		switch control.Name() {
		case RequirementsInputControlName:
			if !control.Present() || control.Digest() != sourceControl.Digest() {
				return RequirementsSelection{}, invalid
			}
		case RequirementsSelectionControlName:
			if control.Present() || len(control.Body()) != 0 {
				return RequirementsSelection{}, invalid
			}
		default:
			return RequirementsSelection{}, invalid
		}
	}
	source, err := ParseRequirementsControl(sourceControl)
	if err != nil {
		return RequirementsSelection{}, err
	}
	selection, err := ParseRequirementsSelection(source, selectedControl, target)
	if err != nil {
		return RequirementsSelection{}, err
	}
	if err := selection.ValidateSnapshot(update.Snapshot(), update.SelectedControls(), target); err != nil {
		return RequirementsSelection{}, err
	}
	return selection, nil
}

// RequirementsSelection keeps actual client output and separately validated
// metadata semantics. It is selection data; only current inspection and Policy
// may produce a ProjectVerifiedSet.
type RequirementsSelection struct {
	source     RequirementsInput
	native     RequirementsNativeInput
	wire       requirementsSelectionWire
	wheels     []UVLockedWheel
	candidates []Candidate
	target     WheelTarget
}

type requirementsSelectionWire struct {
	Version   int                        `json:"version"`
	Source    string                     `json:"source_sha256"`
	Native    []byte                     `json:"native"`
	Delta     []byte                     `json:"delta"`
	Selection []byte                     `json:"selection"`
	Census    []UVInstalledWheelControls `json:"census"`
	Graph     []byte                     `json:"graph"`
}

func (r RequirementsInput) NewSelection(nativeBody, delta, selected []byte, census []UVInstalledWheelControls, target WheelTarget) (RequirementsSelection, error) {
	total := 0
	for _, body := range [][]byte{nativeBody, delta, selected} {
		if len(body) > RequirementsInputByteLimit-total {
			return RequirementsSelection{}, errors.New("requirements native selection exceeds bounds")
		}
		total += len(body)
	}
	if len(census) > UpgradeInputDistributionLimit {
		return RequirementsSelection{}, errors.New("requirements native census exceeds bounds")
	}
	for _, item := range census {
		for _, body := range [][]byte{item.Metadata, item.Wheel} {
			if len(body) > RequirementsInputByteLimit-total {
				return RequirementsSelection{}, errors.New("requirements native census exceeds bounds")
			}
			total += len(body)
		}
	}
	native, err := r.ParseNativeInput(nativeBody)
	if err != nil {
		return RequirementsSelection{}, err
	}
	client, _, _, err := r.Installation()
	if err != nil {
		return RequirementsSelection{}, err
	}
	control, _ := r.Control()
	result := RequirementsSelection{source: r, native: native, wire: requirementsSelectionWire{Version: 1, Source: control.Digest().String(), Native: bytes.Clone(nativeBody), Delta: bytes.Clone(delta), Selection: bytes.Clone(selected)}}
	if !native.HasActiveRoots() {
		if len(delta)+len(selected)+len(census) != 0 {
			return RequirementsSelection{}, errors.New("empty requirements intent contains an unexpected selection")
		}
		result.wheels = []UVLockedWheel{}
		return result, nil
	}
	if _, err := r.SelectionConstraints(native, delta); err != nil {
		return RequirementsSelection{}, err
	}
	if client == "pip" {
		if len(census) != 0 {
			return RequirementsSelection{}, errors.New("pip selection cannot claim a UV materialization")
		}
		report, err := native.ParsePipReport(selected)
		if err != nil {
			return RequirementsSelection{}, err
		}
		for _, candidate := range report.Candidates() {
			result.wheels = append(result.wheels, UVLockedWheel{URL: candidate.url, Filename: candidate.filename, SHA256: candidate.sha256})
		}
	} else {
		lock, err := ParseUVProjectLock(selected, runtimeidentity.PythonVersion)
		if err != nil {
			return RequirementsSelection{}, err
		}
		result.wheels, err = lock.SelectInstalledWheelControls(census, target)
		if err != nil {
			return RequirementsSelection{}, err
		}
		// Freeze all native census bytes, including their literal metadata paths.
		body, err := json.Marshal(census)
		if err != nil || len(body) > RequirementsInputByteLimit {
			return RequirementsSelection{}, errors.New("requirements native census exceeds bounds")
		}
		if json.Unmarshal(body, &result.wire.Census) != nil {
			return RequirementsSelection{}, errors.New("requirements native census is invalid")
		}
	}
	return result, nil
}

func (r RequirementsSelection) Wheels() []UVLockedWheel {
	return append([]UVLockedWheel{}, r.wheels...)
}

func (r RequirementsSelection) SelectResolvedWheels(artifacts []domain.ResolvedArtifact) ([]UVLockedWheel, error) {
	if len(r.wire.Graph) == 0 {
		return nil, errors.New("requirements selected graph is unavailable")
	}
	packages := make([]UVLockedPackage, 0, len(r.wheels))
	for _, wheel := range r.wheels {
		name, version, _, _, _, err := ParseWheelFilename(wheel.Filename)
		if err != nil {
			return nil, err
		}
		packages = append(packages, UVLockedPackage{Name: name, Version: version, Wheels: []UVLockedWheel{wheel}})
	}
	return selectResolvedProjectWheels(packages, artifacts)
}

// ValidateSnapshot binds native subjects, both controls and the exact target.
// Metadata semantics still require authenticated archive/static readback.
func (r RequirementsSelection) ValidateSnapshot(snapshot domain.ProjectDependencySnapshot, controls []domain.ProjectControlFile, target WheelTarget) error {
	invalid := errors.New("requirements snapshot differs from frozen selection")
	if !snapshot.Valid() || snapshot.Context().Mode() != domain.InstallPythonVenv || snapshot.Source() != PublicPyPIProfile().Source() || len(controls) != 2 || snapshot.DependencyFree() == r.native.HasActiveRoots() {
		return invalid
	}
	if _, err := r.SelectResolvedWheels(snapshot.Dependencies()); err != nil {
		return err
	}
	source, _ := r.source.Control()
	selection, err := r.Control()
	if err != nil {
		return err
	}
	digests := make([]domain.ProjectControlDigest, 0, len(controls))
	seen := map[string]bool{}
	for _, control := range controls {
		if seen[control.Name()] || !control.Present() {
			return invalid
		}
		seen[control.Name()] = true
		switch control.Name() {
		case source.Name():
			if control.Digest() != source.Digest() {
				return invalid
			}
		case selection.Name():
			if control.Digest() != selection.Digest() {
				return invalid
			}
		default:
			return invalid
		}
		digest, err := domain.NewProjectControlDigest(control.Name(), control.Digest())
		if err != nil {
			return err
		}
		digests = append(digests, digest)
	}
	digest, err := uvProjectSelectionDigest(snapshot.Context(), runtimeidentity.PythonVersion, target, selection.Body(), digests, snapshot.Dependencies())
	if err != nil {
		return err
	}
	if digest != snapshot.GraphDigest() {
		return invalid
	}
	sort.Slice(digests, func(i, j int) bool { return digests[i].Name() < digests[j].Name() })
	actual := snapshot.ControlDigests()
	if len(digests) != len(actual) {
		return invalid
	}
	for index, control := range digests {
		if control != actual[index] {
			return invalid
		}
	}
	return nil
}

func (r RequirementsSelection) RuntimeIdentity() string {
	client, _, _, err := r.source.Installation()
	if err != nil {
		return ""
	}
	return client + "-requirements:" + runtimeidentity.RuntimeLockSHA256
}

func (r RequirementsSelection) Control() (domain.ProjectControlFile, error) {
	if len(r.wire.Graph) == 0 {
		return domain.ProjectControlFile{}, errors.New("requirements wheel metadata has not been bound")
	}
	body, err := json.Marshal(r.wire)
	if err != nil || len(body) > RequirementsInputByteLimit {
		return domain.ProjectControlFile{}, errors.New("requirements selection provenance exceeds bounds")
	}
	return domain.NewProjectControlFile(RequirementsSelectionControlName, body, true)
}

func ParseRequirementsSelection(source RequirementsInput, control domain.ProjectControlFile, target WheelTarget) (RequirementsSelection, error) {
	invalid := errors.New("requirements selection provenance differs from its frozen source")
	if control.Name() != RequirementsSelectionControlName || !control.Present() || len(control.Body()) > RequirementsInputByteLimit || validateNativeJSONKeys(control.Body(), 5) != nil {
		return RequirementsSelection{}, invalid
	}
	var wire requirementsSelectionWire
	decoder := json.NewDecoder(bytes.NewReader(control.Body()))
	decoder.DisallowUnknownFields()
	before, err := source.Control()
	if err != nil || decoder.Decode(&wire) != nil || ensureSingleJSONValue(decoder) != nil || wire.Version != 1 || wire.Source != before.Digest().String() || len(wire.Graph) == 0 {
		return RequirementsSelection{}, invalid
	}
	selection, err := source.NewSelection(wire.Native, wire.Delta, wire.Selection, wire.Census, target)
	if err != nil {
		return RequirementsSelection{}, err
	}
	selection.wire.Graph = bytes.Clone(wire.Graph)
	canonical, err := selection.Control()
	if err != nil || !bytes.Equal(canonical.Body(), control.Body()) {
		return RequirementsSelection{}, invalid
	}
	return selection, nil
}

type requirementsProjectMetadata struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	SHA256       string   `json:"sha256,omitempty"`
	Python       string   `json:"requires_python"`
	Requirements []string `json:"requires_dist"`
	Extras       []string `json:"provides_extra"`
}

// BindWheels uses the shared archive/RECORD/budget owner once. The callback is
// the fixed native metadata validator on first selection; cache readback uses
// the exact approved projection and must match the same metadata-input digest.
func (r RequirementsSelection) BindWheels(inputs []UVWheelInput, pages []SimpleProject, target WheelTarget, limits WheelLimits, validate func([]byte) ([]byte, error)) (RequirementsSelection, error) {
	payload, candidates, wheels, err := r.metadataInput(inputs, target, limits)
	if err != nil {
		return RequirementsSelection{}, err
	}
	if len(pages) != len(wheels) {
		return RequirementsSelection{}, errors.New("requirements independent source coverage is incomplete")
	}
	if len(wheels) > 0 {
		if _, err := crossCheckReport(InstallationReport{candidates: candidates}, pages, true); err != nil {
			return RequirementsSelection{}, err
		}
	}
	var graph []byte
	if r.native.HasActiveRoots() {
		if validate == nil {
			return RequirementsSelection{}, errors.New("requirements native metadata validator is unavailable")
		}
		graph, err = validate(payload)
		if err != nil {
			return RequirementsSelection{}, err
		}
	} else {
		graph, err = emptyRequirementsMetadataGraph(payload)
		if err != nil {
			return RequirementsSelection{}, err
		}
	}
	bound, err := r.bindMetadataGraph(payload, graph, candidates)
	if err != nil {
		return RequirementsSelection{}, err
	}
	bound.wheels, bound.target = wheels, target
	return bound, nil
}

func (r RequirementsSelection) metadataInput(inputs []UVWheelInput, target WheelTarget, limits WheelLimits) ([]byte, []Candidate, []UVLockedWheel, error) {
	invalid := errors.New("requirements whole wheel metadata binding is unavailable")
	if !r.native.Valid() || len(inputs) != len(r.wheels) {
		return nil, nil, nil, invalid
	}
	byURL := map[string]UVWheelInput{}
	for _, input := range inputs {
		if _, seen := byURL[input.URL]; seen {
			return nil, nil, nil, invalid
		}
		byURL[input.URL] = input
	}
	packages := make([]UVLockedPackage, 0, len(r.wheels))
	wheels := make([]UVLockedWheel, 0, len(r.wheels))
	for _, wheel := range r.wheels {
		input, ok := byURL[wheel.URL]
		if !ok || wheel.Size != 0 && input.Size != wheel.Size {
			return nil, nil, nil, invalid
		}
		wheel.Size = input.Size
		name, version, _, _, _, err := ParseWheelFilename(wheel.Filename)
		if err != nil {
			return nil, nil, nil, err
		}
		packages = append(packages, UVLockedPackage{Name: name, Version: version, Wheels: []UVLockedWheel{wheel}})
		wheels = append(wheels, wheel)
	}
	metadata, candidates, err := inspectRequirementsMetadata(packages, inputs, target, limits)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := verifyRequirementsHashes(r.native, wheels, inputs); err != nil {
		return nil, nil, nil, err
	}
	payload, err := r.metadataPayload(metadata)
	if err != nil {
		return nil, nil, nil, err
	}
	return payload, candidates, wheels, nil
}

// Both requirements installation and UV project selection authenticate the same
// archive metadata without importing the selected packages.
func inspectRequirementsMetadata(packages []UVLockedPackage, inputs []UVWheelInput, target WheelTarget, limits WheelLimits) ([]requirementsProjectMetadata, []Candidate, error) {
	metadata := make([]requirementsProjectMetadata, 0, len(packages))
	candidates := make([]Candidate, 0, len(packages))
	err := inspectSelectedWheelArchives(packages, runtimeidentity.PythonVersion, inputs, target, limits, func(project UVLockedPackage, wheel UVLockedWheel, inspection WheelInspection) error {
		metadata = append(metadata, requirementsProjectMetadata{project.Name, project.Version, wheel.SHA256, inspection.RequiresPython, append([]string{}, inspection.RequiresDist...), append([]string{}, inspection.providesExtra...)})
		candidates = append(candidates, Candidate{project: project.Name, version: project.Version, source: PublicPyPIProfile().Source(), filename: wheel.Filename, url: wheel.URL, sha256: wheel.SHA256, requiresPython: inspection.RequiresPython})
		return nil
	})
	return metadata, candidates, err
}

func (r RequirementsSelection) metadataPayload(metadata []requirementsProjectMetadata) ([]byte, error) {
	var err error
	constraints := []byte{}
	if r.native.HasActiveRoots() {
		constraints, err = r.source.SelectionConstraints(r.native, r.wire.Delta)
		if err != nil {
			return nil, err
		}
	}
	// Original metadata is frozen data, not new requested roots or inspection
	// approval. Only selected wheel entries carry an archive SHA256.
	_, _, files, err := r.source.Installation()
	if err != nil {
		return nil, err
	}
	originals := make([]requirementsProjectMetadata, 0, len(files))
	for _, file := range files {
		project, headers, err := parseInstalledMetadataIdentity(file)
		if err != nil || len(headers.all("requires-python")) > 1 {
			return nil, errors.New("requirements original metadata identity is unavailable")
		}
		python, _ := headers.first("requires-python")
		extras := headers.all("provides-extra")
		for _, extra := range extras {
			if _, err := NormalizeProjectName(extra); err != nil {
				return nil, err
			}
		}
		originals = append(originals, requirementsProjectMetadata{Name: project.project, Version: project.version, Python: python, Requirements: append([]string{}, headers.all("requires-dist")...), Extras: append([]string{}, extras...)})
	}
	return requirementsMetadataPayload(r.wire.Source, r.native, constraints, metadata, originals)
}

func requirementsMetadataPayload(source string, native RequirementsNativeInput, constraints []byte, metadata, originals []requirementsProjectMetadata) ([]byte, error) {
	sort.Slice(metadata, func(i, j int) bool { return metadata[i].Name < metadata[j].Name })
	payload, err := json.Marshal(struct {
		Version       int                           `json:"version"`
		Source        string                        `json:"source_sha256"`
		Clauses       []RequirementsClause          `json:"clauses"`
		Constraints   string                        `json:"selected_constraints"`
		RequireHashes bool                          `json:"require_hashes"`
		Wheels        []requirementsProjectMetadata `json:"wheels"`
		Originals     []requirementsProjectMetadata `json:"originals"`
	}{1, source, native.Clauses(), string(constraints), native.RequireHashes(), metadata, originals})
	if err != nil || len(payload) > RequirementsInputByteLimit {
		return nil, errors.New("requirements metadata payload exceeds bounds")
	}
	return payload, nil
}

func verifyRequirementsHashes(native RequirementsNativeInput, wheels []UVLockedWheel, inputs []UVWheelInput) error {
	byURL := map[string]UVWheelInput{}
	for _, input := range inputs {
		byURL[input.URL] = input
	}
	byProject := map[string][]RequirementsClause{}
	for _, clause := range native.clauses {
		if clause.Active && len(clause.Hashes) > 0 {
			byProject[clause.Project] = append(byProject[clause.Project], clause)
		}
	}
	for _, wheel := range wheels {
		project, _, _, _, _, err := ParseWheelFilename(wheel.Filename)
		if err != nil {
			return err
		}
		digests := map[string]string{"sha256": wheel.SHA256}
		hashers := map[string]hash.Hash{}
		for _, clause := range byProject[project] {
			for algorithm := range clause.Hashes {
				switch algorithm {
				case "sha384":
					hashers[algorithm] = sha512.New384()
				case "sha512":
					hashers[algorithm] = sha512.New()
				}
			}
		}
		if len(hashers) > 0 {
			hashers["sha256"] = sha256.New()
			writers := make([]io.Writer, 0, len(hashers))
			for _, hasher := range hashers {
				writers = append(writers, hasher)
			}
			input := byURL[wheel.URL]
			if count, err := io.Copy(io.MultiWriter(writers...), io.NewSectionReader(input.Reader, 0, input.Size)); err != nil || count != input.Size {
				return errors.New("requirements archive changed while hashing")
			}
			for algorithm, hasher := range hashers {
				digests[algorithm] = hex.EncodeToString(hasher.Sum(nil))
			}
			if digests["sha256"] != wheel.SHA256 {
				return errors.New("requirements archive changed while hashing")
			}
		}
		for _, clause := range byProject[project] {
			matched := false
			for algorithm, values := range clause.Hashes {
				for _, value := range values {
					matched = matched || digests[algorithm] == value
				}
			}
			if !matched {
				return errors.New("requirements hash differs from the exact selected archive")
			}
		}
	}
	return nil
}

type requirementsMetadataGraph struct {
	Version int        `json:"version"`
	Input   string     `json:"input_sha256"`
	Python  string     `json:"python"`
	Pip     string     `json:"pip"`
	Roots   []string   `json:"roots"`
	Edges   [][]string `json:"edges"`
}

func emptyRequirementsMetadataGraph(payload []byte) ([]byte, error) {
	sum := sha256.Sum256(payload)
	return json.Marshal(requirementsMetadataGraph{Version: 1, Input: hex.EncodeToString(sum[:]), Python: runtimeidentity.PythonVersion, Pip: runtimeidentity.PipVersion, Roots: []string{}, Edges: [][]string{}})
}

func (r RequirementsSelection) bindMetadataGraph(payload, body []byte, candidates []Candidate) (RequirementsSelection, error) {
	bound, err := bindRequirementsMetadataGraph(r.native, payload, body, candidates)
	if err != nil {
		return RequirementsSelection{}, err
	}
	r.wire.Graph, r.candidates = bytes.Clone(body), bound
	return r, nil
}

func bindRequirementsMetadataGraph(native RequirementsNativeInput, payload, body []byte, candidates []Candidate) ([]Candidate, error) {
	invalid := errors.New("requirements native graph does not cover the authenticated metadata")
	if len(body) == 0 || len(body) > RequirementsInputByteLimit || validateNativeJSONKeys(body, 4) != nil {
		return nil, invalid
	}
	var graph requirementsMetadataGraph
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	sum := sha256.Sum256(payload)
	if decoder.Decode(&graph) != nil || ensureSingleJSONValue(decoder) != nil || graph.Version != 1 || graph.Input != hex.EncodeToString(sum[:]) || graph.Python != runtimeidentity.PythonVersion || graph.Pip != runtimeidentity.PipVersion || graph.Roots == nil || graph.Edges == nil {
		return nil, invalid
	}
	byName := map[string]int{}
	for index, candidate := range candidates {
		if _, seen := byName[candidate.project]; seen {
			return nil, invalid
		}
		byName[candidate.project] = index
	}
	expectedRoots := map[string]bool{}
	for _, clause := range native.clauses {
		if clause.Active && !clause.Constraint {
			expectedRoots[clause.Project] = true
		}
	}
	if len(graph.Roots) != len(expectedRoots) {
		return nil, invalid
	}
	for _, root := range graph.Roots {
		if !expectedRoots[root] {
			return nil, invalid
		}
		delete(expectedRoots, root)
		if _, ok := byName[root]; !ok {
			return nil, invalid
		}
	}
	edges := map[[2]string]bool{}
	for _, edge := range graph.Edges {
		if len(edge) != 2 || edge[0] == edge[1] {
			return nil, invalid
		}
		key := [2]string{edge[0], edge[1]}
		from, fromOK := byName[edge[0]]
		_, toOK := byName[edge[1]]
		if !fromOK || !toOK || edges[key] {
			return nil, invalid
		}
		edges[key] = true
		candidates[from].dependencies = append(candidates[from].dependencies, edge[1])
	}
	reached := map[string]bool{}
	pending := append([]string(nil), graph.Roots...)
	for len(pending) > 0 {
		name := pending[0]
		pending = pending[1:]
		if reached[name] {
			continue
		}
		reached[name] = true
		pending = append(pending, candidates[byName[name]].dependencies...)
	}
	if len(reached) != len(candidates) {
		return nil, invalid
	}
	for index := range candidates {
		sort.Strings(candidates[index].dependencies)
	}
	return candidates, nil
}

func (r RequirementsSelection) Snapshot(install domain.InstallContext, controls []domain.ProjectControlFile, target WheelTarget) (domain.ProjectDependencySnapshot, error) {
	control, err := r.Control()
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	if r.target != target || len(r.candidates) != len(r.wheels) {
		return domain.ProjectDependencySnapshot{}, errors.New("requirements snapshot needs its bound target and complete metadata")
	}
	sourceControl, _ := r.source.Control()
	digests := make([]domain.ProjectControlDigest, 0, len(controls))
	found, foundSource := false, false
	for _, item := range controls {
		if item.Name() == sourceControl.Name() {
			if !item.Present() || item.Digest() != sourceControl.Digest() {
				return domain.ProjectDependencySnapshot{}, errors.New("requirements source control changed")
			}
			foundSource = true
		}
		if item.Name() == control.Name() {
			if !item.Present() || item.Digest() != control.Digest() {
				return domain.ProjectDependencySnapshot{}, errors.New("requirements selected control changed")
			}
			found = true
		}
		digest, err := domain.NewProjectControlDigest(item.Name(), item.Digest())
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		digests = append(digests, digest)
	}
	if !found || !foundSource || !install.Valid() || install.Mode() != domain.InstallPythonVenv {
		return domain.ProjectDependencySnapshot{}, errors.New("requirements snapshot needs its actual additive venv context and selection")
	}
	artifacts := make([]domain.ResolvedArtifact, 0, len(r.candidates))
	for _, candidate := range r.candidates {
		id, err := domain.NewResolvedArtifactIdentity(candidate.source, candidate.project, candidate.version, "wheel")
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		artifact, err := domain.NewResolvedArtifact(id, candidate.url, "sha256:"+candidate.sha256)
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		artifacts = append(artifacts, artifact)
	}
	digest, err := uvProjectSelectionDigest(install, runtimeidentity.PythonVersion, target, control.Body(), digests, artifacts)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	if len(artifacts) == 0 {
		return domain.NewDependencyFreeProjectSnapshot(install, PublicPyPIProfile().Source(), digests, digest)
	}
	return domain.NewProjectDependencySnapshot(install, PublicPyPIProfile().Source(), digests, artifacts, digest)
}

func (r RequirementsSelection) ProjectRecords(artifacts []domain.ResolvedArtifact, inputs []UVWheelInput, target WheelTarget, limits WheelLimits) ([]domain.LockedDependency, []domain.DependencyEdge, error) {
	if len(r.wire.Graph) == 0 {
		return nil, nil, errors.New("requirements approved metadata projection is missing")
	}
	payload, candidates, _, err := r.metadataInput(inputs, target, limits)
	if err != nil {
		return nil, nil, err
	}
	bound, err := r.bindMetadataGraph(payload, r.wire.Graph, candidates)
	if err != nil {
		return nil, nil, err
	}
	return projectRecordsFromCandidates(artifacts, bound.candidates)
}

// ProjectRecordsFromInspections consumes the composite inspector's completed
// static-all gate. Exact observed/declared digest and metadata-input readback
// bind the saved native semantics; no bounded CUDA grammar is substituted.
func (r RequirementsSelection) ProjectRecordsFromInspections(artifacts []domain.ResolvedArtifact, wheels []WheelInspection) ([]domain.LockedDependency, []domain.DependencyEdge, error) {
	selected, err := r.SelectResolvedWheels(artifacts)
	if err != nil {
		return nil, nil, err
	}
	metadata, candidates, err := requirementsMetadataFromInspections(selected, wheels)
	if err != nil {
		return nil, nil, err
	}
	payload, err := r.metadataPayload(metadata)
	if err != nil {
		return nil, nil, err
	}
	bound, err := r.bindMetadataGraph(payload, r.wire.Graph, candidates)
	if err != nil {
		return nil, nil, err
	}
	return projectRecordsFromCandidates(artifacts, bound.candidates)
}

func requirementsMetadataFromInspections(selected []UVLockedWheel, wheels []WheelInspection) ([]requirementsProjectMetadata, []Candidate, error) {
	if len(wheels) != len(selected) {
		return nil, nil, errors.New("requirements static coverage is incomplete")
	}
	inspections := make(map[string]WheelInspection, len(wheels))
	for _, wheel := range wheels {
		if _, seen := inspections[wheel.Project]; seen {
			return nil, nil, errors.New("requirements static subject is repeated")
		}
		inspections[wheel.Project] = wheel
	}
	metadata := make([]requirementsProjectMetadata, 0, len(selected))
	candidates := make([]Candidate, 0, len(selected))
	for _, wheel := range selected {
		name, version, _, _, _, err := ParseWheelFilename(wheel.Filename)
		if err != nil {
			return nil, nil, err
		}
		inspection, ok := inspections[name]
		if !ok || inspection.Version != version || inspection.Filename != wheel.Filename || inspection.DeclaredSHA256 != wheel.SHA256 || inspection.ObservedSHA256 != wheel.SHA256 {
			return nil, nil, errors.New("requirements static wheel differs from native selection")
		}
		metadata = append(metadata, requirementsProjectMetadata{name, version, wheel.SHA256, inspection.RequiresPython, append([]string{}, inspection.RequiresDist...), append([]string{}, inspection.providesExtra...)})
		candidates = append(candidates, Candidate{project: name, version: version, source: PublicPyPIProfile().Source(), filename: wheel.Filename, url: wheel.URL, sha256: wheel.SHA256})
	}
	return metadata, candidates, nil
}
