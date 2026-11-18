package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// Stateful Docker format fixture: it supplies no native execution evidence.
// Existing volume checks consume the same actual/configured mount views.
type pnpmInstanceRunner struct {
	*environmentBackendModel
	mu                                                   sync.Mutex
	rootOpen                                             bool
	kind                                                 nativeRuntimeToolKind
	t                                                    *testing.T
	volume                                               dockerVolumeInspection
	id                                                   string
	active                                               bool
	payload                                              []byte
	creates, executions, removals, volumeRemovals, calls int
	volumes, ids                                         []string
	events                                               []string
	fault                                                string
	runErr, cleanupErr                                   error
	cancel                                               context.CancelCauseFunc
}

func (r *pnpmInstanceRunner) projectSpec() nativeProjectSpec {
	kind := r.kind
	if kind == 0 {
		kind = nativeRuntimePNPM
	}
	spec, ok := nativeProjectIdentity(kind)
	if !ok {
		r.t.Fatal("unknown fixture native kind")
	}
	return spec
}

func (r *pnpmInstanceRunner) encoded(v any) []byte {
	r.t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		r.t.Fatal(err)
	}
	return b
}
func (r *pnpmInstanceRunner) Output(ctx context.Context, binary string, args ...string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.t.Helper()
	if r.environmentBackendModel == nil {
		r.environmentBackendModel = newEnvironmentBackendState()
	}
	r.calls++
	if binary != "docker" || len(args) == 0 {
		r.t.Fatal("unexpected command")
	}
	r.events = append(r.events, args[0])
	if args[0] == "volume" {
		body, err := r.environmentBackendModel.Output(ctx, binary, args...)
		if args[1] == "create" && err == nil {
			name := args[len(args)-1]
			r.volume = r.environmentBackendModel.volumes[name]
			r.volumes = append(r.volumes, name)
		}
		if args[1] == "rm" && err == nil {
			r.volumeRemovals++
		}
		return body, err
	}
	switch args[0] {
	case "create":
		body, err := r.environmentBackendModel.Output(ctx, binary, args...)
		if err == nil {
			r.creates++
			r.id = string(body)
			r.ids = append(r.ids, r.id)
			r.active = true
		}
		return body, err
	case "ps":
		if r.fault == "foreign-writer" {
			return []byte(strings.Repeat("e", 64)), nil
		}
		return r.environmentBackendModel.Output(ctx, binary, args...)
	case "inspect":
		if args[2] == containerMountInspectTemplate {
			body, err := r.environmentBackendModel.Output(ctx, binary, args...)
			if err != nil {
				return nil, err
			}
			if r.fault == "mount-drift" {
				var view dockerContainerMountInspection
				_ = json.Unmarshal(body, &view)
				view.Mounts[0].Source = "/foreign"
				return r.encoded(view), nil
			}
			return body, nil
		}
		if args[2] == "{{.HostConfig.NetworkMode}}" {
			if r.fault == "network-query" {
				return nil, errors.New("network query unavailable")
			}
			return []byte("none"), nil
		}
		if args[2] == "{{json .NetworkSettings.Networks}}" {
			if r.fault == "network-address" {
				return []byte(`{"none":{"NetworkID":"` + strings.Repeat("b", 64) + `","IPAddress":"1.2.3.4"}}`), nil
			}
			if r.fault == "network-null-json" {
				return []byte("null"), nil
			}
			return []byte(`{"none":{"NetworkID":"` + strings.Repeat("b", 64) + `"}}`), nil
		}
		return r.environmentBackendModel.Output(ctx, binary, args...)
	case "network":
		return []byte(`{"Name":"none","Id":"` + strings.Repeat("b", 64) + `","Driver":"null","Scope":"local"}`), nil
	case "start":
		return nil, r.environmentBackendModel.RunDiscard(ctx, binary, args...)
	case "exec":
		if slices.Contains(args, pnpmTargetLifecycleCommand) || slices.Contains(args, bunTargetLifecycleCommand) || slices.Contains(args, bunRuntimeTargetLifecycleCommand) {
			r.executions++
			r.environmentBackendModel.mu.Lock()
			r.environmentBackendModel.events += 2
			r.environmentBackendModel.mu.Unlock()
			if r.cancel != nil {
				r.cancel(r.runErr)
				return nil, ctx.Err()
			}
			return nil, r.runErr
		}
		if len(args) < 4 || !slices.Equal(args, boundaryReadinessArguments(args[3])) {
			r.t.Fatal("unexpected exec", args)
		}
		return nil, nil
	case "rm":
		r.environmentBackendModel.mu.Lock()
		present := r.environmentBackendModel.containers[args[len(args)-1]] != nil
		r.environmentBackendModel.mu.Unlock()
		body, err := r.environmentBackendModel.Output(ctx, binary, args...)
		r.active = false
		if present {
			r.removals++
		}
		return body, errors.Join(err, r.cleanupErr)
	}
	r.t.Fatal("unhandled command", args)
	return nil, nil
}
func (r *pnpmInstanceRunner) RunDiscard(ctx context.Context, binary string, args ...string) error {
	_, err := r.Output(ctx, binary, args...)
	return err
}
func (r *pnpmInstanceRunner) Create(ctx context.Context, owner, profile string) (ObservationResourceLease, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.rootOpen {
		return ObservationResourceLease{}, errors.New("instance root budget replenished")
	}
	f := r.environmentBackendModel
	f.mu.Lock()
	f.creates = 0
	f.registers = 0
	f.budget = nil
	f.events = 0
	f.usage = 0
	f.snapshots = 0
	f.mu.Unlock()
	lease, err := f.Create(ctx, owner, profile)
	r.rootOpen = err == nil
	return lease, err
}
func (r *pnpmInstanceRunner) Terminate(ctx context.Context, owner, id string) (ObservationResourceLease, error) {
	f := r.environmentBackendModel
	f.mu.Lock()
	present := f.containers[id] != nil
	f.mu.Unlock()
	lease, err := f.Terminate(ctx, owner, id)
	r.mu.Lock()
	defer r.mu.Unlock()
	if present {
		r.removals++
	}
	r.active = false
	return lease, errors.Join(err, r.cleanupErr)
}
func (r *pnpmInstanceRunner) Close(ctx context.Context, owner string) (ObservationResourceLease, error) {
	lease, err := r.environmentBackendModel.Close(ctx, owner)
	r.mu.Lock()
	defer r.mu.Unlock()
	if err == nil {
		r.rootOpen = false
	}
	return lease, err
}

