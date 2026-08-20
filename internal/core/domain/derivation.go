package domain

import "errors"

// DerivedDependency is a controller-produced Artifact which is linked to a
// previously locked source node. It is not a replacement for that source.
type DerivedDependency struct {
	source   DependencyNodeID
	node     LockedDependency
	artifact AcquiredArtifact
}

func NewDerivedDependency(source DependencyNodeID, node LockedDependency, artifact AcquiredArtifact) (DerivedDependency, error) {
	if source.value == "" || node.node.value == "" || node.artifact.identity.variant != "derived-wheel" || artifact.identity != node.artifact.identity || artifact.declaredIntegrity != node.artifact.declaredIntegrity {
		return DerivedDependency{}, errors.New("derived dependency binding is invalid")
	}
	return DerivedDependency{source: source, node: node, artifact: artifact}, nil
}
func (d DerivedDependency) Source() DependencyNodeID   { return d.source }
func (d DerivedDependency) Node() LockedDependency     { return d.node }
func (d DerivedDependency) Artifact() AcquiredArtifact { return d.artifact }

// ExtendLockedDependencyGraph appends controller-derived nodes and source-to-
// derived edges without changing resolver-owned nodes or edges.
func ExtendLockedDependencyGraph(graph LockedDependencyGraph, derived []DerivedDependency) (LockedDependencyGraph, error) {
	nodes, edges := graph.Nodes(), graph.Edges()
	known := map[DependencyNodeID]bool{}
	for _, node := range nodes {
		known[node.Node()] = true
	}
	for _, item := range derived {
		if !known[item.source] || known[item.node.Node()] {
			return LockedDependencyGraph{}, errors.New("derived dependency graph binding is invalid")
		}
		nodes = append(nodes, item.node)
		edge, err := NewDependencyEdge(item.source, item.node.Node())
		if err != nil {
			return LockedDependencyGraph{}, err
		}
		edges = append(edges, edge)
		known[item.node.Node()] = true
	}
	return NewLockedDependencyGraph(nodes, edges)
}
