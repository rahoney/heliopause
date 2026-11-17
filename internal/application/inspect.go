// Package application orchestrates Heliopause use cases through Core contracts.
package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// InspectRequest is a validated ecosystem-neutral Inspect request.
type InspectRequest struct{ reference domain.ArtifactReference }

// NewInspectRequest constructs an Inspect request from a validated Artifact Reference.
func NewInspectRequest(reference domain.ArtifactReference) (InspectRequest, error) {
	if reference.Source().String() == "" {
		return InspectRequest{}, errors.New("artifact reference is required")
	}
	return InspectRequest{reference: reference}, nil
}

// Policy is the deterministic decision boundary consumed by InspectService.
type Policy interface {
	Evaluate(domain.PolicyInput) (domain.PolicyDecision, error)
}

// InspectService executes the M1 Inspect workflow in its canonical order.
type InspectService struct {
	artifact       ports.Artifact
	verification   ports.Verification
	inspection     ports.Inspection
	evidence       ports.Evidence
	policy         Policy
	newOperationID func() (domain.OperationID, error)
	newRunID       func() (domain.RunID, error)
}

// NewInspectService constructs an Inspect workflow from explicit dependencies.
func NewInspectService(artifact ports.Artifact, verification ports.Verification, inspection ports.Inspection, evidence ports.Evidence, policy Policy, newOperationID func() (domain.OperationID, error), newRunID func() (domain.RunID, error)) (*InspectService, error) {
	if artifact == nil || verification == nil || inspection == nil || evidence == nil || policy == nil || newOperationID == nil || newRunID == nil {
		return nil, errors.New("Inspect service requires all Ports, Policy, and ID generators")
	}
	return &InspectService{artifact: artifact, verification: verification, inspection: inspection, evidence: evidence, policy: policy, newOperationID: newOperationID, newRunID: newRunID}, nil
}

