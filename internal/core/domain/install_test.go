package domain_test

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestInstallContextRequiresCanonicalNewTarget(t *testing.T) {
	t.Parallel()
	target, err := domain.NewInstallTarget("/tmp/heliopause-target")
	if err != nil {
		t.Fatal(err)
	}
	context, err := domain.NewInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	if context.Target() != target || !context.RequiresNewTarget() {
		t.Fatalf("InstallContext = %#v", context)
	}
	for _, value := range []string{"relative", "/", "/tmp/../target", "/tmp//target", " /tmp/target"} {
		if _, err := domain.NewInstallTarget(value); err == nil {
			t.Fatalf("NewInstallTarget(%q) error = nil", value)
		}
	}
}

func TestLockedDependencyGraphRejectsIncompleteOrUnsafeShape(t *testing.T) {
	t.Parallel()
	source := mustSource(t, "npm")
	primary := lockedDependency(t, "primary", domain.DependencyPrimary, source, "root", "1.0.0")
	child := lockedDependency(t, "child", domain.DependencyTransitive, source, "child", "2.0.0")
	edge, err := domain.NewDependencyEdge(primary.Node(), child.Node())
	if err != nil {
		t.Fatal(err)
	}
	graph, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{child, primary}, []domain.DependencyEdge{edge})
	if err != nil {
		t.Fatal(err)
	}
	if graph.Primary() != primary.Node() || len(graph.Nodes()) != 2 || len(graph.Edges()) != 1 {
		t.Fatalf("graph = %#v", graph)
	}
	if _, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{primary, child}, nil); err == nil {
		t.Fatal("disconnected graph error = nil")
	}
	if _, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{primary, primary}, nil); err == nil {
		t.Fatal("duplicate node error = nil")
	}
}

func lockedDependency(t *testing.T, node string, role domain.DependencyRole, source domain.SourceID, name, version string) domain.LockedDependency {
	t.Helper()
	nodeID, err := domain.NewDependencyNodeID(node)
	if err != nil {
		t.Fatal(err)
	}
	identity := mustIdentity(t, source, name, version, "tarball")
	resolved, err := domain.NewResolvedArtifact(identity, "registry:npm:"+name+"@"+version, "sha512-"+strings.Repeat("a", 32))
	if err != nil {
		t.Fatal(err)
	}
	dependency, err := domain.NewLockedDependency(nodeID, role, resolved)
	if err != nil {
		t.Fatal(err)
	}
	return dependency
}
