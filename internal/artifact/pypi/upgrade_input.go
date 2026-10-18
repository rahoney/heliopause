package pypi

import (
	"bytes"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const UpgradeInputControlName = "pypi-update-input.json"
const upgradeInputLimit = 4 << 20

// UpgradeRequirement is the supported native-shaped selection, not a generic
// pip argument. All package execution stays behind the existing resolver.
type UpgradeRequirement struct {
	project, requirement, exact string
	bounded                     *BoundedRequirement
	source                      domain.SourceID
}

func ParseUpgradeRequirement(value string) (UpgradeRequirement, error) {
	return ParseUpgradeRequirementForSource(value, PublicPyPIProfile().Source())
}

// ParseUpgradeRequirementForSource binds native upgrade intent to one existing
// canonical profile. A named profile cannot select an ordinary public project.
func ParseUpgradeRequirementForSource(value string, source domain.SourceID) (result UpgradeRequirement, resultErr error) {
	profile, ok := ProfileForSource(source)
	if !ok {
		return result, errors.New("upgrade source profile is unsupported")
	}
	defer func() {
		if resultErr == nil {
			if IsPyTorchSource(source) && !sourceOwnsProject(profile, source, result.project) {
				result, resultErr = UpgradeRequirement{}, errors.New("upgrade root is not owned by the named source")
				return
			}
			result.source = source
		}
	}()
	if value == "" || len(value) > 512 || value != strings.TrimSpace(value) || strings.ContainsAny(value, "[];:/\\\x00\r\n\t") {
		return UpgradeRequirement{}, errors.New("pip upgrade requires one supported project requirement")
	}
	if strings.Contains(value, "@") {
		reference, err := ParseReferenceForSource(value, source)
		if err != nil {
			return UpgradeRequirement{}, err
		}
		project, _ := RequestedProject(reference)
		version, present, _ := RequestedVersion(reference)
		if !present {
			return UpgradeRequirement{}, errors.New("pip upgrade exact reference is invalid")
		}
		return UpgradeRequirement{project: project, requirement: project + "==" + version, exact: version}, nil
	}
	if strings.Count(value, "==") == 1 && !strings.ContainsAny(value, ",<>~*") {
		parts := strings.SplitN(value, "==", 2)
		project, err := NormalizeProjectName(strings.TrimSpace(parts[0]))
		version, versionErr := normalizeVersionForProfile(strings.TrimSpace(parts[1]), IsPyTorchSource(source))
		if err != nil || versionErr != nil {
			return UpgradeRequirement{}, errors.New("pip upgrade exact requirement is invalid")
		}
		return UpgradeRequirement{project: project, requirement: project + "==" + version, exact: version}, nil
	}
	requirement, err := ParseBoundedRequirement(value)
	if err != nil || len(requirement.Extras()) != 0 {
		return UpgradeRequirement{}, errors.New("pip upgrade supports a bare project, exact version or bounded numeric specifier")
	}
	project := requirement.Project()
	match := boundedRequirementHead.FindStringSubmatch(value)
	specifier := strings.TrimSpace(match[3])
	result = UpgradeRequirement{project: project, requirement: project + specifier}
	if specifier != "" {
		result.bounded = &requirement
	}
	return result, nil
}

func (r UpgradeRequirement) Project() string     { return r.project }
func (r UpgradeRequirement) Requirement() string { return r.requirement }
func (r UpgradeRequirement) Reference() domain.ArtifactReference {
	reference, _ := ParseReferenceForSource(r.project, r.source)
	return reference
}

// AcceptsVersion preserves numeric PEP 440 equality across trailing zero
// components. Other explicitly pinned versions retain canonical exact identity;
// automatic root selection never opts into a prerelease.
func (r UpgradeRequirement) AcceptsVersion(version string) bool {
	profile, ok := ProfileForSource(r.source)
	canonical, err := normalizeVersionForProfile(version, IsPyTorchSource(r.source))
	if !ok || r.project == "" || err != nil || canonical != version {
		return false
	}
	comparisonVersion := version
	if IsPyTorchSource(r.source) && !strings.Contains(r.exact, "+") {
		comparisonVersion = strings.SplitN(version, "+", 2)[0]
	}
	if r.exact != "" {
		if numericRelease.MatchString(r.exact) && numericRelease.MatchString(comparisonVersion) {
			constraint, err := ParseBoundedRequirement(r.project + "==" + r.exact)
			return err == nil && constraint.Satisfies(comparisonVersion)
		}
		return comparisonVersion == r.exact
	}
	if !isFinalVersionForProfile(version, profile) {
		return false
	}
	return r.bounded == nil || r.bounded.Satisfies(comparisonVersion)
}

// InstalledMetadataFile contains inert data only. Its directory is one exact
// dist-info basename, never a target interpreter, import path or executable.
type InstalledMetadataFile struct {
	Directory string `json:"dist_info"`
	Body      []byte `json:"metadata"`
}

type InstalledProject struct {
	project, version, directory string
	metadata                    []byte
	requirements                []string
	extraRequirements           []BoundedRequirement
}

func (p InstalledProject) Project() string { return p.project }
func (p InstalledProject) Version() string { return p.version }

// Requirements returns active version constraints. Extra obligations are kept
// separately by UpgradeInput.RequiredUpgradeExtras and the immutable metadata.
func (p InstalledProject) Requirements() []string {
	return append([]string(nil), p.requirements...)
}

func parseUpgradeInstalledMetadata(file InstalledMetadataFile, python string) (InstalledProject, error) {
	return parseUpgradeInstalledMetadataWithExtras(file, python, nil)
}

func parseUpgradeInstalledMetadataWithExtras(file InstalledMetadataFile, python string, activeExtras []string) (InstalledProject, error) {
	if len(file.Body) == 0 || len(file.Body) > defaultWheelMaxMetadata || len(file.Directory) == 0 || len(file.Directory) > 255 {
		return InstalledProject{}, errors.New("installed dependency metadata is empty")
	}
	headers, err := parseRFC822Metadata(file.Body, defaultWheelMaxMetadata)
	if err != nil || len(headers.all("name")) != 1 || len(headers.all("version")) != 1 || len(headers.all("requires-dist")) > maxPyPIReportEntries {
		return InstalledProject{}, errors.New("installed dependency metadata is invalid or ambiguous")
	}
	project, nameErr := NormalizeProjectName(headers.all("name")[0])
	version, versionErr := NormalizeInstalledVersion(headers.all("version")[0])
	if nameErr != nil || versionErr != nil || file.Directory != wheelDistInfo(project, version) {
		return InstalledProject{}, errors.New("installed dist-info identity differs from metadata")
	}
	result := InstalledProject{project: project, version: version, directory: file.Directory, metadata: append([]byte(nil), file.Body...)}
	for _, requirement := range headers.all("requires-dist") {
		if len(requirement) > 1024 || strings.ContainsAny(requirement, "\r\n\x00\t") || requirement != strings.TrimSpace(requirement) {
			return InstalledProject{}, errors.New("installed dependency requirement exceeds supported bounds")
		}
		_, active, err := parseDeclaredDependencyForProfileWithExtras(requirement, PublicPyPIProfile(), python, activeExtras)
		if err != nil {
			return InstalledProject{}, errors.New("installed dependency requirement is unsupported")
		}
		if active {
			value := strings.TrimSpace(strings.SplitN(requirement, ";", 2)[0])
			// Only the existing bounded CUDA subset has a separate typed
			// obligation. A plain pip constraint alone cannot retain extras.
			if strings.ContainsAny(value, "[]") {
				extra, err := ParseBoundedRequirement(requirement)
				if err != nil || len(extra.Extras()) == 0 || extra.Project() == project {
					return InstalledProject{}, errors.New("installed dependency extras cannot be preserved as constraints")
				}
				result.extraRequirements = append(result.extraRequirements, extra)
				value = extra.VersionConstraint()
			}
			result.requirements = append(result.requirements, value)
		}
	}
	sort.Strings(result.requirements)
	return result, nil
}

// UpgradeInput freezes selection intent and the inert installed metadata. It
// does not attest native ownership, integrity, safety or inspection completion.
type UpgradeInput struct {
	requirement UpgradeRequirement
	python      string
	installed   []InstalledProject
}

func NewUpgradeInput(requirement UpgradeRequirement, files []InstalledMetadataFile, python string) (UpgradeInput, error) {
	if requirement.Reference().Source().String() == "" || len(files) > maxPyPIReportEntries || !strings.HasPrefix(python, "3.14.") || len(strings.Split(python, ".")) != 3 || !numericRelease.MatchString(python) {
		return UpgradeInput{}, errors.New("upgrade input requires a bounded supported Python environment")
	}
	canonical, err := NormalizeVersion(python)
	if err != nil || canonical != python {
		return UpgradeInput{}, errors.New("upgrade input Python runtime is invalid")
	}
	result := UpgradeInput{requirement: requirement, python: python}
	seen := map[string]bool{}
	total := int64(len(requirement.requirement) + len(python))
	for _, file := range files {
		total += int64(len(file.Body)) + int64(len(file.Directory))
		if total > upgradeInputLimit {
			return UpgradeInput{}, errors.New("installed upgrade metadata exceeds total input bound")
		}
		project, err := parseUpgradeInstalledMetadata(file, python)
		if err != nil || seen[project.project] {
			return UpgradeInput{}, errors.New("installed upgrade metadata is unavailable, unsupported or duplicated")
		}
		seen[project.project] = true
		result.installed = append(result.installed, project)
	}
	sort.Slice(result.installed, func(i, j int) bool { return result.installed[i].project < result.installed[j].project })
	if _, err := result.Control(); err != nil {
		return UpgradeInput{}, err
	}
	return result, nil
}

func (u UpgradeInput) Requirement() UpgradeRequirement { return u.requirement }
func (u UpgradeInput) Python() string                  { return u.python }
func (u UpgradeInput) Installed() []InstalledProject {
	return append([]InstalledProject(nil), u.installed...)
}
func (u UpgradeInput) MetadataFiles() []InstalledMetadataFile {
	files := make([]InstalledMetadataFile, 0, len(u.installed))
	for _, project := range u.installed {
		files = append(files, InstalledMetadataFile{project.directory, append([]byte(nil), project.metadata...)})
	}
	return files
}

type upgradeInputWire struct {
	Version     int                     `json:"version"`
	Requirement string                  `json:"requirement"`
	Python      string                  `json:"python"`
	Installed   []InstalledMetadataFile `json:"installed"`
	Source      string                  `json:"source_id,omitempty"`
}

func (u UpgradeInput) Control() (domain.ProjectControlFile, error) {
	if u.requirement.project == "" || u.python == "" {
		return domain.ProjectControlFile{}, errors.New("validated upgrade input is required")
	}
	wire := upgradeInputWire{Version: 1, Requirement: u.requirement.requirement, Python: u.python, Installed: u.MetadataFiles()}
	if u.requirement.source != PublicPyPIProfile().Source() {
		wire.Source = u.requirement.source.String()
	}
	body, err := json.Marshal(wire)
	if err != nil {
		return domain.ProjectControlFile{}, err
	}
	return domain.NewProjectControlFile(UpgradeInputControlName, body, true)
}

func ParseUpgradeInput(controls []domain.ProjectControlFile, python string) (UpgradeInput, error) {
	if len(controls) != 1 || controls[0].Name() != UpgradeInputControlName || !controls[0].Present() {
		return UpgradeInput{}, errors.New("upgrade resolver requires exactly one frozen input")
	}
	body := controls[0].Body()
	if len(body) == 0 || len(body) > upgradeInputLimit {
		return UpgradeInput{}, errors.New("frozen upgrade input exceeds bounds")
	}
	var wire upgradeInputWire
	if json.Unmarshal(body, &wire) != nil || wire.Version != 1 || wire.Python != python {
		return UpgradeInput{}, errors.New("frozen upgrade input is invalid")
	}
	source := PublicPyPIProfile().Source()
	if wire.Source != "" {
		var err error
		source, err = domain.NewSourceID(wire.Source)
		if err != nil {
			return UpgradeInput{}, err
		}
	}
	requirement, err := ParseUpgradeRequirementForSource(wire.Requirement, source)
	if err != nil {
		return UpgradeInput{}, err
	}
	input, err := NewUpgradeInput(requirement, wire.Installed, python)
	if err != nil {
		return UpgradeInput{}, err
	}
	canonical, err := input.Control()
	// Canonical roundtrip rejects unknown/duplicate keys, altered order, excess
	// JSON and alternate representations in this controller-generated input.
	if err != nil || !bytes.Equal(canonical.Body(), body) {
		return UpgradeInput{}, errors.New("frozen upgrade input is not canonical")
	}
	return input, nil
}
