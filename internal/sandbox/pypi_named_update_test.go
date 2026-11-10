package sandbox

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

type namedUpgradeRunner struct {
	base   namedLocalGraphRunner
	noop   bool
	calls  [][]string
	inputs [][]byte
}

func (r *namedUpgradeRunner) Output(ctx context.Context, binary string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string(nil), args...))
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "--report "+pypiResolverProjectDir+"/changes.json") {
		for i, arg := range args {
			if arg == "--index-url" && i+2 < len(args) {
				request := args[i+2]
				end := strings.IndexAny(request, "[<>=~; ")
				if end < 0 {
					end = len(request)
				}
				r.base.project = request[:end]
				break
			}
		}
		if !strings.Contains(joined, pyTorchUpgradeReportScript) {
			return nil, nil
		}
	}
	if strings.Contains(joined, "--report "+pypiResolverProjectDir+"/changes.json") || strings.Contains(joined, pypiResolverProjectDir+"/changes.json") && !strings.Contains(joined, pyTorchUpgradeReportScript) {
		body, err := r.base.Output(ctx, binary, pypiResolverProjectDir+"/report.json")
		if err != nil || !r.noop {
			return body, err
		}
		var wire map[string]any
		if err := json.Unmarshal(body, &wire); err != nil {
			return nil, err
		}
		wire["install"] = []any{}
		return json.Marshal(wire)
	}
	if strings.Contains(joined, pyTorchUpgradeFreshScript) {
		if _, err := r.base.Output(ctx, binary, args...); err != nil {
			return nil, err
		}
		return r.base.Output(ctx, binary, pypiResolverProjectDir+"/report.json")
	}
	return r.base.Output(ctx, binary, args...)
}

func (r *namedUpgradeRunner) RunDiscard(ctx context.Context, binary string, args ...string) error {
	_, err := r.Output(ctx, binary, args...)
	return err
}
func (r *namedUpgradeRunner) RunInput(_ context.Context, input io.Reader, _ string, args ...string) error {
	body, err := io.ReadAll(input)
	if err != nil {
		return err
	}
	r.inputs = append(r.inputs, append([]byte(nil), body...))
	r.calls = append(r.calls, append([]string(nil), args...))
	return nil
}

