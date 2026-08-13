// Package application orchestrates Heliopause use cases through Core contracts.
package application

import (
	"context"
	"errors"
	"fmt"

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
	artifact, err := s.artifact.Acquire(ctx, runID, resolvedArtifact)
	if err != nil {
		return failRun(run, request.reference, domain.AcquiredArtifact{}, nil, nil, "ARTIFACT_ACQUIRE_FAILED", "Artifact acquisition failed.", fmt.Errorf("acquire Artifact: %w", err))
	}
	if err := run.BindAcquiredArtifact(artifact); err != nil {
		return failRun(run, request.reference, domain.AcquiredArtifact{}, nil, nil, "ARTIFACT_BINDING_FAILED", "Acquired Artifact binding failed.", fmt.Errorf("bind acquired Artifact: %w", err))
	}
	verification, err := s.verification.Verify(ctx, artifact)
	if err != nil {
		return failRun(run, request.reference, artifact, nil, nil, "VERIFICATION_PROVIDER_FAILED", "Artifact verification failed operationally.", fmt.Errorf("verify Artifact: %w", err))
	}
	checks := []domain.CheckExecution{verification.Execution()}
	inspection, err := s.inspection.Inspect(ctx, artifact)
	if err != nil {
		return failRun(run, request.reference, artifact, checks, nil, "INSPECTION_PROVIDER_FAILED", "Artifact inspection failed operationally.", fmt.Errorf("inspect Artifact: %w", err))
	}
	checks = append(checks, inspection.Executions()...)
	evidence := append(verification.Evidence(), inspection.Evidence()...)
	references, err := s.evidence.Record(ctx, runID, evidence)
	if err != nil {
		return failRun(run, request.reference, artifact, checks, nil, "EVIDENCE_RECORD_FAILED", "Evidence recording failed.", fmt.Errorf("record Evidence: %w", err))
	}
	policyInput, err := domain.NewPolicyInput(runID, artifact, verification, inspection, references)
	if err != nil {
		return failRun(run, request.reference, artifact, checks, references, "POLICY_INPUT_INVALID", "Policy input construction failed.", fmt.Errorf("construct Policy input: %w", err))
	}
	decision, err := s.policy.Evaluate(policyInput)
	if err != nil {
		return failRun(run, request.reference, artifact, checks, references, "POLICY_EVALUATION_FAILED", "Policy evaluation failed operationally.", fmt.Errorf("evaluate Policy: %w", err))
	}
	if err := run.FinalizeCompleted(decision); err != nil {
		return failRun(run, request.reference, artifact, checks, references, "RUN_FINALIZATION_FAILED", "Inspection Run finalization failed.", fmt.Errorf("finalize Inspection Run: %w", err))
	}
	return domain.NewInspectOperationResult(domain.OperationResultData{
		OperationID: operationID, Status: domain.OperationCompleted, Reference: request.reference,
		ResolvedIdentity: identity, Digest: artifact.Digest(), RunID: runID, RunOutcome: domain.RunCompleted,
		Checks: checks, Evidence: references, PolicyDecision: decision,
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

func failRun(run *domain.InspectionRun, reference domain.ArtifactReference, artifact domain.AcquiredArtifact, checks []domain.CheckExecution, references []domain.EvidenceReference, code, message string, cause error) (domain.OperationResult, error) {
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
	if artifact.Digest().String() != "" {
		data.Digest = artifact.Digest()
	}
	result, err := domain.NewInspectOperationResult(data)
	if err != nil {
		return domain.OperationResult{}, errors.Join(cause, err)
	}
	return result, cause
}
