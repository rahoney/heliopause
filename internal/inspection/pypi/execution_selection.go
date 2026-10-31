package pypi

import (
	"context"
	"errors"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// InspectExecutionSelection observes every original once in the complete
// explicitly selected environment. Real metadata edges stay distinct from
// additional root availability; no project or install context is fabricated.
func (i *CompositeInspector) InspectExecutionSelection(ctx context.Context, selection domain.ExecutionDependencySelection, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	if ctx == nil || i == nil || !selection.Valid() {
		return nil, errors.New("PyPI execution inspection requires a complete explicit-root selection")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// The old optional loader is authorized against a single requested graph.
	// It cannot silently supply inputs outside this complete selected set.
	if i.prerequisites != nil {
		return nil, errors.New("graph-specific prerequisites are not an execution-selection input")
	}
	nodes := selection.Nodes()
	if len(nodes) > artifactpypi.ResourcePolicyFromContext(ctx).MaxGraphArtifacts() {
		return nil, errors.New("PyPI execution selection exceeds source artifact bounds")
	}
	for _, node := range nodes {
		if node.Artifact().Identity().Source() != artifactpypi.PublicPyPIProfile().Source() || node.Artifact().Identity().Variant() != "wheel" {
			return nil, errors.New("PyPI execution selection requires only exact public wheels")
		}
	}
	return i.inspectDependencySet(ctx, nodes, acquired, true, func(map[domain.DependencyNodeID]graphStaticState) ([]domain.DependencyEdge, map[domain.DependencyNodeID]domain.AcquiredArtifact, error) {
		return selection.Edges(), nil, nil
	})
}
