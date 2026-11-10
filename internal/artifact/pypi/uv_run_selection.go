package pypi

import (
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// BindUVRunSelection authenticates the joint wheel set once, then retains the
// real reachable graph for each declared root. It never invents a root edge.
func (l UVLock) BindUVRunSelection(request UVRunRequirements, inputs []UVWheelInput, pages []SimpleProject, target WheelTarget, limits WheelLimits) (domain.ExecutionDependencySelection, error) {
	if !request.Valid() || request.Primary().Reference() != l.reference {
		return domain.ExecutionDependencySelection{}, errors.New("UV run requirements differ from the locked primary")
	}
	roots := append([]UVRunRequirement{request.Primary()}, request.Supplemental()...)
	extras := map[string][]string{}
	for _, root := range roots {
		extras[root.Base().Project()] = root.Extras()
	}
	candidates, err := bindUVSelectedWheelsWithSurface(l.packages, l.python, request.Primary().Base().Project(), extras, inputs, pages, target, limits, validatePipxMetadataSurface)
	if err != nil {
		return domain.ExecutionDependencySelection{}, err
	}
	byProject := map[string]Candidate{}
	for _, candidate := range candidates {
		byProject[candidate.project] = candidate
	}
	covered := map[string]bool{}
	graphs := make([]domain.LockedDependencyGraph, 0, len(roots))
	for _, root := range roots {
		name := root.Base().Project()
		candidate, ok := byProject[name]
		if !ok || !root.Base().AcceptsVersion(candidate.version) {
			return domain.ExecutionDependencySelection{}, errors.New("UV run declared root is absent or has a conflicting version")
		}
		visited := map[string]bool{}
		pending := []string{name}
		var reachable []Candidate
		for len(pending) != 0 {
			project := pending[len(pending)-1]
			pending = pending[:len(pending)-1]
			if visited[project] {
				continue
			}
			current, ok := byProject[project]
			if !ok {
				return domain.ExecutionDependencySelection{}, errors.New("UV run dependency is absent from the complete source set")
			}
			visited[project], covered[project] = true, true
			current.primary = project == name
			reachable = append(reachable, current)
			pending = append(pending, current.dependencies...)
		}
		graph, err := BuildLockedGraph(root.Reference(), reachable)
		if err != nil {
			return domain.ExecutionDependencySelection{}, err
		}
		graphs = append(graphs, graph)
	}
	if len(covered) != len(candidates) {
		return domain.ExecutionDependencySelection{}, errors.New("UV run selected undeclared and unreachable packages")
	}
	return domain.NewExecutionDependencySelection(graphs[0], graphs[1:])
}
