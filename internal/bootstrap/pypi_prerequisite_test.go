package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// Explicit local integration: official metadata selection and authenticated
// bytes through the same source router/loader as install. This does not execute
// the input wheel or qualify the requested graph.
func TestLinuxInspectionPrerequisiteAcquisitionIntegration(t *testing.T) {
	if os.Getenv("HELOX_PREREQUISITE_ACQUISITION_INTEGRATION") != "1" {
		t.Skip("requires authenticated production client/helper and pinned runtime")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Fatal("requires Linux amd64")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	launcher, err := hosttool.NewSystemObserverLauncher()
	if err != nil {
		t.Fatal(err)
	}
	observer, err := sandbox.NewObserverSupervisor(ctx, func(start context.Context, remote, output string) (sandbox.ObserverProcess, error) {
		return launcher.StartObserver(start, remote, output)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := observer.Close(); err != nil {
			t.Error(err)
		}
	}()
	resolver, err := pypiInstallDependencyResolver(runtime.GOOS, runtime.GOARCH, executor, observer.Observer())
	if err != nil {
		t.Fatal(err)
	}
	intake, err := artifactpypi.NewPublicIntake(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source, _ := domain.NewSourceID("pypi")
	makeNode := func(name, hash string, role domain.DependencyRole) domain.LockedDependency {
		id, _ := domain.NewResolvedArtifactIdentity(source, name, "1.0", "wheel")
		a, _ := domain.NewResolvedArtifact(id, "https://files.pythonhosted.org/"+name+"-1.0-py3-none-any.whl", "sha256:"+hash)
		n, _ := domain.NewDependencyNodeID(name)
		node, _ := domain.NewLockedDependency(n, role, a)
		return node
	}
	root := makeNode("root", strings.Repeat("a", 64), domain.DependencyPrimary)
	target := makeNode("dependency", strings.Repeat("b", 64), domain.DependencyTransitive)
	edge, _ := domain.NewDependencyEdge(root.Node(), target.Node())
	graph, _ := domain.NewLockedDependencyGraph([]domain.LockedDependency{root, target}, []domain.DependencyEdge{edge})
	request := inspectionPrerequisiteRequest{TargetSHA256: strings.Repeat("b", 64), Source: "pypi", Project: "numpy", Version: "2.4.6", Filename: "numpy-2.4.6-cp314-cp314-manylinux_2_27_x86_64.manylinux_2_28_x86_64.whl", SHA256: "a2c306dea656c12c68f51f4cea133cbe78ca7435eb28c735eac1d3ebe73be6e8", Reason: "BROADER_MODULE_PROBES"}
	raw, _ := json.Marshal(request)
	for _, name := range []string{"cpu", "cu126", "cu130", "cu132"} {
		t.Run(name, func(t *testing.T) {
			profile, _ := artifactpypi.PyTorchProfile(name)
			rootCtx, err := artifactpypi.ContextWithResourcePolicy(ctx, profile)
			if err != nil {
				t.Fatal(err)
			}
			inputs, err := inspectionPrerequisiteLoader(func() (string, error) { return string(raw), nil }, resolver, intake)(rootCtx, graph)
			if err != nil || len(inputs) != 1 {
				t.Fatalf("actual pinned acquisition: %v %v", inputs, err)
			}
			if inputs[0].Artifact.Digest().String() != request.SHA256 || inputs[0].Artifact.Identity().Source() != source || inputs[0].Artifact.SizeBytes() != 16638245 {
				t.Fatal("input pin/source/size changed")
			}
			t.Logf("authenticated inspection input source=pypi version=2.4.6 sha256=%s bytes=%d", request.SHA256, inputs[0].Artifact.SizeBytes())
		})
	}
}

func TestInspectionPrerequisiteConfigurationIsExplicitAndUnambiguous(t *testing.T) {
	request := inspectionPrerequisiteRequest{TargetSHA256: strings.Repeat("a", 64), Source: "pypi", Project: "support", Version: "1.0", Filename: "support-1.0-py3-none-any.whl", SHA256: strings.Repeat("b", 64), Reason: "BROADER_MODULE_PROBES"}
	raw, _ := json.Marshal(request)
	if _, err := parseInspectionPrerequisiteRequest(string(raw)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", string(raw) + " {}", strings.Replace(string(raw), `"source":"pypi"`, `"source":"pypi","source":"pypi"`, 1), strings.Replace(string(raw), `"source":"pypi"`, `"source":"pytorch-cu126"`, 1), strings.Replace(string(raw), `"project":"support"`, `"project":"support","extras":"all"`, 1), strings.Replace(string(raw), `"version":"1.0"`, `"version":1`, 1), strings.Replace(string(raw), "support-1.0-py3-none-any.whl", "../support.whl", 1), strings.Replace(string(raw), strings.Repeat("b", 64), "bad", 1)} {
		if _, err := parseInspectionPrerequisiteRequest(bad); err == nil {
			t.Fatalf("accepted ambiguous configuration %s", bad)
		}
	}
	// Empty default performs no resolver/intake operation, even when neither
	// consumer exists. A missing import cannot activate configuration.
	loader := inspectionPrerequisiteLoader(func() (string, error) { return "", nil }, nil, nil)
	if inputs, err := loader(context.Background(), domain.LockedDependencyGraph{}); err != nil || len(inputs) != 0 {
		t.Fatalf("default=%v %v", inputs, err)
	}
}

func TestInspectionPrerequisitePinSelectionAndIntegrity(t *testing.T) {
	source, _ := domain.NewSourceID("pypi")
	makeNode := func(name, hash string, role domain.DependencyRole) domain.LockedDependency {
		id, _ := domain.NewResolvedArtifactIdentity(source, name, "1.0", "wheel")
		a, _ := domain.NewResolvedArtifact(id, "https://files.pythonhosted.org/"+name+"-1.0-py3-none-any.whl", "sha256:"+hash)
		n, _ := domain.NewDependencyNodeID(name)
		node, _ := domain.NewLockedDependency(n, role, a)
		return node
	}
	root := makeNode("root", strings.Repeat("a", 64), domain.DependencyPrimary)
	child := makeNode("dependency", strings.Repeat("b", 64), domain.DependencyTransitive)
	edge, _ := domain.NewDependencyEdge(root.Node(), child.Node())
	original, _ := domain.NewLockedDependencyGraph([]domain.LockedDependency{root, child}, []domain.DependencyEdge{edge})
	for _, mode := range []string{"valid", "root", "unselected", "replacement", "version", "filename", "hash", "intake hash", "transitive"} {
		t.Run(mode, func(t *testing.T) {
			request := inspectionPrerequisiteRequest{TargetSHA256: strings.Repeat("b", 64), Source: "pypi", Project: "support", Version: "1.0", Filename: "support-1.0-py3-none-any.whl", SHA256: strings.Repeat("c", 64), Reason: "BROADER_MODULE_PROBES"}
			node := makeNode("support", request.SHA256, domain.DependencyPrimary)
			graph, _ := domain.NewLockedDependencyGraph([]domain.LockedDependency{node}, nil)
			digest, _ := domain.NewSHA256Digest(request.SHA256)
			a, _ := domain.NewAcquiredArtifactWithDeclaredIntegrity(node.Artifact().Identity(), digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:wheel", 1, node.Artifact().DeclaredIntegrity())
			switch mode {
			case "root":
				request.TargetSHA256 = strings.Repeat("a", 64)
			case "unselected":
				request.TargetSHA256 = strings.Repeat("f", 64)
			case "replacement":
				request.Project = "dependency"
			case "version":
				request.Version = "2.0"
			case "filename":
				request.Filename = "other.whl"
			case "hash":
				request.SHA256 = strings.Repeat("d", 64)
			case "intake hash":
				bad, _ := domain.NewSHA256Digest(strings.Repeat("d", 64))
				a, _ = domain.NewAcquiredArtifactWithDeclaredIntegrity(a.Identity(), bad, a.ContentHandle(), 1, node.Artifact().DeclaredIntegrity())
			case "transitive":
				extra := makeNode("other", strings.Repeat("d", 64), domain.DependencyTransitive)
				e, _ := domain.NewDependencyEdge(node.Node(), extra.Node())
				graph, _ = domain.NewLockedDependencyGraph([]domain.LockedDependency{node, extra}, []domain.DependencyEdge{e})
			}
			ports := &prerequisiteSelectionPorts{graph: graph, artifact: a}
			raw, _ := json.Marshal(request)
			loader := inspectionPrerequisiteLoader(func() (string, error) { return string(raw), nil }, ports, ports)
			inputs, err := loader(context.Background(), original)
			if mode == "valid" {
				if err != nil || len(inputs) != 1 || ports.acquires != 1 {
					t.Fatalf("%v %v", inputs, err)
				}
			} else if err == nil {
				t.Fatal("invalid input selected")
			}
			if mode == "root" || mode == "unselected" || mode == "replacement" {
				if ports.resolves != 0 || ports.acquires != 0 {
					t.Fatal("unauthorized resolution attempted")
				}
			}
		})
	}
}

type prerequisiteSelectionPorts struct {
	graph              domain.LockedDependencyGraph
	artifact           domain.AcquiredArtifact
	resolves, acquires int
}

func (p *prerequisiteSelectionPorts) ResolveDependencies(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error) {
	p.resolves++
	hash, _ := domain.NewSHA256Digest(strings.Repeat("f", 64))
	return domain.NewDependencyResolution(p.graph, "fixture-runtime", hash)
}
func (p *prerequisiteSelectionPorts) Resolve(context.Context, domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	panic("isolated graph resolution required")
}
func (p *prerequisiteSelectionPorts) Acquire(context.Context, domain.RunID, domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	p.acquires++
	return p.artifact, nil
}
