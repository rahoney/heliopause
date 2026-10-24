package pypi

import (
	"errors"
	"sort"
	"strings"
)

// RequiredUpgradeExtras supplies the existing source-pinned traversal with
// retained and newly selected obligations. A partial selection is data only;
// final completion still requires the complete graph and new inspections.
func (u UpgradeInput) RequiredUpgradeExtras(project string, selected []Candidate) ([]string, error) {
	name, err := NormalizeProjectName(project)
	if err != nil || name != project || len(selected) > maxPyPIReportEntries {
		return nil, errors.New("bounded canonical upgrade project is required")
	}
	profile, ok := ProfileForSource(u.requirement.source)
	if !ok {
		return nil, errors.New("upgrade source profile is invalid")
	}
	versions := map[string]string{}
	for _, item := range u.installed {
		versions[item.project] = item.version
	}
	seen := map[string]bool{}
	for _, item := range selected {
		if seen[item.project] || !sourceOwnsProject(profile, item.source, item.project) || item.primary != (item.project == u.requirement.project) {
			return nil, errors.New("upgrade extra selection has ambiguous source or identity")
		}
		seen[item.project] = true
		versions[item.project] = item.version
	}
	obligations, err := u.upgradeExtraObligations(versions, selected)
	return append([]string(nil), obligations[project]...), err
}

func (u UpgradeInput) upgradeExtraObligations(versions map[string]string, selected []Candidate) (map[string][]string, error) {
	sets := map[string]map[string]bool{}
	add := func(requirement BoundedRequirement) {
		if sets[requirement.project] == nil {
			sets[requirement.project] = map[string]bool{}
		}
		for _, extra := range requirement.extras {
			sets[requirement.project][extra] = true
		}
	}
	for _, item := range u.installed {
		if versions[item.project] == item.version {
			for _, requirement := range item.extraRequirements {
				add(requirement)
			}
		}
	}
	for _, item := range selected {
		for _, value := range item.requirements {
			if !strings.ContainsAny(value, "[]") {
				continue
			}
			requirement, err := ParseBoundedRequirement(value)
			if err != nil || len(requirement.Extras()) == 0 {
				return nil, errors.New("selected upgrade extra requirement is unsupported")
			}
			add(requirement)
		}
	}
	result := map[string][]string{}
	for project, extras := range sets {
		for extra := range extras {
			result[project] = append(result[project], extra)
		}
		sort.Strings(result[project])
	}
	return result, nil
}
