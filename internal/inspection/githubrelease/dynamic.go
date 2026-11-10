package githubrelease

import (
	"context"
	"errors"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
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
	return githubDynamicReport(artifact, result, "github-release-elf-dynamic")
}

// Both fixed GitHub runtime consumers retain the same observation normalization.
func githubDynamicReport(artifact domain.AcquiredArtifact, result domain.SandboxResult, name string) (domain.InspectionReport, error) {
	checkID, _ := domain.NewCheckID(name)
	if result.Status() != domain.SandboxCompleted {
		limitation, _ := result.LimitationCode()
		capability, status := domain.CapabilitySupported, domain.ExecutionIncomplete
		if limitation == "M3_LINUX_ONLY" || limitation == "M3_RUNTIME_UNAVAILABLE" || limitation == "M3_RUNTIME_VERSION_UNSUPPORTED" || limitation == "M3_IMAGE_UNAVAILABLE" || (name == pipxRuntimeCheck && limitation == "P4_PIPX_RUNTIME_UNAVAILABLE") {
			capability, status = domain.CapabilityUnsupported, domain.ExecutionNotExecuted
		}
		check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, capability, status, limitation, githubDynamicProfile(name))
		return domain.NewInspectionReport(check, nil, nil)
	}
	if name != pipxRuntimeCheck {
		states := result.EnvironmentStates()
		if len(states) != 2 || !states[0].VFSOperationsCovered || !states[1].VFSOperationsCovered {
			check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionIncomplete, "M6_DYNAMIC_ENVIRONMENT_INCOMPLETE", githubDynamicProfile(name))
			return domain.NewInspectionReport(check, nil, nil)
		}
	}
	for _, observation := range result.Observations() {
		if observation.Category() == domain.ObservationResource {
			check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionIncomplete, "M6_DYNAMIC_RESOURCE_LIMIT", githubDynamicProfile(name))
			return domain.NewInspectionReport(check, nil, nil)
		}
	}
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", githubDynamicProfile(name))
	summary, err := result.ObservationSummary()
	if err != nil {
		incomplete, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionIncomplete, "M11_DYNAMIC_SUMMARY_INVALID", githubDynamicProfile(name))
		return domain.NewInspectionReport(incomplete, nil, nil)
	}
	evidenceID, _ := domain.NewEvidenceID(name + "-result")
	evidence, err := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), name, summary)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	facts := []domain.Evidence{evidence}
	states := result.EnvironmentStates()
	if name == pipxRuntimeCheck {
		states = nil
	}
	for _, state := range states {
		summary, err := state.Summary()
		if err != nil {
			return domain.InspectionReport{}, err
		}
		id, _ := domain.NewEvidenceID("github-elf-workspace-" + string(state.Profile))
		fact, err := domain.NewEvidence(id, checkID, artifact.Identity(), artifact.Digest(), "workspace-state", summary)
		if err != nil {
			return domain.InspectionReport{}, err
		}
		facts = append(facts, fact)
	}
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
	return domain.NewInspectionReport(check, findings, facts)
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
	if err != nil {
		return static, err
	}
	if len(static.Findings()) != 0 {
		if !githubReleaseRequiresELFProbe(artifact) {
			return static, nil
		}
		id, _ := domain.NewCheckID("github-release-elf-dynamic")
		check, _ := domain.NewCheckExecution(id, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionNotExecuted, "M6_STATIC_PREREQUISITE_FAILED", githubDynamicProfile("github-release-elf-dynamic"))
		skipped, e := domain.NewInspectionReport(check, nil, nil)
		if e != nil {
			return domain.InspectionReport{}, e
		}
		return domain.NewCompositeInspectionReport([]domain.InspectionReport{static, skipped})
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

func githubDynamicProfile(name string) domain.ContentDigest {
	image := runtimeidentity.NodeImageReference
	if name == pipxRuntimeCheck {
		image = runtimeidentity.PythonImageReference + "|" + runtimeidentity.PipxToolSHA256
	}
	revision := name + "-v1"
	if name != pipxRuntimeCheck {
		revision = name + "-environments-v3"
		image += "|" + runtimeidentity.GVisorPatchSHA256 + "|" + domain.SandboxEnvironmentStateSchema + "|" + string(domain.SandboxEnvironmentMinimal) + "|" + string(domain.SandboxEnvironmentDevelopment) + "|" + runtimeidentity.SandboxResponderInspectionIdentity
	}
	profile, _ := domain.CheckProfileDigest(revision, image)
	return profile
}

func githubReleaseRequiresELFProbe(a domain.AcquiredArtifact) bool {
	name := a.Identity().Variant()
	return !strings.HasSuffix(name, ".zip") && !strings.HasSuffix(name, ".tar.gz")
}

func (i *CompositeInspector) ExpectedChecks(ctx context.Context, a domain.AcquiredArtifact, kind domain.CheckKind) ([]domain.CheckRequirement, error) {
	if i == nil || i.static == nil || i.dynamic == nil {
		return nil, errors.New("github release inspection declaration is unavailable")
	}
	result, err := i.static.ExpectedChecks(ctx, a, kind)
	if err != nil || !githubReleaseRequiresELFProbe(a) {
		return result, err
	}
	id, _ := domain.NewCheckID("github-release-elf-dynamic")
	requirement, err := domain.NewCheckRequirement(id, kind, true, githubDynamicProfile("github-release-elf-dynamic"))
	return append(result, requirement), err
}
