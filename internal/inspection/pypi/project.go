package pypi

import (
	"context"
	"errors"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// InspectProject supplies exact per-target closures without fabricating a
// requested primary or broadening any native dynamic observation allowance.
func (i *CompositeInspector) InspectProject(ctx context.Context, snapshot domain.ProjectDependencySnapshot, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	if ctx == nil || !snapshot.Valid() || (snapshot.Context().Mode() != domain.InstallPythonVenv && snapshot.Context().Mode() != domain.InstallPythonProjectControls) || snapshot.Source() != artifactpypi.PublicPyPIProfile().Source() {
		return nil, errors.New("PyPI project inspection requires a complete public wheel snapshot")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	nodes := make([]domain.LockedDependency, 0, len(snapshot.Dependencies()))
	for _, artifact := range snapshot.Dependencies() {
		if artifact.Identity().Variant() != "wheel" {
			return nil, errors.New("PyPI complete project inspection requires exact wheels")
		}
		id, err := domain.ProjectDependencyNodeID(artifact.Identity())
		if err != nil {
			return nil, err
		}
		node, err := domain.NewLockedDependency(id, domain.DependencyTransitive, artifact)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	if snapshot.DependencyFree() {
		if len(acquired) != 0 {
			return nil, errors.New("empty PyPI project inspection contains undeclared artifacts")
		}
		return map[domain.DependencyNodeID]domain.InspectionReport{}, nil
	}
	return i.inspectDependencySet(ctx, nodes, acquired, false, func(states map[domain.DependencyNodeID]graphStaticState) ([]domain.DependencyEdge, map[domain.DependencyNodeID]domain.AcquiredArtifact, error) {
		wheels := make([]artifactpypi.WheelInspection, 0, len(nodes))
		for _, node := range nodes {
			wheels = append(wheels, states[node.Node()].wheel)
		}
		_, edges, err := artifactpypi.ProjectRecordsFromWheels(snapshot.Dependencies(), wheels, sandbox.PinnedPythonRuntime().PythonVersion)
		return edges, nil, err
	})
}
