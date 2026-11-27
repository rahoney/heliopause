package pypi

import (
	"errors"
	"sort"
	"strings"
)

// UVProjectSelection is request data; no selector grants inspection approval.
type UVProjectSelection struct {
	Groups     []string `json:"groups,omitempty"`
	OnlyGroups []string `json:"only_groups,omitempty"`
	NoGroups   []string `json:"no_groups,omitempty"`
	Extras     []string `json:"extras,omitempty"`
	AllExtras  bool     `json:"all_extras,omitempty"`
	Locked     bool     `json:"locked,omitempty"`
	Frozen     bool     `json:"frozen,omitempty"`
}

func (s UVProjectSelection) clone() UVProjectSelection {
	s.Groups = append([]string(nil), s.Groups...)
	s.OnlyGroups = append([]string(nil), s.OnlyGroups...)
	s.NoGroups = append([]string(nil), s.NoGroups...)
	s.Extras = append([]string(nil), s.Extras...)
	return s
}

func (s UVProjectSelection) empty() bool {
	return len(s.Groups)+len(s.OnlyGroups)+len(s.NoGroups)+len(s.Extras) == 0 && !s.AllExtras && !s.Locked && !s.Frozen
}

func canonicalUVProjectSelection(s UVProjectSelection) (UVProjectSelection, error) {
	s = s.clone()
	if s.Locked && s.Frozen || len(s.Groups) > 0 && len(s.OnlyGroups) > 0 || len(s.OnlyGroups) > 0 && (s.AllExtras || len(s.Extras) > 0) {
		return UVProjectSelection{}, errors.New("UV project selection options conflict")
	}
	total := 0
	for _, values := range []*[]string{&s.Groups, &s.OnlyGroups, &s.NoGroups, &s.Extras} {
		seen := map[string]bool{}
		result := make([]string, 0, len(*values))
		for _, value := range *values {
			name, err := NormalizeProjectName(value)
			if err != nil || value != strings.TrimSpace(value) {
				return UVProjectSelection{}, errors.New("UV group or extra name is invalid")
			}
			if !seen[name] {
				seen[name] = true
				result = append(result, name)
				total++
			}
		}
		*values = result
	}
	if total > PublicPyPIProfile().ResourcePolicy().MaxGraphArtifacts() {
		return UVProjectSelection{}, errors.New("UV selection exceeds its input budget")
	}
	return s, nil
}

type uvProjectGroup struct{ requirements, includes []string }

// Use the requirements source boundary here; official PEP 440/508 semantics
// remain the pinned native parser's responsibility.
func uvProjectPublicRequirementName(value string) (string, error) {
	input, err := NewRequirementsInput(nil, nil, []string{value})
	if err != nil {
		return "", err
	}
	if _, err = input.Control(); err != nil {
		return "", err
	}
	head := boundedRequirementHead.FindStringSubmatch(value)
	if head == nil {
		return "", errors.New("UV project requirement has no public name")
	}
	return NormalizeProjectName(head[1])
}

func (p UVProjectInput) Selection() UVProjectSelection { return p.selection.clone() }
func (p UVProjectInput) NeedsMetadataProjection() bool { return p.rich }

func (p UVProjectInput) SelectedRequirements() ([]string, error) {
	if !p.Valid() {
		return nil, errors.New("UV project scope is unavailable")
	}
	result := []string{}
	if len(p.selection.OnlyGroups) == 0 {
		result = append(result, p.rawRequirements...)
		extras := p.selection.Extras
		if p.selection.AllExtras {
			extras = make([]string, 0, len(p.optional))
			for name := range p.optional {
				extras = append(extras, name)
			}
			sort.Strings(extras)
		}
		for _, name := range extras {
			result = append(result, p.optional[name]...)
		}
	}
	names := p.selection.OnlyGroups
	if len(names) == 0 {
		names = append(append([]string{}, p.defaultGroups...), p.selection.Groups...)
	}
	excluded := map[string]bool{}
	for _, name := range p.selection.NoGroups {
		excluded[name] = true
	}
	visited := map[string]bool{}
	var visit func(string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true
		group := p.groups[name]
		result = append(result, group.requirements...)
		for _, included := range group.includes {
			visit(included)
		}
	}
	for _, name := range names {
		if !excluded[name] {
			visit(name)
		}
	}
	if len(result) > PublicPyPIProfile().ResourcePolicy().MaxGraphArtifacts() {
		return nil, errors.New("UV selected requirements exceed bounds")
	}
	return result, nil
}

