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
	return i.inspectWheel(ctx, artifact, static, []domain.AcquiredArtifact{artifact})
}

// InspectWheelWithClosure runs the same required dynamic check with a
// caller-provided exact graph closure installed as a network-disabled fixture.
// The report remains attributed only to the target artifact.
func (i *DynamicInspector) InspectWheelWithClosure(ctx context.Context, artifact domain.AcquiredArtifact, static artifactpypi.WheelInspection, closure []domain.AcquiredArtifact) (domain.InspectionReport, error) {
	runner, ok := i.runner.(sandbox.DependencyAwarePythonWheelRunner)
	if !ok {
		return domain.InspectionReport{}, errors.New("pypi dynamic runner does not support dependency closure")
	}
	return i.inspectWheelWithRunner(ctx, artifact, static, closure, runner)
}

func (i *DynamicInspector) inspectWheel(ctx context.Context, artifact domain.AcquiredArtifact, static artifactpypi.WheelInspection, closure []domain.AcquiredArtifact) (domain.InspectionReport, error) {
	if runner, ok := i.runner.(sandbox.DependencyAwarePythonWheelRunner); ok && len(closure) > 1 {
		return i.inspectWheelWithRunner(ctx, artifact, static, closure, runner)
	}
	return i.inspectWheelWithRunner(ctx, artifact, static, []domain.AcquiredArtifact{artifact}, i.runner)
}

type pythonWheelInspectionRunner interface {
	InspectWheel(context.Context, domain.AcquiredArtifact, []string) (domain.SandboxResult, error)
}

type pythonWheelClosureRunner interface {
	InspectWheelWithClosure(context.Context, domain.AcquiredArtifact, []string, []domain.AcquiredArtifact) (domain.SandboxResult, error)
}

func (i *DynamicInspector) inspectWheelWithRunner(ctx context.Context, artifact domain.AcquiredArtifact, static artifactpypi.WheelInspection, closure []domain.AcquiredArtifact, runner pythonWheelInspectionRunner) (domain.InspectionReport, error) {
	if i == nil || i.runner == nil || ctx == nil || (artifact.Identity().Variant() != "wheel" && artifact.Identity().Variant() != "derived-wheel") || static.Project != artifact.Identity().Name() || static.Version != artifact.Identity().Version() || len(static.ImportNames) == 0 {
		return domain.InspectionReport{}, errors.New("pypi dynamic inspection request is invalid")
	}
	if _, ok := artifactpypi.ProfileForSource(artifact.Identity().Source()); !ok {
		return domain.InspectionReport{}, errors.New("pypi dynamic inspection source is unsupported")
	}
	var result domain.SandboxResult
	var err error
	if closureRunner, ok := runner.(pythonWheelClosureRunner); ok {
		result, err = closureRunner.InspectWheelWithClosure(ctx, artifact, static.ImportNames, closure)
	} else {
		result, err = runner.InspectWheel(ctx, artifact, static.ImportNames)
	}
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
