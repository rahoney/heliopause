package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func qualificationResolutionFixture(t *testing.T, mode string) domain.DependencyResolution {
	t.Helper()
	profile, _ := artifactpypi.PyTorchProfile("cu126")
	makeNode := func(id, name, version, hash string, source domain.SourceID, role domain.DependencyRole) domain.LockedDependency {
		identity, err := domain.NewResolvedArtifactIdentity(source, name, version, "wheel")
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := domain.NewResolvedArtifact(identity, "https://files.pythonhosted.org/fixture.whl", "sha256:"+hash)
		if err != nil {
			t.Fatal(err)
		}
		nodeID, _ := domain.NewDependencyNodeID(id)
		node, err := domain.NewLockedDependency(nodeID, role, artifact)
		if err != nil {
			t.Fatal(err)
		}
		return node
	}
	version := runtimeidentity.PyTorchCU126Version
	if mode == "root version" {
		version = "1.0"
	}
	root := makeNode("root", "torch", version, strings.Repeat("a", 64), profile.Source(), domain.DependencyPrimary)
	source := artifactpypi.PublicPyPIProfile().Source()
	if mode == "wrong source" {
		source = profile.Source()
	}
	name := "cuda-bindings"
	if mode == "replacement" {
		name = "numpy"
	}
	child := makeNode("child", name, "12.9.10", strings.Repeat("b", 64), source, domain.DependencyTransitive)
	nodes := []domain.LockedDependency{root, child}
	edge, _ := domain.NewDependencyEdge(root.Node(), child.Node())
	edges := []domain.DependencyEdge{edge}
	if mode == "missing" {
		nodes = nodes[:1]
		edges = nil
	}
	if mode == "duplicate" {
		extra := makeNode("extra", name, "12.9.11", strings.Repeat("c", 64), source, domain.DependencyTransitive)
		nodes = append(nodes, extra)
		e, _ := domain.NewDependencyEdge(root.Node(), extra.Node())
		edges = append(edges, e)
	}
	graph, err := domain.NewLockedDependencyGraph(nodes, edges)
	if err != nil {
		t.Fatal(err)
	}
	python := sandbox.PinnedPythonRuntime()
	runtimeKey := "python:" + python.PythonVersion + ";pip:" + python.PipVersion + ";target:" + python.InterpreterTag + "/" + python.ABITag + "/" + python.PlatformTag + ";source:" + profile.Name()
	if mode == "runtime" {
		runtimeKey = "unselected-runtime"
	}
	digest, _ := domain.NewSHA256Digest(strings.Repeat("f", 64))
	resolution, err := domain.NewDependencyResolution(graph, runtimeKey, digest)
	if err != nil {
		t.Fatal(err)
	}
	return resolution
}

func TestTrustedQualificationPlanBindsSelectedMetadata(t *testing.T) {
	for _, mode := range []string{"valid", "missing", "duplicate", "wrong source", "replacement", "root version", "runtime", "unknown plan", "cpu"} {
		t.Run(mode, func(t *testing.T) {
			resolution := qualificationResolutionFixture(t, mode)
			profile, plan := "cu126", cudaBindingsNumPyPlan
			if mode == "unknown plan" {
				plan = "artifact-suggested-plan"
			}
			if mode == "cpu" {
				profile = "cpu"
			}
			request, receipt, err := qualificationPrerequisite(resolution, profile, runtimeidentity.PyTorchCU126Version, plan)
			if mode != "valid" {
				if err == nil {
					t.Fatal("invalid qualification metadata accepted")
				}
				return
			}
			if err != nil || request.TargetSHA256 != strings.Repeat("b", 64) || request.Project != "numpy" || request.Version != "2.4.6" || request.Reason != "BROADER_MODULE_PROBES" {
				t.Fatalf("incorrect explicit pin: %+v %v", request, err)
			}
			var data map[string]any
			if json.Unmarshal(receipt, &data) != nil || data["target_version"] != "12.9.10" || data["target_source"] != "pypi" {
				t.Fatal("selected metadata was not frozen")
			}
		})
	}
}

func TestPrerequisiteMetadataCheckRejectsBeforeResolutionReturns(t *testing.T) {
	graph := qualificationResolutionFixture(t, "valid").Graph()
	base := &prerequisiteSelectionPorts{graph: graph}
	request, _, err := qualificationPrerequisite(qualificationResolutionFixture(t, "valid"), "cu126", runtimeidentity.PyTorchCU126Version, cudaBindingsNumPyPlan)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"empty", "valid", "stale", "root", "replacement", "malformed", "read failure"} {
		t.Run(mode, func(t *testing.T) {
			pin := request
			if mode == "stale" {
				pin.TargetSHA256 = strings.Repeat("c", 64)
			}
			if mode == "root" {
				pin.TargetSHA256 = strings.Repeat("a", 64)
			}
			if mode == "replacement" {
				pin.Project = "cuda-bindings"
			}
			body, _ := json.Marshal(pin)
			raw := string(body)
			if mode == "empty" {
				raw = ""
			}
			if mode == "malformed" {
				raw = "{}"
			}
			before := base.resolves
			resolver := prerequisiteCheckingResolver{resolver: base, read: func() (string, error) {
				if mode == "read failure" {
					return "", errors.New("fixture read failure")
				}
				return raw, nil
			}}
			_, err := resolver.ResolveDependencies(context.Background(), domain.ArtifactReference{}, domain.InstallContext{})
			if (mode == "empty" || mode == "valid") != (err == nil) {
				t.Fatalf("mode=%s err=%v", mode, err)
			}
			if (mode == "malformed" || mode == "read failure") && base.resolves != before {
				t.Fatal("invalid configuration reached external resolver")
			}
			if base.acquires != 0 {
				t.Fatal("metadata check acquired payload bytes")
			}
		})
	}
}
