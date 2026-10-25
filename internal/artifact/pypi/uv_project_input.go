package pypi

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/pelletier/go-toml/v2"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type UVProjectIntent string

const (
	UVProjectAdd           UVProjectIntent = "add"
	UVProjectSync          UVProjectIntent = "sync"
	UVProjectLockUpgrade   UVProjectIntent = "lock-upgrade"
	UVProjectInputControl                  = "pyproject.toml"
	UVNativeLockControl                    = "uv.lock"
	UVProjectIntentControl                 = "uv-project-intent.json"
)

// UVProjectInput freezes unapproved virtual-project data. Its optional uv.lock
// is opaque original input, not a parsed selection or ownership/Policy proof.
// The complete target export still needs independent archive binding.
type UVProjectInput struct {
	project, version string
	requirements     []UpgradeRequirement
	body, lock       []byte
	lockPresent      bool
	intent           UVProjectIntent
	selector         string
}

type uvProjectInputWire struct {
	Project *struct {
		Name         string    `toml:"name"`
		Version      string    `toml:"version"`
		Python       string    `toml:"requires-python"`
		Dependencies *[]string `toml:"dependencies"`
	} `toml:"project"`
	Tool *struct {
		UV *struct {
			Package *bool `toml:"package"`
		} `toml:"uv"`
	} `toml:"tool"`
}

// ParseUVProjectInput accepts the first reviewed virtual-project subset. No
// target interpreter, local build, configuration discovery or native execution
// occurs here. Unknown fields fail closed before any native command.
func ParseUVProjectInput(body, lock []byte, lockPresent bool, intent UVProjectIntent, selector string) (UVProjectInput, error) {
	if !validUVProjectText(body, false) || !validUVProjectText(lock, !lockPresent) || !lockPresent && len(lock) != 0 {
		return UVProjectInput{}, errors.New("UV project controls are absent, invalid or exceed bounds")
	}
	var wire uvProjectInputWire
	if err := toml.NewDecoder(bytes.NewReader(body)).DisallowUnknownFields().Decode(&wire); err != nil {
		return UVProjectInput{}, err
	}
	if wire.Project == nil || wire.Project.Dependencies == nil || wire.Tool != nil && (wire.Tool.UV == nil || wire.Tool.UV.Package == nil || *wire.Tool.UV.Package) {
		return UVProjectInput{}, errors.New("UV project must declare an explicit virtual dependency set")
	}
	project, err := NormalizeProjectName(wire.Project.Name)
	version, versionErr := NormalizeVersion(wire.Project.Version)
	python, pythonErr := ParseBoundedRequirement("python" + wire.Project.Python)
	if err != nil || versionErr != nil || version != wire.Project.Version || !IsFinalVersion(version) || wire.Project.Python == "" || pythonErr != nil || python.Project() != "python" || !python.Satisfies(runtimeidentity.PythonVersion) {
		return UVProjectInput{}, errors.New("UV virtual project identity or Python requirement is unsupported")
	}
	dependencies := *wire.Project.Dependencies
	if len(dependencies) > PublicPyPIProfile().ResourcePolicy().MaxGraphArtifacts() {
		return UVProjectInput{}, errors.New("UV project dependency set exceeds bounds")
	}
	result := UVProjectInput{project: project, version: version, body: append([]byte(nil), body...), lock: append([]byte(nil), lock...), lockPresent: lockPresent, intent: intent}
	seen := map[string]bool{}
	for _, value := range dependencies {
		requirement, err := parseUVProjectRequirement(value)
		if err != nil || requirement.Project() == project || seen[requirement.Project()] {
			return UVProjectInput{}, errors.New("UV project dependency is unsupported, repeated or self-referencing")
		}
		seen[requirement.Project()] = true
		result.requirements = append(result.requirements, requirement)
	}
	switch intent {
	case UVProjectAdd:
		requirement, err := parseUVProjectRequirement(selector)
		if err != nil || requirement.Project() == project {
			return UVProjectInput{}, errors.New("UV project add requires one supported public dependency")
		}
		result.selector = requirement.Requirement()
	case UVProjectSync, UVProjectLockUpgrade:
		if selector != "" || intent == UVProjectLockUpgrade && !lockPresent {
			return UVProjectInput{}, errors.New("UV project intent has an unexpected selector or lacks an original lock")
		}
	default:
		return UVProjectInput{}, errors.New("UV project intent is unsupported")
	}
	return result, nil
}

