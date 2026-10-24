package application

import (
	"context"
	"fmt"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// Shared by the existing graph inspection and complete-project inspection.
// All subjects are acquired and verified before the adapter's static-all phase.
func (s *InstallInspectService) inspectDependencyBatch(ctx context.Context, operationID domain.OperationID, nodes []domain.LockedDependency, inspect func(map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error)) ([]domain.DependencyInspection, map[domain.DependencyNodeID]domain.InspectionReport, []DependencyPolicyDiagnostic, error) {
	pending := make([]*pendingDependencyInspection, 0, len(nodes))
	byNode := make(map[domain.DependencyNodeID]*pendingDependencyInspection, len(nodes))
	failPending := func(code string, cause error) ([]domain.DependencyInspection, map[domain.DependencyNodeID]domain.InspectionReport, []DependencyPolicyDiagnostic, error) {
		for _, item := range pending {
			if item.finalized || item.run == nil {
				continue
			}
			_ = item.run.FinalizeFailed(code)
		}
		return nil, nil, nil, cause
	}
	for _, dependency := range nodes {
		runID, err := s.newRunID()
		if err != nil {
			return failPending("RUN_ID_GENERATION_FAILED", fmt.Errorf("generate Run ID for dependency %s: %w", dependency.Node().String(), err))
		}
		reference, err := lockedReference(dependency.Artifact())
		if err != nil {
			return failPending("REFERENCE_INVALID", err)
		}
		run, err := domain.NewInspectionRun(runID, operationID, reference, dependency.Artifact().Identity())
		if err != nil {
			return failPending("RUN_CREATION_FAILED", err)
		}
		if err := run.Activate(); err != nil {
			return failPending("RUN_ACTIVATION_FAILED", err)
		}
		item := &pendingDependencyInspection{dependency: dependency, runID: runID, run: run}
		pending = append(pending, item)
		byNode[dependency.Node()] = item
		ports.ReportProgress(ctx, ports.Progress{Stage: "download"})
		artifact, err := s.artifact.Acquire(ctx, runID, dependency.Artifact())
		if err != nil {
			return failPending("ARTIFACT_ACQUIRE_FAILED", err)
		}
		if err := run.BindAcquiredArtifact(artifact); err != nil {
			return failPending("ARTIFACT_BINDING_FAILED", err)
		}
		if !acquiredMatchesLocked(artifact, dependency, runID) {
			return failPending("ARTIFACT_BINDING_INVALID", fmt.Errorf("acquired artifact binding is invalid for dependency %s", dependency.Node().String()))
		}
		ports.ReportProgress(ctx, ports.Progress{Stage: "verify"})
		verification, err := s.verification.Verify(ctx, artifact)
		if err != nil {
			return failPending("VERIFICATION_PROVIDER_FAILED", err)
		}
		item.artifact, item.verification = artifact, verification
	}
	artifacts := make(map[domain.DependencyNodeID]domain.AcquiredArtifact, len(pending))
	for _, dependency := range nodes {
		item := byNode[dependency.Node()]
		if item == nil {
			return failPending("DEPENDENCY_ARTIFACTS_INVALID", fmt.Errorf("acquired dependency is absent for node %s", dependency.Node().String()))
		}
		artifacts[dependency.Node()] = item.artifact
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "inspect"})
	reports, err := inspect(artifacts)
	if err != nil {
		return failPending("INSPECTION_PROVIDER_FAILED", err)
	}
	if err := validateDependencyReports(nodes, reports); err != nil {
		return failPending("INSPECTION_REPORT_INVALID", err)
	}
	for _, item := range pending {
		report := reports[item.dependency.Node()]
		checks := append([]domain.CheckExecution{item.verification.Execution()}, report.Executions()...)
		evidence := append(item.verification.Evidence(), report.Evidence()...)
		references, err := s.evidence.Record(ctx, item.runID, evidence)
		if err != nil {
			return failPending("EVIDENCE_RECORD_FAILED", err)
		}
		input, err := domain.NewPolicyInput(item.runID, item.artifact, item.verification, report, references)
		if err != nil {
			return failPending("POLICY_INPUT_INVALID", err)
		}
		ports.ReportProgress(ctx, ports.Progress{Stage: "policy"})
		decision, err := s.entryPolicy.Evaluate(input)
		if err != nil {
			return failPending("POLICY_EVALUATION_FAILED", err)
		}
		record, err := newRecordedDependencyInspection(item.dependency.Node(), item.runID, item.artifact, checks, references, decision, evidence)
		if err != nil {
			return failPending("DEPENDENCY_INSPECTION_INVALID", err)
		}
		if err := item.run.FinalizeCompleted(decision); err != nil {
			return failPending("RUN_FINALIZATION_FAILED", err)
		}
		item.finalized = true
		item.decision = decision
		item.inspection = record
	}
	inspections := make([]domain.DependencyInspection, 0, len(pending))
	policyDiagnostics := make([]DependencyPolicyDiagnostic, 0, len(pending))
	for _, item := range pending {
		inspections = append(inspections, item.inspection)
		diagnostic, err := newDependencyPolicyDiagnostic(item.dependency, item.decision, reports[item.dependency.Node()].Findings())
		if err != nil {
			return nil, nil, nil, err
		}
		policyDiagnostics = append(policyDiagnostics, diagnostic)
	}
	return inspections, reports, policyDiagnostics, nil
}
