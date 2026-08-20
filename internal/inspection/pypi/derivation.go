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
		if !allowComplete(source) {
			return nil, errors.New("PyPI sdist source is not a complete ALLOW inspection")
		}
		recipe, err := d.static.InspectSdist(ctx, source.Artifact())
		if err != nil {
			return nil, err
		}
		wheels := map[string]domain.AcquiredArtifact{}
		for _, in := range inspections {
			a := in.Artifact()
			if a.Identity().Variant() == "wheel" {
				if !allowComplete(in) {
					continue
				}
				if _, duplicate := wheels[a.Identity().Name()]; duplicate {
					return nil, errors.New("PyPI verified build wheel is ambiguous")
				}
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
		if err != nil || result.Status() != domain.SandboxCompleted || built.SourceDigest != source.Artifact().Digest() || len(built.BuildRequirementDigests) != len(inputs) {
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
		configDigest, err := domain.NewSHA256Digest(built.BuildConfigSHA256)
		if err != nil {
			return nil, err
		}
		binding, err := domain.NewDerivationBinding(built.SourceDigest, built.BuildRequirementDigests, "pep517-gvisor", configDigest)
		if err != nil {
			return nil, err
		}
		checkID, err := domain.NewCheckID("pypi-pep517-build")
		if err != nil {
			return nil, err
		}
		check, err := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		if err != nil {
			return nil, err
		}
		evidenceID, err := domain.NewEvidenceID("pypi-pep517-build-result")
		if err != nil {
			return nil, err
		}
		evidence, err := domain.NewEvidence(evidenceID, checkID, built.Artifact.Identity(), built.Artifact.Digest(), "pypi-pep517-build", "Trusted gVisor PEP 517 build and observation collection completed.")
		if err != nil {
			return nil, err
		}
		item, err := domain.NewDerivedDependency(source.Node(), node, built.Artifact, binding, check, evidence)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func allowComplete(inspection domain.DependencyInspection) bool {
	if inspection.PolicyDecision().Decision() != domain.DecisionAllow {
		return false
	}
	for _, check := range inspection.Checks() {
		if check.Required() && (check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted) {
			return false
		}
	}
	return true
}
