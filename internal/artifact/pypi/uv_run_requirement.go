package pypi

import (
	"errors"
	"slices"
	"sort"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// UVRunRequirement adds explicitly requested root extras to the existing
// public bare/exact requirement. It is immutable selection DATA, not approval.
// Installed upgrade and CUDA requirement grammars remain separate.
type UVRunRequirement struct {
	base   UpgradeRequirement
	extras []string
}

func ParseUVRunRequirement(value string) (UVRunRequirement, error) {
	invalid := errors.New("UV run requires one public bare or exact package and bounded root extras")
	if value == "" || len(value) > 512 || value != strings.TrimSpace(value) || strings.ContainsAny(value, "\x00\r\n\t") {
		return UVRunRequirement{}, invalid
	}
	baseInput := value
	var extras []string
	if open := strings.IndexByte(value, '['); open >= 0 {
		close := strings.IndexByte(value, ']')
		if open == 0 || close <= open+1 || strings.ContainsAny(value[open+1:close], "[]") || strings.ContainsAny(value[close+1:], "[]") {
			return UVRunRequirement{}, invalid
		}
		baseInput = value[:open] + value[close+1:]
		seen := map[string]bool{}
		for _, raw := range strings.Split(value[open+1:close], ",") {
			extra, err := NormalizeProjectName(strings.TrimSpace(raw))
			if err != nil || seen[extra] {
				return UVRunRequirement{}, invalid
			}
			seen[extra] = true
			extras = append(extras, extra)
		}
		sort.Strings(extras)
	} else if strings.ContainsRune(value, ']') {
		return UVRunRequirement{}, invalid
	}
	base, err := ParseUpgradeRequirement(baseInput)
	if err != nil {
		return UVRunRequirement{}, invalid
	}
	if base.Requirement() != base.Project() {
		reference, err := ParseReference(strings.Replace(base.Requirement(), "==", "@", 1))
		if err != nil {
			return UVRunRequirement{}, invalid
		}
		version, exact, err := RequestedVersion(reference)
		if err != nil || !exact || !IsFinalVersion(version) {
			return UVRunRequirement{}, invalid
		}
		roundtrip, err := ParseUpgradeRequirement(reference.Locator())
		if err != nil || roundtrip.Requirement() != base.Requirement() {
			return UVRunRequirement{}, invalid
		}
	}
	return UVRunRequirement{base, extras}, nil
}

func (r UVRunRequirement) Base() UpgradeRequirement { return r.base }
func (r UVRunRequirement) Extras() []string         { return slices.Clone(r.extras) }
func (r UVRunRequirement) Requirement() string {
	if len(r.extras) == 0 {
		return r.base.Requirement()
	}
	return r.base.Project() + "[" + strings.Join(r.extras, ",") + "]" + strings.TrimPrefix(r.base.Requirement(), r.base.Project())
}
func (r UVRunRequirement) Reference() domain.ArtifactReference {
	reference, _ := ParseReference(strings.Replace(r.base.Requirement(), "==", "@", 1))
	return reference
}
func (r UVRunRequirement) Valid() bool {
	canonical, err := ParseUVRunRequirement(r.Requirement())
	return err == nil && canonical.base.Reference() == r.base.Reference() &&
		canonical.base.Requirement() == r.base.Requirement() && slices.Equal(canonical.extras, r.extras)
}
