package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

// TestPyTorchStaticReplayAllProfiles deterministically replays the production
// PyTorch graph loop. It uses fixed source-shaped pip reports and Simple pages
// to validate parser, aggregation, metadata cross-check, edge, and traversal
// behavior deterministically. It does not authenticate live index metadata,
// use a network, run Docker lifecycle, or perform privileged qualification.
func TestPyTorchStaticReplayAllProfiles(t *testing.T) {
	for _, name := range []string{"cpu", "cu126", "cu130", "cu132"} {
		t.Run(name, func(t *testing.T) {
			profile, ok := artifactpypi.PyTorchProfile(name)
			if !ok {
				t.Fatal("profile is missing")
			}
			runner := &pyTorchReplayRunner{profile: profile}
			resolver := &PyPIResolver{runner: runner, profile: profile}
			reference, err := artifactpypi.ParseReferenceForSource("torch@2.14.0+"+name, profile.Source())
			if err != nil {
				t.Fatal(err)
			}
			candidates, _, err := resolver.resolvePyTorchGraph(context.Background(), "0123456789abcdef", PinnedPythonRuntime(), reference)
			if err != nil {
				t.Fatal(err)
			}
			graph, err := artifactpypi.BuildLockedGraph(reference, candidates)
			if err != nil {
				t.Fatal(err)
			}
			wantMetadataURL := "https://download-r2.pytorch.org/whl/" + name + "/torch-2.14.0%2B" + name + "-cp314-cp314-manylinux_2_28_x86_64.whl.metadata"
			if runner.requestedMetadataURL != wantMetadataURL {
				t.Fatalf("Core Metadata sidecar URL = %q, want %q", runner.requestedMetadataURL, wantMetadataURL)
			}
			if name == "cpu" {
				if len(graph.Nodes()) != 10 || len(graph.Edges()) != 9 {
					t.Fatalf("CPU static graph = %d nodes / %d edges", len(graph.Nodes()), len(graph.Edges()))
				}
				return
			}
			wantEdges := 41
			if name == "cu126" {
				wantEdges = 34
			}
			if len(graph.Nodes()) != 29 || len(graph.Edges()) != wantEdges {
				t.Fatalf("CUDA static graph = %d nodes / %d edges", len(graph.Nodes()), len(graph.Edges()))
			}
			if !runner.sawExtraRequest {
				t.Fatal("cuda-toolkit extras were not retained in the production pip request")
			}
		})
	}
}

func TestPyTorchTraversalQueueBoundFailsClosed(t *testing.T) {
	profile, _ := artifactpypi.PyTorchProfile("cu126")
	runner := &pyTorchReplayRunner{profile: profile, fanout: maxPyTorchGraphEdges + 1}
	resolver := &PyPIResolver{runner: runner, profile: profile}
	reference, _ := artifactpypi.ParseReferenceForSource("torch@2.14.0+cu126", profile.Source())
	if _, _, err := resolver.resolvePyTorchGraph(context.Background(), "0123456789abcdef", PinnedPythonRuntime(), reference); err == nil {
		t.Fatal("resolver accepted dependency queue beyond the traversal edge bound")
	}
}

func TestPyTorchTraversalProjectAndCumulativeReportBoundsFailClosed(t *testing.T) {
	profile, _ := artifactpypi.PyTorchProfile("cu126")
	reference, _ := artifactpypi.ParseReferenceForSource("torch@2.14.0+cu126", profile.Source())
	for _, test := range []struct {
		name   string
		runner pyTorchReplayRunner
	}{
		{"projects", pyTorchReplayRunner{profile: profile, fanout: maxPyTorchReports}},
		{"cumulative reports", pyTorchReplayRunner{profile: profile, chain: 6, paddingBytes: 3 << 20}},
	} {
		t.Run(test.name, func(t *testing.T) {
			resolver := &PyPIResolver{runner: &test.runner, profile: profile}
			if _, _, err := resolver.resolvePyTorchGraph(context.Background(), "0123456789abcdef", PinnedPythonRuntime(), reference); err == nil {
				t.Fatal("resolver accepted a traversal beyond its early resource bound")
			}
		})
	}
}

type pyTorchReplayRunner struct {
	profile              artifactpypi.SourceProfile
	request              string
	fanout               int
	chain                int
	paddingBytes         int
	sawExtraRequest      bool
	requestedMetadataURL string
	overrideSimple       string
	overrideMetadataBody []byte
}

