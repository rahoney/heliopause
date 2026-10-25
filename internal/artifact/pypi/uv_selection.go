package pypi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// These are the three native selection intents, independent of SourceID and
// operation presentation. No caller-supplied native arguments are accepted.
type UVSelectionIntent string

const (
	UVInstallSelection           UVSelectionIntent = "install"
	UVUpgradeSelection           UVSelectionIntent = "upgrade"
	UVUpgradePackageSelection    UVSelectionIntent = "upgrade-package"
	UVSelectionIntentControlName                   = "uv-selection-intent.json"
	UVSelectionDeltaControlName                    = "uv-selection-delta.json"
	UVSelectionLockControlName                     = "uv-selection-pylock.toml"
)

func NewUVSelectionIntentControl(intent UVSelectionIntent) (domain.ProjectControlFile, error) {
	if intent != UVInstallSelection && intent != UVUpgradeSelection && intent != UVUpgradePackageSelection {
		return domain.ProjectControlFile{}, errors.New("unsupported UV selection intent")
	}
	body, err := json.Marshal(struct {
		Version int               `json:"version"`
		Intent  UVSelectionIntent `json:"intent"`
	}{1, intent})
	if err != nil {
		return domain.ProjectControlFile{}, err
	}
	return domain.NewProjectControlFile(UVSelectionIntentControlName, body, true)
}

func ParseUVSelectionRequest(controls []domain.ProjectControlFile, python string) (UpgradeInput, UVSelectionIntent, error) {
	if len(controls) != 2 {
		return UpgradeInput{}, "", errors.New("UV selection requires original input and intent")
	}
	var original, intentControl domain.ProjectControlFile
	for _, control := range controls {
		if !control.Present() {
			return UpgradeInput{}, "", errors.New("UV selection input is absent")
		}
		switch control.Name() {
		case UpgradeInputControlName:
			original = control
		case UVSelectionIntentControlName:
			intentControl = control
		default:
			return UpgradeInput{}, "", errors.New("UV selection input has an unknown control")
		}
	}
	input, err := ParseUpgradeInput([]domain.ProjectControlFile{original}, python)
	if err != nil || input.requirement.source != PublicPyPIProfile().Source() {
		return UpgradeInput{}, "", errors.New("UV selection requires frozen public PyPI input")
	}
	var wire struct {
		Version int               `json:"version"`
		Intent  UVSelectionIntent `json:"intent"`
	}
	if json.Unmarshal(intentControl.Body(), &wire) != nil || wire.Version != 1 {
		return UpgradeInput{}, "", errors.New("UV intent control is invalid")
	}
	canonical, err := NewUVSelectionIntentControl(wire.Intent)
	if err != nil || !bytes.Equal(canonical.Body(), intentControl.Body()) {
		return UpgradeInput{}, "", errors.New("UV intent control is not canonical")
	}
	return input, wire.Intent, nil
}

// UVSelection is an unapproved version proposal. A native change list neither
// authenticates archives nor supplies a complete dependency graph or Policy.
type UVSelection struct {
	input               UpgradeInput
	reference           domain.ArtifactReference
	versions, installed map[string]string
	constraints, delta  []byte
}

