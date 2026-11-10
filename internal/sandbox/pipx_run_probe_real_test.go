//go:build linux && realcorpus

package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/hosttool"
	"os"
	"testing"
	"time"
)

func TestLinuxPipxFrozenRunPreparationProbe(t *testing.T) {
	if os.Getenv("HELOX_PIPX_RUN_PREPARATION_PROBE") != "1" {
		t.Skip("requires explicit fixed original inputs and observed native DATA probe")
	}
	if os.Geteuid() == 0 {
		t.Fatal("ordinary caller required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	input := pipxDiagnosticFrozenInput(t)
	frozen, err := freezePipxNativeInput(ctx, input.lock, input.inputs, input.pages)
	if err != nil {
		t.Fatal(err)
	}
	requirement, err := artifactpypi.ParseUpgradeRequirement("userpath==" + input.version)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(os.Getenv("HELOX_PIPX_RUNTIME_ZIP"))
	if err != nil {
		t.Fatal(err)
	}
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	client, err := hosttool.NewSystemObservationResourceClient()
	if err != nil {
		t.Fatal(err)
	}
	resources := &pipxActualResources{ObservationResourceClient: integrationObservationResources{client: client}}
	runner := admissionAwareRunner(&pipxNativeSessionActualRunner{CommandRunner: executor})
	selected, runErr := runPipxFrozenToolRun(ctx, runner, supervisor.Observer(), resources, &pipxPreparedTool{body: body}, frozen, requirement, "userpath")
	entry := selected.Entry()
	if runErr == nil && (!entry.Valid() || entry.Command() != "userpath" || len(selected.Resolution().Graph().Nodes()) != 2 || selected.Resolution().RuntimeIdentity() != "pipx-run-isolated:"+PipxResolverRuntimeIdentity() || len(selected.Resolution().ProjectControls()) != 0) {
		t.Fatal("native DATA is not bound to the original console")
	}
	resources.mu.Lock()
	creates, registers, closes, cpu := resources.creates, resources.registers, resources.closes, resources.cpu
	resources.mu.Unlock()
	t.Logf("pipx_run_probe original_graph=%d synthetic_lock=true creates=%d registers=%d closes=%d cpu_usec=%d observer_reason=%s selection=BOUND_DATA Policy=NOT_RUN error=%v", len(frozen.graph.Nodes()), creates, registers, closes, cpu, integrationObserverFaultReason(supervisor), runErr)
	if runErr != nil || creates != 1 || registers != 1 || closes != 1 || cpu == 0 || cpu >= 13_000_000 || integrationObserverFaultReason(supervisor) != "" {
		t.Fatal("native pipx run DATA preparation incomplete", runErr)
	}
}

func TestPipxRunPreparationCensusBinding(t *testing.T) {
	filename := os.Getenv("HELOX_PIPX_RUN_PREPARATION_CENSUS")
	if filename == "" {
		t.Skip("requires the recorded fixed-original preparation DATA census")
	}
	input := pipxDiagnosticFrozenInput(t)
	frozen, err := freezePipxNativeInput(context.Background(), input.lock, input.inputs, input.pages)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	var facts []uvRunCensusFact
	if err := json.Unmarshal(body, &facts); err != nil {
		t.Fatal(err)
	}
	entry, err := bindPipxRunConsoleData(frozen, "userpath", facts)
	if err != nil || !entry.Valid() || entry.Command() != "userpath" {
		t.Fatal("recorded native preparation differs from whole frozen original", err)
	}
}

func TestPipxPublicToolRunResolverIntegration(t *testing.T) {
	if os.Getenv("HELOX_PIPX_TOOL_RUN_RESOLVER_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user production pipx run resolver qualification")
	}
	for _, mode := range []string{"normal", "incomplete-refused"} {
		if !t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			supervisor := integrationObserverSupervisor(t)
			defer func() {
				if err := supervisor.Close(); err != nil {
					t.Error(err)
				}
			}()
			executor, err := hosttool.NewSystem(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := executor.Close(); err != nil {
					t.Error(err)
				}
			}()
			client, err := hosttool.NewSystemObservationResourceClient()
			if err != nil {
				t.Fatal(err)
			}
			resources := &pipxActualResources{ObservationResourceClient: integrationObservationResources{client: client}}
			// Reuse the existing actual trace recorder and its bounded EOF fault.
			observed := &uvRunActualObserver{TraceObserver: supervisor.Observer(), rejectFrozen: mode == "incomplete-refused"}
			resolver, err := NewLinuxPipxResolverWithExecutorAndPolicy(executor, observed, integrationResolverPolicyService(t), resources)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := resolver.Close(); err != nil {
					t.Error(err)
				}
			}()
			requirement, err := artifactpypi.ParseUpgradeRequirement("cowsay@6.1")
			if err != nil {
				t.Fatal(err)
			}
			selected, resolveErr := resolver.ResolveToolRun(ctx, requirement, "cowsay")
			entry, resolution := selected.Entry(), selected.Resolution()
			if mode == "normal" {
				if resolveErr != nil || !entry.Valid() || entry.Command() != "cowsay" || entry.Artifact().Digest().String() != "274b1e6fc1b966d53976333eb90ac94cb07a450a700b455af9fbdf882244b30a" || len(resolution.Graph().Nodes()) != 1 || len(resolution.ProjectControls()) != 0 || resolution.RuntimeIdentity() != "pipx-run-isolated:"+PipxResolverRuntimeIdentity() || resolution.LockfileDigest().String() == "" {
					t.Fatal("production pipx selection/source/console differs", resolveErr)
				}
			} else if resolveErr == nil || entry.Valid() || len(resolution.Graph().Nodes()) != 0 {
				t.Fatal("incomplete observation published pipx selection", resolveErr)
			}
			if len(observed.traces) < 3 {
				t.Fatal("public source/native observation missing")
			}
			frozen, injected := 0, 0
			for _, trace := range observed.traces {
				if !trace.rawComplete || !trace.deliveredComplete && !trace.injected {
					t.Fatal("unexpected actual incomplete trace", trace.phase)
				}
				if trace.phase == pipxNativeResolverProfile {
					frozen++
					for _, kind := range []string{"filesystem-outside-workspace", "network-attempt", "honeytoken-access", "process-exec-unexpected"} {
						if trace.kinds[kind] != 0 {
							t.Fatal("pipx offline boundary crossed", kind)
						}
					}
				}
				if trace.injected {
					injected++
				}
				t.Logf("pipx_run_resolver_trace case=%s phase=%s raw_complete=%t delivered_complete=%t injected=%t events=%d bytes=%d kinds=%v", mode, trace.phase, trace.rawComplete, trace.deliveredComplete, trace.injected, trace.events, trace.bytes, trace.kinds)
			}
			if frozen != 1 || mode == "normal" && injected != 0 || mode == "incomplete-refused" && injected != 1 {
				t.Fatal("native trace fault coverage differs")
			}
			resources.mu.Lock()
			creates, registers, closes, cpu := resources.creates, resources.registers, resources.closes, resources.cpu
			resources.mu.Unlock()
			if creates != 1 || registers != 1 || closes != 1 || cpu == 0 || cpu >= 13_000_000 {
				t.Fatal("native pipx accounting differs")
			}
			t.Logf("pipx_run_public case=%s graph=%d creates=%d registers=%d closes=%d cpu_usec=%d error=%v", mode, len(resolution.Graph().Nodes()), creates, registers, closes, cpu, errors.Join(resolveErr, ctx.Err()))
		}) {
			return
		}
	}
}
