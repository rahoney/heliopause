package sandbox

import (
	"context"
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
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	sourceIdentity, _ := domain.NewResolvedArtifactIdentity(sourceID, "example", "1.0", "sdist")
	source, err := domain.NewAcquiredArtifact(sourceIdentity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:sdist", uint64(len("source")))
	if err != nil {
		t.Fatal(err)
	}
	wheelIdentity, _ := domain.NewResolvedArtifactIdentity(sourceID, "setuptools", "70.0", "wheel")
	wheel, err := domain.NewAcquiredArtifact(wheelIdentity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:wheel", uint64(len("wheel")))
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
