package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// Stateful Docker format fixture: it supplies no native execution evidence.
// Existing volume checks consume the same actual/configured mount views.
type pnpmInstanceRunner struct {
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
	r.t.Helper()
	r.calls++
	if binary != "docker" || len(args) == 0 {
		r.t.Fatal("unexpected command")
	}
	r.events = append(r.events, args[0])
	if args[0] == "volume" {
		switch args[1] {
		case "ls":
			return nil, nil
		case "create":
			r.volume = dockerVolumeInspection{Name: args[len(args)-1], Driver: "local", Scope: "local", Mountpoint: "/owned/fixture", CreatedAt: "fixed", Options: map[string]string{}, Labels: map[string]string{}}
			for i := 2; i+1 < len(args); i++ {
				if args[i] == "--opt" || args[i] == "--label" {
					k, v, _ := strings.Cut(args[i+1], "=")
					if args[i] == "--opt" {
						r.volume.Options[k] = v
					} else {
						r.volume.Labels[k] = v
					}
					i++
				}
			}
			r.volumes = append(r.volumes, r.volume.Name)
			return []byte(r.volume.Name), nil
		case "inspect":
			return r.encoded(r.volume), nil
		case "rm":
			r.volumeRemovals++
			return nil, nil
		}
	}
	switch args[0] {
	case "create":
		r.creates++
		r.id = fmt.Sprintf("%064x", r.creates)
		r.ids = append(r.ids, r.id)
		r.active = true
		filtered := slices.Clone(args)
		mountIndex := slices.Index(filtered, "--mount")
		if mountIndex < 0 {
			r.t.Fatal("private mount absent")
		}
		filtered = slices.Delete(filtered, mountIndex, mountIndex+2)
		assertConstrainedCreateCommand(r.t, filtered)
		if !slices.Contains(args, "type=volume,source="+r.volume.Name+",target="+r.projectSpec().root+",volume-nocopy") {
			r.t.Fatal("fresh project volume absent")
		}
		return []byte(r.id), nil
	case "ps":
		if r.active {
			if r.fault == "foreign-writer" {
				return []byte(strings.Repeat("e", 64)), nil
			}
			return []byte(r.id), nil
		}
		return nil, nil
	case "inspect":
		if args[2] == containerMountInspectTemplate {
			configured := dockerHostMountInspection{Type: "volume", Source: r.volume.Name, Target: r.projectSpec().root}
			configured.VolumeOptions.NoCopy = true
			mount := dockerContainerMountInspection{ID: r.id, Mounts: []dockerMountInspection{{Type: "volume", Name: r.volume.Name, Source: r.volume.Mountpoint, Destination: r.projectSpec().root, Driver: "local", RW: true}}, HostMounts: []dockerHostMountInspection{configured}}
			if r.fault == "mount-drift" {
				mount.Mounts[0].Source = "/foreign"
			}
			return r.encoded(mount), nil
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
	case "network":
		return []byte(`{"Name":"none","Id":"` + strings.Repeat("b", 64) + `","Driver":"null","Scope":"local"}`), nil
	case "start":
		return nil, nil
	case "exec":
		command := pnpmTargetLifecycleCommand
		if r.kind == nativeRuntimeBun {
			command = bunTargetLifecycleCommand
		}
		if slices.Contains(args, command) {
			r.executions++
			if r.cancel != nil {
				r.cancel(r.runErr)
				return nil, ctx.Err()
			}
			return nil, r.runErr
		}
		if !slices.Equal(args, boundaryReadinessArguments(r.id)) {
			r.t.Fatal("unexpected exec", args)
		}
		return nil, nil
	case "rm":
		r.removals++
		r.active = false
		return nil, r.cleanupErr
	}
	r.t.Fatal("unhandled command", args)
	return nil, nil
}
func (r *pnpmInstanceRunner) RunInput(_ context.Context, input io.Reader, binary string, args ...string) error {
	r.t.Helper()
	r.calls++
	r.events = append(r.events, "copy")
	if binary != "docker" || !slices.Equal(args, []string{"cp", "--archive", "-", r.id + ":" + r.projectSpec().root}) {
		r.t.Fatal("copy destination is not fixed")
	}
	if r.fault == "copy" {
		return errors.New("copy refused")
	}
	var err error
	r.payload, err = io.ReadAll(io.LimitReader(input, artifactnpm.PNPMLayoutArchiveLimit+1))
	return err
}
func (r *pnpmInstanceRunner) RunOutput(_ context.Context, output io.Writer, binary string, args ...string) error {
	r.t.Helper()
	r.calls++
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
	reader TraceReader
	err    error
	ids    []string
}

func (o *pnpmInstanceObserver) Start(context.Context, string) (TraceReader, error) {
	return nil, errors.New("ordinary profile must not be used")
}
func (o *pnpmInstanceObserver) StartPNPMTargetLifecycleProfile(_ context.Context, id string) (TraceReader, error) {
	o.ids = append(o.ids, id)
	if o.err != nil {
		return nil, o.err
	}
	if o.reader != nil {
		return o.reader, nil
	}
	return &traceReader{records: []TraceRecord{{Kind: "network-attempt", Count: 2}}}, nil
}
func (o *pnpmInstanceObserver) AwaitMountAnchors(context.Context, string) error { return nil }

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
			if fact.Category() == domain.ObservationNetwork && fact.Count() == 2 {
				found = true
			}
		}
		if !found {
			t.Fatal("accepted observation lost")
		}
	}
	if sessions[0] == sessions[1] || r.volumes[0] == r.volumes[1] || r.ids[0] == r.ids[1] || r.executions != 2 || r.removals != 2 || r.volumeRemovals != 2 || !slices.Equal(o.ids, r.ids) {
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
			if err == nil || result.Status() != domain.SandboxIncomplete || r.executions != 0 || r.removals != 1 || r.volumeRemovals != 1 {
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
			if err == nil || result.Status() != domain.SandboxIncomplete || r.executions != 1 || r.removals != 1 || r.volumeRemovals != 1 {
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
