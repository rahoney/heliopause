package policy

import (
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	m2PolicyID      = "m2-npm-static-inspect"
	m2PolicyVersion = 1
)

// M2 evaluates public npm static-inspection results without allowing missing dynamic inspection.
type M2 struct{}

func (M2) Evaluate(input domain.PolicyInput) (domain.PolicyDecision, error) {
	if !input.Valid() {
		return domain.PolicyDecision{}, errors.New("valid Policy input is required")
	}
	if input.Verification().Outcome() == domain.VerificationMismatch {
		return domain.NewPolicyDecision(domain.DecisionBlock, m2PolicyID, m2PolicyVersion, []string{"M2_INTEGRITY_MISMATCH"})
	}
	for _, finding := range input.Inspection().Findings() {
		return domain.NewPolicyDecision(domain.DecisionBlock, m2PolicyID, m2PolicyVersion, []string{finding.Code()})
	}
	for _, check := range []domain.CheckExecution{input.Verification().Execution(), input.Inspection().Execution()} {
		if check.Required() && (check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted) {
			return domain.NewPolicyDecision(domain.DecisionManualReview, m2PolicyID, m2PolicyVersion, []string{"M2_REQUIRED_CHECK_INCOMPLETE"})
		}
	}
	return domain.NewPolicyDecision(domain.DecisionManualReview, m2PolicyID, m2PolicyVersion, []string{"M2_DYNAMIC_INSPECTION_UNAVAILABLE"})
}
