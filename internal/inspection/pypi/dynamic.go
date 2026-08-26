// Package pypi normalizes PyPI static and gVisor dynamic inspection results.
package pypi

import (
	"context"
	"errors"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// DynamicInspector imports only statically declared wheel import names through
// a consumer-owned gVisor runner. It does not control Docker/gVisor directly.
type DynamicInspector struct{ runner sandbox.PythonWheelRunner }

func NewDynamicInspector(runner sandbox.PythonWheelRunner) (*DynamicInspector, error) {
	if runner == nil {
		return nil, errors.New("python wheel runner is required")
	}
	return &DynamicInspector{runner: runner}, nil
}

// InspectWheel translates a completed isolated import run into generic Domain
// evidence. An unavailable, failed or incomplete session is deliberately not a
// successful inspection report.
func (i *DynamicInspector) InspectWheel(ctx context.Context, artifact domain.AcquiredArtifact, static artifactpypi.WheelInspection) (domain.InspectionReport, error) {
	if i == nil || i.runner == nil || ctx == nil || (artifact.Identity().Variant() != "wheel" && artifact.Identity().Variant() != "derived-wheel") || static.Project != artifact.Identity().Name() || static.Version != artifact.Identity().Version() || len(static.ImportNames) == 0 {
		return domain.InspectionReport{}, errors.New("pypi dynamic inspection request is invalid")
	}
	if _, ok := artifactpypi.ProfileForSource(artifact.Identity().Source()); !ok {
		return domain.InspectionReport{}, errors.New("pypi dynamic inspection source is unsupported")
	}
	result, err := i.runner.InspectWheel(ctx, artifact, static.ImportNames)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	checkID, err := domain.NewCheckID("pypi-dynamic-import")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	if result.Status() != domain.SandboxCompleted {
		limitation, _ := result.LimitationCode()
		capability, status := domain.CapabilitySupported, domain.ExecutionIncomplete
		if limitation == "M5_PYPI_DYNAMIC_RUNTIME_UNAVAILABLE" {
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
			return incompleteReport(checkID, "M5_PYPI_DYNAMIC_RESOURCE_LIMIT")
		}
	}
	execution, err := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	summary, err := result.ObservationSummary()
	if err != nil {
		return incompleteReport(checkID, "M11_DYNAMIC_SUMMARY_INVALID")
	}
	evidenceID, err := domain.NewEvidenceID("pypi-dynamic-import-result")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	evidence, err := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), "pypi-dynamic-import", summary)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	findings := make([]domain.Finding, 0)
	for _, code := range pypiDynamicFindingCodes(result.Observations()) {
		finding, findingErr := domain.NewFinding(code, []domain.EvidenceID{evidenceID})
		if findingErr != nil {
			return domain.InspectionReport{}, findingErr
		}
		findings = append(findings, finding)
	}
	return domain.NewInspectionReport(execution, findings, []domain.Evidence{evidence})
}

func incompleteReport(checkID domain.CheckID, limitation string) (domain.InspectionReport, error) {
	execution, err := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionIncomplete, limitation)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	return domain.NewInspectionReport(execution, nil, nil)
}
func pypiDynamicFindingCodes(observations []domain.SandboxObservation) []string {
	seen := map[string]bool{}
	var codes []string
	for _, observation := range observations {
		code := ""
		switch observation.Category() {
		case domain.ObservationHoneytoken:
			code = "M3_HONEYTOKEN_ACCESS"
		case domain.ObservationNetwork:
			code = "M3_NETWORK_ATTEMPT"
		case domain.ObservationFilesystem:
			if observation.Subject() == "filesystem-violation" || observation.Subject() == "filesystem-outside-workspace" {
				code = "M3_FILESYSTEM_VIOLATION"
			}
		case domain.ObservationProcess:
			if observation.Subject() == "process-unexpected" || observation.Subject() == "process-exec-unexpected" {
				code = "M3_UNEXPECTED_PROCESS"
			}
		}
		if code != "" && !seen[code] {
			seen[code] = true
			codes = append(codes, code)
		}
	}
	return codes
}
