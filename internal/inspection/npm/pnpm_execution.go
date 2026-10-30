package npm

import (
	"context"
	"errors"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// PNPMExecutionInspector reuses the complete PNPM inspection owner with a
// frozen execution graph. It never creates an installation target or snapshot.
type PNPMExecutionInspector struct{ shared *PNPMProjectInspector }

func NewPNPMExecutionInspector(frozen artifactnpm.PNPMInspectionContext, static *Inspector, prepare PNPMInspectionPreparer, execute PNPMInstanceExecutor) (*PNPMExecutionInspector, error) {
	if !frozen.MatchesExecutionResolution(frozen.ExecutionResolution()) {
		return nil, errors.New("pnpm execution inspection requires its complete frozen resolution")
	}
	shared, err := NewPNPMProjectInspector(frozen, static, prepare, execute)
	if err != nil {
		return nil, err
	}
	return &PNPMExecutionInspector{shared}, nil
}

func (i *PNPMExecutionInspector) Inspect(ctx context.Context, subject domain.AcquiredArtifact) (domain.InspectionReport, error) {
	if i == nil || i.shared == nil || len(i.shared.frozen.ExecutionResolution().Graph().Nodes()) != 1 {
		return domain.InspectionReport{}, errors.New("pnpm execution inspection requires the complete graph")
	}
	graph := i.shared.frozen.ExecutionResolution().Graph()
	reports, err := i.InspectGraph(ctx, graph, map[domain.DependencyNodeID]domain.AcquiredArtifact{graph.Primary(): subject})
	return reports[graph.Primary()], err
}

func (i *PNPMExecutionInspector) InspectGraph(ctx context.Context, graph domain.LockedDependencyGraph, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	if i == nil || i.shared == nil {
		return nil, errors.New("pnpm execution inspection is unavailable")
	}
	unique, err := i.shared.inspectInput(ctx, func() (artifactnpm.PNPMInspectionInput, error) {
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
			return reports, errors.Join(err, keyErr, errors.New("pnpm instance has no complete original inspection"))
		}
		reports[node.Node()] = report
	}
	return reports, err
}

func (i *PNPMExecutionInspector) PreparedLayoutFor(resolution domain.DependencyResolution, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (artifactnpm.PNPMInspectionLayout, error) {
	if i == nil || i.shared == nil || !i.shared.frozen.MatchesExecutionResolution(resolution) {
		return artifactnpm.PNPMInspectionLayout{}, errors.New("pnpm completed layout differs from current execution")
	}
	input, err := i.shared.frozen.BindExecutionAcquired(resolution.Graph(), acquired)
	if err != nil {
		return artifactnpm.PNPMInspectionLayout{}, err
	}
	i.shared.handoffMu.Lock()
	defer i.shared.handoffMu.Unlock()
	if i.shared.active || !i.shared.prepared.MatchesInput(input) {
		return artifactnpm.PNPMInspectionLayout{}, errors.New("pnpm execution layout lacks complete current observation")
	}
	return i.shared.prepared, nil
}