func (r *pyTorchReplayRunner) Output(_ context.Context, _ string, arguments ...string) ([]byte, error) {
	joined := strings.Join(arguments, " ")
	if strings.Contains(joined, "--report ") {
		r.request = arguments[len(arguments)-1]
		r.sawExtraRequest = r.sawExtraRequest || strings.HasPrefix(r.request, "cuda-toolkit[cublas,cudart]")
		return nil, nil
	}
	if strings.Contains(joined, pypiResolverProjectDir+"/report.json") {
		return []byte(r.report()), nil
	}
	if strings.Contains(joined, pytorchCoreMetadataFetchScript) {
		requestedURL := arguments[len(arguments)-1]
		r.requestedMetadataURL = requestedURL
		if requestedURL != r.expectedMetadataURL() {
			return nil, fmt.Errorf("unexpected metadata sidecar URL requested: %s, want: %s", requestedURL, r.expectedMetadataURL())
		}
		return r.pytorchCoreMetadata(), nil
	}
	if strings.Contains(joined, r.profile.IndexURL()) {
		return []byte(r.pytorchSimple()), nil
	}
	return []byte(pypiResolverSimpleJSON(r.project(), r.filename(), "")), nil
}

func (r *pyTorchReplayRunner) RunDiscard(ctx context.Context, binary string, arguments ...string) error {
	_, err := r.Output(ctx, binary, arguments...)
	return err
}

func (r *pyTorchReplayRunner) project() string {
	switch {
	case strings.HasPrefix(r.request, "torch"):
		return "torch"
	default:
		end := strings.IndexAny(r.request, "[<>=; ")
		if end < 0 {
			end = len(r.request)
		}
		return r.request[:end]
	}
}

func (r *pyTorchReplayRunner) filename() string {
	project := r.project()
	if project == "torch" {
		profileSlug := strings.TrimPrefix(r.profile.Name(), "pytorch:")
		return "torch-2.14.0+" + profileSlug + "-cp314-cp314-manylinux_2_28_x86_64.whl"
	}
	return project + "-1.0.0-py3-none-any.whl"
}

func (r *pyTorchReplayRunner) expectedWheelURL() string {
	profileSlug := strings.TrimPrefix(r.profile.Name(), "pytorch:")
	filename := r.filename()
	return "https://download-r2.pytorch.org/whl/" + profileSlug + "/" + strings.ReplaceAll(filename, "+", "%2B")
}

func (r *pyTorchReplayRunner) expectedMetadataURL() string {
	return r.expectedWheelURL() + ".metadata"
}

func (r *pyTorchReplayRunner) report() string {
	project, filename := r.project(), r.filename()
	version, url, requirements := "1.0.0", "https://files.pythonhosted.org/packages/"+filename, []string{}
	if project == "torch" {
		version = "2.14.0+" + strings.TrimPrefix(r.profile.Name(), "pytorch:")
		url = r.expectedWheelURL()
		if r.fanout > 0 {
			for i := 0; i < r.fanout; i++ {
				requirements = append(requirements, fmt.Sprintf("shared%d", i))
			}
		} else if r.chain > 0 {
			requirements = []string{"shared1"}
		} else if r.profile.Name() == "pytorch:cpu" {
			for i := 1; i <= 9; i++ {
				requirements = append(requirements, fmt.Sprintf("shared%d==1.0.0.*", i))
			}
		} else {
			requirements = []string{"nvidia-cublas", `cuda-toolkit[cublas,cudart]==1.0.0; platform_system == "Linux"`}
			for i := 1; i <= 26; i++ {
				requirements = append(requirements, fmt.Sprintf("shared%d==1.0.0.*", i))
			}
		}
	} else if project == "cuda-toolkit" {
		requirements = []string{`nvidia-cublas==1.0.0.*; extra != "cublas"`}
	} else if project == "shared1" && r.profile.Name() != "pytorch:cpu" {
		extraEdges := 12
		if r.profile.Name() == "pytorch:cu126" {
			extraEdges = 5
		}
		for i := 2; i <= extraEdges+1; i++ {
			requirements = append(requirements, fmt.Sprintf("shared%d", i))
		}
	}
	if r.chain > 0 && strings.HasPrefix(project, "shared") {
		var number int
		if _, err := fmt.Sscanf(project, "shared%d", &number); err == nil && number < r.chain {
			requirements = []string{fmt.Sprintf("shared%d", number+1)}
		}
	}
	requiresPython := ""
	if project == "torch" {
		requiresPython = ">=3.10"
	}
	report := `{"version":"1","pip_version":"26.2.1","environment":{"implementation_name":"cpython","implementation_version":"3.14.7","python_full_version":"3.14.7","platform_machine":"x86_64","sys_platform":"linux"},"install":[{"download_info":{"url":"` + url + `","archive_info":{"hashes":{"sha256":"` + resolverTestSHA256 + `"}}},"is_direct":false,"requested":true,"metadata":{"name":"` + project + `","version":"` + version + `","requires_python":"` + requiresPython + `","requires_dist":` + quotedRequirements(requirements) + `}}]`
	if r.paddingBytes > 0 {
		report += `,"padding":"` + strings.Repeat("x", r.paddingBytes) + `"`
	}
	return report + `}`
}

