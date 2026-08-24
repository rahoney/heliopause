package sandbox

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestObserverSupervisorOwnsOneReceiverHelperAndEndpoints(t *testing.T) {
	paths := supervisorPaths(t)
	launcher := &fakeObserverLauncher{}
	supervisor, err := newObserverSupervisor(context.Background(), launcher.StartObserver, paths.remote, paths.output, paths.lock)
	if err != nil {
		t.Fatal(err)
	}
	if supervisor.Observer() == nil || launcher.starts != 1 {
		t.Fatalf("supervisor = %#v, starts=%d", supervisor, launcher.starts)
	}
	if _, err := os.Lstat(paths.remote); err != nil {
		t.Fatalf("remote endpoint missing: %v", err)
	}
	if _, err := os.Lstat(paths.output); err != nil {
		t.Fatalf("output endpoint missing: %v", err)
	}
	if err := supervisor.Close(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{paths.remote, paths.output, paths.lock} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("supervisor did not clean %q: %v", path, err)
		}
	}
}

func TestObserverSupervisorRefusesExistingOwnerWithoutRemovingIt(t *testing.T) {
	paths := supervisorPaths(t)
	firstLauncher := &fakeObserverLauncher{}
	first, err := newObserverSupervisor(context.Background(), firstLauncher.StartObserver, paths.remote, paths.output, paths.lock)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	secondLauncher := &fakeObserverLauncher{}
	if _, err := newObserverSupervisor(context.Background(), secondLauncher.StartObserver, paths.remote, paths.output, paths.lock); err == nil {
		t.Fatal("second supervisor acquired an owned endpoint")
	}
	for _, path := range []string{paths.remote, paths.output, paths.lock} {
		if _, err := os.Lstat(path); err != nil {
			t.Fatalf("second supervisor removed owned %q: %v", path, err)
		}
	}
}

func TestObserverSupervisorHelperDeathFaultsObservation(t *testing.T) {
	paths := supervisorPaths(t)
	launcher := &fakeObserverLauncher{}
	supervisor, err := newObserverSupervisor(context.Background(), launcher.StartObserver, paths.remote, paths.output, paths.lock)
	if err != nil {
		t.Fatal(err)
	}
	defer supervisor.Close()
	launcher.process.exit(errors.New("helper died"))
	deadline := time.After(time.Second)
	for {
		_, err := supervisor.Observer().Start(context.Background(), "0123456789abcdef")
		if err != nil {
			return
		}
		select {
		case <-deadline:
			t.Fatal("helper death did not fault observer")
		case <-time.After(time.Millisecond):
		}
	}
}

func TestObserverSupervisorStartupFailureReleasesKnownOwnership(t *testing.T) {
	paths := supervisorPaths(t)
	_, err := newObserverSupervisor(context.Background(), failingObserverLauncher{}.StartObserver, paths.remote, paths.output, paths.lock)
	if err == nil {
		t.Fatal("supervisor accepted helper startup failure")
	}
	for _, path := range []string{paths.remote, paths.output, paths.lock} {
		if _, statErr := os.Lstat(path); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("startup failure left known endpoint %q: %v", path, statErr)
		}
	}
}

type supervisorTestPaths struct{ remote, output, lock string }

func supervisorPaths(t *testing.T) supervisorTestPaths {
	t.Helper()
	directory, err := os.MkdirTemp(".", "observer-supervisor-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	directory, err = filepath.Abs(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(filepath.Join(directory, "output.sock")) >= 100 {
		t.Skip("host Unix socket path limit leaves no protected short test directory")
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	return supervisorTestPaths{
		remote: filepath.Join(directory, "remote.sock"),
		output: filepath.Join(directory, "output.sock"),
		lock:   filepath.Join(directory, "lock"),
	}
}

type fakeObserverLauncher struct {
	starts  int
	process *fakeObserverProcess
}

func (l *fakeObserverLauncher) StartObserver(_ context.Context, remote, _ string) (ObserverProcess, error) {
	l.starts++
	listener, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: remote, Net: "unixgram"})
	if err != nil {
		return nil, err
	}
	l.process = &fakeObserverProcess{listener: listener, done: make(chan struct{})}
	return l.process, nil
}

type failingObserverLauncher struct{}

func (failingObserverLauncher) StartObserver(context.Context, string, string) (ObserverProcess, error) {
	return nil, errors.New("helper start failed")
}

type fakeObserverProcess struct {
	listener io.Closer
	done     chan struct{}
	err      error
	once     sync.Once
}

func (p *fakeObserverProcess) Done() <-chan struct{} { return p.done }
func (p *fakeObserverProcess) ExitError() error      { return p.err }
func (p *fakeObserverProcess) Stop(context.Context) error {
	p.exit(nil)
	return nil
}
func (p *fakeObserverProcess) exit(err error) {
	p.once.Do(func() {
		p.err = err
		_ = p.listener.Close()
		close(p.done)
	})
}
