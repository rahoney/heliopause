package pypi

import (
	"errors"
	"sort"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// ProjectRecords derives real file names and dependency edges from the already
// selected, rehashed wheels. It does not synthesize Simple pages or a primary.
func (l UVProjectLock) ProjectRecords(artifacts []domain.ResolvedArtifact, inputs []UVWheelInput, target WheelTarget, limits WheelLimits) ([]domain.LockedDependency, []domain.DependencyEdge, error) {
	wheels, err := l.SelectResolvedWheels(artifacts)
	if err != nil {
		return nil, nil, err
	}
	allowed := map[string]UVLockedWheel{}
	for _, wheel := range wheels {
		allowed[wheel.URL] = wheel
	}
	for _, input := range inputs {
		if _, ok := allowed[input.URL]; !ok {
			return nil, nil, errors.New("project record archive differs from approved native selection")
		}
	}
	candidates, err := uvArchiveCandidates(l.packages, l.python, "", inputs, target, limits)
	if err != nil {
		return nil, nil, err
	}
	if err := validateUVCandidateConstraints(candidates, l.python); err != nil {
		return nil, nil, err
	}
	nodes := make([]domain.LockedDependency, 0, len(artifacts))
	edges := []domain.DependencyEdge{}
	byProject := map[string]domain.DependencyNodeID{}
	for _, candidate := range candidates {
		var actual domain.ResolvedArtifact
		for _, artifact := range artifacts {
			if artifact.Identity().Name() == candidate.project {
				actual = artifact
				break
			}
		}
		if actual.Identity().Version() != candidate.version || actual.AcquisitionLocator() != candidate.url || actual.DeclaredIntegrity() != "sha256:"+candidate.sha256 {
			return nil, nil, errors.New("project record archive identity differs from approval")
		}
		id, err := domain.ProjectDependencyNodeID(actual.Identity())
		if err != nil {
			return nil, nil, err
		}
		node, err := domain.NewLockedDependencyWithRecordPath(id, domain.DependencyTransitive, actual, candidate.filename, false)
		if err != nil {
			return nil, nil, err
		}
		nodes = append(nodes, node)
		byProject[candidate.project] = id
	}
	for _, candidate := range candidates {
		for _, dependency := range candidate.dependencies {
			to, ok := byProject[dependency]
			if !ok {
				return nil, nil, errors.New("project record dependency is absent")
			}
			edge, err := domain.NewDependencyEdge(byProject[candidate.project], to)
			if err != nil {
				return nil, nil, err
			}
			edges = append(edges, edge)
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Node().String() < nodes[j].Node().String() })
	sort.Slice(edges, func(i, j int) bool {
		a, b := edges[i], edges[j]
		return a.From().String() < b.From().String() || a.From() == b.From() && a.To().String() < b.To().String()
	})
	return nodes, edges, nil
}
