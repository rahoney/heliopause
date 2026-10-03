package sandbox

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestPythonSdistBuilderUsesOnlyVerifiedBuildWheels(t *testing.T) {
	root, source, wheel := pythonBuildFixtures(t)
	runner := &buildRunner{recordingRunner: recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil, nil, nil, nil, nil, []byte("example-1.0-py3-none-any.whl\n"), nil}}, output: []byte("derived wheel")}
	introducer, _ := NewPythonArtifactIntroducer(root, runner)
	builder, err := NewPythonSdistBuilder(runner, introducer, &recordingObserver{reader: &traceReader{}}, availablePythonProbe)
	if err != nil {
		t.Fatal(err)
	}
	recipe := artifactpypi.SdistInspection{Project: "example", Version: "1.0", ObservedSHA256: source.Digest().String(), BuildRequirements: []string{"setuptools"}}
	derived, result, err := builder.Build(context.Background(), source, recipe, []domain.AcquiredArtifact{wheel})
	if err != nil || result.Status() != domain.SandboxCompleted || derived.Filename != "example-1.0-py3-none-any.whl" || derived.Artifact.Identity().Variant() != "derived-wheel" || derived.Artifact.SizeBytes() != uint64(len("derived wheel")) || derived.SourceDigest != source.Digest() || len(derived.BuildRequirementDigests) != 1 {
		t.Fatalf("Build() = %#v, %#v, %v", derived, result, err)
	}
	if len(runner.inputCalls) != 2 || len(runner.calls) != 8 {
		t.Fatalf("commands = %#v; inputs = %#v", runner.calls, runner.inputCalls)
	}
	if joined := strings.Join(runner.calls[5].arguments, " "); !strings.Contains(joined, "pip wheel --no-index --no-deps --no-build-isolation") || !strings.Contains(joined, pythonSdistPath(source)) {
		t.Fatalf("build command = %#v", runner.calls[5])
	}
	if got := runner.calls[5].arguments; len(got) < 6 || got[5] != boundaryOriginPythonHandoffMode {
		t.Fatalf("build command does not use the Python artifact handoff: %#v", got)
	}
	if _, err := os.Stat(filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa", "derived.whl")); err != nil {
		t.Fatal(err)
	}
	filename, err := os.ReadFile(filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa", "derived-filename"))
	declared, declaredOK := derived.Artifact.DeclaredIntegrity()
	if err != nil || string(filename) != derived.Filename || !declaredOK || declared != "sha256:"+derived.Artifact.Digest().String() {
		t.Fatalf("derived wheel binding filename=%q declared=%q ok=%v err=%v", filename, declared, declaredOK, err)
	}
}

func TestPythonSdistBuilderRejectsGraphExpansionBeforeDocker(t *testing.T) {
	root, source, wheel := pythonBuildFixtures(t)
	runner := &buildRunner{}
	introducer, _ := NewPythonArtifactIntroducer(root, runner)
	builder, _ := NewPythonSdistBuilder(runner, introducer, &emptyObserver{}, availablePythonProbe)
	recipe := artifactpypi.SdistInspection{Project: "example", Version: "1.0", ObservedSHA256: source.Digest().String(), BuildRequirements: []string{"setuptools", "wheel"}}
	if _, _, err := builder.Build(context.Background(), source, recipe, []domain.AcquiredArtifact{wheel}); err == nil || len(runner.calls) != 0 {
		t.Fatalf("Build accepted incomplete build graph: %#v", runner.calls)
	}
}

