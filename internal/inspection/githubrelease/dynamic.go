package githubrelease

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

type DynamicInspector struct{ sandbox ports.Sandbox }

func NewDynamicInspector(sandbox ports.Sandbox) (*DynamicInspector, error) {
	if sandbox == nil {
		return nil, errors.New("GitHub ELF sandbox is required")
	}
	return &DynamicInspector{sandbox: sandbox}, nil
}

func (i *DynamicInspector) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	if i == nil || i.sandbox == nil || ctx == nil || artifact.Identity().Source().String() != "github-release" {
		return domain.InspectionReport{}, errors.New("GitHub ELF dynamic inspection request is invalid")
	}
	request, err := domain.NewSandboxRequest(artifact)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	result, err := i.sandbox.Execute(ctx, request)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	checkID, _ := domain.NewCheckID("github-release-elf-dynamic")
	if result.Status() != domain.SandboxCompleted {
		limitation, _ := result.LimitationCode()
		capability, status := domain.CapabilitySupported, domain.ExecutionIncomplete
		if limitation == "M3_LINUX_ONLY" || limitation == "M3_RUNTIME_UNAVAILABLE" || limitation == "M3_RUNTIME_VERSION_UNSUPPORTED" || limitation == "M3_IMAGE_UNAVAILABLE" {
			capability, status = domain.CapabilityUnsupported, domain.ExecutionNotExecuted
		}
		check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, capability, status, limitation)
		return domain.NewInspectionReport(check, nil, nil)
	}
	for _, observation := range result.Observations() {
		if observation.Category() == domain.ObservationResource {
			check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionIncomplete, "M6_DYNAMIC_RESOURCE_LIMIT")
			return domain.NewInspectionReport(check, nil, nil)
		}
	}
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidenceID, _ := domain.NewEvidenceID("github-release-elf-dynamic-result")
	evidence, _ := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), "github-release-elf-dynamic", "GitHub Release ELF dynamic inspection completed.")
	findings := []domain.Finding{}
	seen := map[string]bool{}
	for _, observation := range result.Observations() {
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
			finding, _ := domain.NewFinding(code, []domain.EvidenceID{evidenceID})
			findings = append(findings, finding)
			seen[code] = true
		}
	}
	return domain.NewInspectionReport(check, findings, []domain.Evidence{evidence})
}

type CompositeInspector struct {
	static  *StaticInspector
	dynamic *DynamicInspector
}

func NewCompositeInspector(static *StaticInspector, dynamic *DynamicInspector) (*CompositeInspector, error) {
	if static == nil || dynamic == nil {
		return nil, errors.New("GitHub Release composite inspector requires static and dynamic inspectors")
	}
	return &CompositeInspector{static: static, dynamic: dynamic}, nil
}
func (i *CompositeInspector) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	static, err := i.static.Inspect(ctx, artifact)
	if err != nil || len(static.Findings()) != 0 {
		return static, err
	}
	elf := false
	for _, evidence := range static.Evidence() {
		elf = elf || evidence.Kind() == "github-release-elf-static"
	}
	if !elf {
		return static, nil
	}
	dynamic, err := i.dynamic.Inspect(ctx, artifact)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	return domain.NewCompositeInspectionReport([]domain.InspectionReport{static, dynamic})
}
