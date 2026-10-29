//go:build linux && realcorpus

package sandbox

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/hosttool"
)

type uvRunActualObserver struct {
	TraceObserver
	rejectFrozen bool
	traces       []*uvRunActualTrace
}
type uvRunActualTrace struct {
	TraceReader
	phase                                    string
	reject                                   bool
	rawComplete, deliveredComplete, injected bool
	events, bytes                            uint64
	kinds                                    map[string]uint64
}

func (o *uvRunActualObserver) wrap(reader TraceReader, phase string, err error) (TraceReader, error) {
	if err != nil {
		return nil, err
	}
	trace := &uvRunActualTrace{TraceReader: reader, phase: phase, reject: o.rejectFrozen && phase == "frozen-offline", kinds: map[string]uint64{}}
	o.traces = append(o.traces, trace)
	return trace, nil
}
func (o *uvRunActualObserver) StartProfile(ctx context.Context, id, profile string) (TraceReader, error) {
	reader, err := startTrace(ctx, o.TraceObserver, id, profile)
	return o.wrap(reader, profile, err)
}
func (o *uvRunActualObserver) startProfile(ctx context.Context, id, profile string, topology []observerMountExpectation) (TraceReader, error) {
	bound, ok := o.TraceObserver.(interface {
		startProfile(context.Context, string, string, []observerMountExpectation) (TraceReader, error)
	})
	if !ok {
		return nil, errors.New("actual topology observer unavailable")
	}
	reader, err := bound.startProfile(ctx, id, profile, topology)
	return o.wrap(reader, "frozen-offline", err)
}
func (o *uvRunActualObserver) AwaitMountAnchors(ctx context.Context, id string) error {
	return awaitMountAnchors(ctx, o.TraceObserver, id)
}
func (r *uvRunActualTrace) traceBudget() traceBudget {
	if bounded, ok := r.TraceReader.(interface{ traceBudget() traceBudget }); ok {
		return bounded.traceBudget()
	}
	return defaultTraceBudget
}
func (r *uvRunActualTrace) Next(ctx context.Context) (TraceRecord, error) {
	record, err := r.TraceReader.Next(ctx)
	if errors.Is(err, io.EOF) {
		r.rawComplete = true
		if r.reject && !r.injected {
			r.injected = true
			return TraceRecord{}, errors.New("controlled new UV run observation incompleteness")
		}
		r.deliveredComplete = true
	} else if err == nil {
		r.events++
		r.bytes += record.Bytes
		count := record.Count
		if count == 0 {
			count = 1
		}
		r.kinds[record.Kind] += count
	}
	return record, err
}

func TestUVPublicToolRunResolverIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_TOOL_RUN_RESOLVER_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user production UV run resolver qualification")
	}
	for _, mode := range []string{"normal", "incomplete-refused"} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		supervisor := integrationObserverSupervisor(t)
		executor, err := hosttool.NewSystem(ctx)
		if err != nil {
			cancel()
			t.Fatal(err)
		}
		observed := &uvRunActualObserver{TraceObserver: supervisor.Observer(), rejectFrozen: mode == "incomplete-refused"}
		resolver, err := NewLinuxUVResolverWithExecutorAndPolicy(executor, observed, integrationResolverPolicyService(t))
		if err != nil {
			cancel()
			t.Fatal(err)
		}
		requirement, err := artifactpypi.ParseUpgradeRequirement("cowsay@6.1")
		if err != nil {
			cancel()
			t.Fatal(err)
		}
		selected, resolveErr := resolver.ResolveToolRun(ctx, requirement, "cowsay")
		closeErr := errors.Join(resolver.Close(), executor.Close(), supervisor.Close())
		cancel()
		if closeErr != nil {
			t.Fatal("actual production resolver cleanup", closeErr)
		}
		resolution := selected.Resolution()
		entry := selected.Entry()
		if mode == "normal" {
			if resolveErr != nil {
				t.Fatal("actual production native run resolver", resolveErr)
			}
			if resolution.RuntimeIdentity() != "uv-tool-run-isolated:"+UVResolverRuntimeIdentity() || len(resolution.Graph().Nodes()) != 1 || !entry.Valid() || entry.Command() != "cowsay" || entry.Artifact().Digest().String() != "274b1e6fc1b966d53976333eb90ac94cb07a450a700b455af9fbdf882244b30a" || entry.Metadata().SHA256 != "780e250c9ff0a37fe32ec832874a5ebb6c9692915baf899a94e549b4c195262c" || resolution.LockfileDigest().String() == "" || strings.Contains(resolution.RuntimeIdentity(), "uv-tool-install:") {
				t.Fatal("native run identity/source/console differs")
			}
		} else if resolveErr == nil || entry.Valid() || len(resolution.Graph().Nodes()) != 0 || !strings.Contains(resolveErr.Error(), "observation is incomplete") {
			t.Fatal("incomplete observation published selection or lost refusal", resolveErr)
		}
		if len(observed.traces) < 3 {
			t.Fatal("production source/native trace coverage missing", len(observed.traces))
		}
		frozen, injected := 0, 0
		for _, trace := range observed.traces {
			if trace.phase == "frozen-offline" {
				frozen++
				for _, kind := range []string{"filesystem-outside-workspace", "network-attempt", "honeytoken-access", "process-exec-unexpected"} {
					if trace.kinds[kind] != 0 {
						t.Fatal("native frozen DATA trace crossed boundary", kind)
					}
				}
			}
			if !trace.rawComplete || !trace.deliveredComplete && !trace.injected {
				t.Fatal("actual source/native trace incomplete", trace.phase)
			}
			if trace.injected {
				injected++
			}
			t.Logf("uv_run_resolver_trace case=%s phase=%s raw_complete=%t delivered_complete=%t injected_incomplete=%t events=%d bytes=%d kinds=%v", mode, trace.phase, trace.rawComplete, trace.deliveredComplete, trace.injected, trace.events, trace.bytes, trace.kinds)
		}
		if frozen != 1 || mode == "normal" && injected != 0 || mode == "incomplete-refused" && injected != 1 {
			t.Fatal("wrong native observation refusal coverage", frozen, injected)
		}
		t.Logf("uv_run_resolver case=%s source=ACTUAL_PUBLIC_INDEPENDENT original=WHOLE readonly_source=SEALED native=ISOLATED_OFFLINE source_console=BOUND record=UNAPPROVED_DATA native_terminal=EACCES Policy=NOT_RUN program=NOT_RUN selection_returned=%t", mode, entry.Valid())
	}
}
