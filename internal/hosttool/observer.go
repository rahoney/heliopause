package hosttool

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const observerReadinessTimeout = 5 * time.Second

var gVisorObserverCommit = runtimeidentity.GVisorCommit

// ObserverProcess is the bounded lifecycle capability returned after the
// helper has explicitly confirmed that its remote endpoint is ready.
type ObserverProcess interface {
	Done() <-chan struct{}
	ExitError() error
	Stop(context.Context) error
}

// ObserverLauncher starts only the registered observer helper. It is kept
// separate from Docker execution so Sandbox owns the receiver and lifecycle
// while hosttool owns executable identity and the child environment.
type ObserverLauncher interface {
	StartObserver(context.Context, string, string) (ObserverProcess, error)
}

// Launcher launches one root-owned, non-writable observer helper identity.
type Launcher struct {
	helper identity
}

// NewSystemObserverLauncher resolves the root-owned installation path without
// consulting PATH or inherited process configuration.
func NewSystemObserverLauncher() (ObserverLauncher, error) {
	if runtime.GOOS != "linux" {
		return nil, errors.New("trusted observer helper requires Linux")
	}
	config, err := loadSystemConfig()
	if err != nil {
		return nil, err
	}
	return NewObserverLauncher(config.ObserverHelperPath)
}

// NewObserverLauncher validates an absolute helper installation path. It is a
// narrow construction seam for infrastructure tests and controlled installers.
func NewObserverLauncher(path string) (ObserverLauncher, error) {
	if path == "" {
		path = defaultObserverHelper
	}
	helper, err := verifyExecutable(path, "")
	if err != nil {
		return nil, errors.New("verify observer helper executable")
	}
	return &Launcher{helper: helper}, nil
}

// StartObserver waits for the helper's pipe-based readiness acknowledgement;
// a running PID or a socket pathname alone is never treated as ready.
func (l *Launcher) StartObserver(ctx context.Context, remoteEndpoint, outputEndpoint string) (ObserverProcess, error) {
	if l == nil || ctx == nil || !canonicalAbsolutePath(remoteEndpoint) || !canonicalAbsolutePath(outputEndpoint) {
		return nil, errors.New("observer helper start request is invalid")
	}
	current, err := verifyExecutable(l.helper.path, l.helper.digest)
	if err != nil || !os.SameFile(l.helper.info, current.info) {
		return nil, errors.New("observer helper identity changed after validation")
	}
	if err := verifyObserverBuildIdentity(ctx, current.path); err != nil {
		return nil, err
	}
	readinessRead, readinessWrite, err := os.Pipe()
	if err != nil {
		return nil, errors.New("create observer readiness channel")
	}
	command := exec.CommandContext(ctx, current.path, remoteEndpoint, outputEndpoint, "--ready-fd=3")
	command.Env = []string{"LANG=C", "LC_ALL=C"}
	command.ExtraFiles = []*os.File{readinessWrite}
	if err := command.Start(); err != nil {
		_ = readinessRead.Close()
		_ = readinessWrite.Close()
		return nil, errors.New("start trusted observer helper")
	}
	if err := readinessWrite.Close(); err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		_ = readinessRead.Close()
		return nil, errors.New("close observer readiness writer")
	}
	process := &observerProcess{command: command, done: make(chan struct{})}
	go process.wait()
	if err := awaitObserverReady(ctx, readinessRead, process); err != nil {
		stopContext, cancel := context.WithTimeout(context.Background(), observerReadinessTimeout)
		_ = process.Stop(stopContext)
		cancel()
		return nil, err
	}
	return process, nil
}

func verifyObserverBuildIdentity(ctx context.Context, path string) error {
	command := exec.CommandContext(ctx, path, "--identity")
	command.Env = []string{"LANG=C", "LC_ALL=C"}
	output, err := command.Output()
	if err != nil || strings.TrimSpace(string(output)) != "gvisor-commit="+gVisorObserverCommit {
		return errors.New("observer helper build identity is unavailable or mismatched")
	}
	return nil
}

func canonicalAbsolutePath(path string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path
}

func awaitObserverReady(parent context.Context, reader *os.File, process *observerProcess) error {
	defer reader.Close()
	read := make(chan error, 1)
	go func() {
		var byteValue [1]byte
		_, err := io.ReadFull(reader, byteValue[:])
		if err == nil && byteValue[0] != 'R' {
			err = errors.New("observer helper readiness acknowledgement is invalid")
		}
		read <- err
	}()
	deadline, cancel := context.WithTimeout(parent, observerReadinessTimeout)
	defer cancel()
	select {
	case err := <-read:
		if err != nil {
			return errors.New("observer helper readiness failed")
		}
		select {
		case <-process.Done():
			return errors.New("observer helper exited during readiness")
		default:
		}
		return nil
	case <-process.Done():
		return errors.New("observer helper exited before readiness")
	case <-deadline.Done():
		return errors.New("observer helper readiness timed out")
	}
}

type observerProcess struct {
	command *exec.Cmd
	done    chan struct{}
	mu      sync.Mutex
	exitErr error
}

func (p *observerProcess) wait() {
	err := p.command.Wait()
	p.mu.Lock()
	p.exitErr = err
	p.mu.Unlock()
	close(p.done)
}

func (p *observerProcess) Done() <-chan struct{} { return p.done }

func (p *observerProcess) ExitError() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitErr
}

func (p *observerProcess) Stop(ctx context.Context) error {
	if p == nil || p.command == nil || p.command.Process == nil {
		return errors.New("observer helper process is unavailable")
	}
	select {
	case <-p.done:
		return p.ExitError()
	default:
	}
	if err := p.command.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return errors.New("terminate observer helper")
	}
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		return errors.New("observer helper termination timed out")
	}
}