func (r *pyTorchReplayRunner) pytorchSimple() string {
	if r.overrideSimple != "" {
		return r.overrideSimple
	}
	wheelURL := r.expectedWheelURL()
	coreSHA := r.pytorchCoreMetadataSHA256()
	return `<a href="` + wheelURL + `#sha256=` + resolverTestSHA256 + `" data-core-metadata="sha256=` + coreSHA + `" data-dist-info-metadata="sha256=` + coreSHA + `">torch</a>`
}

func (r *pyTorchReplayRunner) pytorchCoreMetadata() []byte {
	if len(r.overrideMetadataBody) > 0 {
		return r.overrideMetadataBody
	}
	return r.canonicalCoreMetadata()
}

func (r *pyTorchReplayRunner) canonicalCoreMetadata() []byte {
	return []byte("Metadata-Version: 2.4\nName: torch\nVersion: 2.14.0+" + strings.TrimPrefix(r.profile.Name(), "pytorch:") + "\nRequires-Python: >=3.10\n\n")
}

func (r *pyTorchReplayRunner) pytorchCoreMetadataSHA256() string {
	sum := sha256.Sum256(r.canonicalCoreMetadata())
	return hex.EncodeToString(sum[:])
}

func quotedRequirements(requirements []string) string {
	quoted := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		quoted = append(quoted, fmt.Sprintf("%q", requirement))
	}
	return "[" + strings.Join(quoted, ",") + "]"
}

func TestPyTorchReplayAdversarialChecks(t *testing.T) {
	profile, ok := artifactpypi.PyTorchProfile("cpu")
	if !ok {
		t.Fatal("CPU profile is missing")
	}
	reference, err := artifactpypi.ParseReferenceForSource("torch@2.14.0+cpu", profile.Source())
	if err != nil {
		t.Fatal(err)
	}

	t.Run("wrong sidecar URL is rejected by replay runner", func(t *testing.T) {
		runner := &pyTorchReplayRunner{profile: profile}
		_, err := runner.Output(context.Background(), "docker", append(boundaryExecArguments("dummy", boundaryLaunchMode, "python", "-I", "-c", pytorchCoreMetadataFetchScript), "https://download-r2.pytorch.org/whl/cpu/wrong.whl.metadata")...)
		if err == nil {
			t.Fatal("expected runner to reject non-bound metadata sidecar URL")
		}
	})

	t.Run("wrong selected link host is rejected", func(t *testing.T) {
		runner := &pyTorchReplayRunner{
			profile:        profile,
			overrideSimple: `<a href="https://evil-untrusted.org/whl/cpu/torch-2.14.0%2Bcpu-cp314-cp314-manylinux_2_28_x86_64.whl#sha256=` + resolverTestSHA256 + `" data-core-metadata="sha256=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa">torch</a>`,
		}
		resolver := &PyPIResolver{runner: runner, profile: profile}
		if _, _, err := resolver.resolvePyTorchGraph(context.Background(), "0123456789abcdef", PinnedPythonRuntime(), reference); err == nil {
			t.Fatal("expected resolution to fail when Simple link host is not trusted")
		}
	})

	t.Run("wrong selected link profile path is rejected", func(t *testing.T) {
		runner := &pyTorchReplayRunner{
			profile:        profile,
			overrideSimple: `<a href="https://download-r2.pytorch.org/whl/cu126/torch-2.14.0%2Bcu126-cp314-cp314-manylinux_2_28_x86_64.whl#sha256=` + resolverTestSHA256 + `" data-core-metadata="sha256=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa">torch</a>`,
		}
		resolver := &PyPIResolver{runner: runner, profile: profile}
		if _, _, err := resolver.resolvePyTorchGraph(context.Background(), "0123456789abcdef", PinnedPythonRuntime(), reference); err == nil {
			t.Fatal("expected resolution to fail when Simple link is outside profile path")
		}
	})

	t.Run("mismatched sidecar hash fails verification", func(t *testing.T) {
		runner := &pyTorchReplayRunner{
			profile:              profile,
			overrideMetadataBody: []byte("Metadata-Version: 2.4\nName: torch\nVersion: 2.14.0+cpu\nRequires-Python: >=3.10\n\nTAMPERED"),
		}
		resolver := &PyPIResolver{runner: runner, profile: profile}
		if _, _, err := resolver.resolvePyTorchGraph(context.Background(), "0123456789abcdef", PinnedPythonRuntime(), reference); err == nil {
			t.Fatal("expected resolution to fail when Core Metadata body hash does not match Simple link attribute")
		}
	})
}
