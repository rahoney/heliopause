package domain

import (
	"errors"
	"slices"
	"sort"
)

// ExecutionDependencySelection joins one console provider and explicitly
// requested supplemental roots. Each input retains its real connected graph;
// the union is selection DATA and grants no inspection or execution authority.
// LockedDependencyGraph keeps its existing single-primary connectivity rule.
type ExecutionDependencySelection struct {
	roots []LockedDependencyGraph
	nodes []LockedDependency
	edges []DependencyEdge
}

func NewExecutionDependencySelection(primary LockedDependencyGraph, supplemental []LockedDependencyGraph) (ExecutionDependencySelection, error) {
	if len(supplemental) == 0 || len(supplemental) >= MaximumExecutionArguments {
		return ExecutionDependencySelection{}, errors.New("execution selection requires bounded supplemental roots")
	}
	roots := append([]LockedDependencyGraph{primary}, supplemental...)
	seenRoots := map[DependencyNodeID]bool{}
	nodes := map[DependencyNodeID]LockedDependency{}
	edges := map[DependencyEdge]bool{}
	outgoing := map[DependencyNodeID][]DependencyEdge{}
	for _, graph := range roots {
		// Nonzero opaque graphs already passed their existing constructor.
		// Derived artifacts remain outside the current execution contract.
		if graph.primary.value == "" || len(graph.nodes) == 0 || len(graph.derivations) != 0 || seenRoots[graph.primary] {
			return ExecutionDependencySelection{}, errors.New("execution selection roots are absent, repeated or derived")
		}
		seenRoots[graph.primary] = true
		currentEdges := map[DependencyNodeID][]DependencyEdge{}
		for _, edge := range graph.edges {
			currentEdges[edge.from] = append(currentEdges[edge.from], edge)
			edges[edge] = true
		}
		for _, node := range graph.nodes {
			// Roles describe this execution request, not an invented edge.
			node.role = DependencyTransitive
			if node.node == primary.primary {
				node.role = DependencyPrimary
			}
			if previous, exists := nodes[node.node]; exists && previous != node {
				return ExecutionDependencySelection{}, errors.New("shared execution node has conflicting source or record facts")
			}
			if previous, exists := outgoing[node.node]; exists && !slices.Equal(previous, currentEdges[node.node]) {
				return ExecutionDependencySelection{}, errors.New("shared execution node has conflicting dependency edges")
			}
			nodes[node.node] = node
			outgoing[node.node] = currentEdges[node.node]
		}
		if len(nodes) > maxDependencyNodes || len(edges) > maxDependencyEdges {
			return ExecutionDependencySelection{}, errors.New("execution selection exceeds existing graph bounds")
		}
	}
	// Preserve the selected console provider first; canonicalize the other roots.
	sort.Slice(roots[1:], func(i, j int) bool { return roots[i+1].primary.value < roots[j+1].primary.value })
	result := ExecutionDependencySelection{roots: roots}
	for _, node := range nodes {
		result.nodes = append(result.nodes, node)
	}
	for edge := range edges {
		result.edges = append(result.edges, edge)
	}
	sort.Slice(result.nodes, func(i, j int) bool { return result.nodes[i].node.value < result.nodes[j].node.value })
	sort.Slice(result.edges, func(i, j int) bool {
		if result.edges[i].from == result.edges[j].from {
			return result.edges[i].to.value < result.edges[j].to.value
		}
		return result.edges[i].from.value < result.edges[j].from.value
	})
	return result, nil
}

func (s ExecutionDependencySelection) Valid() bool {
	return len(s.roots) > 1 && s.roots[0].primary.value != "" && len(s.nodes) != 0
}
func (s ExecutionDependencySelection) Primary() DependencyNodeID {
	if !s.Valid() {
		return DependencyNodeID{}
	}
	return s.roots[0].primary
}
func (s ExecutionDependencySelection) RootGraphs() []LockedDependencyGraph {
	return slices.Clone(s.roots)
}
func (s ExecutionDependencySelection) Nodes() []LockedDependency { return slices.Clone(s.nodes) }
func (s ExecutionDependencySelection) Edges() []DependencyEdge   { return slices.Clone(s.edges) }

// ExecutionSelectionResolution binds all explicit roots and their source
// graph to the actual resolver runtime and complete measured input digest.
type ExecutionSelectionResolution struct {
	selection ExecutionDependencySelection
	runtime   string
	digest    ContentDigest
}

func NewExecutionSelectionResolution(selection ExecutionDependencySelection, runtime string, digest ContentDigest) (ExecutionSelectionResolution, error) {
	if !selection.Valid() || digest.value == "" {
		return ExecutionSelectionResolution{}, errors.New("execution selection resolution requires the complete selection and digest")
	}
	if err := validateBoundedText(runtime, maxInstallTargetLength, "execution selection resolver runtime"); err != nil {
		return ExecutionSelectionResolution{}, err
	}
	return ExecutionSelectionResolution{selection, runtime, digest}, nil
}

func (r ExecutionSelectionResolution) Selection() ExecutionDependencySelection { return r.selection }
func (r ExecutionSelectionResolution) RuntimeIdentity() string                 { return r.runtime }
func (r ExecutionSelectionResolution) LockfileDigest() ContentDigest           { return r.digest }
func (r ExecutionSelectionResolution) Valid() bool {
	return r.selection.Valid() && r.runtime != "" && r.digest.value != ""
}
