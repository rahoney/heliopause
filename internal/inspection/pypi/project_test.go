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
	"github.com/rahoney/heliopause/internal/sandbox"
)

type projectClosureRunner struct {
	*prerequisiteRunner
	closures map[string][]string
}

func (r *projectClosureRunner) InspectWheelWithPlan(ctx context.Context, target domain.AcquiredArtifact, plan artifactpypi.ObservationPlan, closure []domain.AcquiredArtifact) (sandbox.PythonObservationResult, error) {
	names := []string{}
	for _, artifact := range closure {
		names = append(names, artifact.Identity().Name())
	}
	r.closures[target.Identity().Name()] = names
	return r.prerequisiteRunner.InspectWheelWithPlan(ctx, target, plan, closure)
}

func TestPyPICompleteProjectUsesPerTargetClosureAndStaticAllGate(t *testing.T) {
	for _, mode := range []string{"complete", "missing-dependency", "unsatisfied", "static-invalid", "foreign-acquired", "controls-complete", "controls-missing-dependency", "controls-unsatisfied", "controls-static-invalid", "controls-foreign-acquired"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			target, _ := domain.NewInstallTarget(filepath.Join(root, "venv"))
			install, _ := domain.NewPythonVenvInstallContext(target)
			if strings.HasPrefix(mode, "controls-") {
				install, _ = domain.NewPythonProjectControlContext(target)
				mode = strings.TrimPrefix(mode, "controls-")
			}
			dependency := "child>=1.0"
			if mode == "unsatisfied" {
				dependency = "child>=2.0"
			}
			if mode == "missing-dependency" {
				dependency = "missing>=1.0"
			}
			artifacts := []domain.ResolvedArtifact{}
			acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
			var rootArtifact domain.AcquiredArtifact
			for _, name := range []string{"root", "child", "other"} {
				metadata := ""
				if name == "root" {
					metadata = "Requires-Dist: " + dependency + "\n"
				}
				a := prerequisiteFixtureWheel(t, root, name, "py3-none-any", nil, metadata)
				integrity, _ := a.DeclaredIntegrity()
				resolved, err := domain.NewResolvedArtifact(a.Identity(), "https://files.pythonhosted.org/packages/"+name+"-1.0-py3-none-any.whl", integrity)
				if err != nil {
					t.Fatal(err)
				}
				artifacts = append(artifacts, resolved)
				node, _ := domain.ProjectDependencyNodeID(a.Identity())
				acquired[node] = a
				if name == "root" {
					rootArtifact = a
				}
			}
			digest, _ := domain.NewSHA256Digest(strings.Repeat("c", 64))
			control, _ := domain.NewProjectControlDigest("complete-source", digest)
			snapshot, err := domain.NewProjectDependencySnapshot(install, artifacts[0].Identity().Source(), []domain.ProjectControlDigest{control}, artifacts, digest)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "static-invalid" {
				parts := strings.Split(rootArtifact.ContentHandle(), ":")
				if err := os.WriteFile(filepath.Join(root, parts[1], "wheel.whl"), []byte("changed archive"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "foreign-acquired" {
				node, _ := domain.NewDependencyNodeID("unselected")
				acquired[node] = rootArtifact
			}
			runner := &projectClosureRunner{prerequisiteRunner: &prerequisiteRunner{t: t}, closures: map[string][]string{}}
			static, _ := NewStaticInspector(root, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"})
			dynamic, _ := NewDynamicInspector(runner)
			inspector, _ := NewCompositeInspector(static, dynamic)
			reports, err := inspector.InspectProject(context.Background(), snapshot, acquired)
			if mode == "complete" {
				if err != nil || len(reports) != 3 || !reflect.DeepEqual(runner.closures, map[string][]string{"root": {"child", "root"}, "child": {"child"}, "other": {"other"}}) {
					t.Fatal("project mixed disconnected roots or lost a dependency", runner.closures, err)
				}
				return
			}
			if len(runner.calls) != 0 {
				t.Fatal("unsafe static/project data reached dynamic observation", runner.calls)
			}
			if mode == "static-invalid" {
				if err != nil || len(reports) != 3 {
					t.Fatal("static refusal lost reports", err)
				}
				node, _ := domain.ProjectDependencyNodeID(rootArtifact.Identity())
				if len(reports[node].Findings()) == 0 {
					t.Fatal("static refusal was hidden")
				}
			} else if err == nil {
				t.Fatal("invalid complete project accepted", mode)
			}
		})
	}
}
