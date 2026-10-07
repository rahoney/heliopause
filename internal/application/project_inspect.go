package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// ProjectSetPolicy evaluates complete project coverage without constructing a
// primary-artifact graph for project download/build operations.
type ProjectSetPolicy interface {
	EvaluateProjectSet(domain.InspectedProjectSet) (domain.PolicyDecision, error)
}

// ProjectInspectService reuses the existing per-entry acquire/verify/inspect/
// Evidence/Policy/run lifecycle for each exact project snapshot dependency.
type ProjectInspectService struct {
	entry  *InstallInspectService
	policy ProjectSetPolicy
}

// ProjectPolicyFailure preserves the completed, trusted inspection facts and
// decision for reporting. It carries no cache/promotion authorization.
type ProjectPolicyFailure struct {
	inspected domain.InspectedProjectSet
	decision  domain.PolicyDecision
}

func (e *ProjectPolicyFailure) Error() string {
	return "project Policy did not approve the complete snapshot"
}
func (e *ProjectPolicyFailure) Inspected() domain.InspectedProjectSet { return e.inspected }
func (e *ProjectPolicyFailure) Decision() domain.PolicyDecision       { return e.decision }

func NewProjectInspectService(artifact ports.Artifact, verifier ports.Verification, inspector ports.Inspection, evidence ports.Evidence, entryPolicy Policy, setPolicy ProjectSetPolicy, newOperationID func() (domain.OperationID, error), newRunID func() (domain.RunID, error)) (*ProjectInspectService, error) {
	if artifact == nil || verifier == nil || inspector == nil || evidence == nil || entryPolicy == nil || setPolicy == nil || newOperationID == nil || newRunID == nil {
		return nil, errors.New("project inspection requires all entry ports, policies and ID generators")
	}
	// Only the already-resolved per-entry operation is exposed by this service.
	// No resolver, derivation or primary-graph workflow is invoked here.
	entry := &InstallInspectService{artifact: artifact, verification: verifier, inspection: inspector, evidence: evidence, entryPolicy: entryPolicy, newOperationID: newOperationID, newRunID: newRunID}
	return &ProjectInspectService{entry: entry, policy: setPolicy}, nil
}

func (s *ProjectInspectService) InspectProject(ctx context.Context, snapshot domain.ProjectDependencySnapshot) (domain.ProjectVerifiedSet, error) {
	if ctx == nil || s == nil || s.entry == nil || s.policy == nil || !snapshot.Valid() {
		return domain.ProjectVerifiedSet{}, errors.New("valid project snapshot inspection is required")
	}
	if err := ctx.Err(); err != nil {
		return domain.ProjectVerifiedSet{}, err
	}
	operation, err := s.entry.newOperationID()
	if err != nil {
		return domain.ProjectVerifiedSet{}, fmt.Errorf("generate project inspection operation: %w", err)
	}
	inspections := make([]domain.DependencyInspection, 0, len(snapshot.Dependencies()))
	for _, artifact := range snapshot.Dependencies() {
		node, err := domain.ProjectDependencyNodeID(artifact.Identity())
		if err != nil {
			return domain.ProjectVerifiedSet{}, err
		}
		dependency, err := domain.NewLockedDependency(node, domain.DependencyTransitive, artifact)
		if err != nil {
			return domain.ProjectVerifiedSet{}, err
		}
		inspection, _, err := s.entry.inspectDependency(ctx, operation, dependency)
		if err != nil {
			return domain.ProjectVerifiedSet{}, fmt.Errorf("inspect exact project dependency: %w", err)
		}
		inspections = append(inspections, inspection)
	}
	set, err := domain.NewInspectedProjectSet(snapshot, inspections)
	if err != nil {
		return domain.ProjectVerifiedSet{}, err
	}
	decision, err := s.policy.EvaluateProjectSet(set)
	if err != nil {
		return domain.ProjectVerifiedSet{}, fmt.Errorf("evaluate project set Policy: %w", err)
	}
	if decision.Decision() != domain.DecisionAllow {
		return domain.ProjectVerifiedSet{}, &ProjectPolicyFailure{inspected: set, decision: decision}
	}
	return domain.NewProjectVerifiedSet(set, decision)
}
