// Package pypi normalizes PyPI static and gVisor dynamic inspection results.
package pypi

import (
	"context"
	"encoding/json"
	"errors"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// DynamicInspector reconciles the statically admitted observation plan with a
// consumer-owned gVisor runner. It does not control Docker/gVisor directly.
type DynamicInspector struct{ runner sandbox.PythonWheelRunner }

func NewDynamicInspector(runner sandbox.PythonWheelRunner) (*DynamicInspector, error) {
	if runner == nil {
		return nil, errors.New("python wheel runner is required")
	}
	return &DynamicInspector{runner: runner}, nil
}

// InspectWheel translates externally completed bounded observations into
// generic Domain evidence. Completion does not assert normal import return.
// An unavailable, failed or incomplete session is not a successful report.
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

func (i *DynamicInspector) InspectWheelWithPrerequisites(ctx context.Context, artifact domain.AcquiredArtifact, static artifactpypi.WheelInspection, closure, inputs []domain.AcquiredArtifact) (domain.InspectionReport, error) {
	if len(inputs) != 1 {
		return domain.InspectionReport{}, errors.New("inspection prerequisite input must be explicit and bounded")
	}
	return i.inspectWheelWithRunner(ctx, artifact, static, closure, i.runner, inputs...)
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

func (i *DynamicInspector) inspectWheelWithRunner(ctx context.Context, artifact domain.AcquiredArtifact, static artifactpypi.WheelInspection, closure []domain.AcquiredArtifact, runner pythonWheelInspectionRunner, inputs ...domain.AcquiredArtifact) (domain.InspectionReport, error) {
	if i == nil || i.runner == nil || ctx == nil || (artifact.Identity().Variant() != "wheel" && artifact.Identity().Variant() != "derived-wheel") || static.Project != artifact.Identity().Name() || static.Version != artifact.Identity().Version() || static.NoImportSurface && len(static.ImportNames) != 0 {
		return domain.InspectionReport{}, errors.New("pypi dynamic inspection request is invalid")
	}
	if _, ok := artifactpypi.ProfileForSource(artifact.Identity().Source()); !ok {
		return domain.InspectionReport{}, errors.New("pypi dynamic inspection source is unsupported")
	}
	resourcePolicy := artifactpypi.ResourcePolicyFromContext(ctx)
	plan, err := artifactpypi.BuildObservationPlan(static, resourcePolicy)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	if err := artifactpypi.ValidateTypedObservationPlan(plan, resourcePolicy); err != nil {
		return domain.InspectionReport{}, err
	}
	if !plan.Admissible() {
		checkID, err := domain.NewCheckID("pypi-dynamic-import")
		if err != nil {
			return domain.InspectionReport{}, err
		}
		return incompleteReport(checkID, "M5_PYPI_DYNAMIC_SURFACE_UNSUPPORTED")
	}
	var result domain.SandboxResult
	var observed sandbox.PythonObservationResult
	if len(inputs) != 0 {
		planRunner, ok := runner.(sandbox.InspectionPrerequisitePythonWheelRunner)
		if !ok {
			return domain.InspectionReport{}, errors.New("dynamic runner cannot bind inspection prerequisites")
		}
		observed, err = planRunner.InspectWheelWithPrerequisites(ctx, artifact, plan, closure, inputs)
	} else if planRunner, ok := runner.(sandbox.PlanAwarePythonWheelRunner); ok {
		observed, err = planRunner.InspectWheelWithPlan(ctx, artifact, plan, closure)
	} else if static.NoImportSurface {
		noImportRunner, ok := i.runner.(sandbox.NoImportSurfacePythonWheelRunner)
		if !ok {
			return domain.InspectionReport{}, errors.New("pypi dynamic runner cannot inspect a proven no-import wheel")
		}
		result, err = noImportRunner.InspectWheelWithoutImportSurface(ctx, artifact, closure)
	} else if len(plan.SiteStartupHooks) != 0 || len(plan.EntryPointCoverage) != 0 || len(plan.ScriptCoverage) != 0 {
		return domain.InspectionReport{}, errors.New("pypi dynamic runner cannot execute typed observation plan")
	} else if closureRunner, ok := runner.(pythonWheelClosureRunner); ok {
		result, err = closureRunner.InspectWheelWithClosure(ctx, artifact, static.ImportNames, closure)
	} else {
		result, err = runner.InspectWheel(ctx, artifact, static.ImportNames)
	}
	if observed.SessionID().String() != "" {
		result = observed.SandboxResult
	}
	if err != nil {
		return domain.InspectionReport{}, err
	}
	checkName, evidenceName := "pypi-dynamic-import", "pypi-dynamic-import-result"
	if static.NoImportSurface {
		checkName, evidenceName = "pypi-dynamic-import-not-applicable", "pypi-dynamic-import-not-applicable-result"
	}
	if len(inputs) != 0 {
		checkName, evidenceName = "pypi-dynamic-import-supplemented", "pypi-dynamic-import-supplemented-result"
	}
	checkID, err := domain.NewCheckID(checkName)
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
	if len(inputs) != 0 {
		binding, err := inspectionEnvironmentBinding(ctx, artifact, plan, closure, inputs)
		if err != nil {
			return domain.InspectionReport{}, err
		}
		// The supplemental check deliberately makes no assertion that these
		// broader dependency probes work without the inspection-only input.
		identity := inputs[0].Identity()
		encoded, encodeErr := json.Marshal(struct {
			Schema              string          `json:"schema"`
			Scope               string          `json:"scope"`
			OriginalEnvironment string          `json:"original_environment"`
			Source              string          `json:"input_source"`
			Project             string          `json:"input_project"`
			Version             string          `json:"input_version"`
			SHA256              string          `json:"input_sha256"`
			Reason              string          `json:"reason"`
			Environment         string          `json:"environment_sha256"`
			Observations        json.RawMessage `json:"observations"`
		}{"m5-inspection-environment/v1", "inspection-only-prerequisite", "NOT_ATTESTED", identity.Source().String(), identity.Name(), identity.Version(), inputs[0].Digest().String(), "BROADER_MODULE_PROBES", binding, json.RawMessage(summary)})
		if encodeErr != nil {
			return domain.InspectionReport{}, encodeErr
		}
		summary = string(encoded)
	}
	evidenceID, err := domain.NewEvidenceID(evidenceName)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	evidence, err := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), checkName, summary)
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
	commandEvidence, err := commandObservationEvidence(checkID, artifact, plan, inputs, observed.Commands, len(findings) == 0)
	if err != nil {
		return incompleteReport(checkID, "M5_PYPI_COMMAND_EVIDENCE_INCOMPLETE")
	}
	return domain.NewInspectionReport(execution, findings, append([]domain.Evidence{evidence}, commandEvidence...))
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