// Inspect resolves exact identity, runs normalized checks, records Evidence, and evaluates Policy.
func (s *InspectService) Inspect(ctx context.Context, request InspectRequest) (domain.OperationResult, error) {
	if ctx == nil {
		return domain.OperationResult{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return domain.OperationResult{}, err
	}
	if request.reference.Source().String() == "" {
		return domain.OperationResult{}, errors.New("validated Inspect request is required")
	}
	operationID, err := s.newOperationID()
	if err != nil {
		return domain.OperationResult{}, fmt.Errorf("generate operation ID: %w", err)
	}
	ctx = ports.WithStorageOperation(ctx, operationID)
	ports.ReportProgress(ctx, ports.Progress{Stage: "resolve"})
	resolvedArtifact, err := s.artifact.Resolve(ctx, request.reference)
	if err != nil {
		return failedBeforeRun(operationID, request.reference, domain.ResolvedArtifactIdentity{}, "ARTIFACT_RESOLVE_FAILED", "Artifact identity resolution failed.", fmt.Errorf("resolve Artifact: %w", err))
	}
	identity := resolvedArtifact.Identity()
	runID, err := s.newRunID()
	if err != nil {
		return failedBeforeRun(operationID, request.reference, identity, "RUN_ID_GENERATION_FAILED", "Inspection Run identifier generation failed.", fmt.Errorf("generate Run ID: %w", err))
	}
	run, err := domain.NewInspectionRun(runID, operationID, request.reference, identity)
	if err != nil {
		return failedBeforeRun(operationID, request.reference, identity, "RUN_CREATION_FAILED", "Inspection Run creation failed.", fmt.Errorf("create Inspection Run: %w", err))
	}
	if err := run.Activate(); err != nil {
		return failRun(run, request.reference, domain.AcquiredArtifact{}, nil, nil, "RUN_ACTIVATION_FAILED", "Inspection Run activation failed.", fmt.Errorf("activate Inspection Run: %w", err))
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "download"})
	artifact, err := s.artifact.Acquire(ctx, runID, resolvedArtifact)
	if err != nil {
		return failRun(run, request.reference, domain.AcquiredArtifact{}, nil, nil, "ARTIFACT_ACQUIRE_FAILED", "Artifact acquisition failed.", fmt.Errorf("acquire Artifact: %w", err))
	}
	if err := run.BindAcquiredArtifact(artifact); err != nil {
		return failRun(run, request.reference, domain.AcquiredArtifact{}, nil, nil, "ARTIFACT_BINDING_FAILED", "Acquired Artifact binding failed.", fmt.Errorf("bind acquired Artifact: %w", err))
	}
	plans, err := freezeCheckPlan(ctx, runID, artifact, s.verification, s.inspection, s.policy)
	if err != nil {
		return failRun(run, request.reference, artifact, nil, nil, "CHECK_PLAN_UNAVAILABLE", "Required check declaration is unavailable.", err, plans...)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "verify"})
	verification, err := s.verification.Verify(ctx, artifact)
	if err != nil {
		return failRun(run, request.reference, artifact, nil, nil, "VERIFICATION_PROVIDER_FAILED", "Artifact verification failed operationally.", fmt.Errorf("verify Artifact: %w", err), plans...)
	}
	checks := []domain.CheckExecution{verification.Execution()}
	ports.ReportProgress(ctx, ports.Progress{Stage: "inspect"})
	inspection, err := s.inspection.Inspect(ctx, artifact)
	if err != nil {
		checks, references, recordErr := recordFailedInspection(ctx, s.evidence, runID, artifact, verification, inspection)
		return failRun(run, request.reference, artifact, checks, references, "INSPECTION_PROVIDER_FAILED", "Artifact inspection failed operationally.", errors.Join(fmt.Errorf("inspect Artifact: %w", err), recordErr), plans...)
	}
	checks = append(checks, inspection.Executions()...)
	evidence := append(verification.Evidence(), inspection.Evidence()...)
	ports.ReportProgress(ctx, ports.Progress{Stage: "evidence"})
	references, err := recordInspectionEvidence(ctx, s.evidence, runID, evidence, checks...)
	if err != nil {
		return failRun(run, request.reference, artifact, checks, nil, "EVIDENCE_RECORD_FAILED", "Evidence recording failed.", fmt.Errorf("record Evidence: %w", err), plans...)
	}
	policyInput, err := domain.NewPolicyInput(runID, artifact, verification, inspection, references, plans...)
	if err != nil {
		return failRun(run, request.reference, artifact, checks, references, "POLICY_INPUT_INVALID", "Policy input construction failed.", fmt.Errorf("construct Policy input: %w", err), plans...)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "policy"})
	decision, err := s.policy.Evaluate(policyInput)
	if err != nil {
		return failRun(run, request.reference, artifact, checks, references, "POLICY_EVALUATION_FAILED", "Policy evaluation failed operationally.", fmt.Errorf("evaluate Policy: %w", err), plans...)
	}
	if err := run.FinalizeCompleted(decision); err != nil {
		return failRun(run, request.reference, artifact, checks, references, "RUN_FINALIZATION_FAILED", "Inspection Run finalization failed.", fmt.Errorf("finalize Inspection Run: %w", err), plans...)
	}
	return domain.NewInspectOperationResult(domain.OperationResultData{
		OperationID: operationID, Status: domain.OperationCompleted, Reference: request.reference,
		ResolvedIdentity: identity, Digest: artifact.Digest(), RunID: runID, RunOutcome: domain.RunCompleted,
		Checks: checks, CheckPlan: policyInput.CheckPlan(), Evidence: references, PolicyDecision: decision,
	})
}

func failedBeforeRun(operationID domain.OperationID, reference domain.ArtifactReference, identity domain.ResolvedArtifactIdentity, code, message string, cause error) (domain.OperationResult, error) {
	operationError, err := domain.NewOperationError(code, message)
	if err != nil {
		return domain.OperationResult{}, errors.Join(cause, err)
	}
	result, err := domain.NewInspectOperationResult(domain.OperationResultData{OperationID: operationID, Status: domain.OperationFailed, Reference: reference, ResolvedIdentity: identity, OperationalError: operationError})
	if err != nil {
		return domain.OperationResult{}, errors.Join(cause, err)
	}
	return result, cause
}

