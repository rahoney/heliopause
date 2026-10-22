//go:build linux && realcorpus

package sandbox

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

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

func TestUVActualIsolatedPythonInventory(t *testing.T) {
	withActualUVNativeRuntime(t, func(ctx context.Context, runner CommandRunner, id string) {
		body, err := runner.Output(ctx, "docker", boundaryExecArguments(id, boundaryLaunchMode, uvToolGuestPath,
			"pip", "list", "--python", "/usr/local/bin/python", "--offline", "--no-config", "--no-cache", "--no-python-downloads", "--format", "json")...)
		if err != nil {
			t.Fatal("isolated native UV Python inventory failed", err)
		}
		var packages []struct{ Name, Version string }
		if len(body) > 16<<10 || json.Unmarshal(body, &packages) != nil || len(packages) != 1 || packages[0].Name != "pip" || packages[0].Version != runtimeidentity.PipVersion {
			t.Fatal("isolated native UV inventory differs from the pinned pure Python image")
		}
		t.Logf("isolated_native_uv_python_inventory=%s", strings.TrimSpace(string(body)))
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
	baseArguments, err := uvCreateArguments("none", nil, volume)
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
	if err := tool.introduce(ctx, runner, id); err != nil {
		t.Fatal(err)
	}
	if err := volume.verifyTarget(ctx, runner, id); err != nil {
		t.Fatal(err)
	}
	operation(ctx, runner, id)
}
