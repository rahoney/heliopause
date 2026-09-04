package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPythonRuntimeLockMatchesPinnedIdentity(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile(filepath.Join("..", "..", "scripts", "runtimes.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		PythonImage struct {
			Reference     string `json:"reference"`
			PythonVersion string `json:"python_version"`
			PipVersion    string `json:"pip_version"`
			Target        struct {
				Interpreter string `json:"interpreter"`
				ABI         string `json:"abi"`
				Platform    string `json:"platform"`
			} `json:"target"`
		} `json:"python_image"`
	}
	if err := json.Unmarshal(body, &lock); err != nil {
		t.Fatal(err)
	}
	runtime := PinnedPythonRuntime()
	if lock.PythonImage.Reference != runtime.ImageReference || lock.PythonImage.PythonVersion != runtime.PythonVersion || lock.PythonImage.PipVersion != runtime.PipVersion || lock.PythonImage.Target.Interpreter != runtime.InterpreterTag || lock.PythonImage.Target.ABI != runtime.ABITag || lock.PythonImage.Target.Platform != runtime.PlatformTag {
		t.Fatalf("python runtime lock = %#v, pinned runtime = %#v", lock.PythonImage, runtime)
	}
}

func TestProbePython(t *testing.T) {
	t.Parallel()

	base := map[string]string{
		"docker version --format {{.Server.Version}}": "29.6.0",
		"runsc --version":      gVisorRelease,
		"runsc trace metadata": "Name: syscall/open_result\nName: sentry/mount_topology_snapshot\nName: sentry/mount_topology_mutation\n",
		"docker info --format {{json (index .Runtimes \"runsc-trace\")}}":    "{\"path\":\"/usr/local/bin/runsc\"}",
		"docker image inspect " + pythonImageReference + " --format {{.Id}}": "sha256:example",
	}
	tests := []struct {
		name            string
		operatingSystem string
		architecture    string
		executor        fakeExecutor
		available       bool
		limitation      string
	}{
		{name: "non Linux", operatingSystem: "darwin", architecture: "amd64", limitation: "M5_PYPI_LINUX_AMD64_ONLY"},
		{name: "non amd64", operatingSystem: "linux", architecture: "arm64", limitation: "M5_PYPI_LINUX_AMD64_ONLY"},
		{name: "missing runtime", operatingSystem: "linux", architecture: "amd64", executor: fakeExecutor{lookupError: errors.New("missing")}, limitation: "M5_PYPI_RUNTIME_UNAVAILABLE"},
		{name: "old Docker", operatingSystem: "linux", architecture: "amd64", executor: fakeExecutor{outputs: map[string]string{"docker version --format {{.Server.Version}}": "29.5.3"}}, limitation: "M5_PYPI_RUNTIME_VERSION_UNSUPPORTED"},
		{name: "missing image", operatingSystem: "linux", architecture: "amd64", executor: fakeExecutor{outputs: map[string]string{"docker version --format {{.Server.Version}}": "29.6.0", "runsc --version": gVisorRelease, "runsc trace metadata": "Name: syscall/open_result\nName: sentry/mount_topology_snapshot\nName: sentry/mount_topology_mutation\n", "docker info --format {{json (index .Runtimes \"runsc-trace\")}}": "{\"path\":\"/usr/local/bin/runsc\"}"}}, limitation: "M5_PYPI_IMAGE_UNAVAILABLE"},
		{name: "available", operatingSystem: "linux", architecture: "amd64", executor: fakeExecutor{outputs: base}, available: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			capability, err := probePython(context.Background(), test.operatingSystem, test.architecture, test.executor)
			if err != nil || capability.Available != test.available || capability.LimitationCode != test.limitation {
				t.Fatalf("probePython() = %#v, %v", capability, err)
			}
			if capability.Runtime != PinnedPythonRuntime() {
				t.Fatalf("capability runtime = %#v", capability.Runtime)
			}
		})
	}
}

func TestProbePythonPreservesContextFailure(t *testing.T) {
	t.Parallel()

	if _, err := probePython(absentPythonContext(), "linux", "amd64", fakeExecutor{}); err == nil {
		t.Fatal("probePython(nil) error = nil")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := probePython(ctx, "linux", "amd64", fakeExecutor{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("probePython(cancelled) error = %v", err)
	}
}

func absentPythonContext() context.Context { return nil }
