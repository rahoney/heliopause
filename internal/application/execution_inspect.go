package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// ExecutionInspectService inspects an already selected complete graph. It
// reuses the installation/project entry lifecycle without an installation
// target, a resolver, or cached approval state.
type ExecutionInspectService struct {
	entry  *InstallInspectService
	policy DependencySetPolicy
}

// ExecutionInspection retains the operation and resolution that produced the
// current approved set. A failed inspection never contains an approved set.
// This is not a user-program execution result or an admission authorization.
type ExecutionInspection struct {
	operation  domain.OperationID
	resolution domain.DependencyResolution
	set        domain.VerifiedSet
}

func (i ExecutionInspection) Valid() bool                             { return i.operation.String() != "" && i.set.Valid() }
func (i ExecutionInspection) OperationID() domain.OperationID         { return i.operation }
func (i ExecutionInspection) Resolution() domain.DependencyResolution { return i.resolution }
func (i ExecutionInspection) Set() domain.VerifiedSet                 { return i.set }

// ExecutionPolicyFailure preserves complete trusted facts for reporting a
// refusal. It does not grant storage, promotion or execution authorization.
type ExecutionPolicyFailure struct {
	inspected domain.InspectedDependencySet
	decision  domain.PolicyDecision
}

func (e *ExecutionPolicyFailure) Error() string {
	return "execution Policy did not approve the complete dependency graph"
}
func (e *ExecutionPolicyFailure) Inspected() domain.InspectedDependencySet { return e.inspected }
func (e *ExecutionPolicyFailure) Decision() domain.PolicyDecision          { return e.decision }

func NewExecutionInspectService(artifact ports.Artifact, verifier ports.Verification, inspector ports.Inspection, evidence ports.Evidence, entryPolicy Policy, setPolicy DependencySetPolicy, newOperationID func() (domain.OperationID, error), newRunID func() (domain.RunID, error)) (*ExecutionInspectService, error) {
	if artifact == nil || verifier == nil || inspector == nil || evidence == nil || entryPolicy == nil || setPolicy == nil || newOperationID == nil || newRunID == nil {
		return nil, errors.New("execution inspection requires all entry ports, policies and ID generators")
	}
	entry := &InstallInspectService{artifact: artifact, verification: verifier, inspection: inspector, evidence: evidence, entryPolicy: entryPolicy, newOperationID: newOperationID, newRunID: newRunID}
	return &ExecutionInspectService{entry: entry, policy: setPolicy}, nil
}

func (s *ExecutionInspectService) InspectResolved(ctx context.Context, resolution domain.DependencyResolution) (ExecutionInspection, error) {
	graph := resolution.Graph()
	if ctx == nil || s == nil || s.entry == nil || s.policy == nil || resolution.LockfileDigest().String() == "" || graph.Primary().String() == "" || len(graph.Derivations()) != 0 {
		return ExecutionInspection{}, errors.New("execution inspection requires a complete non-derived resolution and configured service")
	}
	if err := ctx.Err(); err != nil {
		return ExecutionInspection{}, err
	}
	operation, err := s.entry.newOperationID()
	if err != nil {
		return ExecutionInspection{}, fmt.Errorf("generate execution inspection operation: %w", err)
	}
	partial := ExecutionInspection{operation: operation, resolution: resolution}
	if operation.String() == "" {
		return partial, errors.New("execution inspection operation is invalid")
	}
	if err := ctx.Err(); err != nil {
		return partial, err
	}
	var batch func(map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error)
	if inspector, ok := s.entry.inspection.(dependencyAwareInspection); ok && len(graph.Edges()) != 0 {
		batch = func(artifacts map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
			return inspector.InspectGraph(ctx, graph, artifacts)
		}
	}
	inspections, err := s.inspectExecutionEntries(ctx, operation, graph.Nodes(), batch)
	if err != nil {
		return partial, err
	}
	if err := ctx.Err(); err != nil {
		return partial, err
	}
	inspected, err := domain.NewInspectedDependencySet(graph, inspections)
	if err != nil {
		return partial, fmt.Errorf("construct complete execution inspection set: %w", err)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "policy"})
	decision, err := s.policy.EvaluateSet(inspected)
	if err != nil {
		return partial, fmt.Errorf("evaluate execution set Policy: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return partial, err
	}
	if decision.Decision() != domain.DecisionAllow {
		return partial, &ExecutionPolicyFailure{inspected, decision}
	}
	approved, err := domain.NewVerifiedSet(inspected, decision)
	if err != nil {
		return partial, fmt.Errorf("construct approved execution set: %w", err)
	}
	return ExecutionInspection{operation, resolution, approved}, nil
}

// The existing current acquire/verify/inspect/Evidence lifecycle is shared by
// ordinary connected graphs and explicit execution-root selections.
func (s *ExecutionInspectService) inspectExecutionEntries(ctx context.Context, operation domain.OperationID, nodes []domain.LockedDependency, batch func(map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error)) ([]domain.DependencyInspection, error) {
	if batch != nil {
		inspections, _, _, err := s.entry.inspectDependencyBatch(ctx, operation, nodes, batch)
		if err != nil {
			return nil, fmt.Errorf("inspect complete execution graph: %w", err)
		}
		return inspections, nil
	}
	var inspections []domain.DependencyInspection
	for _, dependency := range nodes {
		inspection, _, err := s.entry.inspectDependency(ctx, operation, dependency)
		if err != nil {
			return nil, fmt.Errorf("inspect execution dependency: %w", err)
		}
		inspections = append(inspections, inspection)
	}
	return inspections, nil
}
