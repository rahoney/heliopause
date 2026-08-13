// Package policy owns deterministic Heliopause security decision rules.
package policy

import (
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	m1PolicyID      = "m1-fake-inspect"
	m1PolicyVersion = 1
)

// M1 evaluates the minimal fake Inspect policy without performing I/O.
type M1 struct{}

// Evaluate applies ordered block, incomplete-required-check, then allow rules.
func (M1) Evaluate(input domain.PolicyInput) (domain.PolicyDecision, error) {
	if !input.Valid() {
		return domain.PolicyDecision{}, errors.New("valid Policy input is required")
	}
	for _, finding := range input.Inspection().Findings() {
		if finding.Code() == "M1_BLOCK_FINDING" {
			return domain.NewPolicyDecision(domain.DecisionBlock, m1PolicyID, m1PolicyVersion, []string{"M1_BLOCK_FINDING"})
		}
	}
	checks := []domain.CheckExecution{input.Verification().Execution(), input.Inspection().Execution()}
	for _, check := range checks {
		if check.Required() && (check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted) {
			return domain.NewPolicyDecision(domain.DecisionManualReview, m1PolicyID, m1PolicyVersion, []string{"M1_REQUIRED_CHECK_INCOMPLETE"})
		}
	}
	return domain.NewPolicyDecision(domain.DecisionAllow, m1PolicyID, m1PolicyVersion, []string{"M1_REQUIRED_CHECKS_COMPLETED"})
}
