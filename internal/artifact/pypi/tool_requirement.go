package pypi

import "maps"

// SameConstraints compares the same supported clause conjunction regardless
// of order or repetition. It is deliberately conservative about other native
// rewrites: accepting one selected version does not prove equal constraints.
// Original request and native receipt text remain separate immutable controls.
func (r UpgradeRequirement) SameConstraints(other UpgradeRequirement) bool {
	if r.project == "" || r.project != other.project || r.source != other.source {
		return false
	}
	if r.requirement == other.requirement {
		return true
	}
	left, err := ParseBoundedRequirement(r.requirement)
	if err != nil {
		return false
	}
	right, err := ParseBoundedRequirement(other.requirement)
	if err != nil || len(left.extras) != 0 || len(right.extras) != 0 {
		return false
	}
	a, b := map[boundedConstraint]struct{}{}, map[boundedConstraint]struct{}{}
	for _, clause := range left.constraints {
		a[clause] = struct{}{}
	}
	for _, clause := range right.constraints {
		b[clause] = struct{}{}
	}
	return maps.Equal(a, b)
}
