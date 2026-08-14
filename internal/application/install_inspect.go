package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// DependencySetPolicy is the deterministic, graph-level decision boundary for
// an M4 Install. It receives only a complete normalized inspected set.
type DependencySetPolicy interface {
	EvaluateSet(domain.InspectedDependencySet) (domain.PolicyDecision, error)
}

// InstallInspectService resolves the locked dependency graph and independently
// executes the existing acquire, verify, inspect, evidence, and entry-Policy
// sequence for every graph node before evaluating one set-level Policy.
type InstallInspectService struct {
	resolver       ports.DependencyResolver
	artifact       ports.Artifact
	verification   ports.Verification
	inspection     ports.Inspection
	evidence       ports.Evidence
	entryPolicy    Policy
	setPolicy      DependencySetPolicy
	newOperationID func() (domain.OperationID, error)
	newRunID       func() (domain.RunID, error)
}

// NewInstallInspectService constructs the M4 recursive inspection workflow
// from explicit boundary dependencies.
func NewInstallInspectService(resolver ports.DependencyResolver, artifact ports.Artifact, verification ports.Verification, inspection ports.Inspection, evidence ports.Evidence, entryPolicy Policy, setPolicy DependencySetPolicy, newOperationID func() (domain.OperationID, error), newRunID func() (domain.RunID, error)) (*InstallInspectService, error) {
	if resolver == nil || artifact == nil || verification == nil || inspection == nil || evidence == nil || entryPolicy == nil || setPolicy == nil || newOperationID == nil || newRunID == nil {
		return nil, errors.New("install inspection service requires all ports, policies, and ID generators")
	}
	return &InstallInspectService{
		resolver: resolver, artifact: artifact, verification: verification, inspection: inspection, evidence: evidence,
		entryPolicy: entryPolicy, setPolicy: setPolicy, newOperationID: newOperationID, newRunID: newRunID,
	}, nil
}

// Inspect resolves one exact graph and returns only after every locked entry
// has completed its own inspection. Any resolver or entry failure stops the
// workflow before a set-level decision can be produced.
func (s *InstallInspectService) Inspect(ctx context.Context, request InstallRequest) (InspectedInstall, error) {
	if ctx == nil {
		return InspectedInstall{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return InspectedInstall{}, err
	}
	if request.reference.Source().String() == "" || request.context.Target().String() == "" || !request.context.RequiresNewTarget() {
		return InspectedInstall{}, errors.New("validated Install request is required")
	}
	operationID, err := s.newOperationID()
	if err != nil {
		return InspectedInstall{}, fmt.Errorf("generate operation ID: %w", err)
	}
	resolution, err := s.resolver.ResolveDependencies(ctx, request.reference, request.context)
	if err != nil {
		return newPartialInspectedInstall(operationID, request, domain.DependencyResolution{}), fmt.Errorf("resolve locked dependency graph: %w", err)
	}
	partial := newPartialInspectedInstall(operationID, request, resolution)
	graph := resolution.Graph()
	inspections := make([]domain.DependencyInspection, 0, len(graph.Nodes()))
	for _, dependency := range graph.Nodes() {
		inspection, inspectionErr := s.inspectDependency(ctx, operationID, dependency)
		if inspectionErr != nil {
			return partial, inspectionErr
		}
		inspections = append(inspections, inspection)
	}
	set, err := domain.NewInspectedDependencySet(graph, inspections)
	if err != nil {
		return partial, fmt.Errorf("construct inspected dependency set: %w", err)
	}
	decision, err := s.setPolicy.EvaluateSet(set)
	if err != nil {
		return partial, fmt.Errorf("evaluate dependency set policy: %w", err)
	}
	return newInspectedInstall(operationID, request, resolution, set, decision), nil
}

func (s *InstallInspectService) inspectDependency(ctx context.Context, operationID domain.OperationID, dependency domain.LockedDependency) (domain.DependencyInspection, error) {
	runID, err := s.newRunID()
	if err != nil {
		return domain.DependencyInspection{}, fmt.Errorf("generate Run ID for dependency %s: %w", dependency.Node().String(), err)
	}
	reference, err := lockedReference(dependency.Artifact())
	if err != nil {
		return domain.DependencyInspection{}, fmt.Errorf("construct dependency reference: %w", err)
	}
	run, err := domain.NewInspectionRun(runID, operationID, reference, dependency.Artifact().Identity())
	if err != nil {
		return domain.DependencyInspection{}, fmt.Errorf("create dependency Inspection Run: %w", err)
	}
	if err := run.Activate(); err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "RUN_ACTIVATION_FAILED", err)
	}
	artifact, err := s.artifact.Acquire(ctx, runID, dependency.Artifact())
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "ARTIFACT_ACQUIRE_FAILED", err)
	}
	if err := run.BindAcquiredArtifact(artifact); err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "ARTIFACT_BINDING_FAILED", err)
	}
	verification, err := s.verification.Verify(ctx, artifact)
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "VERIFICATION_PROVIDER_FAILED", err)
	}
	checks := []domain.CheckExecution{verification.Execution()}
	inspection, err := s.inspection.Inspect(ctx, artifact)
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "INSPECTION_PROVIDER_FAILED", err)
	}
	checks = append(checks, inspection.Executions()...)
	evidence := append(verification.Evidence(), inspection.Evidence()...)
	references, err := s.evidence.Record(ctx, runID, evidence)
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "EVIDENCE_RECORD_FAILED", err)
	}
	input, err := domain.NewPolicyInput(runID, artifact, verification, inspection, references)
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "POLICY_INPUT_INVALID", err)
	}
	decision, err := s.entryPolicy.Evaluate(input)
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "POLICY_EVALUATION_FAILED", err)
	}
	if err := run.FinalizeCompleted(decision); err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "RUN_FINALIZATION_FAILED", err)
	}
	record, err := domain.NewDependencyInspection(dependency.Node(), runID, artifact, checks, references, decision)
	if err != nil {
		return domain.DependencyInspection{}, fmt.Errorf("construct dependency inspection record: %w", err)
	}
	return record, nil
}

func lockedReference(artifact domain.ResolvedArtifact) (domain.ArtifactReference, error) {
	identity := artifact.Identity()
	return domain.NewArtifactReference(identity.Source(), identity.Name()+"@"+identity.Version())
}

func failDependencyRun(run *domain.InspectionRun, code string, cause error) error {
	if err := run.FinalizeFailed(code); err != nil {
		return errors.Join(cause, fmt.Errorf("finalize failed dependency Inspection Run: %w", err))
	}
	return cause
}
