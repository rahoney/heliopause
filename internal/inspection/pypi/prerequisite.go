package pypi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// InspectionPrerequisite is controller configuration, not wheel metadata.
// Its target must be a non-primary member of the original requested graph.
type InspectionPrerequisite struct {
	TargetDigest string
	Artifact     domain.AcquiredArtifact
	Reason       string
}

type InspectionPrerequisiteLoader func(context.Context, domain.LockedDependencyGraph) ([]InspectionPrerequisite, error)

// WithInspectionPrerequisites is deliberately opt-in. No missing-import output
// or package identity can select an input; the default remains empty.
func (i *CompositeInspector) WithInspectionPrerequisites(loader InspectionPrerequisiteLoader) *CompositeInspector {
	i.prerequisites = loader
	return i
}

func (i *CompositeInspector) preparePrerequisites(ctx context.Context, graph domain.LockedDependencyGraph, states map[domain.DependencyNodeID]graphStaticState, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.AcquiredArtifact, error) {
	selected := make(map[domain.DependencyNodeID]domain.AcquiredArtifact)
	if i.prerequisites == nil {
		return selected, nil
	}
	inputs, err := i.prerequisites(ctx, graph)
	if err != nil {
		return nil, err
	}
	if len(inputs) > 1 {
		return nil, errors.New("inspection prerequisite selection is invalid")
	}
	for _, input := range inputs {
		declared, declaredOK := input.Artifact.DeclaredIntegrity()
		if input.Reason != "BROADER_MODULE_PROBES" || input.Artifact.Identity().Source() != artifactpypi.PublicPyPIProfile().Source() || !declaredOK || declared != "sha256:"+input.Artifact.Digest().String() {
			return nil, errors.New("inspection prerequisite is not explicitly approved")
		}
		var target domain.DependencyNodeID
		originals := make([]artifactpypi.WheelInspection, 0, len(states))
		for _, node := range graph.Nodes() {
			if node.Artifact().DeclaredIntegrity() == "sha256:"+input.TargetDigest {
				if node.Node() == graph.Primary() {
					return nil, errors.New("the original requested artifact cannot use inspection prerequisites")
				}
				target = node.Node()
			}
			originals = append(originals, states[node.Node()].wheel)
		}
		if target.String() == "" {
			return nil, errors.New("inspection prerequisite target is not in the requested graph")
		}
		wheel, report, err := i.static.InspectWheel(ctx, input.Artifact)
		if err != nil || hasBlockingFinding(report) || report.Execution().Status() != domain.ExecutionCompleted {
			return nil, errors.New("inspection prerequisite static validation is incomplete")
		}
		if _, err := artifactpypi.ValidateInspectionPrerequisite(wheel, originals, artifactpypi.ResourcePolicyFromContext(ctx)); err != nil {
			return nil, err
		}
		// Charge storage against the original graph's authorization too.
		var files, expanded, compressed int64
		for _, original := range originals {
			for _, file := range original.Files {
				files++
				expanded += file.Size
			}
		}
		for _, file := range wheel.Files {
			files++
			expanded += file.Size
		}
		for _, node := range graph.Nodes() {
			compressed += int64(acquired[node.Node()].SizeBytes())
		}
		compressed += int64(input.Artifact.SizeBytes())
		policy := artifactpypi.ResourcePolicyFromContext(ctx)
		if len(graph.Nodes())+1 > policy.MaxGraphArtifacts() || files > policy.MaxGraphFiles() || expanded > policy.MaxGraphUncompressed() || compressed > policy.MaxGraphCompressed() || expanded+compressed > policy.MaxTemporaryDisk() {
			return nil, errors.New("inspection prerequisite exceeds original graph resource authorization")
		}
		selected[target] = input.Artifact
	}
	return selected, nil
}

func inspectionEnvironmentBinding(ctx context.Context, target domain.AcquiredArtifact, plan artifactpypi.ObservationPlan, base, inputs []domain.AcquiredArtifact) (string, error) {
	identities := func(artifacts []domain.AcquiredArtifact) []string {
		result := make([]string, 0, len(artifacts))
		for _, a := range artifacts {
			id := a.Identity()
			result = append(result, id.Source().String()+":"+id.Name()+":"+id.Version()+":"+a.Digest().String())
		}
		sort.Strings(result)
		return result
	}
	policy := artifactpypi.ResourcePolicyFromContext(ctx)
	encoded, err := json.Marshal(struct {
		Target, Profile     string
		Base, Inputs        []string
		Runtime             sandbox.PythonRuntime
		Plan                artifactpypi.ObservationPlan
		CPU                 int
		Memory, Tmpfs, Wall int64
	}{target.Digest().String(), artifactpypi.RootSourceProfileNameFromContext(ctx), identities(base), identities(inputs), sandbox.PinnedPythonRuntime(), plan, policy.RuntimeCPUSecs(), policy.RuntimeMemory(), policy.RuntimeTmpfs(), int64(policy.Duration())})
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(encoded)
	return hex.EncodeToString(hash[:]), nil
}
