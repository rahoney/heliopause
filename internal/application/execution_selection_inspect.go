package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

type executionSelectionInspection interface {
	InspectExecutionSelection(context.Context, domain.ExecutionDependencySelection, map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error)
}
type executionSelectionPolicy interface {
	EvaluateExecutionSelection(domain.InspectedExecutionSelection) (domain.PolicyDecision, error)
}

// ExecutionSelectionInspection contains one fresh operation's whole approved
// root selection. It cannot be substituted for a single-graph approval.
type ExecutionSelectionInspection struct {
	operation  domain.OperationID
	resolution domain.ExecutionSelectionResolution
	set        domain.VerifiedExecutionSelection
}

func (i ExecutionSelectionInspection) Valid() bool {
	return i.operation.String() != "" && i.set.Valid()
}
func (i ExecutionSelectionInspection) OperationID() domain.OperationID { return i.operation }
func (i ExecutionSelectionInspection) Resolution() domain.ExecutionSelectionResolution {
	return i.resolution
}
func (i ExecutionSelectionInspection) Set() domain.VerifiedExecutionSelection { return i.set }

type ExecutionSelectionPolicyFailure struct {
	inspected domain.InspectedExecutionSelection
	decision  domain.PolicyDecision
}

func (e *ExecutionSelectionPolicyFailure) Error() string {
	return "execution Policy did not approve the complete explicit-root selection"
}
func (e *ExecutionSelectionPolicyFailure) Inspected() domain.InspectedExecutionSelection {
	return e.inspected
}
func (e *ExecutionSelectionPolicyFailure) Decision() domain.PolicyDecision { return e.decision }

func (s *ExecutionInspectService) InspectSelection(ctx context.Context, resolution domain.ExecutionSelectionResolution) (ExecutionSelectionInspection, error) {
	selection := resolution.Selection()
	if ctx == nil || s == nil || s.entry == nil || s.policy == nil || !selection.Valid() || resolution.LockfileDigest().String() == "" {
		return ExecutionSelectionInspection{}, errors.New("execution inspection requires a complete explicit-root resolution and configured service")
	}
	if err := ctx.Err(); err != nil {
		return ExecutionSelectionInspection{}, errors.Join(err, context.Cause(ctx))
	}
	inspector, ok := s.entry.inspection.(executionSelectionInspection)
	if !ok {
		return ExecutionSelectionInspection{}, errors.New("execution selection requires whole-environment inspection")
	}
	policy, ok := s.policy.(executionSelectionPolicy)
	if !ok {
		return ExecutionSelectionInspection{}, errors.New("execution selection requires its whole-set Policy")
	}
	operation, err := s.entry.newOperationID()
	if err != nil {
		return ExecutionSelectionInspection{}, fmt.Errorf("generate execution inspection operation: %w", err)
	}
	partial := ExecutionSelectionInspection{operation: operation, resolution: resolution}
	if operation.String() == "" {
		return partial, errors.New("execution inspection operation is invalid")
	}
	if err := ctx.Err(); err != nil {
		return partial, errors.Join(err, context.Cause(ctx))
	}
	inspections, err := s.inspectExecutionEntries(ctx, operation, selection.Nodes(), func(artifacts map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
		return inspector.InspectExecutionSelection(ctx, selection, artifacts)
	})
	if err != nil {
		return partial, err
	}
	if err := ctx.Err(); err != nil {
		return partial, errors.Join(err, context.Cause(ctx))
	}
	inspected, err := domain.NewInspectedExecutionSelection(selection, inspections)
	if err != nil {
		return partial, fmt.Errorf("construct complete execution selection inspection: %w", err)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "policy"})
	decision, err := policy.EvaluateExecutionSelection(inspected)
	if err != nil {
		return partial, fmt.Errorf("evaluate execution selection Policy: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return partial, errors.Join(err, context.Cause(ctx))
	}
	if decision.Decision() != domain.DecisionAllow {
		return partial, &ExecutionSelectionPolicyFailure{inspected, decision}
	}
	approved, err := domain.NewVerifiedExecutionSelection(inspected, decision)
	if err != nil {
		return partial, fmt.Errorf("construct approved execution selection: %w", err)
	}
	return ExecutionSelectionInspection{operation, resolution, approved}, nil
}
