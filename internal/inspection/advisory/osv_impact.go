package advisory

import (
	"strings"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"golang.org/x/mod/semver"
)

type versionImpact uint8

const (
	impactUnknown versionImpact = iota
	impactUnaffected
	impactAffected
	impactWithdrawn
)

func (v versionImpact) String() string {
	switch v {
	case impactUnaffected:
		return "UNAFFECTED"
	case impactAffected:
		return "AFFECTED"
	case impactWithdrawn:
		return "WITHDRAWN"
	default:
		return "UNKNOWN"
	}
}

// assessRecord interprets exact package/version facts, never a final Policy
// Decision. An unmatched provider subject is incomplete, not a clean result.
func assessRecord(query osvPackage, record osvRecord) (versionImpact, error) {
	if !validPackage(query) {
		return impactUnknown, errResponse
	}
	if _, err := compareVersions(query.Ecosystem, query.Version, query.Version); err != nil {
		return impactUnknown, nil
	}
	if record.Withdrawn != nil {
		return impactWithdrawn, nil
	}
	matched, unknown, affected := false, false, false
	for _, entry := range record.Affected {
		if !samePackage(query, entry.Package) {
			continue
		}
		matched = true
		entryKnown := len(entry.Versions) != 0 || len(entry.Ranges) != 0
		for _, version := range entry.Versions {
			order, err := compareVersions(query.Ecosystem, query.Version, version)
			if err != nil {
				unknown = true
			} else if order == 0 {
				affected = true
			}
		}
		for _, interval := range entry.Ranges {
			impact, err := assessRange(query, interval)
			if err != nil {
				return impactUnknown, err
			}
			unknown = unknown || impact == impactUnknown
			affected = affected || impact == impactAffected
		}
		unknown = unknown || !entryKnown
	}
	if !matched {
		return impactUnknown, errResponse
	}
	if affected {
		return impactAffected, nil
	}
	if unknown {
		return impactUnknown, nil
	}
	return impactUnaffected, nil
}

func samePackage(query, subject osvPackage) bool {
	if query.Ecosystem != subject.Ecosystem {
		return false
	}
	switch query.Ecosystem {
	case "PyPI":
		left, leftErr := artifactpypi.NormalizeProjectName(query.Name)
		right, rightErr := artifactpypi.NormalizeProjectName(subject.Name)
		return leftErr == nil && rightErr == nil && left == right
	case "crates.io":
		// Reuse the registry's ASCII name grammar before case folding. Unicode
		// lookalikes are not an equivalent public crate identity.
		_, leftErr := artifactcargo.ParseReference(query.Name + "@" + query.Version)
		_, rightErr := artifactcargo.ParseReference(subject.Name + "@" + query.Version)
		return leftErr == nil && rightErr == nil && strings.EqualFold(strings.ReplaceAll(query.Name, "_", "-"), strings.ReplaceAll(subject.Name, "_", "-"))
	default:
		return query.Name == subject.Name
	}
}

func compareVersions(ecosystem, left, right string) (int, error) {
	if ecosystem == "PyPI" {
		return artifactpypi.ComparePublicVersions(left, right)
	}
	if ecosystem != "npm" && ecosystem != "Go" && ecosystem != "crates.io" {
		return 0, errResponse
	}
	var canonical [2]string
	for index, value := range []string{left, right} {
		if len(value) == 0 || len(value) > 256 {
			return 0, errResponse
		}
		version := strings.TrimPrefix(value, "v")
		core := version
		if end := strings.IndexAny(core, "-+"); end >= 0 {
			core = core[:end]
		}
		canonical[index] = "v" + version
		if strings.Count(core, ".") != 2 || !semver.IsValid(canonical[index]) {
			return 0, errResponse
		}
	}
	return semver.Compare(canonical[0], canonical[1]), nil
}

func assessRange(query osvPackage, interval osvRange) (versionImpact, error) {
	if interval.Type != "ECOSYSTEM" && interval.Type != "SEMVER" {
		return impactUnknown, nil
	}
	// PyPI ECOSYSTEM uses PEP 440. Do not evaluate a differently declared
	// ordering as PEP 440 merely because its package is hosted on PyPI.
	if interval.Type == "SEMVER" && query.Ecosystem == "PyPI" {
		return impactUnknown, nil
	}
	if len(interval.Events) == 0 {
		return impactUnknown, errResponse
	}
	active, affected := false, false
	start, previousEnd := "", ""
	previousInclusive := false
	for index, event := range interval.Events {
		if event.Introduced != "" {
			if active || event.Introduced == "0" && index != 0 {
				return impactUnknown, errResponse
			}
			if event.Introduced != "0" {
				if _, err := compareVersions(query.Ecosystem, event.Introduced, event.Introduced); err != nil {
					return impactUnknown, errResponse
				}
				if previousEnd != "" {
					order, err := compareVersions(query.Ecosystem, event.Introduced, previousEnd)
					if err != nil || order < 0 || order == 0 && previousInclusive {
						return impactUnknown, errResponse
					}
				}
			}
			start, active = event.Introduced, true
			continue
		}
		if !active {
			return impactUnknown, errResponse
		}
		end, inclusive := event.Fixed, false
		if event.LastAffected != "" {
			end, inclusive = event.LastAffected, true
		}
		if event.Limit != "" {
			end = event.Limit
		}
		if _, err := compareVersions(query.Ecosystem, end, end); err != nil {
			return impactUnknown, errResponse
		}
		if start != "0" {
			order, err := compareVersions(query.Ecosystem, start, end)
			if err != nil || order > 0 || order == 0 && !inclusive {
				return impactUnknown, errResponse
			}
		}
		lower := 0
		if start != "0" {
			var err error
			lower, err = compareVersions(query.Ecosystem, query.Version, start)
			if err != nil {
				return impactUnknown, nil
			}
		}
		upper, err := compareVersions(query.Ecosystem, query.Version, end)
		if err != nil {
			return impactUnknown, nil
		}
		affected = affected || lower >= 0 && (upper < 0 || upper == 0 && inclusive)
		previousEnd, previousInclusive, active = end, inclusive, false
	}
	if active {
		lower := 0
		if start != "0" {
			var err error
			lower, err = compareVersions(query.Ecosystem, query.Version, start)
			if err != nil {
				return impactUnknown, nil
			}
		}
		affected = affected || lower >= 0
	}
	if affected {
		return impactAffected, nil
	}
	return impactUnaffected, nil
}
