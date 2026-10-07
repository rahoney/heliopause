package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
)

// ProjectDependencyNodeID provides stable opaque entry identity without
// assigning any project dependency the role of an arbitrary primary artifact.
func ProjectDependencyNodeID(identity ResolvedArtifactIdentity) (DependencyNodeID, error) {
	if identity.source.value == "" {
		return DependencyNodeID{}, errors.New("project dependency identity is required")
	}
	hash := sha256.Sum256([]byte(identity.source.value + "\x00" + identity.name + "\x00" + identity.version + "\x00" + identity.variant))
	return NewDependencyNodeID("p" + hex.EncodeToString(hash[:]))
}

// InspectedProjectSet retains exact coverage of one complete project snapshot.
// Entry inspection and Policy facts remain distinct from set-level approval.
type InspectedProjectSet struct {
	snapshot    ProjectDependencySnapshot
	inspections []DependencyInspection
}

func NewInspectedProjectSet(snapshot ProjectDependencySnapshot, inspections []DependencyInspection) (InspectedProjectSet, error) {
	if !snapshot.Valid() || len(inspections) != len(snapshot.dependencies) || len(inspections) > maxDependencyNodes {
		return InspectedProjectSet{}, errors.New("project inspection requires complete snapshot coverage")
	}
	locked := make(map[DependencyNodeID]ResolvedArtifact, len(snapshot.dependencies))
	for _, artifact := range snapshot.dependencies {
		node, err := ProjectDependencyNodeID(artifact.identity)
		if err != nil {
			return InspectedProjectSet{}, err
		}
		if _, duplicate := locked[node]; duplicate {
			return InspectedProjectSet{}, errors.New("project snapshot contains duplicate entry identity")
		}
		locked[node] = artifact
	}
	seen := map[DependencyNodeID]bool{}
	runs := map[RunID]bool{}
	owned := append([]DependencyInspection(nil), inspections...)
	for _, inspection := range owned {
		artifact, known := locked[inspection.node]
		if !known || seen[inspection.node] || runs[inspection.runID] || inspection.runID.value == "" || inspection.decision.policyID == "" || inspection.artifact.identity != artifact.identity || inspection.artifact.digest.value == "" || artifact.declaredIntegrity == "" || inspection.artifact.declaredIntegrity != artifact.declaredIntegrity || inspection.artifact.handle != "intake:"+inspection.runID.value+":"+artifact.identity.variant || len(inspection.evidence) == 0 {
			return InspectedProjectSet{}, errors.New("project inspection subject, integrity or coverage is invalid")
		}
		verification, inspectionCheck := false, false
		checks := map[CheckID]bool{}
		for _, check := range inspection.checks {
			if check.id.value == "" || checks[check.id] {
				return InspectedProjectSet{}, errors.New("project entry checks are invalid or duplicate")
			}
			checks[check.id] = true
			if check.required && check.kind == CheckVerification {
				verification = true
			}
			if check.required && check.kind == CheckInspection {
				inspectionCheck = true
			}
		}
		if !verification || !inspectionCheck {
			return InspectedProjectSet{}, errors.New("project entry requires verification and inspection checks")
		}
		seen[inspection.node] = true
		runs[inspection.runID] = true
	}
	sort.Slice(owned, func(i, j int) bool { return owned[i].node.value < owned[j].node.value })
	return InspectedProjectSet{snapshot: snapshot, inspections: owned}, nil
}

func (s InspectedProjectSet) Valid() bool {
	return s.snapshot.Valid() && len(s.inspections) == len(s.snapshot.dependencies)
}
func (s InspectedProjectSet) Snapshot() ProjectDependencySnapshot { return s.snapshot }
func (s InspectedProjectSet) Inspections() []DependencyInspection {
	return append([]DependencyInspection(nil), s.inspections...)
}

// ProjectVerifiedSet is eligible for project-scoped cache staging only after
// every entry and the complete snapshot have received independent ALLOWs.
type ProjectVerifiedSet struct {
	inspected InspectedProjectSet
	decision  PolicyDecision
}

func NewProjectVerifiedSet(inspected InspectedProjectSet, decision PolicyDecision) (ProjectVerifiedSet, error) {
	if !inspected.Valid() || decision.decision != DecisionAllow || decision.policyID == "" {
		return ProjectVerifiedSet{}, errors.New("project verified set requires a complete inspected snapshot and ALLOW")
	}
	for _, inspection := range inspected.inspections {
		if inspection.decision.decision != DecisionAllow {
			return ProjectVerifiedSet{}, errors.New("project verified set contains an unapproved entry")
		}
		for _, check := range inspection.checks {
			if check.required && (check.capability != CapabilitySupported || check.status != ExecutionCompleted) {
				return ProjectVerifiedSet{}, errors.New("project verified set contains an incomplete required check")
			}
		}
	}
	return ProjectVerifiedSet{inspected: inspected, decision: decision}, nil
}

func (s ProjectVerifiedSet) Valid() bool {
	return s.inspected.Valid() && s.decision.decision == DecisionAllow && s.decision.policyID != ""
}
func (s ProjectVerifiedSet) Inspected() InspectedProjectSet { return s.inspected }
func (s ProjectVerifiedSet) Decision() PolicyDecision       { return s.decision }
