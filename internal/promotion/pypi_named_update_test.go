package promotion

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type namedUpdateSelectionFixture struct {
	t       *testing.T
	profile artifactpypi.SourceProfile
	mode    string
	calls   int
}

func (r *namedUpdateSelectionFixture) ResolveDependencyUpdate(_ context.Context, _ domain.ArtifactReference, _ domain.InstallContext, controls []domain.ProjectControlFile) (domain.DependencyResolution, error) {
	r.t.Helper()
	r.calls++
	runtime := sandbox.PinnedPythonRuntime()
	input, err := artifactpypi.ParseUpgradeInput(controls, runtime.PythonVersion)
	if err != nil {
		r.t.Fatal(err)
	}
	version := "2.0+" + strings.TrimPrefix(r.profile.Name(), "pytorch:")
	wire := map[string]any{"version": "1", "pip_version": runtime.PipVersion, "environment": map[string]string{"implementation_name": "cpython", "implementation_version": runtime.PythonVersion, "python_full_version": runtime.PythonVersion, "platform_machine": "x86_64", "sys_platform": "linux"}, "install": []any{map[string]any{"requested": true, "download_info": map[string]any{"url": r.profile.IndexURL() + "torch-" + version + "-cp314-cp314-manylinux_2_28_x86_64.whl", "archive_info": map[string]any{"hashes": map[string]string{"sha256": strings.Repeat("a", 64)}}}, "metadata": map[string]any{"name": "torch", "version": version}}}}
	body, _ := json.Marshal(wire)
	first := []artifactpypi.SourcePinnedUpgradeRecord{{SourceID: r.profile.Source().String(), Requirement: input.Requirement().Requirement(), SelectionReport: body, CandidateReport: body}}
	final := []artifactpypi.SourcePinnedUpgradeRecord{{SourceID: r.profile.Source().String(), Requirement: "torch==" + version, CandidateReport: body}}
	selected, err := input.BuildSourcePinnedUpgrade(first, final, runtime.PipVersion)
	if err != nil {
		r.t.Fatal(err)
	}
	graph, digest := selected.Graph(), selected.Digest()
	identity := "python:" + runtime.PythonVersion + ";pip:" + runtime.PipVersion + ";target:" + runtime.InterpreterTag + "/" + runtime.ABITag + "/" + runtime.PlatformTag + ";source:" + r.profile.Name()
	output := selected.Controls()
	switch r.mode {
	case "input":
		output[0], _ = domain.NewProjectControlFile(artifactpypi.UpgradeInputControlName, []byte("{}"), true)
	case "constraints":
		output[3], _ = domain.NewProjectControlFile(artifactpypi.SourcePinnedUpgradeConstraintsControlName, []byte("torch==99\n"), true)
	case "missing":
		output = output[:3]
	case "duplicate":
		output[3] = output[0]
	case "runtime":
		identity += ";changed"
	case "digest":
		digest, _ = domain.NewSHA256Digest(strings.Repeat("f", 64))
	case "graph":
		original := graph.Nodes()[0]
		artifact, err := domain.NewResolvedArtifact(original.Artifact().Identity(), original.Artifact().AcquisitionLocator(), "sha256:"+strings.Repeat("b", 64))
		if err != nil {
			r.t.Fatal(err)
		}
		node, err := domain.NewLockedDependency(original.Node(), original.Role(), artifact)
		if err != nil {
			r.t.Fatal(err)
		}
		graph, err = domain.NewLockedDependencyGraph([]domain.LockedDependency{node}, nil)
		if err != nil {
			r.t.Fatal(err)
		}
	}
	resolution, err := domain.NewDependencyResolution(graph, identity, digest)
	if err != nil {
		r.t.Fatal(err)
	}
	return resolution.WithProjectControls(output)
}

func TestPyTorchUpgradeSessionBindsFrozenInputCompleteControlsAndResourceProfile(t *testing.T) {
	for _, name := range []string{"cpu", "cu126", "cu130", "cu132"} {
		for _, mode := range []string{"valid", "input", "constraints", "missing", "duplicate", "runtime", "digest", "graph", "metadata-drift", "no-context", "wrong-context"} {
			t.Run(name+"/"+mode, func(t *testing.T) {
				root, plan := makePypiVenvFixture(t)
				profile, _ := artifactpypi.PyTorchProfile(name)
				old := "1.0+" + name
				oldFiles := venvDestinations(t, map[string]string{"site/torch-" + old + ".dist-info/METADATA": "Name: torch\nVersion: " + old + "\n\n", "site/torch/__init__.py": "old payload"}, "torch")
				for i := range oldFiles {
					oldFiles[i].Version = old
				}
				if err := plan.commit("", oldFiles); err != nil {
					t.Fatal(err)
				}
				before, _, err := plan.readState()
				if err != nil {
					t.Fatal(err)
				}
				resolver := &namedUpdateSelectionFixture{t: t, profile: profile, mode: mode}
				promoter, _ := newPyPIPromotion(filepath.Join(root, "staging"), &venvGuardCheckingRunner{}, "linux", "amd64")
				preparation, _ := NewPyPIUpdatePreparation(promoter, resolver)
				target, _ := domain.NewInstallTarget(root)
				install, _ := domain.NewPythonVenvInstallContext(target)
				ctx := context.Background()
				if mode != "no-context" {
					contextProfile := profile
					if mode == "wrong-context" {
						contextProfile = artifactpypi.PublicPyPIProfile()
					}
					ctx, err = artifactpypi.ContextWithResourcePolicy(ctx, contextProfile)
					if err != nil {
						t.Fatal(err)
					}
				}
				session, err := preparation.BeginInstallUpdate(ctx, profile.Source(), "torch>=1,<3", install, false)
				if mode == "no-context" || mode == "wrong-context" {
					if err == nil || session != nil || resolver.calls != 0 {
						t.Fatal("missing root policy admitted", err)
					}
					if err := plan.verifyState(before); err != nil {
						t.Fatal(err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if second, err := beginPyPIVenvTransaction(plan); err == nil {
					_ = second.close()
					t.Fatal("selection released original guard")
				}
				metadataPath := filepath.Join(plan.site, "torch-"+old+".dist-info/METADATA")
				if mode == "metadata-drift" {
					if err := os.WriteFile(metadataPath, []byte("Name: torch\nVersion: "+old+"\nRequires-Dist: attacker\n\n"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				resolution, err := session.ResolveDependencies(ctx, session.Reference(), install)
				if (mode == "valid") != (err == nil) {
					t.Fatal("unbound named selection admitted", err)
				}
				if mode == "valid" && (len(resolution.Graph().Nodes()) != 1 || len(resolution.ProjectControls()) != 4 || !reflect.DeepEqual(resolution.Graph(), session.(*pypiUpdateSession).resolution.Graph())) {
					t.Fatal("complete bound selection lost")
				}
				if mode == "metadata-drift" && resolver.calls != 0 {
					t.Fatal("drift reached version selection")
				}
				if err := session.Close(); err != nil {
					t.Fatal(err)
				}
				if mode != "metadata-drift" {
					if err := plan.verifyState(before); err != nil {
						t.Fatal(err)
					}
				}
				payload, err := os.ReadFile(filepath.Join(plan.site, "torch/__init__.py"))
				if err != nil || !bytes.Equal(payload, []byte("old payload")) {
					t.Fatal("selection mutated old files", err)
				}
			})
		}
	}
}
