package npm

import (
	"context"
	"errors"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// BunExecutionInspector reuses the complete Bun inspection owner with a
// frozen execution graph. It never creates an installation target or snapshot.
type BunExecutionInspector struct{ shared *BunProjectInspector }

func NewBunExecutionInspector(frozen artifactnpm.BunInspectionContext, static *Inspector, prepare BunInspectionPreparer, execute BunInstanceExecutor) (*BunExecutionInspector, error) {
	if !frozen.MatchesExecutionResolution(frozen.ExecutionResolution()) {
		return nil, errors.New("bun execution inspection requires its complete frozen resolution")
	}
	shared, err := NewBunProjectInspector(frozen, static, prepare, execute)
	if err != nil {
		return nil, err
	}
	return &BunExecutionInspector{shared}, nil
}

func (i *BunExecutionInspector) Inspect(ctx context.Context, subject domain.AcquiredArtifact) (domain.InspectionReport, error) {
	if i == nil || i.shared == nil || len(i.shared.frozen.ExecutionResolution().Graph().Nodes()) != 1 {
		return domain.InspectionReport{}, errors.New("bun execution inspection requires the complete graph")
	}
	graph := i.shared.frozen.ExecutionResolution().Graph()
	reports, err := i.InspectGraph(ctx, graph, map[domain.DependencyNodeID]domain.AcquiredArtifact{graph.Primary(): subject})
	return reports[graph.Primary()], err
}

func (i *BunExecutionInspector) InspectGraph(ctx context.Context, graph domain.LockedDependencyGraph, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	if i == nil || i.shared == nil {
		return nil, errors.New("bun execution inspection is unavailable")
	}
	unique, err := i.shared.inspectInput(ctx, func() (artifactnpm.BunInspectionInput, error) {
		return i.shared.frozen.BindExecutionAcquired(graph, acquired)
	})
	if unique == nil {
		return nil, err
	}
	reports := make(map[domain.DependencyNodeID]domain.InspectionReport, len(acquired))
	for _, node := range graph.Nodes() {
		key, keyErr := domain.ProjectDependencyNodeID(node.Artifact().Identity())
		report, present := unique[key]
		if keyErr != nil || !present {
			return reports, errors.Join(err, keyErr, errors.New("bun instance has no complete original inspection"))
		}
		reports[node.Node()] = report
	}
	return reports, err
}

func (i *BunExecutionInspector) PreparedLayoutFor(resolution domain.DependencyResolution, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (artifactnpm.BunInspectionLayout, error) {
	if i == nil || i.shared == nil || !i.shared.frozen.MatchesExecutionResolution(resolution) {
		return artifactnpm.BunInspectionLayout{}, errors.New("bun completed layout differs from current execution")
	}
	input, err := i.shared.frozen.BindExecutionAcquired(resolution.Graph(), acquired)
	if err != nil {
		return artifactnpm.BunInspectionLayout{}, err
	}
	return i.shared.preparedLayoutFor(input)
}
