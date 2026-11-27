package domain

import (
	"errors"
	"fmt"
	"slices"
)

// InspectedExecutionSelection retains exactly one fresh inspection per union
// node, including explicitly requested independent roots and shared entries.
// It does not approve a set or supply any install target.
type InspectedExecutionSelection struct {
	selection   ExecutionDependencySelection
	inspections []DependencyInspection
}

func NewInspectedExecutionSelection(selection ExecutionDependencySelection, inspections []DependencyInspection) (InspectedExecutionSelection, error) {
	if !selection.Valid() {
		return InspectedExecutionSelection{}, errors.New("execution inspection requires a complete root selection")
	}
	locked := make(map[DependencyNodeID]ResolvedArtifact, len(selection.nodes))
	for _, node := range selection.nodes {
		locked[node.node] = node.artifact
	}
	owned, err := validateCurrentInspectionCoverage(locked, inspections)
	if err != nil {
		return InspectedExecutionSelection{}, fmt.Errorf("execution selection %w", err)
	}
	return InspectedExecutionSelection{selection, owned}, nil
}

func (s InspectedExecutionSelection) Valid() bool {
	return s.selection.Valid() && len(s.inspections) == len(s.selection.nodes)
}
func (s InspectedExecutionSelection) Selection() ExecutionDependencySelection { return s.selection }
func (s InspectedExecutionSelection) Inspections() []DependencyInspection {
	return slices.Clone(s.inspections)
}

// VerifiedExecutionSelection binds the complete independently inspected union
// to one current whole-selection ALLOW. A primary-only approval is insufficient.
type VerifiedExecutionSelection struct {
	inspected InspectedExecutionSelection
	decision  PolicyDecision
}

func NewVerifiedExecutionSelection(inspected InspectedExecutionSelection, decision PolicyDecision) (VerifiedExecutionSelection, error) {
	if !inspected.Valid() || decision.decision != DecisionAllow || decision.policyID == "" {
		return VerifiedExecutionSelection{}, errors.New("verified execution selection requires complete current inspection and whole-set ALLOW")
	}
	for _, inspection := range inspected.inspections {
		if inspection.decision.decision != DecisionAllow {
			return VerifiedExecutionSelection{}, errors.New("verified execution selection contains an unapproved entry")
		}
	}
	return VerifiedExecutionSelection{inspected, decision}, nil
}

func (s VerifiedExecutionSelection) Valid() bool {
	return s.inspected.Valid() && s.decision.decision == DecisionAllow && s.decision.policyID != ""
}
func (s VerifiedExecutionSelection) Inspected() InspectedExecutionSelection { return s.inspected }
func (s VerifiedExecutionSelection) Decision() PolicyDecision               { return s.decision }
