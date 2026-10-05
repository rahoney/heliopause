package gomodule

import (
	"context"
	"errors"
	"reflect"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// BuildInspector interprets build observations. It cannot operate a runtime,
// manufacture verification or authorize output publication.
type BuildSandbox interface {
	Build(context.Context, domain.ProjectBuildInputs) (domain.ProjectBuildObservation, error)
}

type BuildInspector struct{ sandbox BuildSandbox }

func NewBuildInspector(backend BuildSandbox) (*BuildInspector, error) {
	if backend == nil {
		return nil, errors.New("project build inspector requires a sandbox")
	}
	return &BuildInspector{backend}, nil
}

func (i *BuildInspector) InspectBuild(ctx context.Context, inputs domain.ProjectBuildInputs) (domain.ProjectBuildReport, error) {
	if i == nil || ctx == nil || !inputs.Valid() {
		return domain.ProjectBuildReport{}, errors.New("project build inspection request is invalid")
	}
	observation, err := i.sandbox.Build(ctx, inputs)
	if err != nil {
		return domain.ProjectBuildReport{}, err
	}
	if !reflect.DeepEqual(observation.Inputs(), inputs) || observation.SandboxResult().Status() != domain.SandboxCompleted {
		return domain.ProjectBuildReport{}, errors.New("project build observation is incomplete or substituted")
	}
	result := observation.SandboxResult()
	checkID, _ := domain.NewCheckID("project-build-observation")
	status, limitation := domain.ExecutionCompleted, ""
	for _, fact := range result.Observations() {
		if fact.Category() == domain.ObservationResource {
			status, limitation = domain.ExecutionIncomplete, "M12_GO_BUILD_RESOURCE_LIMIT"
		}
	}
	summary, err := result.ObservationSummary()
	if err != nil {
		return domain.ProjectBuildReport{}, errors.New("project build observation summary is invalid")
	}
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, status, limitation)
	observationID, _ := domain.NewEvidenceID("project-build-observation-result")
	bindingID, _ := domain.NewEvidenceID("project-build-output-binding")
	facts := make([]domain.Evidence, 0, 2)
	for _, item := range []struct {
		id            domain.EvidenceID
		kind, summary string
	}{{observationID, "go-build-observation", summary}, {bindingID, "go-build-output-binding", domain.BuildBindingSummary(inputs, observation.Output(), observation.Binding())}} {
		evidence, err := domain.NewEvidence(item.id, checkID, inputs.Source().Identity(), inputs.Source().Digest(), item.kind, item.summary)
		if err != nil {
			return domain.ProjectBuildReport{}, err
		}
		facts = append(facts, evidence)
	}
	seen := map[string]bool{}
	findings := []domain.Finding{}
	for _, fact := range result.Observations() {
		code := ""
		switch fact.Category() {
		case domain.ObservationHoneytoken:
			code = "M3_HONEYTOKEN_ACCESS"
		case domain.ObservationNetwork:
			code = "M3_NETWORK_ATTEMPT"
		case domain.ObservationFilesystem:
			if fact.Subject() == "filesystem-violation" || fact.Subject() == "filesystem-outside-workspace" {
				code = "M3_FILESYSTEM_VIOLATION"
			}
		case domain.ObservationProcess:
			if fact.Subject() == "process-unexpected" || fact.Subject() == "process-exec-unexpected" {
				code = "M3_UNEXPECTED_PROCESS"
			}
		}
		if code != "" && !seen[code] {
			finding, err := domain.NewFinding(code, []domain.EvidenceID{observationID})
			if err != nil {
				return domain.ProjectBuildReport{}, err
			}
			findings = append(findings, finding)
			seen[code] = true
		}
	}
	report, err := domain.NewInspectionReport(check, findings, facts)
	if err != nil {
		return domain.ProjectBuildReport{}, err
	}
	return domain.NewProjectBuildReport(observation, report)
}
