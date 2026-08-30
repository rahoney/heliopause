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
	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil, nil, nil, nil, nil}}
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
	if len(runner.calls) != 6 {
		t.Fatalf("commands = %#v", runner.calls)
	}
	if len(runner.timeline) < 4 || !sameStrings(runner.timeline[2].arguments, boundaryExecArguments("0123456789abcdef", boundaryLaunchMode, "/bin/true")) || runner.timeline[3].binary != "docker" || !strings.Contains(strings.Join(runner.timeline[3].arguments, " "), "python -I -c") {
		t.Fatalf("helper was not ready before artifact introduction: %#v", runner.timeline)
	}
	assertPythonDynamicCreate(t, runner.calls[0].arguments)
	if !strings.Contains(strings.Join(runner.calls[3].arguments, " "), "--no-index --no-deps --no-compile") || !strings.Contains(strings.Join(runner.calls[3].arguments, " "), "--target /haa-site") || !strings.Contains(strings.Join(runner.calls[3].arguments, " "), "/tmp/example-1.0-py3-none-any.whl") {
		t.Fatalf("pip install command = %#v", runner.calls[3])
	}
	if !strings.Contains(strings.Join(runner.calls[4].arguments, " "), "/haa-site") || strings.Contains(strings.Join(runner.calls[4].arguments, " "), "/tmp/haa-site") || !sameStrings(runner.calls[4].arguments[len(runner.calls[4].arguments)-1:], []string{"example"}) {
		t.Fatalf("import command = %#v", runner.calls[4])
	}
}

func TestPythonDynamicBackendFailsClosedForIncompleteObservation(t *testing.T) {
	root, artifact := pythonWheelFixture(t)
	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil, nil, nil, nil, nil}}
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
	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil, nil, nil, nil, nil}}
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
	if !strings.Contains(joined, "--tmpfs /tmp:rw,noexec,nosuid,nodev,size=2147483648") {
		t.Fatalf("general temporary space is not noexec: %q", joined)
	}
	if !strings.Contains(joined, "--tmpfs /haa-site:rw,exec,nosuid,nodev,size=2147483648") {
		t.Fatalf("dedicated Python site is not bounded exec tmpfs: %q", joined)
	}
}