func TestPyTorchUpgradeUsesInertOnlyIfNeededThenFreshExactConstrainedTraversal(t *testing.T) {
	for _, name := range []string{"cpu", "cu126", "cu130", "cu132"} {
		for _, mode := range []string{"changed", "noop"} {
			t.Run(name+"/"+mode, func(t *testing.T) {
				profile, _ := artifactpypi.PyTorchProfile(name)
				requirement, err := artifactpypi.ParseUpgradeRequirementForSource("torchvision>=1,<3", profile.Source())
				if err != nil {
					t.Fatal(err)
				}
				old := "1.0+" + name
				if mode == "noop" {
					old = "2.14.0+" + name
				}
				files := []artifactpypi.InstalledMetadataFile{{Directory: "torchvision-" + old + ".dist-info", Body: []byte("Name: torchvision\nVersion: " + old + "\nRequires-Dist: torch==2.14\n\n")}, {Directory: "torch-" + old + ".dist-info", Body: []byte("Name: torch\nVersion: " + old + "\n\n")}}
				input, err := artifactpypi.NewUpgradeInput(requirement, files, PinnedPythonRuntime().PythonVersion)
				if err != nil {
					t.Fatal(err)
				}
				runner := &namedUpgradeRunner{base: namedLocalGraphRunner{profile: profile}, noop: mode == "noop"}
				resolver := &PyPIResolver{runner: runner, profile: profile}
				ctx, err := artifactpypi.ContextWithResourcePolicy(context.Background(), profile)
				if err != nil {
					t.Fatal(err)
				}
				resolution, err := resolver.resolvePyTorchDependencyUpgrade(ctx, "0123456789abcdef", PinnedPythonRuntime(), requirement.Reference(), input)
				if err != nil || len(resolution.Graph().Nodes()) != 2 || len(resolution.ProjectControls()) != 4 || len(runner.inputs) != 2 {
					t.Fatal("named update did not preserve complete controls", resolution, err)
				}
				parsed, err := input.ParseSourcePinnedUpgrade(resolution.ProjectControls(), PinnedPythonRuntime().PipVersion)
				if err != nil || parsed.Digest() != resolution.LockfileDigest() {
					t.Fatal("runtime result does not bind emitted controls", err)
				}
				if !strings.Contains(string(runner.inputs[1]), "torch==2.14.0+"+name+"\n") {
					t.Fatal("proposed installed-version constraints lost")
				}
				rootSelections, dependencySelections, fresh, constrained := 0, 0, 0, 0
				for _, call := range runner.calls {
					text := strings.Join(call, " ")
					if strings.Contains(text, "--extra-index-url") {
						t.Fatal("multi-index selection returned")
					}
					if strings.Contains(text, "--report "+pypiResolverProjectDir+"/changes.json") {
						if !strings.Contains(text, "--no-deps") {
							t.Fatal("single-source version selection delegated dependencies")
						}
						if strings.Contains(text, pypiUpgradeMetadataScript) {
							rootSelections++
							if !strings.Contains(text, "--upgrade --upgrade-strategy only-if-needed") {
								t.Fatal("root upgrade intent lost")
							}
						}
						if strings.Contains(text, pyTorchUpgradeSelectionScript) {
							dependencySelections++
							if strings.Contains(text, "--upgrade") {
								t.Fatal("dependency became eager")
							}
						}
					}
					if strings.Contains(text, "--report "+pypiResolverProjectDir+"/report.json") {
						fresh++
						if !strings.Contains(text, "--ignore-installed") {
							t.Fatal("empty delta did not trigger fresh exact lookup")
						}
						if strings.Contains(text, "--constraint "+pypiResolverProjectDir+"/upgrade-constraints.txt") {
							constrained++
						}
					}
				}
				if rootSelections != 1 || dependencySelections != 1 || fresh != 4 || constrained != 2 {
					t.Fatal(rootSelections, dependencySelections, fresh, constrained)
				}
			})
		}
	}
}

func TestPyTorchUpgradeDoesNotLaunchSeparateReportReaders(t *testing.T) {
	profile, _ := artifactpypi.PyTorchProfile("cu126")
	requirement, _ := artifactpypi.ParseUpgradeRequirementForSource("torchvision", profile.Source())
	input, err := artifactpypi.NewUpgradeInput(requirement, []artifactpypi.InstalledMetadataFile{
		{Directory: "torchvision-2.14.0+cu126.dist-info", Body: []byte("Name: torchvision\nVersion: 2.14.0+cu126\nRequires-Dist: torch==2.14\n\n")},
		{Directory: "torch-2.14.0+cu126.dist-info", Body: []byte("Name: torch\nVersion: 2.14.0+cu126\n\n")},
	}, PinnedPythonRuntime().PythonVersion)
	if err != nil {
		t.Fatal(err)
	}
	runner := &namedUpgradeRunner{base: namedLocalGraphRunner{profile: profile}, noop: true}
	resolver := &PyPIResolver{runner: runner, profile: profile}
	ctx, _ := artifactpypi.ContextWithResourcePolicy(context.Background(), profile)
	resolution, err := resolver.resolvePyTorchDependencyUpgrade(ctx, "0123456789abcdef", PinnedPythonRuntime(), requirement.Reference(), input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := input.ParseSourcePinnedUpgrade(resolution.ProjectControls(), PinnedPythonRuntime().PipVersion); err != nil {
		t.Fatal(err)
	}
	readers := 0
	for _, call := range runner.calls {
		for _, arg := range call {
			if arg == boundedReadScript {
				readers++
			}
		}
	}
	if readers != 0 {
		t.Fatalf("named update launches %d separate report readers; reports must be returned by the successful pip invocation", readers)
	}
}
