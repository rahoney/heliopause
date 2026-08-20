package policy

import (
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// M6 evaluates one standalone GitHub Release asset. Unsupported formats and
// uncompleted required checks are reviewable; integrity and unsafe archive
// findings are blocking.
type M6 struct{}

func (M6) Evaluate(input domain.PolicyInput) (domain.PolicyDecision, error) {
	if !input.Valid() || input.Artifact().Identity().Source().String() != "github-release" {
		return domain.PolicyDecision{}, errors.New("valid GitHub Release Policy input is required")
	}
	if input.Verification().Outcome() == domain.VerificationMismatch {
		return m6Decision(domain.DecisionBlock, "M6_INTEGRITY_MISMATCH")
	}
	for _, finding := range input.Inspection().Findings() {
		switch finding.Code() {
		case "M6_FORMAT_UNSUPPORTED", "M6_FORMAT_NAME_MISMATCH", "M6_ELF_PLATFORM_UNSUPPORTED":
			return m6Decision(domain.DecisionManualReview, finding.Code())
		default:
			return m6Decision(domain.DecisionBlock, finding.Code())
		}
	}
	for _, check := range append([]domain.CheckExecution{input.Verification().Execution()}, input.Inspection().Executions()...) {
		if check.Required() && (check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted) {
			return m6Decision(domain.DecisionManualReview, "M6_REQUIRED_CHECK_INCOMPLETE")
		}
	}
	elf, dynamic := false, false
	for _, evidence := range input.Inspection().Evidence() {
		elf = elf || evidence.Kind() == "github-release-elf-static"
		dynamic = dynamic || evidence.Kind() == "github-release-elf-dynamic"
	}
	if elf && !dynamic {
		return m6Decision(domain.DecisionManualReview, "M6_DYNAMIC_INSPECTION_REQUIRED")
	}
	return m6Decision(domain.DecisionAllow, "M6_REQUIRED_CHECKS_COMPLETED")
}

func m6Decision(decision domain.Decision, reason string) (domain.PolicyDecision, error) {
	return domain.NewPolicyDecision(decision, "m6-github-release", 1, []string{reason})
}
