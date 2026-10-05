package sandbox

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// This initial minimal gate proves the actual registered image/role/open
// chain before introducing runtime-read semantics. It is not build Policy,
// retained cache, output publication or CLI qualification.
func TestLinuxGoBuildIntegration(t *testing.T) {
	if os.Getenv("HELOX_GO_BUILD_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	runner := admissionAwareRunner(integrationRunner{t: t})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	capability, err := ProbeGo(ctx, integrationRunner{t: t})
	if err != nil || !capability.Available || capability.Runtime != PinnedGoRuntime() {
		t.Fatalf("pinned Go build runtime: %v", err)
	}
	created, err := runner.Output(ctx, "docker", goBuildCreateArguments()...)
	id := strings.TrimSpace(string(created))
	if err != nil || !containerIDPattern.MatchString(id) {
		t.Fatalf("create Go build runtime: %s", pythonCommandErrorReason(err))
	}
	var trace TraceReader
	phase := "REGISTER"
	var primary error
	defer func() {
		cleanupCtx, cleanupCancel := resolverCleanupContext()
		_, cleanup := runner.Output(cleanupCtx, "docker", "rm", "--force", id)
		cleanupCancel()
		if trace == nil {
			if primary != nil || cleanup != nil {
				t.Errorf("go build phase=%s primary=%s cleanup=%s", phase, pythonCommandErrorReason(primary), pythonCommandErrorReason(cleanup))
			}
			return
		}
		collectCtx, collectCancel := resolverCleanupContext()
		_, limitation, diagnostic := collectTraceDiagnostic(collectCtx, trace)
		collectCancel()
		for _, kind := range []string{"process-exec-unexpected", "filesystem-outside-workspace", "honeytoken-access", "network-attempt"} {
			if diagnostic.KindCounts[kind] != 0 {
				t.Errorf("go build boundary signal=%s count=%d", kind, diagnostic.KindCounts[kind])
			}
		}
		if primary != nil || cleanup != nil || limitation != "" {
			t.Errorf("go build phase=%s primary=%s cleanup=%s trace={%s}", phase, pythonCommandErrorReason(primary), pythonCommandErrorReason(cleanup), diagnostic.String())
		}
	}()
	trace, primary = startTrace(ctx, supervisor.Observer(), id, goBuildProfile)
	if primary != nil {
		return
	}
	phase = "START"
	if _, primary = runner.Output(ctx, "docker", "start", id); primary != nil {
		return
	}
	phase = "HELPER"
	if primary = awaitBoundaryHelper(ctx, runner, id); primary != nil {
		return
	}
	phase = "TOPOLOGY"
	if primary = awaitMountAnchors(ctx, supervisor.Observer(), id); primary != nil {
		return
	}
	input, ok := runner.(inputCommandRunner)
	if !ok {
		primary = errors.New("missing bounded build input")
		return
	}
	phase = "CONFIG"
	if primary = input.RunInput(ctx, strings.NewReader("off\n"), "docker", boundaryInputExecArguments(id, boundaryLaunchMode, "/bin/sh", "-ceu", "umask 077; mkdir -p /tmp/.config/go/telemetry; cat > /tmp/.config/go/telemetry/mode")...); primary != nil {
		return
	}
	phase = "SOURCE"
	for _, member := range []struct{ name, body string }{{"go.mod", "module example.com/haa-build-fixture\ngo 1.26\n"}, {"main.go", "package main\nfunc main() {}\n"}} {
		command := "umask 077; mkdir -p " + goResolverGuestProject + "; cat > " + goResolverGuestProject + "/" + member.name
		if primary = input.RunInput(ctx, strings.NewReader(member.body), "docker", boundaryInputExecArguments(id, boundaryLaunchMode, "/bin/sh", "-ceu", command)...); primary != nil {
			return
		}
	}
	phase = "BUILD"
	_, primary = runner.Output(ctx, "docker", boundaryExecArguments(id, boundaryELFHandoffMode, goResolverBinary, "-C", goResolverGuestProject, "build", "-mod=readonly", "-o", "/tmp/haa-go-output", ".")...)
}