func TestPythonDynamicObserverProfileSelectsRootTransactionPolicyNotNodeSource(t *testing.T) {
	transitiveRunID := "run_" + strings.Repeat("a", 26)
	root, transitiveArtifact := wheelArtifactWithFilename(t, transitiveRunID, "pypi", "networkx", "3.6.1", "networkx-3.6.1-py3-none-any.whl")
	if transitiveArtifact.Identity().Source().String() != "pypi" {
		t.Fatalf("transitive artifact source = %s, want pypi", transitiveArtifact.Identity().Source())
	}

	// 1. Root transaction is pytorch:cpu, current node is ordinary PyPI transitive dependency
	cpuProfile, ok := artifactpypi.PyTorchProfile("cpu")
	if !ok {
		t.Fatal("missing cpu profile")
	}
	cpuCtx, err := artifactpypi.ContextWithResourcePolicy(context.Background(), cpuProfile)
	if err != nil {
		t.Fatal(err)
	}

	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil, nil, nil, nil, nil}}
	introducer, err := NewPythonArtifactIntroducer(root, runner)
	if err != nil {
		t.Fatal(err)
	}
	observer := &recordingObserver{reader: &traceReader{}}
	backend, err := NewPythonDynamicBackend(runner, introducer, observer, availablePythonProbe)
	if err != nil {
		t.Fatal(err)
	}

	result, err := backend.InspectWheel(cpuCtx, transitiveArtifact, []string{"networkx"})
	if err != nil || result.Status() != domain.SandboxCompleted {
		t.Fatalf("InspectWheel() under CPU root = %#v, %v", result, err)
	}
	if observer.profile != "pypi-wheel-pytorch-cpu" {
		t.Fatalf("observer profile for PyPI node under CPU root = %q, want %q", observer.profile, "pypi-wheel-pytorch-cpu")
	}
	cpuBudget := traceBudgetForProfile(observer.profile)
	if cpuBudget.events != 500_000 || cpuBudget.bytes != 128<<20 {
		t.Fatalf("CPU trace budget = %#v, want 500k/128MiB", cpuBudget)
	}

	// 2. Root transaction is ordinary PyPI (or default background context)
	defaultRunner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil, nil, nil, nil, nil}}
	defaultIntroducer, err := NewPythonArtifactIntroducer(root, defaultRunner)
	if err != nil {
		t.Fatal(err)
	}
	defaultObserver := &recordingObserver{reader: &traceReader{}}
	defaultBackend, err := NewPythonDynamicBackend(defaultRunner, defaultIntroducer, defaultObserver, availablePythonProbe)
	if err != nil {
		t.Fatal(err)
	}
	result, err = defaultBackend.InspectWheel(context.Background(), transitiveArtifact, []string{"networkx"})
	if err != nil || result.Status() != domain.SandboxCompleted {
		t.Fatalf("InspectWheel() under default root = %#v, %v", result, err)
	}
	if defaultObserver.profile != "pypi-wheel" {
		t.Fatalf("observer profile under default root = %q, want %q", defaultObserver.profile, "pypi-wheel")
	}
	defaultBudget := traceBudgetForProfile(defaultObserver.profile)
	if defaultBudget.events != 10_000 || defaultBudget.bytes != 2<<20 {
		t.Fatalf("default trace budget = %#v, want 10k/2MiB", defaultBudget)
	}

	// 3. Root transaction is pytorch:cu126
	cu126Profile, ok := artifactpypi.PyTorchProfile("cu126")
	if !ok {
		t.Fatal("missing cu126 profile")
	}
	cu126Ctx, err := artifactpypi.ContextWithResourcePolicy(context.Background(), cu126Profile)
	if err != nil {
		t.Fatal(err)
	}
	cu126Runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil, nil, nil, nil, nil}}
	cu126Introducer, err := NewPythonArtifactIntroducer(root, cu126Runner)
	if err != nil {
		t.Fatal(err)
	}
	cu126Observer := &recordingObserver{reader: &traceReader{}}
	cu126Backend, err := NewPythonDynamicBackend(cu126Runner, cu126Introducer, cu126Observer, availablePythonProbe)
	if err != nil {
		t.Fatal(err)
	}
	result, err = cu126Backend.InspectWheel(cu126Ctx, transitiveArtifact, []string{"networkx"})
	if err != nil || result.Status() != domain.SandboxCompleted {
		t.Fatalf("InspectWheel() under cu126 root = %#v, %v", result, err)
	}
	if cu126Observer.profile != "pypi-wheel-pytorch-cu126" {
		t.Fatalf("observer profile under cu126 root = %q, want %q", cu126Observer.profile, "pypi-wheel-pytorch-cu126")
	}
	cu126Budget := traceBudgetForProfile(cu126Observer.profile)
	if cu126Budget.events != 100_000 || cu126Budget.bytes != 16<<20 {
		t.Fatalf("cu126 trace budget = %#v, want 100k/16MiB", cu126Budget)
	}
}

