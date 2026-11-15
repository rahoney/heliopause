package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type requirementsRequestRecorder struct {
	requests []PythonRequirementsRequest
	outcome  PythonRequirementsOutcome
}

func (r *requirementsRequestRecorder) InstallRequirements(_ context.Context, request PythonRequirementsRequest) (PythonRequirementsOutcome, error) {
	r.requests = append(r.requests, request)
	return r.outcome, nil
}

func TestRequirementsCLICollectsFilesAndRootsOnceAcrossPreflightAndExecute(t *testing.T) {
	for _, client := range []PackageClientID{PipPackageClient, UVPipPackageClient} {
		t.Run(string(client), func(t *testing.T) {
			directory := t.TempDir()
			files := []string{}
			for _, name := range []string{"first.txt", "second.txt", "constraints.txt"} {
				path := filepath.Join(directory, name)
				if err := os.WriteFile(path, []byte("alpha>=1\n"), 0600); err != nil {
					t.Fatal(err)
				}
				files = append(files, path)
			}
			var output bytes.Buffer
			root, err := New(&output, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			legacy := &uvClientRequestRecorder{}
			if client == PipPackageClient {
				err = AddPyPIInstallSourcesAndUpdates(root, map[string]Installer{"pypi": legacy}, nil)
			} else {
				err = AddUVPipInstall(root, legacy)
			}
			if err != nil {
				t.Fatal(err)
			}
			recorder := &requirementsRequestRecorder{outcome: PythonRequirementsOutcome{Noop: true}}
			if err := AddPythonRequirementsInstall(root, client, recorder); err != nil {
				t.Fatal(err)
			}
			prefix := []string{"pip", "install"}
			if client == UVPipPackageClient {
				prefix = []string{"uv", "pip", "install"}
			}
			args := append([]string{"--output", "json", "--lang", "ko"}, prefix...)
			args = append(args, "-r", files[0], "--requirement="+files[1], "-c", files[2], "--target", "/tmp/requirements-cli", "--require-hashes", "--upgrade", "--adopt", "alpha>=1", "beta[socks]; sys_platform == 'linux'")
			if _, err := PreparePresentation(root, args); err != nil {
				t.Fatal(err)
			}
			root.SetArgs(args)
			if err := root.ExecuteContext(context.Background()); err != nil {
				t.Fatal(err, output.String())
			}
			if len(recorder.requests) != 1 || len(legacy.requests) != 0 {
				t.Fatal("batch request split or entered legacy path")
			}
			request := recorder.requests[0].Requirements
			if len(request.Sources) != 2 || len(request.Constraints) != 1 || len(request.Positional) != 2 || !request.Upgrade || !request.Adopt || !request.RequireHashes {
				t.Fatal("flags accumulated or intent was lost", request)
			}
			var report struct {
				Schema string `json:"schema_version"`
			}
			if json.Unmarshal(output.Bytes(), &report) != nil || report.Schema != "helox.project-install-result/v1" {
				t.Fatal("no-op changed the install result schema", output.String())
			}
			if !strings.Contains(output.String(), "SKIPPED") || strings.Contains(output.String(), "ALLOW") {
				t.Fatal("no-op was reported as inspection approval", output.String())
			}
		})
	}
}

func TestRequirementsCLIExpandedGrammarKeepsExistingSingleRootPaths(t *testing.T) {
	for _, client := range []PackageClientID{PipPackageClient, UVPipPackageClient} {
		for _, arguments := range [][]string{{"alpha", "beta"}, {"alpha>=1,<3"}, {"alpha[socks]"}, {"alpha; sys_platform == 'linux'"}, {"-U", "alpha", "beta"}, {"--upgrade", "alpha[socks]"}} {
			root, _ := New(io.Discard, io.Discard)
			legacy := &uvClientRequestRecorder{}
			if client == PipPackageClient {
				_ = AddPyPIInstallSourcesAndUpdates(root, map[string]Installer{"pypi": legacy}, nil)
			} else {
				_ = AddUVPipInstall(root, legacy)
			}
			recorder := &requirementsRequestRecorder{outcome: PythonRequirementsOutcome{Noop: true}}
			_ = AddPythonRequirementsInstall(root, client, recorder)
			prefix := []string{"pip", "install"}
			if client == UVPipPackageClient {
				prefix = []string{"uv", "pip", "install"}
			}
			args := append(prefix, arguments...)
			args = append(args, "--target", "/tmp/requirements-cli")
			if _, err := PreparePresentation(root, args); err != nil {
				t.Fatal(arguments, err)
			}
			root.SetArgs(args)
			if err := root.ExecuteContext(context.Background()); err != nil {
				t.Fatal(arguments, err)
			}
			if len(recorder.requests) != 1 || len(legacy.requests) != 0 {
				t.Fatal("expanded grammar did not use one complete selection")
			}
		}
	}
}

func TestRequirementsCLIRejectsMissingFilesAndUnsupportedSourcesBeforeService(t *testing.T) {
	for _, tail := range [][]string{{"--requirement="}, {"-r", "/nonexistent/requirements.txt"}, {"alpha", "beta", "--source", "pytorch:cpu"}, {"alpha @ https://evil.invalid/a.whl"}, {"-c", ""}} {
		root, _ := New(io.Discard, io.Discard)
		if _, err := PreparePresentation(root, append([]string{"pip", "install", "--target", "/tmp/requirements-cli"}, tail...)); err == nil {
			t.Fatal("unsupported request passed preflight", tail)
		}
	}
}
