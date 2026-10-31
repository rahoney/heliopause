package sandbox

import (
	"context"
	"errors"
	"slices"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type bunInstanceObserver struct{ *pnpmInstanceObserver }

func (o *bunInstanceObserver) StartBunTargetLifecycleProfile(ctx context.Context, id string) (TraceReader, error) {
	return o.StartPNPMTargetLifecycleProfile(ctx, id)
}

func bunInstanceFixture(t *testing.T) (artifactnpm.BunInspectionLayout, domain.AcquiredArtifact, string) {
	t.Helper()
	layout, input := bunBoundPayloadInputs(t)
	subject := input.Artifacts()[0]
	instances, err := input.InstancesFor(subject)
	if err != nil || len(instances) != 1 {
		t.Fatal("complete Bun instance fixture", err)
	}
	return layout, subject, instances[0].Key
}
func bunInstanceBackend(t *testing.T, r *pnpmInstanceRunner, o *bunInstanceObserver) *Backend {
	t.Helper()
	b, err := NewBackend(r, &recordingIntroducer{}, o, availableProbe)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestBunInstanceRunsExactBoundTargetWithFreshSessions(t *testing.T) {
	layout, subject, key := bunInstanceFixture(t)
	r := &pnpmInstanceRunner{kind: nativeRuntimeBun, t: t}
	o := &bunInstanceObserver{pnpmInstanceObserver: &pnpmInstanceObserver{}}
	b := bunInstanceBackend(t, r, o)
	var sessions []string
	for range 2 {
		result, err := b.ExecuteBunInstance(context.Background(), layout, subject, key)
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
func TestBunInstancePreflightDoesNoRuntimeWork(t *testing.T) {
	layout, subject, key := bunInstanceFixture(t)
	for _, mode := range []string{"zero", "foreign-instance", "foreign-subject", "cancel", "transport", "observer"} {
		t.Run(mode, func(t *testing.T) {
			r := &pnpmInstanceRunner{kind: nativeRuntimeBun, t: t}
			b := bunInstanceBackend(t, r, &bunInstanceObserver{pnpmInstanceObserver: &pnpmInstanceObserver{}})
			candidate := layout
			item := subject
			instance := key
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("canceled before runtime")
			switch mode {
			case "zero":
				candidate = artifactnpm.BunInspectionLayout{}
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
			_, err := b.ExecuteBunInstance(ctx, candidate, item, instance)
			if err == nil || r.calls != 0 {
				t.Fatal("preflight performed runtime work", err)
			}
			if mode == "cancel" && !errors.Is(err, cause) {
				t.Fatal("cancel cause lost")
			}
		})
	}
}
func TestBunInstanceNeverRunsAfterUnprovenPreparation(t *testing.T) {
	layout, subject, key := bunInstanceFixture(t)
	for _, mode := range []string{"foreign-writer", "mount-drift", "copy", "readback", "readback-drift", "network-query", "network-address", "network-null-json", "observer"} {
		t.Run(mode, func(t *testing.T) {
			r := &pnpmInstanceRunner{kind: nativeRuntimeBun, t: t, fault: mode}
			o := &bunInstanceObserver{pnpmInstanceObserver: &pnpmInstanceObserver{}}
			if mode == "observer" {
				o.err = errors.New("observer unavailable")
			}
			result, err := bunInstanceBackend(t, r, o).ExecuteBunInstance(context.Background(), layout, subject, key)
			if err == nil || result.Status() != domain.SandboxIncomplete || r.executions != 0 || r.removals != 1 || r.volumeRemovals != 1 {
				t.Fatal("failed preparation executed target or lost cleanup", err, r.executions, r.removals, r.volumeRemovals)
			}
		})
	}
}
func TestBunInstanceRetainsFirstFailureAndPartialSuspiciousFacts(t *testing.T) {
	layout, subject, key := bunInstanceFixture(t)
	for _, mode := range []string{"execution", "cancel", "partial", "cleanup"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New("original target failure")
			cleanup := errors.New("container removal failed")
			r := &pnpmInstanceRunner{kind: nativeRuntimeBun, t: t}
			o := &bunInstanceObserver{pnpmInstanceObserver: &pnpmInstanceObserver{}}
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
			result, err := bunInstanceBackend(t, r, o).ExecuteBunInstance(ctx, layout, subject, key)
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
