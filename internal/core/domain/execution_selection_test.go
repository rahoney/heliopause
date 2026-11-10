package domain_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestExecutionSelectionPreservesAggregateGraphBounds(t *testing.T) {
	for _, dense := range []bool{false, true} {
		name, size := "nodes", 2049
		if dense {
			name, size = "edges", 130
		}
		t.Run(name, func(t *testing.T) {
			// Each input satisfies the existing graph limit. Their union exceeds
			// the existing 4096-node or 16384-edge limit respectively.
			source := mustSource(t, "npm")
			var roots []domain.LockedDependencyGraph
			for group := 0; group < 2; group++ {
				var nodes []domain.LockedDependency
				for i := 0; i < size; i++ {
					role := domain.DependencyTransitive
					if i == 0 {
						role = domain.DependencyPrimary
					}
					label := fmt.Sprintf("r%d-n%d", group, i)
					nodes = append(nodes, lockedDependency(t, label, role, source, label, "1.0.0"))
				}
				var pairs [][2]domain.DependencyNodeID
				for from := range nodes {
					if from != 0 && !dense {
						break
					}
					for to := from + 1; to < size; to++ {
						pairs = append(pairs, [2]domain.DependencyNodeID{nodes[from].Node(), nodes[to].Node()})
					}
				}
				roots = append(roots, selectionTestGraph(t, nodes, pairs...))
			}
			selection, err := domain.NewExecutionDependencySelection(roots[0], roots[1:])
			if err == nil || selection.Valid() {
				t.Fatal("separate roots bypassed the existing aggregate graph budget")
			}
		})
	}
}

func selectionTestGraph(t *testing.T, nodes []domain.LockedDependency, pairs ...[2]domain.DependencyNodeID) domain.LockedDependencyGraph {
	t.Helper()
	var edges []domain.DependencyEdge
	for _, pair := range pairs {
		edge, err := domain.NewDependencyEdge(pair[0], pair[1])
		if err != nil {
			t.Fatal(err)
		}
		edges = append(edges, edge)
	}
	graph, err := domain.NewLockedDependencyGraph(nodes, edges)
	if err != nil {
		t.Fatal(err)
	}
	return graph
}

func TestExecutionSelectionRetainsRealRootsAndSharedDependencyOnce(t *testing.T) {
	source := mustSource(t, "npm")
	main := lockedDependency(t, "main", domain.DependencyPrimary, source, "main", "1.0.0")
	other := lockedDependency(t, "other", domain.DependencyPrimary, source, "other", "1.0.0")
	shared := lockedDependency(t, "shared", domain.DependencyTransitive, source, "shared", "1.0.0")
	primary := selectionTestGraph(t, []domain.LockedDependency{main, shared}, [2]domain.DependencyNodeID{main.Node(), shared.Node()})
	another := lockedDependency(t, "another", domain.DependencyPrimary, source, "another", "1.0.0")
	supplemental := []domain.LockedDependencyGraph{
		selectionTestGraph(t, []domain.LockedDependency{other, shared}, [2]domain.DependencyNodeID{other.Node(), shared.Node()}),
		selectionTestGraph(t, []domain.LockedDependency{another, shared}, [2]domain.DependencyNodeID{another.Node(), shared.Node()}),
	}
	selection, err := domain.NewExecutionDependencySelection(primary, supplemental)
	if err != nil || !selection.Valid() || selection.Primary() != main.Node() || len(selection.Nodes()) != 4 || len(selection.Edges()) != 3 || len(selection.RootGraphs()) != 3 || selection.RootGraphs()[1].Primary() != another.Node() {
		t.Fatal("whole real selection missing or shared entry duplicated", err)
	}
	if _, err := domain.NewLockedDependencyGraph(selection.Nodes(), selection.Edges()); err == nil {
		t.Fatal("old connected graph contract was weakened")
	}
	if primary.Primary() != main.Node() || !slices.Equal(primary.Nodes(), selection.RootGraphs()[0].Nodes()) || !slices.Equal(primary.Edges(), selection.RootGraphs()[0].Edges()) {
		t.Fatal("existing primary graph changed")
	}
	supplemental[0] = domain.LockedDependencyGraph{}
	nodes, edges, roots := selection.Nodes(), selection.Edges(), selection.RootGraphs()
	nodes[0], edges[0], roots[0] = domain.LockedDependency{}, domain.DependencyEdge{}, domain.LockedDependencyGraph{}
	if !selection.Valid() || selection.Nodes()[0].Node().String() == "" || selection.Edges()[0].From().String() == "" || selection.RootGraphs()[0].Primary().String() == "" {
		t.Fatal("caller mutated the frozen selection")
	}
}

