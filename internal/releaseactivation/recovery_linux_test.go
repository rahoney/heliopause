//go:build linux

package releaseactivation

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type terminalRecoveryFixture struct {
	kind   string
	events []string
	cancel context.CancelFunc
}

func (f *terminalRecoveryFixture) active(context.Context, string, uint32) (*serviceProcess, error) {
	f.events = append(f.events, "active")
	if f.kind == "active" {
		return &serviceProcess{pid: 200}, nil
	}
	if f.kind == "empty-success" {
		return nil, nil
	}
	if f.kind == "cancel" {
		f.cancel()
	}
	return nil, ErrTransition
}
func (f *terminalRecoveryFixture) stopped(context.Context, *serviceProcess) error {
	f.events = append(f.events, "stopped")
	if f.kind == "unknown" {
		return ErrUntrusted
	}
	return nil
}
func (f *terminalRecoveryFixture) resourcesDrained(context.Context, uint32, int, int) error {
	f.events = append(f.events, "drain")
	if f.kind == "busy" {
		return ErrBusy
	}
	return nil
}
func (f *terminalRecoveryFixture) start(_ context.Context, version string, gid uint32) (*serviceProcess, error) {
	f.events = append(f.events, "start")
	if version != "v1.0.0" || gid != 1000 {
		return nil, ErrUntrusted
	}
	if f.kind == "empty-start" {
		return nil, nil
	}
	if f.kind == "start-failed" {
		return nil, ErrHealth
	}
	return &serviceProcess{pid: 201}, nil
}
func TestTerminalRecoveryStartsOnlyConfirmedStoppedAndDrainedService(t *testing.T) {
	for _, kind := range []string{"active", "inactive", "unknown", "busy", "start-failed", "cancel", "empty-success", "empty-start"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fixture := &terminalRecoveryFixture{kind: kind, cancel: cancel}
			installed := InstalledRelease{record: registration{registrationSchema, "NATIVE", "v1.0.0", bodySHA256([]byte("manifest")), 1000, 1000}}
			process, err := recoverTerminalService(ctx, fixture, installed)
			success := kind == "active" || kind == "inactive"
			if (err == nil) != success || success && process == nil {
				t.Fatal("recovery result differs", kind, err)
			}
			if process != nil {
				if err := process.Close(); err != nil {
					t.Fatal(err)
				}
			}
			expected := []string{"active"}
			switch kind {
			case "unknown":
				expected = append(expected, "stopped")
			case "busy":
				expected = append(expected, "stopped", "drain")
			case "inactive", "start-failed", "empty-start":
				expected = append(expected, "stopped", "drain", "start")
			}
			if !reflect.DeepEqual(fixture.events, expected) {
				t.Fatal("unsafe command sequence", fixture.events, expected)
			}
			if kind == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation lost", err)
			}
			if kind == "busy" && !errors.Is(err, ErrBusy) {
				t.Fatal("drain cause lost", err)
			}
			if kind == "start-failed" && !errors.Is(err, ErrHealth) {
				t.Fatal("start cause lost", err)
			}
		})
	}
}
