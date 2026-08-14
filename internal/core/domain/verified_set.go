package domain

import (
	"errors"
	"sort"
)

// DependencyInspection retains the completed, per-node inspection record that
// a set-level Policy needs. It deliberately contains only normalized Domain
// values and trusted Evidence references.
type DependencyInspection struct {
	node     DependencyNodeID
	runID    RunID
	artifact AcquiredArtifact
	checks   []CheckExecution
	evidence []EvidenceReference
	decision PolicyDecision
}

// NewDependencyInspection constructs one completed inspection record for an
// exact locked dependency.
func NewDependencyInspection(node DependencyNodeID, runID RunID, artifact AcquiredArtifact, checks []CheckExecution, evidence []EvidenceReference, decision PolicyDecision) (DependencyInspection, error) {
	if node.value == "" || runID.value == "" || artifact.identity.source.value == "" || decision.policyID == "" {
		return DependencyInspection{}, errors.New("dependency inspection requires node, completed run subject, and policy decision")
	}
	if len(checks) == 0 || len(evidence) == 0 {
		return DependencyInspection{}, errors.New("dependency inspection requires checks and recorded evidence")
	}
	for _, check := range checks {
		if check.id.value == "" {
			return DependencyInspection{}, errors.New("dependency inspection contains an invalid check")
		}
	}
	seenEvidence := make(map[EvidenceID]bool, len(evidence))
	for _, reference := range evidence {
		if reference.id.value == "" || seenEvidence[reference.id] {
			return DependencyInspection{}, errors.New("dependency inspection contains invalid evidence references")
		}
		seenEvidence[reference.id] = true
	}
	return DependencyInspection{
		node: node, runID: runID, artifact: artifact,
		checks:   append([]CheckExecution(nil), checks...),
		evidence: append([]EvidenceReference(nil), evidence...), decision: decision,
	}, nil
}

func (i DependencyInspection) Node() DependencyNodeID         { return i.node }
func (i DependencyInspection) RunID() RunID                   { return i.runID }
func (i DependencyInspection) Artifact() AcquiredArtifact     { return i.artifact }
func (i DependencyInspection) PolicyDecision() PolicyDecision { return i.decision }
func (i DependencyInspection) Checks() []CheckExecution {
	return append([]CheckExecution(nil), i.checks...)
}
func (i DependencyInspection) Evidence() []EvidenceReference {
	return append([]EvidenceReference(nil), i.evidence...)
}

// InspectedDependencySet is a complete graph-bound collection of independently
// inspected entries. A partial set cannot be constructed.
type InspectedDependencySet struct {
	graph       LockedDependencyGraph
	inspections []DependencyInspection
}

// NewInspectedDependencySet accepts exactly one inspection per locked graph
// node and binds every acquired subject to its locked identity and integrity.
func NewInspectedDependencySet(graph LockedDependencyGraph, inspections []DependencyInspection) (InspectedDependencySet, error) {
	nodes := graph.Nodes()
	if len(nodes) == 0 || len(inspections) != len(nodes) {
		return InspectedDependencySet{}, errors.New("inspected dependency set requires every locked graph node")
	}
	locked := make(map[DependencyNodeID]LockedDependency, len(nodes))
	for _, node := range nodes {
		locked[node.node] = node
	}
	seen := make(map[DependencyNodeID]bool, len(inspections))
	owned := make([]DependencyInspection, len(inspections))
	for index, inspection := range inspections {
		node, ok := locked[inspection.node]
		if !ok || seen[inspection.node] {
			return InspectedDependencySet{}, errors.New("inspected dependency set contains an unknown or duplicate node")
		}
		if inspection.artifact.identity != node.artifact.identity || inspection.artifact.declaredIntegrity != node.artifact.declaredIntegrity {
			return InspectedDependencySet{}, errors.New("dependency inspection subject does not match locked dependency")
		}
		seen[inspection.node] = true
		owned[index] = inspection
	}
	sort.Slice(owned, func(i, j int) bool { return owned[i].node.value < owned[j].node.value })
	return InspectedDependencySet{graph: graph, inspections: owned}, nil
}

func (s InspectedDependencySet) Graph() LockedDependencyGraph { return s.graph }
func (s InspectedDependencySet) Inspections() []DependencyInspection {
	return append([]DependencyInspection(nil), s.inspections...)
}

// Valid reports whether this set was constructed with complete graph coverage.
func (s InspectedDependencySet) Valid() bool {
	return len(s.graph.nodes) != 0 && len(s.inspections) == len(s.graph.nodes)
}
