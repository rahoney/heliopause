package pypi

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// RequirementsNativeInput is a pinned parser's projection of frozen input,
// not artifact selection or Policy approval. Every clause remains separate
// for native conjunction, extra and hash handling.
type RequirementsNativeInput struct {
	inputDigest domain.ContentDigest
	clauses     []RequirementsClause
	hashes      bool
	body        []byte
}

type RequirementsClause struct {
	Active      bool                `json:"active"`
	Requirement string              `json:"requirement"`
	Project     string              `json:"project"`
	Extras      []string            `json:"extras"`
	Constraint  bool                `json:"constraint"`
	Hashes      map[string][]string `json:"hashes"`
}

func (r RequirementsInput) ParseNativeInput(body []byte) (RequirementsNativeInput, error) {
	invalid := errors.New("native requirements projection differs from frozen input or runtime")
	control, err := r.Control()
	parserControl, parserErr := r.ParserControl()
	if err != nil || parserErr != nil || len(body) == 0 || len(body) > maxReportBytes || validateNativeJSONKeys(body, 5) != nil {
		return RequirementsNativeInput{}, invalid
	}
	var wire struct {
		Version       int    `json:"version"`
		InputSHA256   string `json:"input_sha256"`
		ParserSHA256  string `json:"parser_sha256"`
		Python        string `json:"python"`
		Pip           string `json:"pip"`
		RequireHashes *bool  `json:"require_hashes"`
		Error         *struct {
			Source string `json:"source"`
			Line   int    `json:"line"`
			Code   string `json:"code"`
		} `json:"error"`
		Clauses []struct {
			Requirement string              `json:"requirement"`
			Project     string              `json:"project"`
			Extras      []string            `json:"extras"`
			Constraint  *bool               `json:"constraint"`
			Active      *bool               `json:"active"`
			Source      string              `json:"source"`
			Line        int                 `json:"line"`
			TextSHA256  string              `json:"text_sha256"`
			Hashes      map[string][]string `json:"hashes"`
		} `json:"clauses"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&wire) != nil || ensureSingleJSONValue(decoder) != nil || wire.Version != 1 || wire.InputSHA256 != control.Digest().String() || wire.ParserSHA256 != parserControl.Digest().String() || wire.Python != runtimeidentity.PythonVersion || wire.Pip != runtimeidentity.PipVersion || len(wire.Clauses) > maxPyPIReportEntries {
		return RequirementsNativeInput{}, invalid
	}
	if wire.Error != nil {
		// A parser refusal is bounded DATA, never a successful empty selection.
		if wire.Error.Code != "INVALID_REQUIREMENT" || wire.Clauses != nil || wire.RequireHashes != nil {
			return RequirementsNativeInput{}, invalid
		}
		var source requirementsInputWire
		if json.Unmarshal(r.body, &source) != nil {
			return RequirementsNativeInput{}, invalid
		}
		found := wire.Error.Source == "@positional" && wire.Error.Line > 0 && wire.Error.Line <= len(source.Positional)
		if body, ok := source.Files[wire.Error.Source]; ok {
			document, err := ParseRequirementsFile(body)
			if err != nil {
				return RequirementsNativeInput{}, invalid
			}
			for _, line := range document.Lines() {
				if line.Number == wire.Error.Line {
					found = true
					break
				}
			}
		}
		if !found {
			return RequirementsNativeInput{}, invalid
		}
		return RequirementsNativeInput{}, fmt.Errorf("requirements %q:%d: invalid or unsupported syntax for the pinned runtime", wire.Error.Source, wire.Error.Line)
	}
	if wire.Clauses == nil || wire.RequireHashes == nil {
		return RequirementsNativeInput{}, invalid
	}
	var original requirementsInputWire
	if json.Unmarshal(r.body, &original) != nil || original.RequireHashes && !*wire.RequireHashes {
		return RequirementsNativeInput{}, invalid
	}
	expectedSources, err := r.nativeClauseSources()
	if err != nil || len(expectedSources) != len(wire.Clauses) {
		return RequirementsNativeInput{}, invalid
	}
	clauses := make([]RequirementsClause, 0, len(wire.Clauses))
	for index, clause := range wire.Clauses {
		source := expectedSources[index]
		if clause.Active == nil || clause.Constraint == nil || clause.Source != source.file || clause.Line != source.line || clause.TextSHA256 != source.hash || *clause.Constraint != source.constraint || clause.Project != source.project {
			return RequirementsNativeInput{}, invalid
		}
		name, err := NormalizeProjectName(clause.Project)
		if err != nil || name != clause.Project || clause.Requirement == "" || len(clause.Requirement) > 4096 || strings.ContainsAny(clause.Requirement, "@:/\\;\x00\r\n\t ") || clause.Extras == nil || clause.Constraint == nil || clause.Hashes == nil {
			return RequirementsNativeInput{}, invalid
		}
		head := boundedRequirementHead.FindStringSubmatch(clause.Requirement)
		if len(head) != 4 || head[1] != clause.Project {
			return RequirementsNativeInput{}, invalid
		}
		expected := map[string]bool{}
		if head[2] != "" {
			for _, extra := range strings.Split(head[2], ",") {
				expected[extra] = true
			}
		}
		if len(expected) != len(clause.Extras) || !sort.StringsAreSorted(clause.Extras) {
			return RequirementsNativeInput{}, invalid
		}
		for _, extra := range clause.Extras {
			name, err := NormalizeProjectName(extra)
			if err != nil || name != extra || !expected[extra] {
				return RequirementsNativeInput{}, invalid
			}
			delete(expected, extra)
		}
		for algorithm, values := range clause.Hashes {
			length := 0
			switch algorithm {
			case "sha256":
				length = sha256.Size * 2
			case "sha384":
				length = sha512.Size384 * 2
			case "sha512":
				length = sha512.Size * 2
			default:
				return RequirementsNativeInput{}, invalid
			}
			if len(values) == 0 || len(values) > maxUVWheelAlternatives {
				return RequirementsNativeInput{}, invalid
			}
			for _, value := range values {
				if len(value) != length {
					return RequirementsNativeInput{}, invalid
				}
				for _, c := range value {
					if !(c >= 'a' && c <= 'f' || c >= '0' && c <= '9') {
						return RequirementsNativeInput{}, invalid
					}
				}
			}
		}
		clauses = append(clauses, RequirementsClause{*clause.Active, clause.Requirement, clause.Project, clause.Extras, *clause.Constraint, clause.Hashes})
	}
	return RequirementsNativeInput{inputDigest: control.Digest(), clauses: clauses, hashes: *wire.RequireHashes, body: bytes.Clone(body)}, nil
}

func (r RequirementsNativeInput) Valid() bool {
	return r.clauses != nil && r.inputDigest.String() != ""
}
func (r RequirementsNativeInput) RequireHashes() bool { return r.hashes }
func (r RequirementsNativeInput) Clauses() []RequirementsClause {
	result := append([]RequirementsClause{}, r.clauses...)
	for i := range result {
		result[i].Extras = append([]string{}, result[i].Extras...)
		result[i].Hashes = map[string][]string{}
		for name, values := range r.clauses[i].Hashes {
			result[i].Hashes[name] = append([]string(nil), values...)
		}
	}
	return result
}

// Source membership is derived from frozen bytes, not the parser's returned
// clause list. An omitted conditional clause must not become an empty success.
type requirementsClauseSource struct {
	file       string
	line       int
	hash       string
	constraint bool
	project    string
}

func (r RequirementsInput) nativeClauseSources() ([]requirementsClauseSource, error) {
	var wire requirementsInputWire
	if json.Unmarshal(r.body, &wire) != nil {
		return nil, errors.New("requirements source is unavailable")
	}
	documents := map[string]RequirementsFile{}
	for name, body := range wire.Files {
		document, err := ParseRequirementsFile(body)
		if err != nil {
			return nil, err
		}
		documents[name] = document
	}
	var result []requirementsClauseSource
	add := func(file string, line int, text string, constraint bool) error {
		if text == "--require-hashes" {
			return nil
		}
		head := boundedRequirementHead.FindStringSubmatch(text)
		if len(head) != 4 {
			return errors.New("requirements project name is unavailable")
		}
		project, err := NormalizeProjectName(head[1])
		if err != nil {
			return err
		}
		hash := sha256.Sum256([]byte(text))
		result = append(result, requirementsClauseSource{file, line, hex.EncodeToString(hash[:]), constraint, project})
		if len(result) > RequirementsClauseLimit {
			return errors.New("requirements clauses exceed bounds")
		}
		return nil
	}
	type visitKey struct {
		name       string
		constraint bool
	}
	visited := map[visitKey]bool{}
	var visit func(string, bool) error
	visit = func(name string, constraint bool) error {
		key := visitKey{name, constraint}
		if visited[key] {
			return nil
		}
		visited[key] = true
		document, ok := documents[name]
		if !ok {
			return errors.New("requirements include is unavailable")
		}
		for _, line := range document.lines {
			if line.Include != "" {
				if err := visit(path.Join(path.Dir(name), line.Include), line.Constraint); err != nil {
					return err
				}
			} else if err := add(name, line.Number, line.Text, constraint); err != nil {
				return err
			}
		}
		return nil
	}
	for _, source := range wire.Sources {
		if err := visit(source.Name, source.Constraint); err != nil {
			return nil, err
		}
	}
	for index, text := range wire.Positional {
		if err := add("@positional", index+1, strings.TrimSpace(text), false); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (r RequirementsNativeInput) InputDigest() domain.ContentDigest { return r.inputDigest }

func (r RequirementsNativeInput) Body() []byte { return bytes.Clone(r.body) }

func (r RequirementsNativeInput) HasActiveRoots() bool {
	for _, clause := range r.clauses {
		if clause.Active && !clause.Constraint {
			return true
		}
	}
	return false
}

// ParsePipReport validates the actual multi-root native report as selection
// data. Dependency semantics are bound later to authenticated whole wheels;
// neither report metadata nor requested=true supplies inspection authority.
func (r RequirementsNativeInput) ParsePipReport(body []byte) (InstallationReport, error) {
	if !r.Valid() || !r.HasActiveRoots() {
		return InstallationReport{}, errors.New("requirements report has no active request")
	}
	report, err := parseUpgradeDeltaReport(body, runtimeidentity.PipVersion, runtimeidentity.PythonVersion)
	if err != nil {
		return InstallationReport{}, err
	}
	roots := map[string]bool{}
	for _, clause := range r.clauses {
		if clause.Active && !clause.Constraint {
			roots[clause.Project] = true
		}
	}
	seen := map[string]bool{}
	candidates := make([]Candidate, 0, len(report.Install))
	for _, item := range report.Install {
		candidate, err := parseReportCandidateWithDependencies(item, PublicPyPIProfile(), nil, func(string) (string, bool, error) { return "", false, nil })
		if err != nil || seen[candidate.project] || !IsFinalVersion(candidate.version) || candidate.primary != roots[candidate.project] {
			return InstallationReport{}, errors.New("requirements final pip subjects differ or repeat")
		}
		name, version, _, _, _, err := ParseWheelFilename(candidate.filename)
		if err != nil || name != candidate.project || version != candidate.version {
			return InstallationReport{}, errors.New("requirements selection requires matching wheel identities")
		}
		seen[candidate.project] = true
		candidates = append(candidates, candidate)
	}
	for root := range roots {
		if !seen[root] {
			return InstallationReport{}, errors.New("requirements final pip selection omitted a root")
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].project < candidates[j].project })
	return InstallationReport{candidates: candidates}, nil
}

// ResolverFiles contains normalized declarations only. Include traversal and
// marker evaluation have already consumed the complete frozen source closure.
func (r RequirementsNativeInput) ResolverFiles() (requirements, constraints []byte, err error) {
	if !r.Valid() {
		return nil, nil, errors.New("requirements native input is unavailable")
	}
	var roots, limits strings.Builder
	if r.hashes {
		roots.WriteString("--require-hashes\n")
	}
	for _, clause := range r.clauses {
		if !clause.Active {
			continue
		}
		line := clause.Requirement
		algorithms := make([]string, 0, len(clause.Hashes))
		for algorithm := range clause.Hashes {
			algorithms = append(algorithms, algorithm)
		}
		sort.Strings(algorithms)
		for _, algorithm := range algorithms {
			for _, value := range clause.Hashes[algorithm] {
				line += " --hash=" + algorithm + ":" + value
			}
		}
		if clause.Constraint {
			limits.WriteString(line + "\n")
		} else {
			roots.WriteString(line + "\n")
		}
	}
	return []byte(roots.String()), []byte(limits.String()), nil
}

// SelectionConstraints freezes the installed-aware native proposal without
// promoting unrelated installed packages into requested graph roots.
func (r RequirementsInput) SelectionConstraints(native RequirementsNativeInput, delta []byte) ([]byte, error) {
	control, err := r.Control()
	if err != nil || !native.Valid() || native.InputDigest() != control.Digest() {
		return nil, errors.New("requirements proposal source differs")
	}
	client, _, files, err := r.Installation()
	if err != nil {
		return nil, err
	}
	installed := make([]InstalledProject, 0, len(files))
	versions := map[string]string{}
	for _, file := range files {
		project, _, err := parseInstalledMetadataIdentity(file)
		if err != nil {
			return nil, err
		}
		installed = append(installed, project)
		versions[project.project] = project.version
	}
	if client == "uv" {
		selected, err := parseUVSelectionDelta(delta, installed)
		if err != nil {
			return nil, err
		}
		versions = selected.versions
	} else {
		report, err := parseUpgradeDeltaReport(delta, runtimeidentity.PipVersion, runtimeidentity.PythonVersion)
		if err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		for _, item := range report.Install {
			candidate, err := parseReportCandidateWithDependencies(item, PublicPyPIProfile(), nil, func(string) (string, bool, error) { return "", false, nil })
			if err != nil || seen[candidate.project] || !IsFinalVersion(candidate.version) {
				return nil, errors.New("requirements pip proposal is invalid or repeated")
			}
			seen[candidate.project] = true
			versions[candidate.project] = candidate.version
		}
	}
	if len(versions) > maxPyPIReportEntries {
		return nil, errors.New("requirements selected versions exceed bounds")
	}
	for _, clause := range native.clauses {
		if clause.Active && !clause.Constraint && versions[clause.Project] == "" {
			return nil, errors.New("requirements native proposal omitted a root")
		}
	}
	// Reuse exact version constraints. General retained metadata obligations are
	// validated by the pinned PEP parser together with the authenticated wheels;
	// the legacy bounded grammar remains exclusive to the existing upgrade lane.
	return selectedInstalledVersionConstraints(installed, runtimeidentity.PythonVersion, versions, nil)
}
