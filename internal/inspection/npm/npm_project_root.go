package npm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type NPMProjectRootSourceReader interface {
	ReadSource(context.Context, domain.AcquiredArtifact) (artifactnpm.PNPMLayoutArchive, error)
}

type NPMProjectRootInspector struct {
	input   artifactnpm.NPMProjectRootInspectionInput
	reader  NPMProjectRootSourceReader
	execute func(context.Context, artifactnpm.NPMProjectRootInspectionInput) (domain.SandboxResult, error)
}

func NewNPMProjectRootInspector(input artifactnpm.NPMProjectRootInspectionInput, reader NPMProjectRootSourceReader, execute func(context.Context, artifactnpm.NPMProjectRootInspectionInput) (domain.SandboxResult, error)) (*NPMProjectRootInspector, error) {
	if !input.Valid() || reader == nil || execute == nil {
		return nil, errors.New("npm root inspection requires complete current data and actual consumers")
	}
	return &NPMProjectRootInspector{input, reader, execute}, nil
}

func (i *NPMProjectRootInspector) Inspect(ctx context.Context, subject domain.AcquiredArtifact) (domain.InspectionReport, error) {
	if i == nil || ctx == nil || subject != i.input.Source() {
		return domain.InspectionReport{}, errors.New("npm root inspection differs from the frozen local subject")
	}
	source, err := i.reader.ReadSource(ctx, subject)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	manifest, err := source.ReadFile("project/package.json")
	controls := i.input.Dependencies().Context().Controls()
	if err != nil || string(manifest) != string(controls[0].Body()) {
		return domain.InspectionReport{}, errors.Join(errors.New("npm root manifest differs from complete dependency selection"), err)
	}
	phases, err := artifactnpm.NPMProjectRootLifecycleScripts(manifest)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	implicit := false
	for _, entry := range source.Entries() {
		if entry.Name == "project/binding.gyp" {
			implicit = true
		}
	}
	staticID, _ := domain.NewCheckID("npm-project-root-static")
	staticCheck, _ := domain.NewCheckExecution(staticID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", npmRootCheckProfile())
	staticEvidenceID, _ := domain.NewEvidenceID("npm-project-root-source-structure")
	staticFact, err := domain.NewEvidence(staticEvidenceID, staticID, subject.Identity(), subject.Digest(), "npm-project-root-source-structure", fmt.Sprintf("Bounded local source entries=%d declared_install_hooks=%d binding_gyp=%t. Local bytes are not registry provenance.", len(source.Entries()), len(phases), implicit))
	if err != nil {
		return domain.InspectionReport{}, err
	}
	actionID, _ := domain.NewEvidenceID("npm-project-root-host-action")
	actionKind := domain.EvidenceHostInstallActionNotRequired
	if len(phases) != 0 || implicit {
		actionKind = domain.EvidenceHostInstallActionRequired
	}
	action, err := domain.NewEvidence(actionID, staticID, subject.Identity(), subject.Digest(), actionKind, "Local project install action recorded; observation grants no Host script execution.")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	static, err := domain.NewInspectionReport(staticCheck, nil, []domain.Evidence{staticFact, action})
	if err != nil {
		return domain.InspectionReport{}, err
	}
	result, executionErr := i.execute(ctx, i.input)
	dynamicID, _ := domain.NewCheckID("npm-project-root-lifecycle")
	factsID, _ := domain.NewCheckID("npm-project-root-observation-facts")
	status, limitation := domain.ExecutionCompleted, ""
	if executionErr != nil || result.Status() != domain.SandboxCompleted || ctx.Err() != nil {
		status, limitation = domain.ExecutionIncomplete, "M13_NPM_PROJECT_ROOT_INCOMPLETE"
		if code, known := result.LimitationCode(); known {
			limitation = code
		}
	}
	if status == domain.ExecutionCompleted && !completeDynamicEnvironments(result) {
		status, limitation = domain.ExecutionIncomplete, "M13_DYNAMIC_ENVIRONMENT_INCOMPLETE"
	}
	for _, fact := range result.Observations() {
		if fact.Category() == domain.ObservationResource {
			status, limitation = domain.ExecutionIncomplete, "M3_DYNAMIC_RESOURCE_LIMIT"
		}
	}
	if result.SessionID().String() == "" {
		if len(result.Observations()) != 0 {
			return domain.InspectionReport{}, errors.Join(errors.New("npm root observations lack an actual session"), executionErr)
		}
		check, _ := domain.NewCheckExecution(dynamicID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionNotExecuted, "M13_NPM_PROJECT_ROOT_NOT_EXECUTED", npmRootCheckProfile())
		dynamic, err := domain.NewInspectionReport(check, nil, nil)
		if err != nil {
			return domain.InspectionReport{}, err
		}
		factsCheck, _ := domain.NewCheckExecution(factsID, domain.CheckInspection, false, domain.CapabilitySupported, domain.ExecutionNotExecuted, "M13_NPM_PROJECT_ROOT_NOT_EXECUTED", npmRootCheckProfile())
		facts, err := domain.NewInspectionReport(factsCheck, nil, nil)
		if err != nil {
			return domain.InspectionReport{}, err
		}
		return domain.NewCompositeInspectionReport([]domain.InspectionReport{static, facts, dynamic})
	}
	summary, summaryErr := npmProjectInstanceSummary(result, projectInspectionNPM)
	if summaryErr != nil {
		status, limitation = domain.ExecutionIncomplete, "M11_DYNAMIC_SUMMARY_INVALID"
	}
	if summary == "" {
		return domain.InspectionReport{}, errors.Join(executionErr, summaryErr)
	}
	check, err := domain.NewCheckExecution(factsID, domain.CheckInspection, false, domain.CapabilitySupported, domain.ExecutionCompleted, "", npmRootCheckProfile())
	if err != nil {
		return domain.InspectionReport{}, err
	}
	observedID, _ := domain.NewEvidenceID("npm-project-root-observation")
	observed, err := domain.NewEvidence(observedID, factsID, subject.Identity(), subject.Digest(), "npm-project-root-observation", summary)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	binding, err := json.Marshal(struct{ Graph, Runtime, Source, Session string }{i.input.Dependencies().Context().Snapshot().GraphDigest().String(), pnpmKeyDigest(i.input.Dependencies().Context().RuntimeIdentity()), subject.Digest().String(), result.SessionID().String()})
	if err != nil {
		return domain.InspectionReport{}, err
	}
	bindingID, _ := domain.NewEvidenceID("npm-project-root-binding")
	bound, err := domain.NewEvidence(bindingID, factsID, subject.Identity(), subject.Digest(), "npm-project-root-binding", string(binding))
	if err != nil {
		return domain.InspectionReport{}, err
	}
	findings := []domain.Finding{}
	for _, code := range dynamicFindingCodes(result.Observations()) {
		finding, err := domain.NewFinding(code, []domain.EvidenceID{observedID, bindingID})
		if err != nil {
			return domain.InspectionReport{}, err
		}
		findings = append(findings, finding)
	}
	workspaces, err := workspaceEvidence(subject, factsID, result, "npm-project-root-ws")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	dynamic, err := domain.NewInspectionReport(check, findings, append([]domain.Evidence{observed, bound}, workspaces...))
	if err != nil {
		return domain.InspectionReport{}, err
	}
	coverageCheck, err := domain.NewCheckExecution(dynamicID, domain.CheckInspection, true, domain.CapabilitySupported, status, limitation, npmRootCheckProfile())
	if err != nil {
		return domain.InspectionReport{}, err
	}
	coverageID, _ := domain.NewEvidenceID("npm-project-root-lifecycle-coverage")
	coverageFact, err := domain.NewEvidence(coverageID, dynamicID, subject.Identity(), subject.Digest(), "npm-project-root-lifecycle-coverage", fmt.Sprintf("Native root execution completed=%t; actual session=%s.", status == domain.ExecutionCompleted, result.SessionID().String()))
	if err != nil {
		return domain.InspectionReport{}, err
	}
	coverage, err := domain.NewInspectionReport(coverageCheck, nil, []domain.Evidence{coverageFact})
	if err != nil {
		return domain.InspectionReport{}, err
	}
	return domain.NewCompositeInspectionReport([]domain.InspectionReport{static, dynamic, coverage})
}

func npmRootCheckProfile() domain.ContentDigest {
	profile, _ := domain.CheckProfileDigest("npm-project-root-environments-v2", dynamicCheckRuntimeDescriptor)
	return profile
}

func (i *NPMProjectRootInspector) ExpectedChecks(ctx context.Context, a domain.AcquiredArtifact, kind domain.CheckKind) ([]domain.CheckRequirement, error) {
	if i == nil || ctx == nil || ctx.Err() != nil || kind != domain.CheckInspection || a != i.input.Source() {
		return nil, errors.New("npm root declaration differs from the frozen subject")
	}
	var result []domain.CheckRequirement
	for _, name := range []string{"npm-project-root-static", "npm-project-root-lifecycle", "npm-project-root-observation-facts"} {
		id, _ := domain.NewCheckID(name)
		requirement, err := domain.NewCheckRequirement(id, kind, name != "npm-project-root-observation-facts", npmRootCheckProfile())
		if err != nil {
			return nil, err
		}
		result = append(result, requirement)
	}
	return result, nil
}
