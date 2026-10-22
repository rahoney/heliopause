//go:build linux && realcorpus

package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// This opt-in probe requires a daemon that actually sees the selected bundle,
// exact images and the candidate observer. A client-only mount view is insufficient.
func TestUVActualIsolatedNativeStartup(t *testing.T) {
	withActualUVNativeRuntime(t, func(ctx context.Context, runner CommandRunner, id string) {
		version, err := runner.Output(ctx, "docker", boundaryExecArguments(id, boundaryLaunchMode, uvToolGuestPath, "--version")...)
		if err != nil || !strings.HasPrefix(strings.TrimSpace(string(version)), "uv "+runtimeidentity.UVVersion+" ") {
			t.Fatal("isolated native UV startup failed", err)
		}
		t.Logf("isolated_native_uv_version=%s", strings.TrimSpace(string(version)))
	})
}

// The empty private target still triggers the native interpreter query. It
// does not require UV to inspect the image's globally installed pip metadata.
func TestUVActualIsolatedEmptyTargetQuery(t *testing.T) {
	withActualUVNativeRuntime(t, func(ctx context.Context, runner CommandRunner, id string) {
		body, err := runner.Output(ctx, "docker", boundaryExecArguments(id, boundaryLaunchMode, uvToolGuestPath,
			"pip", "list", "--target", pythonSitePath, "--python", "/usr/local/bin/python", "--offline", "--no-config", "--no-cache", "--no-python-downloads", "--format", "json")...)
		if err != nil {
			t.Fatal("isolated native UV empty-target interpreter query failed", err)
		}
		if strings.TrimSpace(string(body)) != "[]" {
			t.Fatal("isolated native UV empty target unexpectedly contains packages")
		}
		t.Log("isolated_native_uv_empty_target_query=[]")
	})
}

// This fixture checks native output transport only. Its local wheel source is
// intentionally outside the public-source lock contract and never promoted.
func TestUVActualIsolatedNativeLockCollection(t *testing.T) {
	withActualUVNativeRuntime(t, func(ctx context.Context, runner CommandRunner, id string) {
		input, ok := runner.(inputCommandRunner)
		if !ok {
			t.Fatal("native UV stdin transport unavailable")
		}
		wheel := linuxDynamicProjectWheel(t, "haa_uv_probe", "raise RuntimeError('probe must never execute')\n", nil)
		const filename = "haa_uv_probe-1.0-py3-none-any.whl"
		var transport bytes.Buffer
		archive := tar.NewWriter(&transport)
		if err := archive.WriteHeader(&tar.Header{Name: "site/", Typeflag: tar.TypeDir, Mode: 0o700, Uid: 1000, Gid: 1000}); err != nil {
			t.Fatal(err)
		}
		if err := archive.WriteHeader(&tar.Header{Name: "site/" + filename, Typeflag: tar.TypeReg, Size: int64(len(wheel)), Mode: 0o644, Uid: 1000, Gid: 1000}); err != nil {
			t.Fatal(err)
		}
		if _, err := archive.Write(wheel); err != nil {
			t.Fatal(err)
		}
		if err := archive.Close(); err != nil {
			t.Fatal(err)
		}
		if err := input.RunInput(ctx, &transport, "docker", "cp", "--archive", "-", id+":"+pythonSitePath); err != nil {
			t.Fatal("native inert wheel data copy failed", err)
		}
		common := []string{"--offline", "--no-build", "--no-config", "--no-cache", "--no-python-downloads", "--no-managed-python", "--python", "/usr/local/bin/python"}
		compile := []string{"pip", "compile", "-", "--format", "pylock.toml", "--output-file", pythonSitePath + "/pylock.toml", "--no-index", "--python-version", PinnedPythonRuntime().PythonVersion, "--python-platform", "x86_64-manylinux_2_36"}
		compile = append(compile, common...)
		requirement := "haa-uv-probe @ file://" + pythonSitePath + "/site/" + filename + "\n"
		if err := input.RunInput(ctx, strings.NewReader(requirement), "docker", boundaryInputExecArguments(id, boundaryLaunchMode, append([]string{uvToolGuestPath}, compile...)...)...); err != nil {
			t.Fatal("native UV stdin compile failed", err)
		}
		sync := append([]string{"pip", "sync", pythonSitePath + "/pylock.toml", "--target", pythonSitePath + "/site", "--no-index", "--link-mode", "copy"}, common...)
		if _, err := runner.Output(ctx, "docker", boundaryExecArguments(id, boundaryLaunchMode, append([]string{uvToolGuestPath}, sync...)...)...); err != nil {
			t.Fatal("native UV pylock materialization failed", err)
		}
		data, err := readUVNativeOutput(ctx, runner, id)
		if err != nil || len(data.controls) != 1 || data.controls[0].DistInfo != "haa_uv_probe-1.0.dist-info" {
			t.Fatal("native UV output collection failed", err)
		}
		reference, _ := artifactpypi.ParseReference("haa-uv-probe@1.0")
		if _, err := artifactpypi.ParseUVLock(reference, data.lock, PinnedPythonRuntime().PythonVersion); err == nil {
			t.Fatal("local fixture source became an approved public lock")
		}
		t.Logf("isolated_native_uv_lock_collection=OK controls=%d lock_bytes=%d", len(data.controls), len(data.lock))
	})
}