func (r *pnpmInstanceRunner) RunInput(ctx context.Context, input io.Reader, binary string, args ...string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	_, destination, _ := strings.Cut(args[len(args)-1], ":")
	if destination == "/tmp" {
		return r.environmentBackendModel.RunInput(ctx, input, binary, args...)
	}
	r.events = append(r.events, "copy")
	if binary != "docker" || !slices.Equal(args, []string{"cp", "--archive", "-", r.id + ":" + r.projectSpec().root}) {
		r.t.Fatal("copy destination is not fixed")
	}
	if r.fault == "copy" {
		return errors.New("copy refused")
	}
	var err error
	r.payload, err = io.ReadAll(io.LimitReader(input, artifactnpm.PNPMLayoutArchiveLimit+1))
	if err != nil {
		return err
	}
	return r.environmentBackendModel.RunInput(ctx, bytes.NewReader(r.payload), binary, args...)
}
func (r *pnpmInstanceRunner) RunOutput(ctx context.Context, output io.Writer, binary string, args ...string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	if strings.HasSuffix(args[1], ":/tmp/.") {
		return r.environmentBackendModel.RunOutput(ctx, output, binary, args...)
	}
	r.events = append(r.events, "readback")
	if binary != "docker" || !slices.Equal(args, []string{"cp", r.id + ":" + r.projectSpec().directory, "-"}) {
		r.t.Fatal("readback is not whole project")
	}
	if r.fault == "readback" {
		return errors.New("readback unavailable")
	}
	body := r.payload
	if r.fault == "readback-drift" {
		body = bytes.Clone(body)
		body[0] ^= 1
	}
	_, err := output.Write(body)
	return err
}

