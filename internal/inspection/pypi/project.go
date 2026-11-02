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
	return i.inspectProject(ctx, snapshot, acquired, nil)
}

func (i *CompositeInspector) InspectProjectSelection(ctx context.Context, update domain.ProjectDependencyUpdate, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	if !update.Valid() {
		return nil, errors.New("python inspection needs the actual frozen project selection")
	}
	var sourceControl, selectionControl domain.ProjectControlFile
	for _, control := range update.SelectedControls() {
		switch control.Name() {
		case artifactpypi.RequirementsInputControlName:
			sourceControl = control
		case artifactpypi.RequirementsSelectionControlName:
			selectionControl = control
		}
	}
	if !sourceControl.Present() && !selectionControl.Present() {
		for _, control := range update.SelectedControls() {
			if control.Name() == artifactpypi.UVProjectIntentControl {
				selection, err := sandbox.ParseUVProjectUpdate(update)
				if err != nil {
					return nil, err
				}
				return i.inspectProject(ctx, update.Snapshot(), acquired, selection)
			}
		}
		return i.InspectProject(ctx, update.Snapshot(), acquired)
	}
	runtime := sandbox.PinnedPythonRuntime()
	target := artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}
	selection, err := artifactpypi.ParseRequirementsUpdate(update, target)
	if err != nil {
		return nil, err
	}
	return i.inspectProject(ctx, update.Snapshot(), acquired, selection)
}

// The two actual producers retain their own provenance and share static readback.
type projectMetadataRecords interface {
	ProjectRecordsFromInspections([]domain.ResolvedArtifact, []artifactpypi.WheelInspection) ([]domain.LockedDependency, []domain.DependencyEdge, error)
}

func (i *CompositeInspector) inspectProject(ctx context.Context, snapshot domain.ProjectDependencySnapshot, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact, selection projectMetadataRecords) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
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
		if selection != nil {
			_, edges, err := selection.ProjectRecordsFromInspections(snapshot.Dependencies(), wheels)
			return edges, nil, err
		}
		_, edges, err := artifactpypi.ProjectRecordsFromWheels(snapshot.Dependencies(), wheels, sandbox.PinnedPythonRuntime().PythonVersion)
		return edges, nil, err
	})
}
