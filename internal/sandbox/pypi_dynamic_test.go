package sandbox

import (
	"context"
	"errors"
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

func TestPythonDynamicBackendInstallsExactClosureInOneOfflineInvocation(t *testing.T) {
	root, target := pythonWheelFixture(t)
	dependencyRunID := "run_" + strings.Repeat("a", 25) + "i"
	dependencyRun := filepath.Join(root, dependencyRunID)
	if err := os.MkdirAll(dependencyRun, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dependencyRun, "wheel.whl"), []byte("dependency wheel"), 0o600); err != nil {
		t.Fatal(err)
	}
	source, _ := domain.NewSourceID("pypi")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "dependency", "1.0", "wheel")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("b", 64))
	dependency, err := domain.NewAcquiredArtifact(identity, digest, "intake:"+dependencyRunID+":wheel", uint64(len("dependency wheel")))
	if err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil, nil, nil, nil}}
	introducer, err := NewPythonArtifactIntroducer(root, runner)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := NewPythonDynamicBackend(runner, introducer, &recordingObserver{reader: &traceReader{}}, availablePythonProbe)
	if err != nil {
		t.Fatal(err)
	}
	result, err := backend.InspectWheelWithClosure(context.Background(), target, []string{"example"}, []domain.AcquiredArtifact{target, dependency})
	if err != nil || result.Status() != domain.SandboxCompleted {
		t.Fatalf("InspectWheelWithClosure() = %#v, %v", result, err)
	}
	if len(runner.inputCalls) != 2 {
		t.Fatalf("introduced closure artifacts = %#v", runner.inputCalls)
	}
	if len(runner.calls) < 3 || !strings.Contains(strings.Join(runner.calls[2].arguments, " "), "--no-index --no-deps") || !strings.Contains(strings.Join(runner.calls[2].arguments, " "), pythonWheelPath(target)) || !strings.Contains(strings.Join(runner.calls[2].arguments, " "), pythonWheelPath(dependency)) {
		t.Fatalf("closure install command = %#v", runner.calls)
	}
}

func TestPythonDynamicBackendClassifiesBoundedInstallFailureWithoutExposingOutput(t *testing.T) {
	root, artifact := pythonWheelFixture(t)
	runner := &recordingRunner{
		responses:     [][]byte{[]byte("0123456789abcdef")},
		errors:        []error{nil, nil, errors.New("exit status 1")},
		boundedOutput: []byte("ERROR: package installation failed: No space left on device"),
	}
	introducer, err := NewPythonArtifactIntroducer(root, runner)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := NewPythonDynamicBackend(runner, introducer, &recordingObserver{reader: &traceReader{}}, availablePythonProbe)
	if err != nil {
		t.Fatal(err)
	}
	result, err := backend.InspectWheel(context.Background(), artifact, []string{"example"})
	if err != nil {
		t.Fatal(err)
	}
	limitation, _ := result.LimitationCode()
	if result.Status() != domain.SandboxIncomplete || limitation != "M5_PYPI_DYNAMIC_INSTALL_FAILED_ENOSPC" || strings.Contains(limitation, "No space") {
		t.Fatalf("result = %#v", result)
	}
}

func TestClassifyDynamicInstallFailureUsesBoundedVocabulary(t *testing.T) {
	for _, test := range []struct {
		output string
		want   string
	}{
		{"ERROR: invalid requirement", dynamicInstallFailurePipArgument},
		{"ERROR: not a supported wheel on this platform", dynamicInstallFailureWheelPlatform},
		{"ERROR: invalid wheel metadata", dynamicInstallFailureWheelMetadata},
		{"ERROR: ResolutionImpossible: conflicting dependencies", dynamicInstallFailurePackageConflict},
		{"ERROR: duplicate distribution", dynamicInstallFailureDuplicate},
		{"ERROR: No space left on device", dynamicInstallFailureENOSPC},
		{"ERROR: Cannot allocate memory", dynamicInstallFailureMemory},
		{"ERROR: Permission denied", dynamicInstallFailurePermission},
		{"ERROR: OCI runtime runsc failure", dynamicInstallFailureSandboxRuntime},
		{"unrecognized", dynamicInstallFailureOther},
	} {
		if got := classifyDynamicInstallFailure(context.Background(), test.output); got != test.want {
			t.Fatalf("classifyDynamicInstallFailure(%q) = %q, want %q", test.output, got, test.want)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := classifyDynamicInstallFailure(ctx, ""); got != dynamicInstallFailureOther {
		t.Fatalf("canceled context class = %q", got)
	}
	deadline, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	<-deadline.Done()
	if got := classifyDynamicInstallFailure(deadline, ""); got != dynamicInstallFailureTimeout {
		t.Fatalf("deadline class = %q", got)
	}
}

func TestPythonDynamicBackendRejectsNonWheelClosure(t *testing.T) {
	root, target := pythonWheelFixture(t)
	source, _ := domain.NewSourceID("pypi")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "source", "1.0", "sdist")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("b", 64))
	sdist, _ := domain.NewAcquiredArtifact(identity, digest, "intake:run_bbbbbbbbbbbbbbbbbbbbbbbbbb:sdist", 1)
	introducer, _ := NewPythonArtifactIntroducer(root, &recordingRunner{})
	backend, _ := NewPythonDynamicBackend(&recordingRunner{}, introducer, &recordingObserver{reader: &traceReader{}}, availablePythonProbe)
	if _, err := backend.InspectWheelWithClosure(context.Background(), target, []string{"example"}, []domain.AcquiredArtifact{target, sdist}); err == nil {
		t.Fatal("non-wheel closure accepted")
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
