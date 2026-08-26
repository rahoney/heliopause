package npm

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// DynamicInspector interprets raw Sandbox results; it never controls a runtime.
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
	if err != nil {
		return domain.InspectionReport{}, err
	}
	return normalizeDynamicResult(artifact, result)
}

func normalizeDynamicResult(artifact domain.AcquiredArtifact, result domain.SandboxResult) (domain.InspectionReport, error) {
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
		execution, err := domain.NewCheckExecution(checkID, domain.CheckInspection, true, capability, status, limitation)
		if err != nil {
			return domain.InspectionReport{}, err
		}
		return domain.NewInspectionReport(execution, nil, nil)
	}
	for _, observation := range result.Observations() {
		if observation.Category() == domain.ObservationResource {
			execution, err := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionIncomplete, "M3_DYNAMIC_RESOURCE_LIMIT")
			if err != nil {
				return domain.InspectionReport{}, err
			}
			return domain.NewInspectionReport(execution, nil, nil)
		}
	}
	execution, err := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	summary, err := result.ObservationSummary()
	if err != nil {
		incomplete, checkErr := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionIncomplete, "M11_DYNAMIC_SUMMARY_INVALID")
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
	codes := dynamicFindingCodes(result.Observations())
	findings := make([]domain.Finding, 0, len(codes))
	for _, code := range codes {
		finding, err := domain.NewFinding(code, []domain.EvidenceID{evidenceID})
		if err != nil {
			return domain.InspectionReport{}, err
		}
		findings = append(findings, finding)
	}
	return domain.NewInspectionReport(execution, findings, []domain.Evidence{evidence})
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