func failRun(run *domain.InspectionRun, reference domain.ArtifactReference, artifact domain.AcquiredArtifact, checks []domain.CheckExecution, references []domain.EvidenceReference, code, message string, cause error, plans ...domain.CheckPlan) (domain.OperationResult, error) {
	if err := run.FinalizeFailed(code); err != nil {
		return domain.OperationResult{}, errors.Join(cause, fmt.Errorf("finalize failed Inspection Run: %w", err))
	}
	operationError, err := domain.NewOperationError(code, message)
	if err != nil {
		return domain.OperationResult{}, errors.Join(cause, err)
	}
	data := domain.OperationResultData{
		OperationID: run.OperationID(), Status: domain.OperationFailed, Reference: reference,
		ResolvedIdentity: run.Identity(), RunID: run.ID(), RunOutcome: domain.RunFailed,
		Checks: checks, Evidence: references, OperationalError: operationError,
	}
	if len(plans) == 1 {
		data.CheckPlan = plans[0]
	}
	if artifact.Digest().String() != "" {
		data.Digest = artifact.Digest()
	}
	result, err := domain.NewInspectOperationResult(data)
	if err != nil {
		return domain.OperationResult{}, errors.Join(cause, err)
	}
	return result, cause
}

// Only legacy policies that do not claim current M3 coverage can omit a plan.
// Every configured current provider declares before any check executes.
func freezeCheckPlan(ctx context.Context, run domain.RunID, artifact domain.AcquiredArtifact, verification, inspection any, policy Policy) ([]domain.CheckPlan, error) {
	required, ok := policy.(interface{ RequiresCheckPlan() bool })
	if !ok || !required.RequiresCheckPlan() {
		return nil, nil
	}
	verifier, ok := verification.(ports.CheckPlanner)
	if !ok {
		return nil, errors.New("verification check declaration is unavailable")
	}
	inspector, ok := inspection.(ports.CheckPlanner)
	if !ok {
		return nil, errors.New("inspection check declaration is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	checks, err := verifier.ExpectedChecks(ctx, artifact, domain.CheckVerification)
	if err != nil || ctx.Err() != nil {
		return nil, errors.Join(err, ctx.Err())
	}
	inspections, err := inspector.ExpectedChecks(ctx, artifact, domain.CheckInspection)
	if err != nil || ctx.Err() != nil {
		return nil, errors.Join(err, ctx.Err())
	}
	checks = append(append([]domain.CheckRequirement(nil), checks...), inspections...)
	plan, err := domain.NewCheckPlan(run, artifact, checks)
	if err != nil {
		return nil, err
	}
	return []domain.CheckPlan{plan}, nil
}

// Keep already confirmed facts on an operational failure before payload finish.
// This bounded, cancellation-independent write never evaluates Policy or creates
// a complete inspection. Foreign partial facts cannot replace verified facts.
func recordFailedInspection(ctx context.Context, store ports.Evidence, run domain.RunID, artifact domain.AcquiredArtifact, verification domain.VerificationReport, inspection domain.InspectionReport) ([]domain.CheckExecution, []domain.EvidenceReference, error) {
	checks := []domain.CheckExecution{verification.Execution()}
	facts := verification.Evidence()
	var partialErr error
	for _, fact := range inspection.Evidence() {
		if fact.Identity() != artifact.Identity() || fact.Digest() != artifact.Digest() {
			partialErr = errors.New("failed inspection Evidence has a foreign subject")
			break
		}
	}
	if partialErr == nil {
		checks = append(checks, inspection.Executions()...)
		facts = append(facts, inspection.Evidence()...)
	}
	if len(facts) == 0 {
		return checks, nil, partialErr
	}
	for _, fact := range facts {
		if fact.Identity() != artifact.Identity() || fact.Digest() != artifact.Digest() {
			return checks, nil, errors.Join(partialErr, errors.New("failed verification Evidence has a foreign subject"))
		}
	}
	recordContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	refs, err := store.Record(recordContext, run, facts, checks...)
	if err != nil {
		err = fmt.Errorf("retain failed inspection Evidence: %w", err)
	}
	return checks, refs, errors.Join(partialErr, err)
}

// The existing application owner, rather than a filesystem adapter, interprets
// check completion. Finalization still waits for the outer command/transactions.
func recordInspectionEvidence(ctx context.Context, store ports.Evidence, run domain.RunID, facts []domain.Evidence, checks ...domain.CheckExecution) ([]domain.EvidenceReference, error) {
	refs, err := store.Record(ctx, run, facts, checks...)
	if err != nil {
		return refs, err
	}
	if lifetime, ok := ports.StorageFromContext(ctx).(ports.InspectionStorage); ok {
		complete := len(checks) != 0
		for _, check := range checks {
			if check.Required() && check.Status() != domain.ExecutionCompleted {
				complete = false
			}
		}
		if err := lifetime.NoteInspection(ctx, run, complete); err != nil {
			return refs, err
		}
	}
	return refs, nil
}
