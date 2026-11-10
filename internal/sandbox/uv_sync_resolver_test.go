package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func uvSyncTestInput(t *testing.T, body []byte, allow bool) artifactpypi.UVSyncInput {
	t.Helper()
	input, err := artifactpypi.ParseUVSyncInput(artifactpypi.UVSyncRequirements, body, allow, PinnedPythonRuntime().PythonVersion)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func uvSyncTestContext(t *testing.T) domain.InstallContext {
	t.Helper()
	target, err := domain.NewInstallTarget("/tmp/uv-sync-resolver-test")
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewPythonVenvInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	return install
}

func uvSyncTestSelectedControls(t *testing.T, input artifactpypi.UVSyncInput, body []byte) []domain.ProjectControlFile {
	t.Helper()
	controls, err := NewUVSyncOriginalControls(input)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := domain.NewProjectControlFile(artifactpypi.UVSelectionLockControlName, body, true)
	if err != nil {
		t.Fatal(err)
	}
	for i := range controls {
		if controls[i].Name() == lock.Name() {
			controls[i] = lock
		}
	}
	return controls
}

func TestUVCompleteResolverRejectsBeforeNativeOrNetwork(t *testing.T) {
	for _, outcome := range []string{"nil context", "canceled", "selector", "context mode", "missing source", "duplicate", "unknown", "runtime substitution", "selected original", "skip", "runtime unavailable", "untrusted tool"} {
		t.Run(outcome, func(t *testing.T) {
			_, runner := uvReaderFixture(t)
			endpoints := &uvCountingEndpoints{}
			probe := availablePythonProbe
			if outcome == "runtime unavailable" {
				probe = func(context.Context) (PythonCapability, error) {
					return PythonCapability{}, errors.New("runtime unavailable")
				}
			}
			resolver, err := NewUVResolver(runner, endpoints, &recordingObserver{reader: &traceReader{}}, probe, &recordingResolverPolicyService{})
			if err != nil {
				t.Fatal(err)
			}
			input := uvSyncTestInput(t, []byte("probe==1.0\n"), false)
			controls, err := NewUVSyncOriginalControls(input)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			install, selector := uvSyncTestContext(t), ""
			switch outcome {
			case "nil context":
				ctx = nil
			case "canceled":
				cancel()
			case "selector":
				selector = "probe"
			case "context mode":
				install = domain.InstallContext{}
			case "missing source":
				controls[0], _ = domain.NewProjectControlFile(controls[0].Name(), nil, false)
			case "duplicate":
				controls[0] = controls[1]
			case "unknown":
				controls[0], _ = domain.NewProjectControlFile("foreign", nil, true)
			case "runtime substitution":
				controls[2], _ = domain.NewProjectControlFile(uvSyncRuntimeControlName, []byte(`{"version":1,"resolver_runtime":"uv:foreign"}`), true)
			case "selected original":
				controls[3], _ = domain.NewProjectControlFile(artifactpypi.UVSelectionLockControlName, []byte("unapproved"), true)
			case "skip":
				source, _ := uvSyncTestInput(t, nil, false).Controls()
				controls[0], controls[1] = source[0], source[1]
			case "untrusted tool":
				runner.responses = [][]byte{[]byte("changed immutable image identity")}
			}
			result, err := resolver.ResolveProjectLockUpdate(ctx, install, controls, selector)
			if err == nil || result.Valid() || result.Snapshot().Valid() || endpoints.calls != 0 || len(runner.inputCalls) != 0 {
				t.Fatal("invalid complete request reached selection", err)
			}
			if outcome == "untrusted tool" {
				if !errors.Is(err, errUVToolSupply) || len(runner.calls) != 1 || runner.calls[0].arguments[0] != "image" {
					t.Fatal("tool supply preflight bypassed", err)
				}
			} else if len(runner.calls) != 0 {
				t.Fatal("refused complete source mutated Docker")
			}
		})
	}
	if _, err := NewUVSyncOriginalControls(uvSyncTestInput(t, nil, false)); err == nil {
		t.Fatal("default blank skip became positive original controls")
	}
	foreign, err := artifactpypi.ParseUVSyncInput(artifactpypi.UVSyncRequirements, []byte("probe==1.0\n"), false, "3.14.8")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewUVSyncOriginalControls(foreign); err == nil {
		t.Fatal("foreign Python target accepted")
	}
}

func TestUVCompleteIndependentReaderReusesWholeArchiveValidation(t *testing.T) {
	for _, outcome := range []string{"complete", "archive checksum", "metadata mismatch", "source mismatch", "cleanup failure", "trace incomplete"} {
		t.Run(outcome, func(t *testing.T) {
			native, runner := uvReaderFixture(t)
			trace := &traceReader{}
			source := []byte("probe==1.0\n")
			switch outcome {
			case "archive checksum":
				runner.archive = append([]byte(nil), runner.archive...)
				runner.archive[0] ^= 1
			case "metadata mismatch":
				runner.metadata = bytes.Replace(runner.metadata, []byte(`"version":"1.0"`), []byte(`"version":"9.0"`), 1)
			case "source mismatch":
				source = []byte("probe==2.0\n")
			case "cleanup failure":
				runner.cleanupErr = errors.New("cleanup failure")
			case "trace incomplete":
				trace.err = errors.New("stream failed")
			}
			reader, err := NewPyPIResolver(runner, pypiStaticEndpoints{}, &recordingObserver{reader: trace}, availablePythonProbe, &recordingResolverPolicyService{})
			if err != nil {
				t.Fatal(err)
			}
			input := uvSyncTestInput(t, source, false)
			controls := uvSyncTestSelectedControls(t, input, native.lock)
			install := uvSyncTestContext(t)
			result, err := reader.bindUVNativeProjectSelection(context.Background(), install, controls, input, native)
			if outcome == "complete" {
				if err != nil || !result.Valid() || result.DependencyFree() || result.Context() != install || len(result.Dependencies()) != 1 || len(result.ControlDigests()) != 4 {
					t.Fatal("complete independent binding failed", err)
				}
				original, _ := NewUVSyncOriginalControls(input)
				update, err := domain.NewProjectDependencyUpdateFromSnapshot(original, controls, result)
				if err != nil || !update.Valid() || update.HasRequestedResolution() {
					t.Fatal("complete selection acquired a fake primary", err)
				}
			} else if err == nil || result.Valid() {
				t.Fatal("invalid complete archive/metadata retained snapshot", outcome, err)
			}
			if runner.filePath != "" {
				if _, err := os.Lstat(runner.filePath); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("complete reader archive workspace retained", err)
				}
			}
		})
	}
}

