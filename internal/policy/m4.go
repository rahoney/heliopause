package policy

import (
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	m4PolicyID      = "m4-npm-install-promotion"
	m4PolicyVersion = 1
)

// M4 evaluates a complete dependency set after every entry has received its
// own M3-compatible decision. It never promotes a partial or uncertain set.
type M4 struct{}

func (M4) EvaluateSet(set domain.InspectedDependencySet) (domain.PolicyDecision, error) {
	if !set.Valid() {
		return domain.PolicyDecision{}, errors.New("complete inspected dependency set is required")
	}
	inspections := set.Inspections()
	for _, inspection := range inspections {
		if inspection.PolicyDecision().Decision() == domain.DecisionBlock {
			return m4Decision(domain.DecisionBlock, "M4_DEPENDENCY_BLOCKED")
		}
	}
	for _, dependency := range set.Graph().Nodes() {
		if dependency.HostInstallAction() {
			return m4Decision(domain.DecisionManualReview, "M4_HOST_LIFECYCLE_UNSUPPORTED")
		}
	}
	for _, inspection := range inspections {
		if inspection.PolicyDecision().Decision() != domain.DecisionAllow {
			return m4Decision(domain.DecisionManualReview, "M4_DEPENDENCY_REVIEW_REQUIRED")
		}
		for _, check := range inspection.Checks() {
			if check.Required() && (check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted) {
				return m4Decision(domain.DecisionManualReview, "M4_REQUIRED_CHECK_INCOMPLETE")
			}
		}
	}
	return m4Decision(domain.DecisionAllow, "M4_VERIFIED_SET_COMPLETED")
}

func m4Decision(decision domain.Decision, reason string) (domain.PolicyDecision, error) {
	return domain.NewPolicyDecision(decision, m4PolicyID, m4PolicyVersion, []string{reason})
}
