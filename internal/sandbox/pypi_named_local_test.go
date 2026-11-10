package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

// This two-node source-shaped fixture exercises the actual traversal. It is
// not live source authentication, Docker execution or artifact qualification.
type namedLocalGraphRunner struct {
	profile artifactpypi.SourceProfile
	project string
}

func (r *namedLocalGraphRunner) Output(_ context.Context, _ string, arguments ...string) ([]byte, error) {
	joined := strings.Join(arguments, " ")
	if strings.Contains(joined, "--report ") {
		request := arguments[len(arguments)-1]
		end := strings.IndexAny(request, "[<>=~; ")
		if end < 0 {
			end = len(request)
		}
		r.project = request[:end]
		if r.project != "torchvision" && r.project != "torch" {
			return nil, fmt.Errorf("unexpected fixture project %q", r.project)
		}
		return nil, nil
	}
	version := "2.14.0+" + strings.TrimPrefix(r.profile.Name(), "pytorch:")
	filename := r.project + "-" + version + "-cp314-cp314-manylinux_2_28_x86_64.whl"
	url := r.profile.IndexURL() + filename
	digest := strings.Repeat("a", 64)
	if strings.Contains(joined, pypiResolverProjectDir+"/report.json") {
		runtime := PinnedPythonRuntime()
		requirements := []string{}
		if r.project == "torchvision" {
			requirements = []string{"torch==2.14"}
		}
		return json.Marshal(map[string]any{"version": "1", "pip_version": runtime.PipVersion, "environment": map[string]string{"implementation_name": "cpython", "implementation_version": runtime.PythonVersion, "python_full_version": runtime.PythonVersion, "platform_machine": "x86_64", "sys_platform": "linux"}, "install": []any{map[string]any{"download_info": map[string]any{"url": url, "archive_info": map[string]any{"hashes": map[string]string{"sha256": digest}}}, "is_direct": false, "requested": true, "metadata": map[string]any{"name": r.project, "version": version, "requires_python": ">=3.14", "requires_dist": requirements}}}})
	}
	if strings.Contains(joined, pytorchHTMLFetchScript) {
		return []byte(`<a href="` + url + `#sha256=` + digest + `" data-requires-python=">=3.14">wheel</a>`), nil
	}
	return nil, errors.New("unexpected source-shaped fixture command")
}

func (r *namedLocalGraphRunner) RunDiscard(ctx context.Context, binary string, args ...string) error {
	_, err := r.Output(ctx, binary, args...)
	return err
}

func TestPyTorchTraversalAcceptsSourceBoundNamedLocalDependency(t *testing.T) {
	for _, name := range []string{"cpu", "cu126", "cu130", "cu132"} {
		t.Run(name, func(t *testing.T) {
			profile, _ := artifactpypi.PyTorchProfile(name)
			runner := &namedLocalGraphRunner{profile: profile}
			resolver := &PyPIResolver{runner: runner, profile: profile}
			reference, _ := artifactpypi.ParseReferenceForSource("torchvision", profile.Source())
			candidates, _, err := resolver.resolvePyTorchGraph(context.Background(), "0123456789abcdef", PinnedPythonRuntime(), reference)
			if err != nil {
				t.Fatal(err)
			}
			graph, err := artifactpypi.BuildLockedGraph(reference, candidates)
			if err != nil || len(graph.Nodes()) != 2 || len(graph.Edges()) != 1 {
				t.Fatal("incoming named local dependency lost", graph, err)
			}
			for _, node := range graph.Nodes() {
				if node.Artifact().Identity().Source() != profile.Source() {
					t.Fatal("named source changed")
				}
			}
		})
	}
}
