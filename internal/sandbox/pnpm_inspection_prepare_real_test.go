//go:build linux && realcorpus

package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/hosttool"
)

type pnpmPrepareCaptureRunner struct {
	CommandRunner
	layout []byte
}

func (r *pnpmPrepareCaptureRunner) RunInput(ctx context.Context, input io.Reader, name string, args ...string) error {
	return r.CommandRunner.(inputCommandRunner).RunInput(ctx, input, name, args...)
}
func (r *pnpmPrepareCaptureRunner) RunOutput(ctx context.Context, output io.Writer, name string, args ...string) error {
	if name == "docker" && len(args) == 3 && args[0] == "cp" && strings.HasSuffix(args[1], "/node_modules") && args[2] == "-" {
		capture := nativeToolArchiveOutput{limit: artifactnpm.PNPMLayoutArchiveLimit}
		err := r.CommandRunner.(streamingDockerOutput).RunOutput(ctx, io.MultiWriter(output, &capture), name, args...)
		r.layout = capture.body
		return err
	}
	return r.CommandRunner.(streamingDockerOutput).RunOutput(ctx, output, name, args...)
}

func logPNPMPrepareLayoutFailure(t *testing.T, ctx context.Context, body []byte, inventories []artifactnpm.ArchiveInventory) {
	t.Helper()
	layout, err := artifactnpm.ParsePNPMLayoutArchive(ctx, body)
	if err != nil {
		t.Log("bounded diagnostic layout unavailable", err)
		return
	}
	type fact struct {
		Name, Link, Digest, Generated string
		Kind                          byte
		Mode, Size                    int64
	}
	var facts []fact
	for _, entry := range layout.Entries() {
		f := fact{Name: entry.Name, Link: entry.Link, Digest: entry.Digest.String(), Kind: entry.Kind, Mode: entry.Mode, Size: entry.Size}
		if (entry.Name == "node_modules/.modules.yaml" || entry.Name == "node_modules/.pnpm/lock.yaml") && entry.Size <= 64<<10 {
			content, err := layout.ReadFile(entry.Name)
			if err == nil {
				f.Generated = string(content)
			}
		}
		facts = append(facts, f)
	}
	encoded, err := json.Marshal(facts)
	if err == nil && len(encoded) <= 1<<20 {
		t.Logf("pnpm_prepare_rejected_native_layout=%s", encoded)
	}
	for _, inventory := range inventories {
		facts = nil
		for _, entry := range inventory.Entries() {
			facts = append(facts, fact{Name: entry.Path, Digest: entry.Digest.String(), Mode: entry.Mode, Size: entry.Size})
		}
		encoded, err := json.Marshal(facts)
		if err == nil && len(encoded) <= 1<<20 {
			t.Logf("pnpm_prepare_original_inventory package=%s entries=%s", inventory.Artifact().Identity().Name(), encoded)
		}
	}
}

type pnpmPrepareTrace struct {
	TraceReader
	complete bool
	kinds    map[string]uint64
}

func (r *pnpmPrepareTrace) Next(ctx context.Context) (TraceRecord, error) {
	record, err := r.TraceReader.Next(ctx)
	if errors.Is(err, io.EOF) {
		r.complete = true
	}
	if err == nil {
		r.kinds[record.Kind] += max(1, record.Count)
	}
	return record, err
}
func (r *pnpmPrepareTrace) traceBudget() traceBudget {
	if configured, ok := r.TraceReader.(interface{ traceBudget() traceBudget }); ok {
		return configured.traceBudget()
	}
	return defaultTraceBudget
}

type pnpmPrepareTraceObserver struct {
	TraceObserver
	traces []*pnpmPrepareTrace
}

func (o *pnpmPrepareTraceObserver) StartProfile(ctx context.Context, id, profile string) (TraceReader, error) {
	reader, err := startTrace(ctx, o.TraceObserver, id, profile)
	if err != nil {
		return nil, err
	}
	wrapped := &pnpmPrepareTrace{TraceReader: reader, kinds: map[string]uint64{}}
	o.traces = append(o.traces, wrapped)
	return wrapped, nil
}
func (o *pnpmPrepareTraceObserver) AwaitMountAnchors(ctx context.Context, id string) error {
	return awaitMountAnchors(ctx, o.TraceObserver, id)
}

