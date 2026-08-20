package pypi

import (
	"context"
	"errors"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type Deriver struct {
	static  *StaticInspector
	builder *sandbox.PythonSdistBuilder
}

func NewDeriver(static *StaticInspector, builder *sandbox.PythonSdistBuilder) (*Deriver, error) {
	if static == nil || builder == nil {
		return nil, errors.New("PyPI deriver requires static inspector and builder")
	}
	return &Deriver{static, builder}, nil
}
func (d *Deriver) Derive(ctx context.Context, inspections []domain.DependencyInspection) ([]domain.DerivedDependency, error) {
	var out []domain.DerivedDependency
	for _, source := range inspections {
		if source.Artifact().Identity().Variant() != "sdist" {
			continue
		}
		recipe, err := d.static.InspectSdist(ctx, source.Artifact())
		if err != nil {
			return nil, err
		}
		wheels := map[string]domain.AcquiredArtifact{}
		for _, in := range inspections {
			a := in.Artifact()
			if a.Identity().Variant() == "wheel" {
				wheels[a.Identity().Name()] = a
			}
		}
		inputs := make([]domain.AcquiredArtifact, 0, len(recipe.BuildRequirements))
		for _, name := range recipe.BuildRequirements {
			a, ok := wheels[name]
			if !ok {
				return nil, errors.New("PyPI sdist build requirement is absent from verified graph")
			}
			inputs = append(inputs, a)
		}
		built, result, err := d.builder.Build(ctx, source.Artifact(), recipe, inputs)
		if err != nil || result.Status() != domain.SandboxCompleted {
			return nil, errors.New("PyPI sdist build is incomplete")
		}
		nodeID, err := domain.NewDependencyNodeID(source.Node().String() + "-derived")
		if err != nil {
			return nil, err
		}
		resolved, err := domain.NewResolvedArtifact(built.Artifact.Identity(), "derived:"+source.Node().String(), "sha256:"+built.Artifact.Digest().String())
		if err != nil {
			return nil, err
		}
		node, err := domain.NewLockedDependencyWithRecordPath(nodeID, domain.DependencyTransitive, resolved, built.Filename, false)
		if err != nil {
			return nil, err
		}
		item, err := domain.NewDerivedDependency(source.Node(), node, built.Artifact)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}
