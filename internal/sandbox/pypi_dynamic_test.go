package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestPythonDynamicBackendRunsOnlyLocalWheelAndDeclaredImports(t *testing.T) {
	root, artifact := pythonWheelFixture(t)
	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil, nil, nil, nil}}
	introducer, err := NewPythonArtifactIntroducer(root, runner)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := NewPythonDynamicBackend(runner, introducer, &recordingObserver{reader: &traceReader{}}, availablePythonProbe)
	if err != nil {
		t.Fatal(err)
	}
	result, err := backend.InspectWheel(context.Background(), artifact, []string{"example"})
	if err != nil || result.Status() != domain.SandboxCompleted {
		t.Fatalf("InspectWheel() = %#v, %v", result, err)
	}
	if len(runner.inputCalls) != 1 || string(runner.input) != "wheel fixture" {
		t.Fatalf("wheel stream = %#v/%q", runner.inputCalls, runner.input)
	}
	if len(runner.calls) != 5 {
		t.Fatalf("commands = %#v", runner.calls)
	}
	assertPythonDynamicCreate(t, runner.calls[0].arguments)
	if !strings.Contains(strings.Join(runner.calls[2].arguments, " "), "--no-index --no-deps --no-compile") || !strings.Contains(strings.Join(runner.calls[2].arguments, " "), pythonWheelPath(artifact)) {
		t.Fatalf("pip install command = %#v", runner.calls[2])
	}
	if !sameStrings(runner.calls[3].arguments[len(runner.calls[3].arguments)-1:], []string{"example"}) {
		t.Fatalf("import command = %#v", runner.calls[3])
	}
}

func TestPythonDynamicBackendFailsClosedForIncompleteObservation(t *testing.T) {
	root, artifact := pythonWheelFixture(t)
	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil, nil, nil, nil}}
	introducer, _ := NewPythonArtifactIntroducer(root, runner)
	backend, _ := NewPythonDynamicBackend(runner, introducer, &recordingObserver{reader: &traceReader{err: os.ErrClosed}}, availablePythonProbe)
	result, err := backend.InspectWheel(context.Background(), artifact, []string{"example"})
	if err != nil || result.Status() != domain.SandboxIncomplete {
		t.Fatalf("result = %#v, %v", result, err)
	}
	if limitation, _ := result.LimitationCode(); limitation != "M5_PYPI_DYNAMIC_OBSERVATION_INCOMPLETE" {
		t.Fatalf("limitation = %q", limitation)
	}
}

func TestPythonDynamicBackendUsesNamedRootProfileResources(t *testing.T) {
	root, artifact := pythonWheelFixture(t)
	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil, nil, nil, nil}}
	introducer, _ := NewPythonArtifactIntroducer(root, runner)
	backend, _ := NewPythonDynamicBackend(runner, introducer, &recordingObserver{reader: &traceReader{}}, availablePythonProbe)
	profile, _ := artifactpypi.PyTorchProfile("cpu")
	ctx, err := artifactpypi.ContextWithResourcePolicy(context.Background(), profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := backend.InspectWheel(ctx, artifact, []string{"example"}); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(runner.calls[0].arguments, " ")
	if !strings.Contains(joined, "--memory 2147483648") || !strings.Contains(joined, "size=2147483648") {
		t.Fatalf("CPU resource policy did not reach dynamic sandbox: %q", joined)
	}
}

func TestPythonDynamicBackendRejectsEmptyOrArbitraryImportSurface(t *testing.T) {
	root, artifact := pythonWheelFixture(t)
	introducer, _ := NewPythonArtifactIntroducer(root, &recordingRunner{})
	backend, _ := NewPythonDynamicBackend(&recordingRunner{}, introducer, &emptyObserver{}, availablePythonProbe)
	if _, err := backend.InspectWheel(context.Background(), artifact, []string{}); err == nil {
		t.Fatal("empty imports accepted")
	}
	if _, err := backend.InspectWheel(context.Background(), artifact, []string{"example;os.system('x')"}); err == nil {
		t.Fatal("arbitrary import accepted")
	}
}

func pythonWheelFixture(t *testing.T) (string, domain.AcquiredArtifact) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa", "wheel.whl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("wheel fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	source, _ := domain.NewSourceID("pypi")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "example", "1.0", "wheel")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:wheel", uint64(len("wheel fixture")))
	if err != nil {
		t.Fatal(err)
	}
	return root, artifact
}

func assertPythonDynamicCreate(t *testing.T, arguments []string) {
	t.Helper()
	joined := strings.Join(arguments, " ")
	for _, required := range []string{"--pull never", "--runtime " + gVisorRuntimeName, "--network none", "--read-only", "--cap-drop ALL", "no-new-privileges", "--pids-limit 64", "--memory 536870912", pythonImageReference} {
		if !strings.Contains(joined, required) {
			t.Errorf("create command missing %q: %q", required, joined)
		}
	}
	for _, forbidden := range []string{"--mount", "--volume", "-v ", "--env", "--privileged", "--network host", "/var/run/docker.sock"} {
		if strings.Contains(joined, forbidden) {
			t.Errorf("create command contains forbidden %q: %q", forbidden, joined)
		}
	}
}