type pnpmInstanceObserver struct {
	*environmentBackendModel
	reader TraceReader
	err    error
	ids    []string
}

func (o *pnpmInstanceObserver) Start(context.Context, string) (TraceReader, error) {
	return nil, errors.New("ordinary profile must not be used")
}
func (o *pnpmInstanceObserver) AwaitMountAnchors(context.Context, string) error { return nil }

func (o *pnpmInstanceObserver) startNPMInstanceEnvironmentProfile(ctx context.Context, id string, _ nativeRuntimeToolKind, _ bool, reader bool, budget *observationTraceLedger) (TraceReader, error) {
	o.ids = append(o.ids, id)
	if o.err != nil {
		return nil, o.err
	}
	f := o.environmentBackendModel
	f.mu.Lock()
	f.traceRecords = nil
	f.traceError = nil
	if !reader {
		f.traceRecords = []TraceRecord{{Kind: "network-attempt", Count: 2}}
		if custom, ok := o.reader.(*traceReader); ok {
			f.traceRecords = append([]TraceRecord(nil), custom.records...)
			f.traceError = custom.err
		}
	}
	f.mu.Unlock()
	return f.StartEnvironmentProfile(ctx, id, reader, budget)
}
func wirePNPMInstanceEnvironment(b *Backend, r *pnpmInstanceRunner, o *pnpmInstanceObserver) {
	f := newEnvironmentBackendState()
	r.environmentBackendModel = f
	o.environmentBackendModel = f
	b.resources = r
}

