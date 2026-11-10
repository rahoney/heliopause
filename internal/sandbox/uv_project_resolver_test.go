package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func uvProjectTestInput(t *testing.T, intent artifactpypi.UVProjectIntent) artifactpypi.UVProjectInput {
	t.Helper()
	body := []byte("[project]\nname='haa-project'\nversion='0.1.0'\nrequires-python='==" + PinnedPythonRuntime().PythonVersion + "'\ndependencies=[]\n")
	selector := ""
	if intent == artifactpypi.UVProjectAdd {
		selector = "packaging==25.0"
	}
	input, err := artifactpypi.ParseUVProjectInput(body, nil, false, intent, selector)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func TestUVProjectResolverPreflightRejectsBeforeProbeNetworkAndTool(t *testing.T) {
	for _, name := range []string{"nil context", "canceled", "selector", "missing", "duplicate", "runtime", "selected export", "malformed original lock", "invalid add selector", "control-only sync", "control-only add"} {
		t.Run(name, func(t *testing.T) {
			_, runner := uvReaderFixture(t)
			endpoints := &uvCountingEndpoints{}
			probes := 0
			native, err := NewUVResolver(runner, endpoints, &recordingObserver{reader: &traceReader{}}, func(context.Context) (PythonCapability, error) {
				probes++
				return availablePythonProbe(context.Background())
			}, &recordingResolverPolicyService{})
			if err != nil {
				t.Fatal(err)
			}
			resolver, err := NewUVProjectResolver(native)
			if err != nil {
				t.Fatal(err)
			}
			intent := artifactpypi.UVProjectSync
			if name == "invalid add selector" || name == "control-only add" {
				intent = artifactpypi.UVProjectAdd
			}
			controls, err := NewUVProjectOriginalControls(uvProjectTestInput(t, intent))
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			selector := ""
			switch name {
			case "nil context":
				ctx = nil
			case "canceled":
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
			case "selector":
				selector = "packaging"
			case "missing":
				controls = controls[:5]
			case "duplicate":
				controls[1] = controls[0]
			case "runtime":
				controls[3], err = domain.NewProjectControlFile(uvProjectRuntimeControlName, []byte("stale"), true)
			case "selected export":
				controls[4], err = domain.NewProjectControlFile(artifactpypi.UVProjectExportControl, []byte("selected"), true)
			case "malformed original lock":
				controls[1], err = domain.NewProjectControlFile(artifactpypi.UVNativeLockControl, []byte("version=99"), true)
			case "invalid add selector":
				controls[2], err = domain.NewProjectControlFile(artifactpypi.UVProjectIntentControl, bytes.ReplaceAll(controls[2].Body(), []byte("packaging==25.0"), []byte("https://example.org/payload")), true)
			}
			if err != nil {
				t.Fatal(err)
			}
			install := uvSyncTestContext(t)
			if strings.HasPrefix(name, "control-only") {
				install, err = domain.NewPythonProjectControlContext(install.Target())
				if err != nil {
					t.Fatal(err)
				}
			}
			if result, err := resolver.ResolveProjectLockUpdate(ctx, install, controls, selector); err == nil || result.Valid() {
				t.Fatal("invalid project request accepted")
			}
			if probes != 0 || endpoints.calls != 0 || len(runner.calls) != 0 || len(runner.inputCalls) != 0 {
				t.Fatal("rejected project reached external infrastructure")
			}
		})
	}
	if _, err := NewUVProjectResolver(nil); err == nil {
		t.Fatal("nil native infrastructure accepted")
	}
}

func TestUVProjectNamedControlTransportPreservesProducerFailures(t *testing.T) {
	for _, name := range []string{artifactpypi.UVProjectInputControl, artifactpypi.UVNativeLockControl, artifactpypi.UVProjectExportControl} {
		body := testClosureTar(t, map[string][]byte{uvProjectFileName(name): []byte("inert")}, 1000)
		runner := &uvOutputRunner{body: body}
		got, err := readUVProjectControl(context.Background(), runner, strings.Repeat("a", 64), name)
		if err != nil || string(got) != "inert" {
			t.Fatal("named control failed", name, err)
		}
		cause := errors.New("control producer failed after data")
		runner = &uvOutputRunner{body: body, err: cause}
		if got, err := readUVProjectControl(context.Background(), runner, strings.Repeat("a", 64), name); !errors.Is(err, cause) || len(got) != 0 {
			t.Fatal("control bytes replaced producer completion", err)
		}
		for _, bad := range []map[string][]byte{{"project/" + uvProjectFileName(name): []byte("x")}, {uvProjectFileName(name): []byte("x"), "extra": []byte("x")}} {
			runner = &uvOutputRunner{body: testClosureTar(t, bad, 1000)}
			if got, err := readUVProjectControl(context.Background(), runner, strings.Repeat("a", 64), name); err == nil || len(got) != 0 {
				t.Fatal("alias/extra control accepted")
			}
		}
	}
	runner := &uvOutputRunner{}
	if _, err := readUVProjectControl(context.Background(), runner, strings.Repeat("a", 64), "../uv.lock"); err == nil {
		t.Fatal("unbounded project path accepted")
	}
	if len(runner.call) != 0 {
		t.Fatal("rejected control path reached Docker")
	}
}

func TestUVProjectFrozenControlsJoinExistingCompleteCensus(t *testing.T) {
	files := map[string][]byte{artifactpypi.UVProjectInputControl: []byte("manifest"), artifactpypi.UVNativeLockControl: []byte("native"), artifactpypi.UVProjectExportControl: []byte("export")}
	runner := &recordingRunner{}
	if err := introduceUVProjectFiles(context.Background(), runner, strings.Repeat("a", 64), files, uvExplicitEmptyLock()); err != nil {
		t.Fatal(err)
	}
	expected := map[string][]byte{}
	for name, body := range files {
		expected["project/"+uvProjectFileName(name)] = body
	}
	result, err := scanUVCompleteNativeOutputWithInput(bytes.NewReader(runner.input), PinnedPythonRuntime().PythonVersion, expected)
	if err != nil || result.controls == nil || len(result.controls) != 0 {
		t.Fatal("explicit empty census rejected frozen project controls", err)
	}
	if _, err := scanUVCompleteNativeOutput(bytes.NewReader(runner.input), PinnedPythonRuntime().PythonVersion); err == nil {
		t.Fatal("old UV pip scope accepted project controls without frozen expectation")
	}
	for name := range expected {
		changed := map[string][]byte{}
		for k, v := range expected {
			changed[k] = v
		}
		changed[name] = []byte("tampered")
		if _, err := scanUVCompleteNativeOutputWithInput(bytes.NewReader(runner.input), PinnedPythonRuntime().PythonVersion, changed); err == nil {
			t.Fatal("frozen project control mismatch accepted", name)
		}
	}
	archive := tar.NewReader(bytes.NewReader(runner.input))
	seen := 0
	for {
		h, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			if h.Mode != 0400 || h.Uid != 1000 || h.Gid != 1000 {
				t.Fatal("frozen control is writable or foreign")
			}
			seen++
		}
	}
	if seen != 4 {
		t.Fatal("frozen transport omitted provenance or view")
	}
}