func TestExecutionSelectionAllowsDeclaredRootDependingOnConsoleProvider(t *testing.T) {
	source := mustSource(t, "npm")
	main := lockedDependency(t, "main", domain.DependencyPrimary, source, "main", "1.0.0")
	mainAsDependency := lockedDependency(t, "main", domain.DependencyTransitive, source, "main", "1.0.0")
	other := lockedDependency(t, "other", domain.DependencyPrimary, source, "other", "1.0.0")
	primary := selectionTestGraph(t, []domain.LockedDependency{main})
	second := selectionTestGraph(t, []domain.LockedDependency{other, mainAsDependency}, [2]domain.DependencyNodeID{other.Node(), main.Node()})
	selection, err := domain.NewExecutionDependencySelection(primary, []domain.LockedDependencyGraph{second})
	if err != nil || len(selection.Nodes()) != 2 || len(selection.Edges()) != 1 || selection.Primary() != main.Node() {
		t.Fatal("real supplemental dependency was replaced by an invented edge", err)
	}
	if _, err := domain.NewLockedDependencyGraph(selection.Nodes(), selection.Edges()); err == nil {
		t.Fatal("old primary-inbound prohibition was changed")
	}
}

func TestExecutionSelectionRejectsPartialConflictingOrRepeatedRoots(t *testing.T) {
	source := mustSource(t, "npm")
	main := lockedDependency(t, "main", domain.DependencyPrimary, source, "main", "1.0.0")
	other := lockedDependency(t, "other", domain.DependencyPrimary, source, "other", "1.0.0")
	shared := lockedDependency(t, "shared", domain.DependencyTransitive, source, "shared", "1.0.0")
	leaf := lockedDependency(t, "leaf", domain.DependencyTransitive, source, "leaf", "1.0.0")
	primary := selectionTestGraph(t, []domain.LockedDependency{main, shared, leaf}, [2]domain.DependencyNodeID{main.Node(), shared.Node()}, [2]domain.DependencyNodeID{shared.Node(), leaf.Node()})
	second := selectionTestGraph(t, []domain.LockedDependency{other, shared, leaf}, [2]domain.DependencyNodeID{other.Node(), shared.Node()}, [2]domain.DependencyNodeID{shared.Node(), leaf.Node()})
	changed := lockedDependency(t, "shared", domain.DependencyTransitive, source, "shared", "2.0.0")
	changedGraph := selectionTestGraph(t, []domain.LockedDependency{other, changed, leaf}, [2]domain.DependencyNodeID{other.Node(), changed.Node()}, [2]domain.DependencyNodeID{changed.Node(), leaf.Node()})
	missingEdge := selectionTestGraph(t, []domain.LockedDependency{other, shared}, [2]domain.DependencyNodeID{other.Node(), shared.Node()})
	for _, row := range []struct {
		primary domain.LockedDependencyGraph
		other   []domain.LockedDependencyGraph
	}{
		{domain.LockedDependencyGraph{}, []domain.LockedDependencyGraph{second}},
		{primary, nil}, {primary, []domain.LockedDependencyGraph{{}}},
		{primary, []domain.LockedDependencyGraph{primary}},
		{primary, []domain.LockedDependencyGraph{second, second}},
		{primary, []domain.LockedDependencyGraph{changedGraph}},
		{primary, []domain.LockedDependencyGraph{missingEdge}},
		{primary, make([]domain.LockedDependencyGraph, domain.MaximumExecutionArguments)},
	} {
		selection, err := domain.NewExecutionDependencySelection(row.primary, row.other)
		if err == nil || selection.Valid() || len(selection.Nodes()) != 0 {
			t.Fatal("invalid selection published partial source facts")
		}
	}
}

func TestExecutionSelectionResolutionBindsCompleteSelection(t *testing.T) {
	source := mustSource(t, "npm")
	main := lockedDependency(t, "main", domain.DependencyPrimary, source, "main", "1.0.0")
	other := lockedDependency(t, "other", domain.DependencyPrimary, source, "other", "1.0.0")
	primary := selectionTestGraph(t, []domain.LockedDependency{main})
	second := selectionTestGraph(t, []domain.LockedDependency{other})
	selection, err := domain.NewExecutionDependencySelection(primary, []domain.LockedDependencyGraph{second})
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	resolution, err := domain.NewExecutionSelectionResolution(selection, "pinned-runtime", digest)
	if err != nil || resolution.LockfileDigest() != digest || resolution.RuntimeIdentity() != "pinned-runtime" || len(resolution.Selection().RootGraphs()) != 2 {
		t.Fatal("complete source/runtime binding lost", err)
	}
	if _, err := domain.NewExecutionSelectionResolution(selection, "", digest); err == nil {
		t.Fatal("empty resolver identity accepted")
	}
	if _, err := domain.NewExecutionSelectionResolution(selection, "pinned-runtime", domain.ContentDigest{}); err == nil {
		t.Fatal("unmeasured input accepted")
	}
	if _, err := domain.NewExecutionSelectionResolution(domain.ExecutionDependencySelection{}, "pinned-runtime", digest); err == nil {
		t.Fatal("zero selection accepted")
	}
}