func uvExplicitEmptyLock() []byte {
	return []byte(fmt.Sprintf("lock-version='1.0'\ncreated-by='uv'\nrequires-python='>=%s'\npackages=[]\n", PinnedPythonRuntime().PythonVersion))
}

func TestUVCompleteEmptyBindingDoesNotInventArchivesOrSkipApproval(t *testing.T) {
	_, runner := uvReaderFixture(t)
	reader, err := NewPyPIResolver(runner, pypiStaticEndpoints{}, &recordingObserver{reader: &traceReader{}}, availablePythonProbe, &recordingResolverPolicyService{})
	if err != nil {
		t.Fatal(err)
	}
	input := uvSyncTestInput(t, nil, true)
	native := uvNativeOutput{lock: uvExplicitEmptyLock(), controls: []artifactpypi.UVInstalledWheelControls{}}
	controls := uvSyncTestSelectedControls(t, input, native.lock)
	install := uvSyncTestContext(t)
	result, err := reader.bindUVNativeProjectSelection(context.Background(), install, controls, input, native)
	if err != nil || !result.DependencyFree() || len(result.Dependencies()) != 0 || len(runner.calls) != 0 || runner.streams != 0 {
		t.Fatal("explicit empty binding invented archive traffic", err)
	}
	for _, outcome := range []string{"missing census", "extra census", "missing lock", "skip", "canceled", "invalid target", "nil context"} {
		t.Run(outcome, func(t *testing.T) {
			candidate, source, target := native, input, install
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch outcome {
			case "missing census":
				candidate.controls = nil
			case "extra census":
				candidate.controls = []artifactpypi.UVInstalledWheelControls{{DistInfo: "extra"}}
			case "missing lock":
				candidate.lock = nil
			case "skip":
				source = uvSyncTestInput(t, nil, false)
			case "canceled":
				cancel()
			case "invalid target":
				target = domain.InstallContext{}
			case "nil context":
				ctx = nil
			}
			result, err := reader.bindUVNativeProjectSelection(ctx, target, controls, source, candidate)
			if err == nil || result.Valid() {
				t.Fatal("missing/skip/invalid state became positive empty", err)
			}
		})
	}
}

func uvEmptyNativeOutputTar(t *testing.T, site bool, payload string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	add := func(name string, kind byte, mode int64, body []byte) {
		if err := writer.WriteHeader(&tar.Header{Name: name, Typeflag: kind, Mode: mode, Uid: 1000, Gid: 1000, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if site {
		add("site/", tar.TypeDir, 0700, nil)
	}
	add("pylock.toml", tar.TypeReg, 0400, uvExplicitEmptyLock())
	switch payload {
	case "file":
		add("site/unrecorded", tar.TypeReg, 0400, nil)
	case "directory":
		add("site/hidden/", tar.TypeDir, 0700, nil)
	case "auxiliary lock":
		add("site/.lock", tar.TypeReg, 0666, nil)
	case "nonempty lock":
		add("site/.lock", tar.TypeReg, 0400, []byte("x"))
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestUVCompleteEmptyOutputRequiresExplicitLockAndNoPrivatePayload(t *testing.T) {
	for _, payload := range []string{"", "auxiliary lock", "file", "directory", "nonempty lock"} {
		t.Run(payload, func(t *testing.T) {
			body := uvEmptyNativeOutputTar(t, true, payload)
			result, err := scanUVCompleteNativeOutput(bytes.NewReader(body), PinnedPythonRuntime().PythonVersion)
			positive := payload == "" || payload == "auxiliary lock"
			if positive {
				if err != nil || result.controls == nil || len(result.controls) != 0 || len(result.lock) == 0 {
					t.Fatal("valid explicit empty output rejected", err)
				}
			} else if err == nil || len(result.lock) != 0 {
				t.Fatal("private payload became empty complete selection", err)
			}
			if _, err := scanUVNativeOutput(bytes.NewReader(body)); err == nil {
				t.Fatal("single-root empty refusal relaxed")
			}
		})
	}
	if _, err := scanUVCompleteNativeOutput(bytes.NewReader(uvEmptyNativeOutputTar(t, false, "")), PinnedPythonRuntime().PythonVersion); err == nil {
		t.Fatal("missing site became empty census")
	}
	if _, err := scanUVCompleteNativeOutput(bytes.NewReader(uvEmptyNativeOutputTar(t, true, "")), "3.13.1"); err == nil {
		t.Fatal("empty output bypassed Python target")
	}
}