func pnpmInstanceFixture(t *testing.T) (artifactnpm.PNPMInspectionLayout, domain.AcquiredArtifact, string) {
	t.Helper()
	layout, input := pnpmBoundPayloadInputs(t, true)
	return layout, input.Artifacts()[0], input.Context().Lock().Snapshots()[0].Key
}
func pnpmInstanceBackend(t *testing.T, r *pnpmInstanceRunner, o *pnpmInstanceObserver) *Backend {
	t.Helper()
	b, err := NewBackend(r, &recordingIntroducer{}, o, availableProbe)
	if err != nil {
		t.Fatal(err)
	}
	wirePNPMInstanceEnvironment(b, r, o)
	return b
}
func TestPNPMInstanceRunsExactBoundTargetWithFreshSessions(t *testing.T) {
	layout, subject, key := pnpmInstanceFixture(t)
	r := &pnpmInstanceRunner{t: t}
	o := &pnpmInstanceObserver{}
	b := pnpmInstanceBackend(t, r, o)
	var sessions []string
	for range 2 {
		result, err := b.ExecutePNPMInstance(context.Background(), layout, subject, key)
		if err != nil || result.Status() != domain.SandboxCompleted {
			t.Fatal("fresh execution failed", err, result.Status())
		}
		sessions = append(sessions, result.SessionID().String())
		found := false
		for _, fact := range result.Observations() {
			if fact.Category() == domain.ObservationNetwork && fact.Count() == 4 {
				found = true
			}
		}
		if !found {
			t.Fatal("accepted observation lost")
		}
	}
	if sessions[0] == sessions[1] || r.volumes[0] == r.volumes[1] || r.ids[0] == r.ids[1] || r.executions != 4 || r.removals != 8 || r.volumeRemovals != 8 || !slices.Equal(o.ids, r.ids) {
		t.Fatal("session/resources reused or not finalized")
	}
	if slices.Index(r.events, "readback") < slices.Index(r.events, "copy") {
		t.Fatal("readback preceded copy")
	}
}
func TestPNPMInstancePreflightDoesNoRuntimeWork(t *testing.T) {
	layout, subject, key := pnpmInstanceFixture(t)
	for _, mode := range []string{"zero", "foreign-instance", "foreign-subject", "cancel", "transport", "observer"} {
		t.Run(mode, func(t *testing.T) {
			r := &pnpmInstanceRunner{t: t}
			b := pnpmInstanceBackend(t, r, &pnpmInstanceObserver{})
			candidate := layout
			item := subject
			instance := key
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("canceled before runtime")
			switch mode {
			case "zero":
				candidate = artifactnpm.PNPMInspectionLayout{}
			case "foreign-instance":
				instance += "-foreign"
			case "foreign-subject":
				item = domain.AcquiredArtifact{}
			case "cancel":
				cancel(cause)
			case "transport":
				b.runner = uvStreamOnlyRunner{}
			case "observer":
				b.observer = &emptyObserver{}
			}
			_, err := b.ExecutePNPMInstance(ctx, candidate, item, instance)
			if err == nil || r.calls != 0 {
				t.Fatal("preflight performed runtime work", err)
			}
			if mode == "cancel" && !errors.Is(err, cause) {
				t.Fatal("cancel cause lost")
			}
		})
	}
}
func TestPNPMInstanceNeverRunsAfterUnprovenPreparation(t *testing.T) {
	layout, subject, key := pnpmInstanceFixture(t)
	for _, mode := range []string{"foreign-writer", "mount-drift", "copy", "readback", "readback-drift", "network-query", "network-address", "network-null-json", "observer"} {
		t.Run(mode, func(t *testing.T) {
			r := &pnpmInstanceRunner{t: t, fault: mode}
			o := &pnpmInstanceObserver{}
			if mode == "observer" {
				o.err = errors.New("observer unavailable")
			}
			result, err := pnpmInstanceBackend(t, r, o).ExecutePNPMInstance(context.Background(), layout, subject, key)
			if err == nil || result.Status() != domain.SandboxIncomplete || r.executions != 0 || len(r.environmentBackendModel.containers) != 0 || (mode != "foreign-writer" && len(r.environmentBackendModel.volumes) != 0) || (mode == "foreign-writer" && r.volumeRemovals != 0) {
				t.Fatal("failed preparation executed target or lost cleanup", err, r.executions, r.removals, r.volumeRemovals)
			}
		})
	}
}
func TestPNPMInstanceRetainsFirstFailureAndPartialSuspiciousFacts(t *testing.T) {
	layout, subject, key := pnpmInstanceFixture(t)
	for _, mode := range []string{"execution", "cancel", "partial", "cleanup"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New("original target failure")
			cleanup := errors.New("container removal failed")
			r := &pnpmInstanceRunner{t: t}
			o := &pnpmInstanceObserver{}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			if mode == "execution" || mode == "cancel" {
				r.runErr = cause
				r.cleanupErr = cleanup
			}
			if mode == "cleanup" {
				r.cleanupErr = cleanup
			}
			if mode == "cancel" {
				r.cancel = cancel
			}
			if mode == "partial" {
				o.reader = &traceReader{records: []TraceRecord{{Kind: "network-attempt", Count: 3}}, err: errors.New("incomplete stream"), errAfter: 1}
			}
			result, err := pnpmInstanceBackend(t, r, o).ExecutePNPMInstance(ctx, layout, subject, key)
			if err == nil || result.Status() != domain.SandboxIncomplete || r.executions != 1 || len(r.environmentBackendModel.containers) != 0 || len(r.environmentBackendModel.volumes) != 0 {
				t.Fatal("failure lost or cleanup skipped", err)
			}
			if (mode == "execution" || mode == "cancel") && (!errors.Is(err, cause) || !errors.Is(err, cleanup)) {
				t.Fatal("first or cleanup cause lost", err)
			}
			if mode == "cleanup" && !errors.Is(err, cleanup) {
				t.Fatal("cleanup cause lost")
			}
			facts := result.Observations()
			if len(facts) == 0 || facts[0].Category() != domain.ObservationNetwork {
				t.Fatal("observed suspicious facts discarded")
			}
			for _, fact := range facts {
				if fact.Subject() == "lifecycle-completed" {
					t.Fatal("incomplete run gained completion fact")
				}
			}
			if mode == "execution" || mode == "cancel" {
				code, _ := result.LimitationCode()
				if code != "M3_DYNAMIC_EXECUTION_FAILED" {
					t.Fatal("cleanup overwrote primary code", code)
				}
			}
		})
	}
}