func (u UpgradeInput) ParseUVSelection(body []byte) (UVSelection, error) {
	if u.python == "" || u.requirement.source != PublicPyPIProfile().Source() || len(body) == 0 || len(body) > maxReportBytes {
		return UVSelection{}, errors.New("UV selection requires bounded native delta and original input")
	}
	if err := validateUVJSONKeys(body); err != nil {
		return UVSelection{}, err
	}
	var wire struct {
		Schema struct {
			Version string `json:"version"`
		} `json:"schema"`
		Changes []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Action  string `json:"action"`
		} `json:"changes"`
		DryRun *bool `json:"dry_run"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&wire) != nil || ensureSingleJSONValue(decoder) != nil || wire.Schema.Version != "preview" || wire.DryRun == nil || !*wire.DryRun || wire.Changes == nil || len(wire.Changes) > 2*maxPyPIReportEntries {
		return UVSelection{}, errors.New("UV native delta schema, dry-run or bounds are invalid")
	}
	result := UVSelection{input: u, versions: map[string]string{}, installed: map[string]string{}, delta: append([]byte(nil), body...)}
	before := map[string]string{}
	for _, project := range u.installed {
		before[project.project] = project.version
		result.versions[project.project] = project.version
	}
	removed := map[string]string{}
	for _, change := range wire.Changes {
		name, err := NormalizeProjectName(change.Name)
		version, versionErr := NormalizeVersion(change.Version)
		if err != nil || name != change.Name || versionErr != nil || version != change.Version || !IsFinalVersion(version) {
			return UVSelection{}, errors.New("UV native change identity is unsupported")
		}
		switch change.Action {
		case "uninstalled":
			if removed[name] != "" || before[name] != version {
				return UVSelection{}, errors.New("UV uninstall differs from frozen original")
			}
			removed[name] = version
		case "installed":
			if result.installed[name] != "" {
				return UVSelection{}, errors.New("UV install action is ambiguous")
			}
			result.installed[name] = version
			result.versions[name] = version
		default:
			return UVSelection{}, errors.New("UV native change action is unsupported")
		}
	}
	for name := range removed {
		if result.installed[name] == "" {
			return UVSelection{}, errors.New("UV install cannot remove without replacement")
		}
	}
	for name := range result.installed {
		if before[name] != "" && removed[name] != before[name] {
			return UVSelection{}, errors.New("UV replacement omitted original uninstall")
		}
	}
	version := result.versions[u.requirement.project]
	if len(result.versions) > maxPyPIReportEntries || !u.requirement.AcceptsVersion(version) {
		return UVSelection{}, errors.New("UV selection does not satisfy requested root")
	}
	var err error
	result.reference, err = ParseReference(u.requirement.project + "@" + version)
	if err != nil {
		return UVSelection{}, err
	}
	extras, err := u.upgradeExtraObligations(result.versions, nil)
	if err != nil {
		return UVSelection{}, err
	}
	result.constraints, err = u.selectedVersionConstraints(result.versions, extras)
	if err != nil {
		return UVSelection{}, err
	}
	return result, nil
}

func (s UVSelection) Reference() domain.ArtifactReference { return s.reference }
func (s UVSelection) Constraints() []byte                 { return append([]byte(nil), s.constraints...) }
func (s UVSelection) Delta() []byte                       { return append([]byte(nil), s.delta...) }

// ValidateGraph binds independently authenticated graph identities to the
// native version choices. Unrelated installed packages stay constraints.
func (s UVSelection) ValidateGraph(graph domain.LockedDependencyGraph) error {
	if s.reference.Source() != PublicPyPIProfile().Source() || len(graph.Nodes()) == 0 {
		return errors.New("UV proposal requires complete independently bound graph")
	}
	seen := map[string]bool{}
	root := false
	for _, node := range graph.Nodes() {
		id := node.Artifact().Identity()
		if seen[id.Name()] || id.Source() != PublicPyPIProfile().Source() || id.Variant() != "wheel" || s.versions[id.Name()] != id.Version() || node.HostInstallAction() {
			return errors.New("UV graph differs from frozen native choices")
		}
		seen[id.Name()] = true
		if node.Node() == graph.Primary() {
			root = id.Name() == s.input.requirement.project && s.input.requirement.AcceptsVersion(id.Version())
		}
	}
	if !root || len(graph.Derivations()) != 0 {
		return errors.New("UV graph root or derivation differs from selection")
	}
	for name := range s.installed {
		if !seen[name] {
			return errors.New("UV changed distribution is missing from graph")
		}
	}
	for _, project := range s.input.installed {
		if s.versions[project.project] != project.version {
			continue
		}
		for _, value := range project.requirements {
			requirement, err := ParseBoundedRequirement(value)
			if err != nil || !requirement.Satisfies(s.versions[requirement.Project()]) {
				return errors.New("UV selection violates retained original requirement")
			}
		}
	}
	return nil
}

// Native JSON may reorder keys or whitespace, but duplicate keys at any of its
// bounded schema levels must never silently override earlier selection data.
func validateUVJSONKeys(body []byte) error {
	return validateNativeJSONKeys(body, 4)
}

func validateNativeJSONKeys(body []byte, maxDepth int) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	var value func(int) error
	value = func(depth int) error {
		if depth > maxDepth {
			return errors.New("UV JSON nesting exceeds schema")
		}
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, compound := token.(json.Delim)
		if !compound {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return errors.New("UV JSON contains duplicate field")
				}
				seen[name] = true
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for decoder.More() {
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		default:
			return errors.New("UV JSON delimiter is invalid")
		}
		_, err = decoder.Token()
		return err
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("UV JSON has trailing data")
	}
	return nil
}
