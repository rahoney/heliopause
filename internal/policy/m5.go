package policy

import (
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// M5 is the PyPI set-level policy. It preserves M4's complete-set rule while
// making the ecosystem-specific policy identity explicit in public results.
type M5 struct{}

func (M5) EvaluateSet(set domain.InspectedDependencySet) (domain.PolicyDecision, error) {
	if !set.Valid() {
		return domain.PolicyDecision{}, errors.New("complete PyPI inspected dependency set is required")
	}
	for _, inspection := range set.Inspections() {
		if inspection.PolicyDecision().Decision() == domain.DecisionBlock {
			return m5Decision(domain.DecisionBlock, "M5_DISTRIBUTION_BLOCKED")
		}
	}
	derived := map[domain.DependencyNodeID]bool{}
	for _, edge := range set.Graph().Edges() {
		for _, node := range set.Graph().Nodes() {
			if node.Node() == edge.To() && node.Artifact().Identity().Variant() == "derived-wheel" {
				derived[edge.From()] = true
			}
		}
	}
	for _, dependency := range set.Graph().Nodes() {
		variant := dependency.Artifact().Identity().Variant()
		if dependency.Artifact().Identity().Source().String() != "pypi" || (variant != "wheel" && variant != "derived-wheel" && !(variant == "sdist" && derived[dependency.Node()])) {
			return m5Decision(domain.DecisionManualReview, "M5_NON_WHEEL_PROMOTION_UNAVAILABLE")
		}
	}
	for _, inspection := range set.Inspections() {
		if inspection.PolicyDecision().Decision() != domain.DecisionAllow {
			return m5Decision(domain.DecisionManualReview, "M5_DISTRIBUTION_REVIEW_REQUIRED")
		}
		for _, check := range inspection.Checks() {
			if check.Required() && (check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted) {
				return m5Decision(domain.DecisionManualReview, "M5_REQUIRED_CHECK_INCOMPLETE")
			}
		}
	}
	return m5Decision(domain.DecisionAllow, "M5_VERIFIED_SET_COMPLETED")
}

func m5Decision(decision domain.Decision, reason string) (domain.PolicyDecision, error) {
	return domain.NewPolicyDecision(decision, "m5-pypi-pip", 1, []string{reason})
}
