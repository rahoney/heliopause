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

func TestExecutionSelectionInspectsEveryOriginalInWholeEnvironment(t *testing.T) {
	for _, mode := range []string{"complete", "missing", "foreign", "static-blocked", "legacy-prerequisite"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			nodes := map[string]domain.LockedDependency{}
			acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
			for _, name := range []string{"root", "child", "other"} {
				metadata := ""
				if name == "root" {
					metadata = "Requires-Dist: child>=1.0\n"
				}
				a := prerequisiteFixtureWheel(t, root, name, "py3-none-any", nil, metadata)
				integrity, _ := a.DeclaredIntegrity()
				resolved, _ := domain.NewResolvedArtifact(a.Identity(), "https://files.pythonhosted.org/packages/"+name+"-1.0-py3-none-any.whl", integrity)
				id, _ := domain.NewDependencyNodeID(name)
				role := domain.DependencyPrimary
				if name == "child" {
					role = domain.DependencyTransitive
				}
				node, err := domain.NewLockedDependency(id, role, resolved)
				if err != nil {
					t.Fatal(err)
				}
				nodes[name], acquired[id] = node, a
			}
			edge, _ := domain.NewDependencyEdge(nodes["root"].Node(), nodes["child"].Node())
			primary, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{nodes["root"], nodes["child"]}, []domain.DependencyEdge{edge})
			if err != nil {
				t.Fatal(err)
			}
			other, _ := domain.NewLockedDependencyGraph([]domain.LockedDependency{nodes["other"]}, nil)
			selection, err := domain.NewExecutionDependencySelection(primary, []domain.LockedDependencyGraph{other})
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "missing":
				delete(acquired, nodes["other"].Node())
			case "foreign":
				acquired[nodes["other"].Node()] = acquired[nodes["root"].Node()]
			case "static-blocked":
				parts := strings.Split(acquired[nodes["other"].Node()].ContentHandle(), ":")
				if err := os.WriteFile(filepath.Join(root, parts[1], "wheel.whl"), []byte("changed original"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			runner := &projectClosureRunner{prerequisiteRunner: &prerequisiteRunner{t: t}, closures: map[string][]string{}}
			static, _ := NewStaticInspector(root, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"})
			dynamic, _ := NewDynamicInspector(runner)
			inspector, _ := NewCompositeInspector(static, dynamic)
			if mode == "legacy-prerequisite" {
				inspector.WithInspectionPrerequisites(func(context.Context, domain.LockedDependencyGraph) ([]InspectionPrerequisite, error) {
					t.Fatal("single-graph prerequisite authority used outside its contract")
					return nil, nil
				})
			}
			reports, err := inspector.InspectExecutionSelection(context.Background(), selection, acquired)
			if mode == "complete" {
				want := map[string][]string{"root": {"child", "other", "root"}, "child": {"child", "other", "root"}, "other": {"child", "other", "root"}}
				if err != nil || len(reports) != 3 || len(runner.calls) != 3 || !reflect.DeepEqual(runner.closures, want) || len(selection.Edges()) != 1 {
					t.Fatal("whole context lost, repeated inspection or invented edge", runner.closures, err)
				}
				return
			}
			if len(runner.calls) != 0 {
				t.Fatal("incomplete or unapproved selected original reached dynamic inspection")
			}
			if mode == "static-blocked" {
				if err != nil || len(reports) != 3 || len(reports[nodes["other"].Node()].Findings()) == 0 {
					t.Fatal("whole static refusal lost", err)
				}
			} else if err == nil {
				t.Fatal("unbound execution selection accepted", mode)
			}
		})
	}
}