func TestPythonDynamicObserverProfileMapping(t *testing.T) {
	tests := []struct {
		rootProfile string
		wantProfile string
		wantEvents  int
		wantBytes   uint64
		wantErr     bool
	}{
		{"pypi", "pypi-wheel", 10_000, 2 << 20, false},
		{"pytorch:cpu", "pypi-wheel-pytorch-cpu", 500_000, 128 << 20, false},
		{"pytorch:cu126", "pypi-wheel-pytorch-cu126", 100_000, 16 << 20, false},
		{"unknown", "", 0, 0, true},
		{"", "", 0, 0, true},
	}
	for _, test := range tests {
		t.Run(test.rootProfile, func(t *testing.T) {
			profile, err := pythonDynamicObserverProfile(test.rootProfile)
			if test.wantErr {
				if err == nil {
					t.Fatalf("pythonDynamicObserverProfile(%q) unexpectedly succeeded with %q", test.rootProfile, profile)
				}
				return
			}
			if err != nil || profile != test.wantProfile {
				t.Fatalf("pythonDynamicObserverProfile(%q) = (%q, %v), want %q", test.rootProfile, profile, err, test.wantProfile)
			}
			budget := traceBudgetForProfile(profile)
			if budget.events != test.wantEvents || budget.bytes != test.wantBytes {
				t.Fatalf("budget for %q = %#v, want events=%d bytes=%d", profile, budget, test.wantEvents, test.wantBytes)
			}
		})
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
	if err := os.WriteFile(filepath.Join(dependencyRun, "filename"), []byte("dependency-1.0-py3-none-any.whl"), 0o400); err != nil {
		t.Fatal(err)
	}
	source, _ := domain.NewSourceID("pypi")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "dependency", "1.0", "wheel")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("b", 64))
	dependency, err := domain.NewAcquiredArtifact(identity, digest, "intake:"+dependencyRunID+":wheel", uint64(len("dependency wheel")))
	if err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil, nil, nil, nil, nil}}
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
	if len(runner.calls) < 4 || !strings.Contains(strings.Join(runner.calls[3].arguments, " "), "--no-index --no-deps") || !strings.Contains(strings.Join(runner.calls[3].arguments, " "), "--target /haa-site") || !strings.Contains(strings.Join(runner.calls[3].arguments, " "), "/tmp/example-1.0-py3-none-any.whl") || !strings.Contains(strings.Join(runner.calls[3].arguments, " "), "/tmp/dependency-1.0-py3-none-any.whl") {
		t.Fatalf("closure install command = %#v", runner.calls)
	}
}