func withActualUVNativeRuntime(t *testing.T, operation func(context.Context, CommandRunner, string)) {
	t.Helper()
	if os.Geteuid() == 0 || os.Getenv("HELOX_UV_NATIVE_RUNTIME") != "1" {
		t.Fatal("requires declared ordinary-user isolated UV runtime qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	runner := admissionAwareRunner(executor)
	tool, err := prepareUVTool(ctx, runner)
	if err != nil {
		t.Fatal(err)
	}
	volume, err := createUVRuntimeVolume(ctx, runner)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("owned_uv_runtime_volume name=%s", volume.name)
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		if err := volume.close(cleanup, runner); err != nil {
			t.Error(err)
		}
	}()
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	name := "haa-uv-native-" + session.String()
	output, err := createClosureVolume(ctx, runner, session.String()+"-uv-output", runtimeidentity.UVToolSHA256, artifactpypi.PublicPyPIProfile().ResourcePolicy().PromotionTmpfs())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("owned_uv_output_volume name=%s", output.name)
	defer func() {
		cleanup, done := resolverCleanupContext()
		defer done()
		if err := closeUVOutputVolume(cleanup, runner, &output); err != nil {
			t.Error(err)
		}
	}()
	baseArguments, err := uvCreateArguments("none", nil, volume, output)
	if err != nil {
		t.Fatal(err)
	}
	arguments := append([]string{"create", "--name", name, "--label", "io.heliopause.uv-native=" + session.String()}, baseArguments[1:]...)
	created, err := runner.Output(ctx, "docker", arguments...)
	id := strings.TrimSpace(string(created))
	if err != nil || !exactObservationContainerID(id) {
		t.Fatal("UV native create outcome unavailable", name, err)
	}
	t.Logf("owned_uv_native_container name=%s id=%s", name, id)
	var trace TraceReader
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		if _, err := runner.Output(cleanup, "docker", "rm", "--force", id); err != nil {
			t.Error("UV native cleanup unproven", err)
		}
		if trace != nil {
			_, limitation, diagnostic := collectTraceDiagnostic(cleanup, trace)
			t.Logf("uv_native_trace limitation=%s diagnostic=%+v", limitation, diagnostic)
			if limitation != "" {
				t.Error("UV native observation incomplete", limitation)
			}
		}
	}()
	trace, err = startTrace(ctx, supervisor.Observer(), id, uvResolverProfile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Output(ctx, "docker", "start", id); err != nil {
		t.Fatal(err)
	}
	if err := awaitMountAnchors(ctx, supervisor.Observer(), id); err != nil {
		t.Fatal(err)
	}
	if err := awaitBoundaryHelper(ctx, runner, id); err != nil {
		t.Fatal(err)
	}
	if err := volume.verifyTarget(ctx, runner, id); err != nil {
		t.Fatal(err)
	}
	if err := output.verifyContainerMount(ctx, runner, id, false); err != nil {
		t.Fatal(err)
	}
	if err := output.verifyAttachments(ctx, runner, map[string]bool{id: false}); err != nil {
		t.Fatal(err)
	}
	if err := tool.introduce(ctx, runner, id); err != nil {
		t.Fatal(err)
	}
	if err := volume.verifyTarget(ctx, runner, id); err != nil {
		t.Fatal(err)
	}
	operation(ctx, runner, id)
}
