package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

// Own source-shaped data exercises the real traversal/control producer, without
// claiming live source authentication or CUDA runtime qualification.
type namedCUDAUpgradeRunner struct {
	profile           artifactpypi.SourceProfile
	project           string
	calls             []string
	inputs            [][]byte
	shared            bool
	sharedRequirement string
}

func (r *namedCUDAUpgradeRunner) Output(_ context.Context, _ string, args ...string) ([]byte, error) {
	joined := strings.Join(args, " ")
	r.calls = append(r.calls, joined)
	if strings.Contains(joined, "--report ") {
		for i, arg := range args {
			if arg == "--index-url" {
				request := args[len(args)-1]
				if args[i+2] != "--ignore-installed" && args[i+2] != "--constraint" {
					request = args[i+2]
				}
				end := strings.IndexAny(request, "[<>=~; ")
				if end < 0 {
					end = len(request)
				}
				r.project = request[:end]
				break
			}
		}
		if !strings.Contains(joined, pyTorchUpgradeReportScript) {
			return nil, nil
		}
	}
	version := "13.2"
	requirements := []string{}
	profile := artifactpypi.PublicPyPIProfile()
	switch r.project {
	case "torch":
		version = "2.14.0+" + strings.TrimPrefix(r.profile.Name(), "pytorch:")
		profile = r.profile
		requirements = []string{`cuda-toolkit[cudart]==13.2; platform_system == "Linux"`}
		if r.shared {
			bound := r.sharedRequirement
			if bound == "" {
				bound = ">=13.2"
			}
			requirements = append(requirements, "nvidia-cublas"+bound)
		}
	case "cuda-toolkit":
		requirements = []string{`nvidia-cublas>=13.2; extra == "cublas"`, `nvidia-cudart>=13.2; extra == "cudart"`}
	case "nvidia-cublas", "nvidia-cudart":
	default:
		return nil, errors.New("unexpected CUDA fixture project: " + r.project)
	}
	filename := strings.ReplaceAll(r.project, "-", "_") + "-" + version + "-py3-none-any.whl"
	url := "https://files.pythonhosted.org/packages/" + filename
	if r.project == "torch" {
		url = profile.IndexURL() + filename
	}
	if strings.Contains(joined, pypiResolverProjectDir+"/changes.json") || strings.Contains(joined, pypiResolverProjectDir+"/report.json") {
		runtime := PinnedPythonRuntime()
		install := []any{}
		if strings.Contains(joined, "--report "+pypiResolverProjectDir+"/report.json") || !strings.Contains(joined, pyTorchUpgradeReportScript) && strings.Contains(joined, "/report.json") {
			metadata := map[string]any{"name": r.project, "version": version, "requires_dist": requirements}
			if r.project == "torch" {
				metadata["requires_python"] = ">=3.14"
			}
			install = append(install, map[string]any{"requested": true, "download_info": map[string]any{"url": url, "archive_info": map[string]any{"hashes": map[string]string{"sha256": resolverTestSHA256}}}, "metadata": metadata})
		}
		return json.Marshal(map[string]any{"version": "1", "pip_version": runtime.PipVersion, "environment": map[string]string{"implementation_name": "cpython", "implementation_version": runtime.PythonVersion, "python_full_version": runtime.PythonVersion, "platform_machine": "x86_64", "sys_platform": "linux"}, "install": install})
	}
	if strings.Contains(joined, pytorchHTMLFetchScript) {
		return []byte(`<a href="` + url + `#sha256=` + resolverTestSHA256 + `" data-requires-python=">=3.14">wheel</a>`), nil
	}
	if strings.Contains(joined, simpleJSONFetchScript) {
		return []byte(pypiResolverSimpleJSON(r.project, filename, "")), nil
	}
	return nil, errors.New("unexpected CUDA fixture command")
}
func (r *namedCUDAUpgradeRunner) RunDiscard(ctx context.Context, binary string, args ...string) error {
	_, err := r.Output(ctx, binary, args...)
	return err
}
func (r *namedCUDAUpgradeRunner) RunInput(_ context.Context, input io.Reader, _ string, _ ...string) error {
	body, err := io.ReadAll(input)
	r.inputs = append(r.inputs, body)
	return err
}

