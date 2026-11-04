package policy

import (
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	m3PolicyID      = "m3-npm-dynamic-inspect"
	m3PolicyVersion = 4
)

// M3 requires completed integrity, static, and dynamic checks before allowing an npm Artifact.
type M3 struct{}

func (M3) RequiresCheckPlan() bool { return true }

func (M3) Evaluate(input domain.PolicyInput) (domain.PolicyDecision, error) {
	if !input.Valid() {
		return domain.PolicyDecision{}, errors.New("valid Policy input is required")
	}
	if input.Verification().Outcome() == domain.VerificationMismatch {
		return m3Decision(domain.DecisionBlock, "M3_INTEGRITY_MISMATCH")
	}
	reviewReason := ""
	for _, finding := range input.Inspection().Findings() {
		switch finding.Code() {
		case "M3_HONEYTOKEN_ACCESS", "M3_FILESYSTEM_VIOLATION":
			return m3Decision(domain.DecisionBlock, finding.Code())
		case "M3_NETWORK_ATTEMPT", "M3_UNEXPECTED_PROCESS", "M13_ADVISORY_REVIEW", "M13_SOURCE_REVIEW", "M13_BINARY_REVIEW":
			if reviewReason == "" {
				reviewReason = finding.Code()
			}
		default:
			return m3Decision(domain.DecisionBlock, finding.Code())
		}
	}
	if !input.CheckPlan().Valid() {
		return m3Decision(domain.DecisionManualReview, "M3_CHECK_PLAN_UNDECLARED")
	}
	if reviewReason != "" {
		return m3Decision(domain.DecisionManualReview, reviewReason)
	}
	checks := append([]domain.CheckExecution{input.Verification().Execution()}, input.Inspection().Executions()...)
	for _, check := range checks {
		if check.Required() && (check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted) {
			return m3Decision(domain.DecisionManualReview, "M3_REQUIRED_CHECK_INCOMPLETE")
		}
	}
	return m3Decision(domain.DecisionAllow, "M3_REQUIRED_CHECKS_COMPLETED")
}

func m3Decision(decision domain.Decision, reason string) (domain.PolicyDecision, error) {
	return domain.NewPolicyDecision(decision, m3PolicyID, m3PolicyVersion, []string{reason})
}
