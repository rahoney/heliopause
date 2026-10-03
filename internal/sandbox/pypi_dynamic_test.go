package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

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
		{"pytorch:cu130", "pypi-wheel-pytorch-cu130", 100_000, 16 << 20, false},
		{"pytorch:cu132", "pypi-wheel-pytorch-cu132", 100_000, 16 << 20, false},
		{"pytorch:cu128", "", 0, 0, true},
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

func TestPythonDynamicBackendRequiresTrustedTransaction(t *testing.T) {
	root, artifact := pythonWheelFixture(t)
	runner := &recordingRunner{}
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
	if result.Status() != domain.SandboxIncomplete || limitation != "M5_PYPI_DYNAMIC_TRUSTED_TRANSACTION_UNAVAILABLE" {
		t.Fatalf("result = %#v", result)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("artifact execution was launched: %#v", runner.calls)
	}
}
