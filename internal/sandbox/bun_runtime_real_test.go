//go:build linux && realcorpus

package sandbox

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestBunActualIsolatedNativeQueries(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_BUN_NATIVE_RUNTIME") != "1" {
		t.Fatal("requires declared ordinary-user isolated Bun runtime qualification")
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
	tool, err := prepareBunTool(ctx)
	if err != nil {
		t.Fatal(err)
	}
	volume, err := createNativeRuntimeVolume(ctx, runner, nativeRuntimeBun)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("owned_bun_runtime_volume name=%s", volume.name)
	defer func() {
		cleanup, done := resolverCleanupContext()
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
	name := "haa-bun-native-" + session.String()
	args, err := bunQueryCreateArguments(volume)
	if err != nil {
		t.Fatal(err)
	}
	args = append([]string{"create", "--name", name, "--label", "io.heliopause.bun-native=" + session.String()}, args[1:]...)
	created, err := runner.Output(ctx, "docker", args...)
	id := strings.TrimSpace(string(created))
	if err != nil || !exactObservationContainerID(id) {
		t.Fatal("Bun native create outcome unavailable", name, err)
	}
	t.Logf("owned_bun_native_container name=%s id=%s", name, id)
	var trace TraceReader
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		if _, err := runner.Output(cleanup, "docker", "rm", "--force", id); err != nil {
			t.Error("Bun native cleanup unproven", err)
		}
		if trace != nil {
			_, limitation, diagnostic := collectTraceDiagnostic(cleanup, trace)
			t.Logf("bun_native_trace limitation=%s diagnostic=%+v", limitation, diagnostic)
			if limitation != "" {
				t.Error("Bun native observation incomplete", limitation)
			}
			for _, kind := range []string{"process-exec-unexpected", "filesystem-outside-workspace", "honeytoken-access", "network-attempt"} {
				if diagnostic.KindCounts[kind] != 0 {
					t.Errorf("Bun query contains %s=%d", kind, diagnostic.KindCounts[kind])
				}
			}
		}
	}()
	trace, err = startTrace(ctx, supervisor.Observer(), id, bunResolverProfile)
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
	for _, query := range []struct{ arg, want string }{{"--version", runtimeidentity.BunVersion}, {"--revision", runtimeidentity.BunVersion + "+" + runtimeidentity.BunSourceCommitObserved[:9]}} {
		output, err := runner.Output(ctx, "docker", boundaryExecArguments(id, boundaryLaunchMode, bunToolGuestPath, query.arg)...)
		if err != nil || strings.TrimSpace(string(output)) != query.want {
			t.Fatalf("Bun isolated %s failed output=%q: %v", query.arg, output, err)
		}
		t.Logf("isolated_native_bun_query=%s result=%s", query.arg, strings.TrimSpace(string(output)))
	}
}