func TestPythonDynamicBackendClassifiesBoundedInstallFailureWithoutExposingOutput(t *testing.T) {
	root, artifact := pythonWheelFixture(t)
	runner := &recordingRunner{
		responses:     [][]byte{[]byte("0123456789abcdef")},
		errors:        []error{nil, nil, nil, errors.New("exit status 1")},
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

func TestPythonDynamicBackendClassifiesBoundedImportFailureWithoutExposingOutput(t *testing.T) {
	root, artifact := pythonWheelFixture(t)
	runner := &recordingRunner{
		responses:     [][]byte{[]byte("0123456789abcdef")},
		errors:        []error{nil, nil, nil, nil, errors.New("exit status 1")},
		boundedOutput: []byte("ImportError: libtorch_cpu.so: cannot open shared object file"),
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
	if result.Status() != domain.SandboxIncomplete || limitation != "M5_PYPI_DYNAMIC_IMPORT_FAILED_MISSING_SHARED_LIBRARY" || strings.Contains(limitation, "libtorch") {
		t.Fatalf("result = %#v", result)
	}
}

func TestPythonDynamicBackendClassifiesObserverStartFailure(t *testing.T) {
	root, artifact := pythonWheelFixture(t)
	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef")}}
	introducer, err := NewPythonArtifactIntroducer(root, runner)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := NewPythonDynamicBackend(runner, introducer, &recordingObserver{err: observerFault{reason: "HELPER_CRASHED"}}, availablePythonProbe)
	if err != nil {
		t.Fatal(err)
	}
	result, err := backend.InspectWheel(context.Background(), artifact, []string{"example"})
	if err != nil {
		t.Fatal(err)
	}
	limitation, _ := result.LimitationCode()
	if result.Status() != domain.SandboxIncomplete || limitation != "M5_PYPI_DYNAMIC_OBSERVER_FAILED_START_HELPER_CRASHED" {
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

func TestClassifyDynamicImportAndObserverFailureUseBoundedVocabulary(t *testing.T) {
	for _, test := range []struct {
		output string
		want   string
	}{
		{"ImportError: cannot open shared object file", dynamicImportFailureMissingLibrary},
		{"ImportError: failed to map segment from shared object: Operation not permitted", dynamicImportFailureDlopenPermission},
		{"ImportError: wrong ELF class", dynamicImportFailureELFLoader},
		{"ImportError: GLIBCXX_3.4.99 not found", dynamicImportFailureSymbolVersion},
		{"ImportError: Python ABI version mismatch", dynamicImportFailurePythonABI},
		{"MemoryError: cannot allocate memory", dynamicImportFailureMemory},
		{"Resource temporarily unavailable", dynamicImportFailurePID},
		{"CPU time limit exceeded", dynamicImportFailureCPU},
		{"OCI runtime runsc failure", dynamicImportFailureSandboxRuntime},
		{"Traceback (most recent call last)", dynamicImportFailureException},
		{"unrecognized", dynamicImportFailureOther},
	} {
		if got := classifyDynamicImportFailure(context.Background(), test.output); got != test.want {
			t.Fatalf("classifyDynamicImportFailure(%q) = %q, want %q", test.output, got, test.want)
		}
	}
	deadline, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	<-deadline.Done()
	if got := classifyDynamicImportFailure(deadline, ""); got != dynamicImportFailureTimeout {
		t.Fatalf("deadline class = %q", got)
	}
	for _, test := range []struct {
		reason string
		want   string
	}{
		{"HELPER_UNAVAILABLE", "HELPER_UNAVAILABLE"},
		{"HELPER_CRASHED", "HELPER_CRASHED"},
		{"EVENT_LIMIT", "EVENT_LIMIT"},
		{"BYTE_LIMIT", "BYTE_LIMIT"},
		{"CHANNEL_OVERFLOW", "SESSION_LIMIT"},
		{"STREAM_FAULT", "TRACE_COLLECTION_FAILED"},
		{"ATTRIBUTION_FAILURE", "LIFECYCLE_ERROR"},
		{"unrecognized", "OTHER"},
	} {
		if got := classifyDynamicObserverFailure(observerFault{reason: test.reason}); got != test.want {
			t.Fatalf("classifyDynamicObserverFailure(%q) = %q, want %q", test.reason, got, test.want)
		}
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

func TestPythonArtifactIntroducerPreservesExactVerifiedWheelFilename(t *testing.T) {
	tests := []struct {
		name, source, project, version, filename string
	}{
		{"torch local version and tags", "pytorch-cpu", "torch", "2.9.1+cpu", "torch-2.9.1+cpu-cp314-cp314-manylinux_2_28_x86_64.whl"},
		{"MarkupSafe compressed platform tags", "pypi", "markupsafe", "3.0.3", "MarkupSafe-3.0.3-cp314-cp314-manylinux2014_x86_64.manylinux_2_17_x86_64.manylinux_2_28_x86_64.whl"},
		{"ordinary universal wheel", "pypi", "example", "1.0", "example-1.0-py3-none-any.whl"},
		{"build tag", "pypi", "example", "1.0", "example-1.0-1-cp314-cp314-manylinux_2_28_x86_64.whl"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, artifact := wheelArtifactWithFilename(t, "run_"+strings.Repeat("a", 26), test.source, test.project, test.version, test.filename)
			introducer, err := NewPythonArtifactIntroducer(root, &recordingRunner{})
			if err != nil {
				t.Fatal(err)
			}
			got, err := introducer.validatedWheelDestination(artifact)
			if err != nil || got != "/tmp/"+test.filename {
				t.Fatalf("validatedWheelDestination() = %q, %v", got, err)
			}
		})
	}
}

func TestPythonArtifactIntroducerRejectsUntrustedWheelFilenameRecord(t *testing.T) {
	tests := []struct {
		name, filename string
	}{
		{"empty", ""},
		{"absolute", "/tmp/example-1.0-py3-none-any.whl"},
		{"traversal", "../example-1.0-py3-none-any.whl"},
		{"backslash escape", `..\\example-1.0-py3-none-any.whl`},
		{"invalid wheel filename", "example-not-a-wheel"},
		{"project mismatch", "other-1.0-py3-none-any.whl"},
		{"version mismatch", "example-2.0-py3-none-any.whl"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, artifact := wheelArtifactWithFilename(t, "run_"+strings.Repeat("a", 26), "pypi", "example", "1.0", test.filename)
			introducer, err := NewPythonArtifactIntroducer(root, &recordingRunner{})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := introducer.validatedWheelDestination(artifact); err == nil {
				t.Fatalf("filename %q was accepted", test.filename)
			}
		})
	}
}

func TestPythonArtifactIntroducerRejectsFilenameFromWrongRun(t *testing.T) {
	root := t.TempDir()
	wheelArtifactInRoot(t, root, "run_"+strings.Repeat("a", 26), "pypi", "example", "1.0", "example-1.0-py3-none-any.whl")

	wrongRun := "run_" + strings.Repeat("b", 26)
	identity, err := domain.NewResolvedArtifactIdentity(mustSourceID(t, "pypi"), "example", "1.0", "wheel")
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	wrongArtifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:"+wrongRun+":wheel", uint64(len("wheel fixture")))
	if err != nil {
		t.Fatal(err)
	}
	introducer, err := NewPythonArtifactIntroducer(root, &recordingRunner{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := introducer.validatedWheelDestination(wrongArtifact); err == nil {
		t.Fatal("filename record from another run was accepted")
	}
}

func TestPythonArtifactIntroducerRejectsDuplicateExactDestinations(t *testing.T) {
	root := t.TempDir()
	filename := "example-1.0-py3-none-any.whl"
	targetRun := "run_" + strings.Repeat("a", 26)
	dependencyRun := "run_" + strings.Repeat("b", 26)
	target := wheelArtifactInRoot(t, root, targetRun, "pypi", "example", "1.0", filename)
	dependency := wheelArtifactInRoot(t, root, dependencyRun, "pypi", "example", "1.0", filename)
	introducer, err := NewPythonArtifactIntroducer(root, &recordingRunner{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := introducer.validatedWheelDestinations(target, []domain.AcquiredArtifact{target, dependency}); err == nil {
		t.Fatal("duplicate destination basename was accepted")
	}
}

func wheelArtifactWithFilename(t *testing.T, runID, sourceName, project, version, filename string) (string, domain.AcquiredArtifact) {
	t.Helper()
	root := t.TempDir()
	return root, wheelArtifactInRoot(t, root, runID, sourceName, project, version, filename)
}

func wheelArtifactInRoot(t *testing.T, root, runID, sourceName, project, version, filename string) domain.AcquiredArtifact {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, runID), 0o700); err != nil {
		t.Fatal(err)
	}
	body := []byte("wheel fixture")
	if err := os.WriteFile(filepath.Join(root, runID, "wheel.whl"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, runID, "filename"), []byte(filename), 0o400); err != nil {
		t.Fatal(err)
	}
	source := mustSourceID(t, sourceName)
	identity, err := domain.NewResolvedArtifactIdentity(source, project, version, "wheel")
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:"+runID+":wheel", uint64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func mustSourceID(t *testing.T, value string) domain.SourceID {
	t.Helper()
	source, err := domain.NewSourceID(value)
	if err != nil {
		t.Fatal(err)
	}
	return source
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
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "filename"), []byte("example-1.0-py3-none-any.whl"), 0o400); err != nil {
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
	for _, required := range []string{"--pull never", "--runtime " + gVisorRuntimeName, "--network none", "--read-only", "--cap-drop ALL", "no-new-privileges", "--pids-limit 64", "--memory 536870912", "--tmpfs " + boundaryHelperMount, pythonImageReference} {
		if !strings.Contains(joined, required) {
			t.Errorf("create command missing %q: %q", required, joined)
		}
	}
	if strings.Contains(joined, "--user 1000:1000") || !strings.Contains(joined, boundaryContainerCommand()) {
		t.Errorf("create command does not establish root-owned boundary helper: %q", joined)
	}
	for _, forbidden := range []string{"--mount", "--volume", "-v ", "--env", "--privileged", "--network host", "/var/run/docker.sock"} {
		if strings.Contains(joined, forbidden) {
			t.Errorf("create command contains forbidden %q: %q", forbidden, joined)
		}
	}
}
