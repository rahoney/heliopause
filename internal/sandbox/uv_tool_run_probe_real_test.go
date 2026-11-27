//go:build linux && realcorpus

package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

type uvToolRunProbeRecorder struct {
	CommandRunner
	called bool
	err    error
}

func (*uvToolRunProbeRecorder) RequiresDirectExecAdmission() {}

func (r *uvToolRunProbeRecorder) Output(ctx context.Context, binary string, arguments ...string) ([]byte, error) {
	r.called = true
	output, err := r.CommandRunner.Output(ctx, binary, arguments...)
	r.err = err
	return output, err
}

// This exercises native environment preparation only. The absolute selected
// command is a non-executable null device, never a package script or interpreter.
// Local raw fixture data grants no public authenticity, Policy or execution status.
func TestUVActualToolRunSelectionOnly(t *testing.T) {
	if os.Getenv("HELOX_UV_TOOL_RUN_SELECTION_ONLY_INTEGRATION") != "1" {
		t.Fatal("declare isolated native tool-run DATA-only diagnostic")
	}
	withActualUVNativeRuntime(t, func(ctx context.Context, runner CommandRunner, id string) {
		input, ok := runner.(inputCommandRunner)
		if !ok {
			t.Fatal("native UV stdin transport unavailable")
		}
		wheel := linuxDynamicProjectWheel(t, "haa_uv_probe", "raise RuntimeError('native selection probe must never execute package body')\n", nil)
		const filename = "haa_uv_probe-1.0-py3-none-any.whl"
		runtime := PinnedPythonRuntime()
		sum := sha256.Sum256(wheel)
		inspected, err := artifactpypi.InspectWheel(bytes.NewReader(wheel), int64(len(wheel)), filename, fmt.Sprintf("%x", sum), artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}, artifactpypi.DefaultWheelLimits())
		if err != nil || inspected.Project != "haa-uv-probe" || inspected.Version != "1.0" || len(inspected.Surface.SiteHookLines) != 0 {
			t.Fatal("controlled whole fixture metadata changed", err)
		}
		var transport bytes.Buffer
		archive := tar.NewWriter(&transport)
		if err := archive.WriteHeader(&tar.Header{Name: "native-wheels/", Typeflag: tar.TypeDir, Mode: 0o700, Uid: 1000, Gid: 1000}); err != nil {
			t.Fatal(err)
		}
		if err := archive.WriteHeader(&tar.Header{Name: "native-wheels/" + filename, Typeflag: tar.TypeReg, Size: int64(len(wheel)), Mode: 0o400, Uid: 1000, Gid: 1000}); err != nil {
			t.Fatal(err)
		}
		if _, err := archive.Write(wheel); err != nil {
			t.Fatal(err)
		}
		if err := archive.Close(); err != nil {
			t.Fatal(err)
		}
		if err := input.RunInput(ctx, &transport, "docker", "cp", "--archive", "-", id+":"+pythonSitePath); err != nil {
			t.Fatal("native inert fixture copy failed", err)
		}
		admitted, ok := runner.(*admissionAwareCommandRunner)
		if !ok {
			t.Fatal("actual native fixture lost its admission-aware validated Host executor")
		}
		recorder := &uvToolRunProbeRecorder{CommandRunner: admitted.runner}
		args := []string{uvToolGuestPath, "--verbose", "--verbose", "tool", "run", "--isolated", "--from", "haa-uv-probe==1.0", "--offline", "--no-index", "--find-links", pythonSitePath + "/native-wheels", "--no-build", "--no-config", "--no-cache", "--no-python-downloads", "--no-managed-python", "--python", "/usr/local/bin/python", "--link-mode", "copy", "--no-sources", "/dev/null"}
		output, runErr := admissionAwareRunner(recorder).Output(ctx, "docker", boundaryExecArguments(id, boundaryLaunchMode, args...)...)
		var nativeExit *exec.ExitError
		stderr := ""
		if errors.As(recorder.err, &nativeExit) {
			stderr = string(nativeExit.Stderr)
		}
		t.Logf("uv_tool_run_probe_native called=%t command_error=%v stderr_bytes=%d stderr_prefix=%q", recorder.called, recorder.err, len(stderr), stderr[:min(len(stderr), 64<<10)])
		var exited *exec.ExitError
		if !errors.As(runErr, &exited) || exited.ExitCode() == 0 || len(output) != 0 {
			t.Fatal("selection-only native probe did not refuse the non-executable terminal", runErr)
		}
		stderr = string(exited.Stderr)
		if !strings.Contains(stderr, "/dev/null") || !strings.Contains(stderr, "Permission denied") {
			t.Fatalf("native preparation failed before the expected non-executable terminal: %v; stderr=%q", runErr, stderr)
		}
		t.Logf("uv_tool_run_probe phase=prepared expected_terminal=EACCES selected_program=NOT_RUN fixture_source=LOCAL_RAW public_authentication=NOT_RUN Policy=NOT_RUN wheel_sha256=%x", sum)
	})
}
