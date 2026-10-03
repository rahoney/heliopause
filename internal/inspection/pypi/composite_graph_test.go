package pypi

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestGraphClosureUsesExactTransitiveMembersInNodeOrder(t *testing.T) {
	graph, acquired := graphFixture(t)
	closure, err := graphClosure(graph, graph.Primary(), acquired)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(closure))
	for _, artifact := range closure {
		got = append(got, artifact.Identity().Name())
	}
	if want := []string{"dependency", "root", "transitive"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("closure=%v want=%v", got, want)
	}
}

func TestGraphClosureRejectsAcquiredIdentityMismatch(t *testing.T) {
	graph, acquired := graphFixture(t)
	nodes := graph.Nodes()
	wrongIdentity, _ := domain.NewResolvedArtifactIdentity(nodes[1].Artifact().Identity().Source(), "unlocked", "1.0", "wheel")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("f", 64))
	acquired[nodes[1].Node()], _ = domain.NewAcquiredArtifactWithDeclaredIntegrity(wrongIdentity, digest, "intake:run_bbbbbbbbbbbbbbbbbbbbbbbbbb:wheel", 1, nodes[1].Artifact().DeclaredIntegrity())
	if _, err := graphClosure(graph, graph.Primary(), acquired); err == nil {
		t.Fatal("identity-mismatched closure artifact accepted")
	}
}

func TestValidateRuntimeClosureFailsBeforeDynamicExecution(t *testing.T) {
	graph, acquired := graphFixture(t)
	policy := artifactpypi.PublicPyPIProfile().ResourcePolicy()
	for node, artifact := range acquired {
		acquired[node], _ = domain.NewAcquiredArtifactWithDeclaredIntegrity(artifact.Identity(), artifact.Digest(), artifact.ContentHandle(), uint64(policy.MaxGraphCompressed())+1, mustDeclaredIntegrity(t, artifact))
	}
	closure, err := graphClosure(graph, graph.Primary(), acquired)
	if err != nil {
		t.Fatal(err)
	}
	states := make(map[domain.DependencyNodeID]graphStaticState, len(graph.Nodes()))
	for _, node := range graph.Nodes() {
		states[node.Node()] = graphStaticState{dynamic: true}
	}
	if err := validateRuntimeClosure(context.Background(), closure, graph, states); err == nil {
		t.Fatal("oversized closure passed runtime preflight")
	}
}

func TestInspectGraphSkipsEveryDynamicRunAfterStaticBlocking(t *testing.T) {
	root := t.TempDir()
	graph, acquired := invalidGraphFixture(t, root)
	static, err := NewStaticInspector(root, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"})
	if err != nil {
		t.Fatal(err)
	}
	dynamic, err := NewDynamicInspector(panicWheelRunner{})
	if err != nil {
		t.Fatal(err)
	}
	inspector, err := NewCompositeInspector(static, dynamic)
	if err != nil {
		t.Fatal(err)
	}
	reports, err := inspector.InspectGraph(context.Background(), graph, acquired)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != len(graph.Nodes()) {
		t.Fatalf("reports=%#v", reports)
	}
	for _, node := range graph.Nodes() {
		if len(reports[node.Node()].Findings()) == 0 {
			t.Fatalf("static finding missing for %s", node.Node().String())
		}
	}
}

func graphFixture(t *testing.T) (domain.LockedDependencyGraph, map[domain.DependencyNodeID]domain.AcquiredArtifact) {
	t.Helper()
	source, _ := domain.NewSourceID("pypi")
	makeNode := func(id, name, run string, role domain.DependencyRole) (domain.LockedDependency, domain.AcquiredArtifact) {
		identity, _ := domain.NewResolvedArtifactIdentity(source, name, "1.0", "wheel")
		resolved, _ := domain.NewResolvedArtifact(identity, "https://files.pythonhosted.org/"+name+".whl", "sha256:"+strings.Repeat("a", 64))
		nodeID, _ := domain.NewDependencyNodeID(id)
		node, _ := domain.NewLockedDependency(nodeID, role, resolved)
		digestCharacter := map[string]string{"root": "a", "dependency": "b", "transitive": "c"}[name]
		digest, _ := domain.NewSHA256Digest(strings.Repeat(digestCharacter, 64))
		artifact, _ := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+run+":wheel", 1, resolved.DeclaredIntegrity())
		return node, artifact
	}
	rootRun, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	dependencyRun, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	transitiveRun, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	root, rootArtifact := makeNode("root", "root", rootRun.String(), domain.DependencyPrimary)
	dependency, dependencyArtifact := makeNode("dependency", "dependency", dependencyRun.String(), domain.DependencyTransitive)
	transitive, transitiveArtifact := makeNode("transitive", "transitive", transitiveRun.String(), domain.DependencyTransitive)
	first, _ := domain.NewDependencyEdge(root.Node(), dependency.Node())
	second, _ := domain.NewDependencyEdge(dependency.Node(), transitive.Node())
	graph, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{root, dependency, transitive}, []domain.DependencyEdge{first, second})
	if err != nil {
		t.Fatal(err)
	}
	return graph, map[domain.DependencyNodeID]domain.AcquiredArtifact{root.Node(): rootArtifact, dependency.Node(): dependencyArtifact, transitive.Node(): transitiveArtifact}
}

func invalidGraphFixture(t *testing.T, root string) (domain.LockedDependencyGraph, map[domain.DependencyNodeID]domain.AcquiredArtifact) {
	t.Helper()
	graph, acquired := graphFixture(t)
	for _, node := range graph.Nodes() {
		artifact := acquired[node.Node()]
		parts := strings.Split(artifact.ContentHandle(), ":")
		directory := filepath.Join(root, parts[1])
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		body := []byte("not a valid wheel")
		if err := os.WriteFile(filepath.Join(directory, "filename"), []byte(artifact.Identity().Name()+"-1.0-cp314-cp314-linux_x86_64.whl"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "wheel.whl"), body, 0o600); err != nil {
			t.Fatal(err)
		}
		updated, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(artifact.Identity(), artifact.Digest(), artifact.ContentHandle(), uint64(len(body)), mustDeclaredIntegrity(t, artifact))
		if err != nil {
			t.Fatal(err)
		}
		acquired[node.Node()] = updated
	}
	return graph, acquired
}

func mustDeclaredIntegrity(t *testing.T, artifact domain.AcquiredArtifact) string {
	t.Helper()
	declared, ok := artifact.DeclaredIntegrity()
	if !ok {
		t.Fatal("declared integrity missing")
	}
	return declared
}
