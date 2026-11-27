package policy

import (
	"errors"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// M5 is the PyPI set-level policy. It preserves M4's complete-set rule while
// making the ecosystem-specific policy identity explicit in public results.
type M5 struct{}

func (M5) EvaluateSet(set domain.InspectedDependencySet) (domain.PolicyDecision, error) {
	if !set.Valid() {
		return domain.PolicyDecision{}, errors.New("complete PyPI inspected dependency set is required")
	}
	return evaluateM5Selection(set.Graph().Nodes(), set.Graph().Edges(), set.Inspections())
}

// EvaluateExecutionSelection applies the same whole-wheel rules to explicit
// roots without presenting an independent package as a fabricated dependency.
func (M5) EvaluateExecutionSelection(set domain.InspectedExecutionSelection) (domain.PolicyDecision, error) {
	if !set.Valid() {
		return domain.PolicyDecision{}, errors.New("complete PyPI execution selection inspection is required")
	}
	return evaluateM5Selection(set.Selection().Nodes(), set.Selection().Edges(), set.Inspections())
}

func evaluateM5Selection(nodes []domain.LockedDependency, edges []domain.DependencyEdge, inspections []domain.DependencyInspection) (domain.PolicyDecision, error) {
	for _, inspection := range inspections {
		if inspection.PolicyDecision().Decision() == domain.DecisionBlock {
			return m5Decision(domain.DecisionBlock, "M5_DISTRIBUTION_BLOCKED")
		}
	}
	derived := map[domain.DependencyNodeID]bool{}
	for _, edge := range edges {
		for _, node := range nodes {
			if node.Node() == edge.To() && node.Artifact().Identity().Variant() == "derived-wheel" {
				derived[edge.From()] = true
			}
		}
	}
	for _, dependency := range nodes {
		variant := dependency.Artifact().Identity().Variant()
		if _, supported := artifactpypi.ProfileForSource(dependency.Artifact().Identity().Source()); !supported || (variant != "wheel" && variant != "derived-wheel" && !(variant == "sdist" && derived[dependency.Node()])) {
			return m5Decision(domain.DecisionManualReview, "M5_NON_WHEEL_PROMOTION_UNAVAILABLE")
		}
	}
	for _, inspection := range inspections {
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