func TestPNPMActualProductionInspectionPrepare(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_PNPM_NATIVE_RUNTIME") != "1" || os.Getenv("HELOX_PNPM_PUBLIC_METADATA_FIXTURE") != "1" || os.Getenv("HELOX_PNPM_INSPECTION_PREPARE_FIXTURE") != "1" {
		t.Fatal("declare ordinary-user production PNPM preparation and public metadata qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
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
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	observer := &pnpmPrepareTraceObserver{TraceObserver: supervisor.Observer()}
	native, err := NewLinuxPNPMResolverWithExecutorAndPolicy(executor, observer, integrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	capture := &pnpmPrepareCaptureRunner{CommandRunner: native.runner}
	native.runner = admissionAwareRunner(capture)
	for _, fixture := range pnpmOfflineFixtures() {
		if !t.Run(fixture.name, func(t *testing.T) {
			// All public acquisition/SRI/static work precedes resource creation.
			data := loadPNPMOfflineInputs(t, ctx, fixture)
			preparer, err := NewPNPMInspectionPreparer(native, data.intake)
			if err != nil {
				t.Fatal(err)
			}
			before := len(observer.traces)
			capture.layout = nil
			bound, err := preparer.PrepareInspection(ctx, data.inspection, data.inventories)
			if err != nil {
				logPNPMPrepareLayoutFailure(t, ctx, capture.layout, data.inventories)
				t.Fatal("production preparation rejected complete public inputs", err)
			}
			if len(observer.traces) != before+1 {
				t.Fatal("metadata preparation did not have one independent observed session")
			}
			trace := observer.traces[before]
			if !trace.complete {
				t.Fatal("production returned layout before complete observation")
			}
			for _, kind := range []string{"process-exec-unexpected", "filesystem-outside-workspace", "honeytoken-access", "network-attempt"} {
				if trace.kinds[kind] != 0 {
					t.Fatal("native metadata has forbidden observation", kind, trace.kinds[kind])
				}
			}
			instances := 0
			for _, subject := range data.inspection.Artifacts() {
				contexts, err := data.inspection.InstancesFor(subject)
				if err != nil {
					t.Fatal(err)
				}
				for _, instance := range contexts {
					if _, err := bound.Target(subject, instance.Key); err != nil {
						t.Fatal(err)
					}
					instances++
				}
			}
			if _, err := bound.ReadFile("node_modules/.pnpm-workspace-state-v1.json"); err == nil {
				t.Fatal("native freshness cache exposed")
			}
			if len(bound.InspectionManifest()) == 0 || bound.Digest().String() == "" {
				t.Fatal("production preparation returned an empty binding")
			}
			t.Logf("production_pnpm_prepare case=%s archives=%d instances=%d layout_digest=%s trace_complete=%t forbidden=0 kinds=%v native_runs=1 scripts=NOT_EXECUTED dynamic=NOT_EXECUTED Host_install=NOT_EXECUTED cleanup_before_return=true", fixture.name, len(data.inventories), instances, bound.Digest().String(), trace.complete, trace.kinds)
			if fixture.name == "transitive" {
				// The callback has already computed a usable binding when the
				// real removal succeeds but its acknowledgement is faulted.
				// The production consumer must discard that captured value.
				cause := errors.New("injected production cleanup acknowledgement failure")
				runner := &pnpmNativeFaultRunner{CommandRunner: admissionAwareRunner(executor), mode: "cleanup acknowledgement", cause: cause}
				faultNative, err := NewPNPMResolver(runner, systemEndpointResolver{}, observer, integrationCapabilityProbe(executor), integrationResolverPolicyService(t))
				if err != nil {
					t.Fatal(err)
				}
				faultPreparer, err := NewPNPMInspectionPreparer(faultNative, data.intake)
				if err != nil {
					t.Fatal(err)
				}
				before := len(observer.traces)
				discarded, err := faultPreparer.PrepareInspection(ctx, data.inspection, data.inventories)
				if !errors.Is(err, cause) || discarded.Digest().String() != "" || runner.cleanupFaults != 1 || len(observer.traces) != before+1 || !observer.traces[before].complete {
					t.Fatal("cleanup failure retained bound output or lost its original cause", err)
				}
				t.Log("production_pnpm_prepare_negative cleanup_ack_failure=1 bound_value_discarded=true first_cause_preserved=true trace_complete=true public_inputs_reused=true")
			}
		}) {
			t.FailNow()
		}
	}
}
