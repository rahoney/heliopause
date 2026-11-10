package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type uvSelectionInputRunner struct {
	body      []byte
	arguments []string
	calls     int
}

func (r *uvSelectionInputRunner) RunInput(_ context.Context, input io.Reader, binary string, args ...string) error {
	r.calls++
	r.arguments = append([]string{binary}, args...)
	var err error
	r.body, err = io.ReadAll(input)
	return err
}

func TestUVSelectionInputTransfersOnlyFixedMetadataAndChecksReturnedBytes(t *testing.T) {
	files := map[string][]byte{"original/probe-1.0.dist-info/METADATA": []byte("Name: probe\nVersion: 1.0\n\n"), "constraints.txt": []byte("probe==1.0\n")}
	runner := &uvSelectionInputRunner{}
	id := strings.Repeat("a", 64)
	if err := introduceUVSelectionFiles(context.Background(), runner, id, files, true); err != nil {
		t.Fatal(err)
	}
	if strings.Join(runner.arguments, " ") != "docker cp --archive - "+id+":"+pythonSitePath {
		t.Fatal("input escaped the fixed mount", runner.arguments)
	}
	archive := tar.NewReader(bytes.NewReader(runner.body))
	count := 0
	for {
		h, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if h.Uid != 1000 || h.Gid != 1000 {
			t.Fatal("input changed native owner")
		}
		if h.Typeflag == tar.TypeReg {
			count++
			body, err := io.ReadAll(archive)
			if err != nil || h.Mode != 0400 || !bytes.Equal(body, files[h.Name]) {
				t.Fatal("input tar changed metadata")
			}
		}
	}
	if count != 2 {
		t.Fatal("input included additional payload", count)
	}
	for _, mode := range []string{"valid", "changed", "missing", "mode", "extra", "lock"} {
		output := uvOutputFiles()
		for name, body := range files {
			output[name] = body
		}
		if mode == "changed" {
			output["constraints.txt"] = []byte("probe==9.0\n")
		}
		if mode == "missing" {
			delete(output, "constraints.txt")
		}
		if mode == "extra" {
			output["original/probe-1.0.dist-info/RECORD"] = []byte("untrusted payload\n")
		}
		var out bytes.Buffer
		writer := tar.NewWriter(&out)
		for name, body := range output {
			permission := int64(0600)
			if files[name] != nil {
				permission = 0400
			}
			if mode == "mode" && name == "constraints.txt" {
				permission = 0600
			}
			if err := writer.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: permission, Uid: 1000, Gid: 1000, Size: int64(len(body))}); err != nil {
				t.Fatal(err)
			}
			if _, err := writer.Write(body); err != nil {
				t.Fatal(err)
			}
		}
		if mode == "lock" {
			if err := writer.WriteHeader(&tar.Header{Name: "original/.lock", Typeflag: tar.TypeReg, Mode: 0666, Uid: 1000, Gid: 1000}); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		got, err := scanUVNativeOutputWithInput(bytes.NewReader(out.Bytes()), files)
		if mode == "valid" || mode == "lock" {
			if err != nil || len(got.controls) != 1 {
				t.Fatal(mode, err)
			}
		} else if err == nil {
			t.Fatal("altered original input accepted", mode)
		}
		if _, err := scanUVNativeOutput(bytes.NewReader(out.Bytes())); err == nil {
			t.Fatal("fresh resolver accepted update-only input paths")
		}
	}
	for _, name := range []string{"../../escape", "original/x.py", "original/probe-1.0.dist-info/RECORD", "/constraints.txt"} {
		before := runner.calls
		if err := introduceUVSelectionFiles(context.Background(), runner, id, map[string][]byte{name: []byte("x")}, false); err == nil || runner.calls != before {
			t.Fatal("unsafe input launched transport", name)
		}
	}
}

func TestUVSelectionRefusesInvalidFrozenRequestBeforeRuntime(t *testing.T) {
	requirement, _ := artifactpypi.ParseUpgradeRequirement("packaging==25.0")
	input, _ := artifactpypi.NewUpgradeInput(requirement, nil, PinnedPythonRuntime().PythonVersion)
	original, _ := input.Control()
	intent, _ := artifactpypi.NewUVSelectionIntentControl(artifactpypi.UVUpgradeSelection)
	for _, mode := range []string{"missing", "mismatch", "foreign", "cancel", "nil-context"} {
		_, runner := uvReaderFixture(t)
		endpoints := &uvCountingEndpoints{}
		resolver, err := NewUVResolver(runner, endpoints, &recordingObserver{reader: &traceReader{}}, func(context.Context) (PythonCapability, error) {
			return PythonCapability{}, errors.New("runtime probe must not run")
		}, &recordingResolverPolicyService{})
		if err != nil {
			t.Fatal(err)
		}
		reference := requirement.Reference()
		controls := []domain.ProjectControlFile{original, intent}
		var ctx context.Context = context.Background()
		switch mode {
		case "missing":
			controls = controls[:1]
		case "mismatch":
			reference, _ = artifactpypi.ParseReference("idna")
		case "foreign":
			profile, _ := artifactpypi.PyTorchProfile("cpu")
			reference, _ = artifactpypi.ParseReferenceForSource("torch", profile.Source())
		case "cancel":
			c, cancel := context.WithCancel(ctx)
			cancel()
			ctx = c
		case "nil-context":
			ctx = nil
		}
		result, err := resolver.ResolveDependencyUpdate(ctx, reference, domain.InstallContext{}, controls)
		if err == nil || len(result.Graph().Nodes()) != 0 || len(runner.calls) != 0 || len(runner.inputCalls) != 0 || endpoints.calls != 0 {
			t.Fatal("invalid selection reached runtime", mode, err)
		}
	}
}

func TestUVNativeDeltaOutputIsBoundedAndFailurePreserved(t *testing.T) {
	id := strings.Repeat("a", 64)
	for _, mode := range []string{"valid", "too-large", "failed"} {
		runner := &uvOutputRunner{body: []byte(`{"changes":[]}`)}
		if mode == "too-large" {
			runner.body = bytes.Repeat([]byte("x"), uvNativeLockLimit+1)
		}
		cause := errors.New("producer failed")
		if mode == "failed" {
			runner.err = cause
		}
		body, err := uvBoundedSelectionOutput(context.Background(), runner, id, []string{uvToolGuestPath, "pip", "install"})
		if mode == "valid" {
			if err != nil || !bytes.Equal(body, runner.body) {
				t.Fatal(err)
			}
		} else if err == nil || len(body) != 0 || mode == "failed" && !errors.Is(err, cause) {
			t.Fatal("invalid delta output accepted or cause lost", mode, err)
		}
	}
}

func TestUVOriginalMetadataLimitDoesNotTreatRetainedDistributionsAsSelectedGraph(t *testing.T) {
	files := map[string][]byte{}
	count := artifactpypi.PublicPyPIProfile().ResourcePolicy().MaxGraphArtifacts() + 1
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("probe%d", i)
		files["original/"+name+"-1.0.dist-info/METADATA"] = []byte("Name: " + name + "\nVersion: 1.0\n\n")
	}
	runner := &uvSelectionInputRunner{}
	if err := introduceUVSelectionFiles(context.Background(), runner, strings.Repeat("a", 64), files, true); err != nil || runner.calls != 1 {
		t.Fatal("retained original metadata was limited as a selected artifact graph", err)
	}
}