func (p *UVProjectInput) readScope(wire uvProjectInputWire) error {
	p.rawRequirements = append([]string(nil), (*wire.Project.Dependencies)...)
	p.optional = map[string][]string{}
	p.groups = map[string]uvProjectGroup{}
	total := len(p.rawRequirements)
	validate := func(value string) error {
		name, err := uvProjectPublicRequirementName(value)
		if err != nil {
			return err
		}
		if name == p.project {
			return errors.New("UV virtual project cannot depend on itself")
		}
		return nil
	}
	for name, values := range wire.Project.OptionalDependencies {
		canonical, err := NormalizeProjectName(name)
		if err != nil || p.optional[canonical] != nil {
			return errors.New("UV extra identity is invalid or repeated")
		}
		p.optional[canonical] = append([]string{}, values...)
		total += len(values)
		for _, value := range values {
			if err = validate(value); err != nil {
				return err
			}
		}
	}
	for name, values := range wire.Groups {
		canonical, err := NormalizeProjectName(name)
		if err != nil {
			return err
		}
		if _, exists := p.groups[canonical]; exists {
			return errors.New("UV group identity repeats")
		}
		group := uvProjectGroup{}
		for _, value := range values {
			total++
			switch entry := value.(type) {
			case string:
				if err = validate(entry); err != nil {
					return err
				}
				group.requirements = append(group.requirements, entry)
			case map[string]any:
				include, ok := entry["include-group"].(string)
				if !ok || len(entry) != 1 {
					return errors.New("UV group member is unsupported")
				}
				included, err := NormalizeProjectName(include)
				if err != nil {
					return err
				}
				group.includes = append(group.includes, included)
			default:
				return errors.New("UV group member is unsupported")
			}
		}
		p.groups[canonical] = group
	}
	if total > PublicPyPIProfile().ResourcePolicy().MaxGraphArtifacts() || len(p.groups)+len(p.optional) > PublicPyPIProfile().ResourcePolicy().MaxGraphArtifacts() {
		return errors.New("UV project group or extra input exceeds bounds")
	}
	states := map[string]uint8{}
	var visit func(string) error
	visit = func(name string) error {
		group, exists := p.groups[name]
		if !exists {
			return errors.New("UV included group does not exist")
		}
		if states[name] == 1 {
			return errors.New("UV group includes form a cycle")
		}
		if states[name] == 2 {
			return nil
		}
		states[name] = 1
		for _, included := range group.includes {
			if err := visit(included); err != nil {
				return err
			}
		}
		states[name] = 2
		return nil
	}
	for name := range p.groups {
		if err := visit(name); err != nil {
			return err
		}
	}
	if wire.Tool != nil && wire.Tool.UV != nil && wire.Tool.UV.DefaultGroups != nil {
		switch defaults := wire.Tool.UV.DefaultGroups.(type) {
		case string:
			if defaults != "all" {
				return errors.New("UV default groups setting is unsupported")
			}
			for name := range p.groups {
				p.defaultGroups = append(p.defaultGroups, name)
			}
			sort.Strings(p.defaultGroups)
		case []any:
			p.defaultGroups = []string{}
			seen := map[string]bool{}
			for _, value := range defaults {
				name, ok := value.(string)
				if !ok {
					return errors.New("UV default group is invalid")
				}
				canonical, err := NormalizeProjectName(name)
				if err != nil || seen[canonical] {
					return errors.New("UV default group is invalid or repeated")
				}
				seen[canonical] = true
				p.defaultGroups = append(p.defaultGroups, canonical)
			}
		default:
			return errors.New("UV default groups setting is unsupported")
		}
	} else if _, exists := p.groups["dev"]; exists {
		p.defaultGroups = []string{"dev"}
	}
	for _, values := range [][]string{p.defaultGroups, p.selection.Groups, p.selection.OnlyGroups, p.selection.NoGroups} {
		for _, name := range values {
			if _, exists := p.groups[name]; !exists {
				return errors.New("UV selected group does not exist")
			}
		}
	}
	for _, name := range p.selection.Extras {
		if _, exists := p.optional[name]; !exists {
			return errors.New("UV selected extra does not exist")
		}
	}
	p.rich = p.rich || wire.Project.OptionalDependencies != nil || wire.Groups != nil || !p.selection.empty()
	return nil
}