func TestPythonArtifactIntroductionRejectsChangedStreamBytes(t *testing.T) {
	root, source, _ := pythonBuildFixtures(t)
	path := filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa", "sdist.tar.gz")
	if err := os.WriteFile(path, []byte("sourcE"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{}
	introducer, err := NewPythonArtifactIntroducer(root, runner)
	if err != nil {
		t.Fatal(err)
	}
	if err := introducer.introduce(context.Background(), "0123456789abcdef", source, "/tmp/source.tar.gz", "sdist"); err == nil {
		t.Fatal("same-size intake mutation was accepted after its bytes were streamed")
	}
}

func pythonBuildFixtures(t *testing.T) (string, domain.AcquiredArtifact, domain.AcquiredArtifact) {
	t.Helper()
	root := t.TempDir()
	sourcePath := filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa", "sdist.tar.gz")
	wheelPath := filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa", "wheel.whl")
	for path, body := range map[string][]byte{sourcePath: []byte("source"), wheelPath: []byte("wheel")} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(wheelPath), "filename"), []byte("setuptools-70.0-py3-none-any.whl"), 0o400); err != nil {
		t.Fatal(err)
	}
	sourceID, _ := domain.NewSourceID("pypi")
	sourceHash, _ := domain.NewSHA256Digest(fmt.Sprintf("%x", sha256.Sum256([]byte("source"))))
	wheelHash, _ := domain.NewSHA256Digest(fmt.Sprintf("%x", sha256.Sum256([]byte("wheel"))))
	sourceIdentity, _ := domain.NewResolvedArtifactIdentity(sourceID, "example", "1.0", "sdist")
	source, err := domain.NewAcquiredArtifact(sourceIdentity, sourceHash, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:sdist", uint64(len("source")))
	if err != nil {
		t.Fatal(err)
	}
	wheelIdentity, _ := domain.NewResolvedArtifactIdentity(sourceID, "setuptools", "70.0", "wheel")
	wheel, err := domain.NewAcquiredArtifact(wheelIdentity, wheelHash, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:wheel", uint64(len("wheel")))
	if err != nil {
		t.Fatal(err)
	}
	return root, source, wheel
}

type buildRunner struct {
	recordingRunner
	output []byte
}

func (r *buildRunner) RunOutput(_ context.Context, output io.Writer, _ string, _ ...string) error {
	_, err := output.Write(r.output)
	return err
}

func TestPythonSdistBuildPreservesPrimaryAndObserverFailure(t *testing.T) {
	for _, cleanupFails := range []bool{false, true} {
		t.Run(fmt.Sprintf("cleanup-%t", cleanupFails), func(t *testing.T) {
			root, source, wheel := pythonBuildFixtures(t)
			runner := &sdistFailureRunner{cleanupFails: cleanupFails}
			introducer, _ := NewPythonArtifactIntroducer(root, runner)
			builder, _ := NewPythonSdistBuilder(runner, introducer, &recordingObserver{reader: terminationDiagnosticTrace{err: observerFault{reason: "STREAM_FAULT", site: "OPEN_RESULT_CLASSIFICATION_IMAGE", imageLocator: 123}}}, availablePythonProbe)
			recipe := artifactpypi.SdistInspection{Project: "example", Version: "1.0", ObservedSHA256: source.Digest().String(), BuildRequirements: []string{"setuptools"}}
			built, result, err := builder.Build(context.Background(), source, recipe, []domain.AcquiredArtifact{wheel})
			code, _ := result.LimitationCode()
			if err == nil || code != "M5_PYPI_BUILD_FAILED" || result.Status() != domain.SandboxIncomplete || built.Filename != "" || !runner.removed {
				t.Fatalf("failure lost: %#v %s %v removed=%t", built, code, err, runner.removed)
			}
			for _, want := range []string{"primary_code=M5_PYPI_BUILD_FAILED", "command=COMMAND_ERROR", "reason=STREAM_FAULT", "fault_site=OPEN_RESULT_CLASSIFICATION_IMAGE", "image_locator_fnv1a64=000000000000007b"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("missing %q in %v", want, err)
				}
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("raw command text leaked")
			}
			if cleanupFails && !strings.Contains(err.Error(), "cleanup=M5_PYPI_DYNAMIC_CLEANUP_FAILED") {
				t.Fatalf("cleanup lost: %v", err)
			}
		})
	}
}

type sdistFailureRunner struct {
	buildRunner
	cleanupFails, removed bool
}

func (r *sdistFailureRunner) Output(ctx context.Context, binary string, args ...string) ([]byte, error) {
	if len(args) > 0 && args[0] == "create" {
		return []byte("0123456789abcdef"), nil
	}
	return r.buildRunner.Output(ctx, binary, args...)
}
func (r *sdistFailureRunner) RunDiscard(_ context.Context, _ string, args ...string) error {
	if len(args) > 0 && args[0] == "rm" {
		r.removed = true
		if r.cleanupFails {
			return errors.New("secret cleanup output")
		}
	}
	if strings.Contains(strings.Join(args, " "), "--no-build-isolation") {
		return errors.New("secret backend output")
	}
	return nil
}