func TestPyTorchUpgradeTraversalPreservesRetainedCUDAExtrasAndRealComponentEdges(t *testing.T) {
	for _, name := range []string{"cu130", "cu132"} {
		t.Run(name, func(t *testing.T) {
			profile, _ := artifactpypi.PyTorchProfile(name)
			requirement, _ := artifactpypi.ParseUpgradeRequirementForSource("torch", profile.Source())
			files := []artifactpypi.InstalledMetadataFile{}
			for _, item := range []struct{ name, version, requires string }{
				{"torch", "2.14.0+" + name, "Requires-Dist: cuda-toolkit[cudart]==13.2; platform_system == \"Linux\"\n"},
				{"retained", "1.0", "Requires-Dist: cuda-toolkit[cublas]==13.2; platform_system == \"Linux\"\n"},
				{"cuda-toolkit", "13.2", "Requires-Dist: nvidia-cublas>=13.2; extra == \"cublas\"\nRequires-Dist: nvidia-cudart>=13.2; extra == \"cudart\"\n"},
				{"nvidia-cublas", "13.2", ""}, {"nvidia-cudart", "13.2", ""},
			} {
				files = append(files, artifactpypi.InstalledMetadataFile{Directory: strings.ReplaceAll(item.name, "-", "_") + "-" + item.version + ".dist-info", Body: []byte("Name: " + item.name + "\nVersion: " + item.version + "\n" + item.requires + "\n")})
			}
			input, err := artifactpypi.NewUpgradeInput(requirement, files, PinnedPythonRuntime().PythonVersion)
			if err != nil {
				t.Fatal(err)
			}
			runner := &namedCUDAUpgradeRunner{profile: profile}
			resolver := &PyPIResolver{runner: runner, profile: profile}
			ctx, _ := artifactpypi.ContextWithResourcePolicy(context.Background(), profile)
			resolution, err := resolver.resolvePyTorchDependencyUpgrade(ctx, "0123456789abcdef", PinnedPythonRuntime(), requirement.Reference(), input)
			if err != nil || len(resolution.Graph().Nodes()) != 4 || len(resolution.Graph().Edges()) != 3 {
				t.Fatal("CUDA obligations lost in producer", err)
			}
			if len(runner.inputs) != 2 || !strings.Contains(string(runner.inputs[1]), "nvidia-cublas>=13.2\n") {
				t.Fatal("retained conditional constraints lost")
			}
			constrainedToolkit := false
			for _, call := range runner.calls {
				if strings.Contains(call, "--constraint ") && strings.Contains(call, `cuda-toolkit[cublas,cudart]==13.2; platform_system == "Linux"`) {
					constrainedToolkit = true
				}
			}
			if !constrainedToolkit {
				t.Fatal("final exact toolkit lookup lost retained extras")
			}
		})
	}
}

func TestPyTorchUpgradeSharedDependencyKeepsOneFreshLookupPerPhase(t *testing.T) {
	for _, scenario := range []struct {
		name, bound string
		calls       int
	}{{"identical", ">=13.2", 3}, {"changed", ">=13.1", 6}} {
		t.Run(scenario.name, func(t *testing.T) {
			profile, _ := artifactpypi.PyTorchProfile("cu132")
			requirement, _ := artifactpypi.ParseUpgradeRequirementForSource("torch", profile.Source())
			files := []artifactpypi.InstalledMetadataFile{}
			for _, item := range []struct{ name, version, requires string }{
				{"torch", "2.14.0+cu132", "Requires-Dist: cuda-toolkit[cudart]==13.2; platform_system == \"Linux\"\nRequires-Dist: nvidia-cublas>=13.2\n"},
				{"retained", "1.0", "Requires-Dist: cuda-toolkit[cublas]==13.2; platform_system == \"Linux\"\n"},
				{"cuda-toolkit", "13.2", "Requires-Dist: nvidia-cublas>=13.2; extra == \"cublas\"\nRequires-Dist: nvidia-cudart>=13.2; extra == \"cudart\"\n"},
				{"nvidia-cublas", "13.2", ""}, {"nvidia-cudart", "13.2", ""},
			} {
				files = append(files, artifactpypi.InstalledMetadataFile{Directory: strings.ReplaceAll(item.name, "-", "_") + "-" + item.version + ".dist-info", Body: []byte("Name: " + item.name + "\nVersion: " + item.version + "\n" + item.requires + "\n")})
			}
			input, err := artifactpypi.NewUpgradeInput(requirement, files, PinnedPythonRuntime().PythonVersion)
			if err != nil {
				t.Fatal(err)
			}
			runner := &namedCUDAUpgradeRunner{profile: profile, shared: true, sharedRequirement: scenario.bound}
			resolver := &PyPIResolver{runner: runner, profile: profile}
			ctx, _ := artifactpypi.ContextWithResourcePolicy(context.Background(), profile)
			resolution, err := resolver.resolvePyTorchDependencyUpgrade(ctx, "0123456789abcdef", PinnedPythonRuntime(), requirement.Reference(), input)
			if err != nil {
				t.Fatal(err)
			}
			if len(resolution.Graph().Nodes()) != 4 || len(resolution.Graph().Edges()) != 4 {
				t.Fatal("shared edge lost")
			}
			if _, err := input.ParseSourcePinnedUpgrade(resolution.ProjectControls(), PinnedPythonRuntime().PipVersion); err != nil {
				t.Fatal(err)
			}
			count := 0
			for _, call := range runner.calls {
				if strings.Contains(call, "--report ") && strings.Contains(call, "--index-url https://pypi.org/simple/ nvidia-cublas") || strings.Contains(call, "--report ") && strings.Contains(call, "nvidia-cublas==13.2") {
					count++
				}
			}
			if count != scenario.calls {
				t.Fatalf("shared dependency got %d pip selections; want %d; only identical intent may reuse a same-phase candidate", count, scenario.calls)
			}
		})
	}
}