func validUVProjectText(body []byte, allowEmpty bool) bool {
	return (allowEmpty || len(body) != 0) && len(body) <= maxReportBytes && utf8.Valid(body) && bytes.IndexByte(body, 0) < 0
}

func parseUVProjectRequirement(value string) (UpgradeRequirement, error) {
	// @ is also the CLI exact-reference shorthand; it is not project PEP 508.
	if strings.Contains(value, "@") {
		return UpgradeRequirement{}, errors.New("UV project direct or shorthand dependency is unsupported")
	}
	return ParseUpgradeRequirement(value)
}

func (p UVProjectInput) Valid() bool             { return p.project != "" }
func (p UVProjectInput) Project() string         { return p.project }
func (p UVProjectInput) Version() string         { return p.version }
func (p UVProjectInput) Intent() UVProjectIntent { return p.intent }
func (p UVProjectInput) Selector() string        { return p.selector }
func (p UVProjectInput) Body() []byte            { return append([]byte(nil), p.body...) }
func (p UVProjectInput) NativeLock() ([]byte, bool) {
	return append([]byte(nil), p.lock...), p.lockPresent
}
func (p UVProjectInput) Requirements() []UpgradeRequirement {
	return append([]UpgradeRequirement(nil), p.requirements...)
}

type uvProjectIntentWire struct {
	Version  int             `json:"version"`
	Intent   UVProjectIntent `json:"intent"`
	Selector string          `json:"selector"`
	UV       string          `json:"uv_version"`
	Python   string          `json:"python_version"`
	ToolHash string          `json:"uv_tool_sha256"`
}

func (p UVProjectInput) Controls() ([]domain.ProjectControlFile, error) {
	if !p.Valid() {
		return nil, errors.New("UV project input is unavailable")
	}
	body, err := json.Marshal(uvProjectIntentWire{1, p.intent, p.selector, runtimeidentity.UVVersion, runtimeidentity.PythonVersion, runtimeidentity.UVToolSHA256})
	if err != nil {
		return nil, err
	}
	project, err := domain.NewProjectControlFile(UVProjectInputControl, p.body, true)
	if err != nil {
		return nil, err
	}
	lock, err := domain.NewProjectControlFile(UVNativeLockControl, p.lock, p.lockPresent)
	if err != nil {
		return nil, err
	}
	intent, err := domain.NewProjectControlFile(UVProjectIntentControl, body, true)
	if err != nil {
		return nil, err
	}
	return []domain.ProjectControlFile{project, lock, intent}, nil
}

// ParseUVProjectControls requires the exact canonical intent/runtime bytes and
// preserves lock absence. Successful parsing grants no installation authority.
func ParseUVProjectControls(controls []domain.ProjectControlFile) (UVProjectInput, error) {
	if len(controls) != 3 {
		return UVProjectInput{}, errors.New("UV project requires three frozen controls")
	}
	byName := map[string]domain.ProjectControlFile{}
	for _, control := range controls {
		if _, seen := byName[control.Name()]; seen {
			return UVProjectInput{}, errors.New("UV project controls repeat")
		}
		switch control.Name() {
		case UVProjectInputControl, UVProjectIntentControl:
			if !control.Present() {
				return UVProjectInput{}, errors.New("UV project input or intent is absent")
			}
		case UVNativeLockControl:
		default:
			return UVProjectInput{}, errors.New("UV project control is unknown")
		}
		byName[control.Name()] = control
	}
	var wire uvProjectIntentWire
	if json.Unmarshal(byName[UVProjectIntentControl].Body(), &wire) != nil {
		return UVProjectInput{}, errors.New("UV project intent is invalid")
	}
	result, err := ParseUVProjectInput(byName[UVProjectInputControl].Body(), byName[UVNativeLockControl].Body(), byName[UVNativeLockControl].Present(), wire.Intent, wire.Selector)
	if err != nil {
		return UVProjectInput{}, err
	}
	canonical, err := result.Controls()
	if err != nil || !bytes.Equal(canonical[2].Body(), byName[UVProjectIntentControl].Body()) {
		return UVProjectInput{}, errors.New("UV project intent or runtime differs from canonical input")
	}
	return result, nil
}
