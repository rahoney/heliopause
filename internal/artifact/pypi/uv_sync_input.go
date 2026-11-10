package pypi

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type UVSyncSourceFormat string

const (
	UVSyncRequirements      UVSyncSourceFormat = "requirements"
	UVSyncPylock            UVSyncSourceFormat = "pylock"
	UVSyncSourceControlName                    = "uv-sync-source"
	UVSyncIntentControlName                    = "uv-sync-intent.json"
	UVSyncSourceByteLimit                      = maxReportBytes
)

// UVSyncInput freezes unapproved complete source data. A blank requirements
// file normally skips selection; it is not an approved empty project.
type UVSyncInput struct {
	format       UVSyncSourceFormat
	body         []byte
	allowEmpty   bool
	python       string
	requirements []UpgradeRequirement
}

// ParsePinnedUVSyncInput provides command preflight with the same generated
// Python identity used by native selection, without importing a runtime backend.
func ParsePinnedUVSyncInput(format UVSyncSourceFormat, body []byte, allowEmpty bool) (UVSyncInput, error) {
	return ParseUVSyncInput(format, body, allowEmpty, runtimeidentity.PythonVersion)
}

func ParseUVSyncInput(format UVSyncSourceFormat, body []byte, allowEmpty bool, python string) (UVSyncInput, error) {
	canonical, err := NormalizeVersion(python)
	if err != nil || canonical != python || !numericRelease.MatchString(python) || len(strings.Split(python, ".")) != 3 || len(body) > UVSyncSourceByteLimit || !utf8.Valid(body) || bytes.IndexByte(body, 0) >= 0 {
		return UVSyncInput{}, errors.New("UV sync source or Python target is unsupported")
	}
	result := UVSyncInput{format: format, body: append([]byte(nil), body...), allowEmpty: allowEmpty, python: python}
	switch format {
	case UVSyncPylock:
		if _, err := ParseUVProjectLock(body, python); err != nil {
			return UVSyncInput{}, err
		}
	case UVSyncRequirements:
		seen := map[string]bool{}
		for _, line := range strings.Split(string(body), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if at := strings.IndexByte(line, '#'); at >= 0 {
				if at == 0 || line[at-1] != ' ' {
					return UVSyncInput{}, errors.New("UV sync inline comment is unsupported")
				}
				line = strings.TrimSpace(line[:at])
			}
			// No includes, options, URLs, extras, markers or range expansion.
			if strings.Count(line, "==") != 1 || strings.ContainsAny(line, "@[];:/\\\x00\r\n\t,<>~*") {
				return UVSyncInput{}, errors.New("UV sync requires bounded exact public pins")
			}
			pin, err := ParseUpgradeRequirement(line)
			if err != nil || pin.exact == "" || !IsFinalVersion(pin.exact) || seen[pin.Project()] || len(result.requirements) >= PublicPyPIProfile().ResourcePolicy().MaxGraphArtifacts() {
				return UVSyncInput{}, errors.New("UV sync exact pins are invalid, repeated or exceed bounds")
			}
			seen[pin.Project()] = true
			result.requirements = append(result.requirements, pin)
		}
	default:
		return UVSyncInput{}, errors.New("UV sync source format is unsupported")
	}
	return result, nil
}

func (s UVSyncInput) Valid() bool                      { return s.python != "" }
func (s UVSyncInput) Python() string                   { return s.python }
func (s UVSyncInput) SourceFormat() UVSyncSourceFormat { return s.format }
func (s UVSyncInput) Body() []byte                     { return append([]byte(nil), s.body...) }
func (s UVSyncInput) AllowEmptyRequirements() bool     { return s.allowEmpty }
func (s UVSyncInput) ShouldSkip() bool {
	return s.Valid() && s.format == UVSyncRequirements && len(s.requirements) == 0 && !s.allowEmpty
}

func (s UVSyncInput) Controls() ([]domain.ProjectControlFile, error) {
	if !s.Valid() {
		return nil, errors.New("UV sync input is unavailable")
	}
	source, err := domain.NewProjectControlFile(UVSyncSourceControlName, s.body, true)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(struct {
		Version    int                `json:"version"`
		Format     UVSyncSourceFormat `json:"format"`
		AllowEmpty bool               `json:"allow_empty_requirements"`
	}{1, s.format, s.allowEmpty})
	if err != nil {
		return nil, err
	}
	intent, err := domain.NewProjectControlFile(UVSyncIntentControlName, body, true)
	if err != nil {
		return nil, err
	}
	return []domain.ProjectControlFile{source, intent}, nil
}

func ParseUVSyncControls(controls []domain.ProjectControlFile, python string) (UVSyncInput, error) {
	if len(controls) != 2 {
		return UVSyncInput{}, errors.New("UV sync requires frozen source and intent")
	}
	byName := map[string]domain.ProjectControlFile{}
	for _, control := range controls {
		if !control.Present() || byName[control.Name()].Present() || control.Name() != UVSyncSourceControlName && control.Name() != UVSyncIntentControlName {
			return UVSyncInput{}, errors.New("UV sync controls are absent, repeated or unknown")
		}
		byName[control.Name()] = control
	}
	var intent struct {
		Version    int                `json:"version"`
		Format     UVSyncSourceFormat `json:"format"`
		AllowEmpty bool               `json:"allow_empty_requirements"`
	}
	if json.Unmarshal(byName[UVSyncIntentControlName].Body(), &intent) != nil || intent.Version != 1 {
		return UVSyncInput{}, errors.New("UV sync intent is invalid")
	}
	input, err := ParseUVSyncInput(intent.Format, byName[UVSyncSourceControlName].Body(), intent.AllowEmpty, python)
	if err != nil {
		return UVSyncInput{}, err
	}
	canonical, err := input.Controls()
	if err != nil || !bytes.Equal(canonical[1].Body(), byName[UVSyncIntentControlName].Body()) {
		return UVSyncInput{}, errors.New("UV sync intent is not canonical")
	}
	return input, nil
}

// ValidateSelection checks only source intent. The caller must still validate
// native census and independently bind all archives/metadata/constraints.
func (s UVSyncInput) ValidateSelection(lock UVProjectLock) error {
	if !s.Valid() || s.ShouldSkip() || len(lock.body) == 0 || lock.python != s.python {
		return errors.New("UV sync lacks a positive native selection for its target")
	}
	if s.format == UVSyncPylock {
		if !bytes.Equal(s.body, lock.body) {
			return errors.New("UV sync native pylock source changed")
		}
		return nil
	}
	if len(s.requirements) != len(lock.packages) {
		return errors.New("UV sync native package set differs from source pins")
	}
	selected := map[string]string{}
	for _, project := range lock.packages {
		selected[project.Name] = project.Version
	}
	for _, pin := range s.requirements {
		if !pin.AcceptsVersion(selected[pin.Project()]) {
			return errors.New("UV sync native version differs from source pin")
		}
	}
	return nil
}
