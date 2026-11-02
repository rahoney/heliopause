package npm

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// DynamicInspector interprets raw Sandbox results; it never controls a runtime.
// Required fresh experiments and state evidence are bound to the common
// version owners; one-shot or earlier count-only evidence cannot satisfy them.
const (
	dynamicCheckProfileName       = "npm-dynamic-environment-state-v1"
	dynamicCheckRuntimeDescriptor = runtimeidentity.NodeImageReference + "|" + runtimeidentity.GVisorPatchSHA256 + "|" + string(domain.SandboxEnvironmentMinimal) + "|" + string(domain.SandboxEnvironmentDevelopment) + "|" + domain.SandboxEnvironmentStateSchema
)

type DynamicInspector struct{ sandbox ports.Sandbox }

// NewDynamicInspector constructs an npm lifecycle inspector from the consumer-owned Sandbox Port.
func NewDynamicInspector(sandbox ports.Sandbox) (*DynamicInspector, error) {
	if sandbox == nil {
		return nil, errors.New("sandbox is required")
	}
	return &DynamicInspector{sandbox: sandbox}, nil
}

// Inspect converts bounded M3 raw observations into normalized inspection facts.
func (i *DynamicInspector) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	if ctx == nil {
		return domain.InspectionReport{}, errors.New("context is required")
	}
	if i == nil || i.sandbox == nil || artifact.Identity().Source().String() != "npm" {
		return domain.InspectionReport{}, errors.New("npm dynamic inspection requires an acquired npm Artifact")
	}
	request, err := domain.NewSandboxRequest(artifact)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	result, err := i.sandbox.Execute(ctx, request)
	// A trusted backend may retain an operational cause alongside its valid
	// incomplete result. Keep that first limitation in the required check.
	if err != nil && (result.Status() != domain.SandboxIncomplete || result.SessionID().String() == "") {
		return domain.InspectionReport{}, err
	}
	return normalizeDynamicResult(artifact, result)
}

func normalizeDynamicResult(artifact domain.AcquiredArtifact, result domain.SandboxResult) (domain.InspectionReport, error) {
	profile, _ := domain.CheckProfileDigest(dynamicCheckProfileName, dynamicCheckRuntimeDescriptor)
	checkID, err := domain.NewCheckID("npm-dynamic-lifecycle")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	if result.Status() != domain.SandboxCompleted {
		limitation, _ := result.LimitationCode()
		capability, status := domain.CapabilitySupported, domain.ExecutionIncomplete
		if limitation == "M3_LINUX_ONLY" || limitation == "M3_RUNTIME_UNAVAILABLE" || limitation == "M3_RUNTIME_VERSION_UNSUPPORTED" || limitation == "M3_IMAGE_UNAVAILABLE" {
			capability, status = domain.CapabilityUnsupported, domain.ExecutionNotExecuted
		}
		execution, err := domain.NewCheckExecution(checkID, domain.CheckInspection, true, capability, status, limitation, profile)
		if err != nil {
			return domain.InspectionReport{}, err
		}
		return domain.NewInspectionReport(execution, nil, nil)
	}
	for _, observation := range result.Observations() {
		if observation.Category() == domain.ObservationResource {
			execution, err := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionIncomplete, "M3_DYNAMIC_RESOURCE_LIMIT", profile)
			if err != nil {
				return domain.InspectionReport{}, err
			}
			return domain.NewInspectionReport(execution, nil, nil)
		}
	}
	states := result.EnvironmentStates()
	if len(states) != 2 {
		execution, err := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionIncomplete, "M13_DYNAMIC_ENVIRONMENT_INCOMPLETE", profile)
		if err != nil {
			return domain.InspectionReport{}, err
		}
		return domain.NewInspectionReport(execution, nil, nil)
	}
	execution, err := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", profile)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	summary, err := result.ObservationSummary()
	if err != nil {
		incomplete, checkErr := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionIncomplete, "M11_DYNAMIC_SUMMARY_INVALID", profile)
		if checkErr != nil {
			return domain.InspectionReport{}, checkErr
		}
		return domain.NewInspectionReport(incomplete, nil, nil)
	}
	evidenceID, err := domain.NewEvidenceID("npm-dynamic-lifecycle-result")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	evidence, err := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), "npm-dynamic-lifecycle", summary)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	retained := []domain.Evidence{evidence}
	for _, state := range states {
		summary, err := state.Summary()
		if err != nil {
			return domain.InspectionReport{}, err
		}
		id, err := domain.NewEvidenceID("npm-workspace-" + string(state.Profile))
		if err != nil {
			return domain.InspectionReport{}, err
		}
		fact, err := domain.NewEvidence(id, checkID, artifact.Identity(), artifact.Digest(), "workspace-state", summary)
		if err != nil {
			return domain.InspectionReport{}, err
		}
		retained = append(retained, fact)
	}
	codes := dynamicFindingCodes(result.Observations())
	findings := make([]domain.Finding, 0, len(codes))
	for _, code := range codes {
		finding, err := domain.NewFinding(code, []domain.EvidenceID{evidenceID})
		if err != nil {
			return domain.InspectionReport{}, err
		}
		findings = append(findings, finding)
	}
	return domain.NewInspectionReport(execution, findings, retained)
}

func dynamicFindingCodes(observations []domain.SandboxObservation) []string {
	codes := make([]string, 0)
	seen := map[string]bool{}
	for _, observation := range observations {
		var code string
		switch observation.Category() {
		case domain.ObservationHoneytoken:
			code = "M3_HONEYTOKEN_ACCESS"
		case domain.ObservationNetwork:
			code = "M3_NETWORK_ATTEMPT"
		case domain.ObservationProcess:
			if observation.Subject() == "process-unexpected" || observation.Subject() == "process-exec-unexpected" {
				code = "M3_UNEXPECTED_PROCESS"
			}
		case domain.ObservationFilesystem:
			if observation.Subject() == "filesystem-violation" || observation.Subject() == "filesystem-outside-workspace" {
				code = "M3_FILESYSTEM_VIOLATION"
			}
		}
		if code != "" && !seen[code] {
			seen[code] = true
			codes = append(codes, code)
		}
	}
	return codes
}

func (i *DynamicInspector) ExpectedChecks(ctx context.Context, artifact domain.AcquiredArtifact, kind domain.CheckKind) ([]domain.CheckRequirement, error) {
	if i == nil || i.sandbox == nil || ctx == nil || kind != domain.CheckInspection || artifact.Identity().Source().String() != "npm" {
		return nil, errors.New("npm dynamic declaration is invalid")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id, _ := domain.NewCheckID("npm-dynamic-lifecycle")
	profile, _ := domain.CheckProfileDigest(dynamicCheckProfileName, dynamicCheckRuntimeDescriptor)
	requirement, err := domain.NewCheckRequirement(id, kind, true, profile)
	if err != nil {
		return nil, err
	}
	return []domain.CheckRequirement{requirement}, nil
}
