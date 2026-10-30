package pypi

import (
	"errors"
	"slices"
	"sort"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// UVRunRequirements is one console provider and explicitly requested extra
// roots. Every member uses the same bounded public bare/exact/extra grammar.
type UVRunRequirements struct {
	primary      UVRunRequirement
	supplemental []UVRunRequirement
}

func NewUVRunRequirements(primary UVRunRequirement, supplemental []UVRunRequirement) (UVRunRequirements, error) {
	invalid := errors.New("UV run requires distinct bounded public primary and supplemental requirements")
	if !primary.Valid() || len(supplemental) == 0 || len(supplemental) >= domain.MaximumExecutionArguments {
		return UVRunRequirements{}, invalid
	}
	seen := map[string]bool{primary.Base().Project(): true}
	text := []string{primary.Requirement()}
	owned := slices.Clone(supplemental)
	for _, request := range owned {
		if !request.Valid() || seen[request.Base().Project()] {
			return UVRunRequirements{}, invalid
		}
		seen[request.Base().Project()] = true
		text = append(text, request.Requirement())
	}
	if err := domain.ValidateExecutionArguments(text); err != nil {
		return UVRunRequirements{}, invalid
	}
	sort.Slice(owned, func(i, j int) bool { return owned[i].Requirement() < owned[j].Requirement() })
	return UVRunRequirements{primary, owned}, nil
}
func (r UVRunRequirements) Valid() bool                      { return r.primary.Valid() && len(r.supplemental) != 0 }
func (r UVRunRequirements) Primary() UVRunRequirement        { return r.primary }
func (r UVRunRequirements) Supplemental() []UVRunRequirement { return slices.Clone(r.supplemental) }
func (r UVRunRequirements) Requirements() []string {
	if !r.Valid() {
		return nil
	}
	text := []string{r.primary.Requirement()}
	for _, request := range r.supplemental {
		text = append(text, request.Requirement())
	}
	return text
}
