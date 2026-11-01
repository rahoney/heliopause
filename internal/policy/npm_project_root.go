package policy

import "github.com/rahoney/heliopause/internal/core/domain"

// NPMProjectRoot retains M3 behavior findings and the existing M4 restriction on
// executing untrusted install actions on the Host. Local code is not a package
// dependency/primary, and successful observation cannot grant Host execution.
type NPMProjectRoot struct{}

func (NPMProjectRoot) Evaluate(input domain.PolicyInput) (domain.PolicyDecision, error) {
	decision, err := (M3{}).Evaluate(input)
	if err != nil || decision.Decision() != domain.DecisionAllow {
		return decision, err
	}
	if input.Artifact().Identity().Source().String() != "project-local" || input.Artifact().Identity().Variant() != "npm-source" {
		return m4Decision(domain.DecisionManualReview, "M4_HOST_LIFECYCLE_UNCONFIRMED")
	}
	expected := map[string]bool{"project-build-input-binding": true, "npm-project-root-static": true, "npm-project-root-lifecycle": true, "npm-project-root-observation-facts": false}
	seen := map[string]bool{}
	checks := append([]domain.CheckExecution{input.Verification().Execution()}, input.Inspection().Executions()...)
	for _, check := range checks {
		required, known := expected[check.ID().String()]
		if !known || seen[check.ID().String()] || check.Required() != required || check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted {
			return m4Decision(domain.DecisionManualReview, "M4_REQUIRED_CHECK_INCOMPLETE")
		}
		seen[check.ID().String()] = true
	}
	if len(checks) != len(expected) {
		return m4Decision(domain.DecisionManualReview, "M4_REQUIRED_CHECK_INCOMPLETE")
	}
	var action domain.Evidence
	count := 0
	for _, fact := range input.Inspection().Evidence() {
		if fact.Kind() != domain.EvidenceHostInstallActionRequired && fact.Kind() != domain.EvidenceHostInstallActionNotRequired {
			continue
		}
		if fact.Identity() != input.Artifact().Identity() || fact.Digest() != input.Artifact().Digest() || fact.CheckID().String() != "npm-project-root-static" {
			return m4Decision(domain.DecisionManualReview, "M4_HOST_LIFECYCLE_UNCONFIRMED")
		}
		action = fact
		count++
	}
	if count != 1 {
		return m4Decision(domain.DecisionManualReview, "M4_HOST_LIFECYCLE_UNCONFIRMED")
	}
	if action.Kind() == domain.EvidenceHostInstallActionRequired {
		return m4Decision(domain.DecisionManualReview, "M4_HOST_LIFECYCLE_UNSUPPORTED")
	}
	return decision, nil
}
