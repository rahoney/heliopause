package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

const (
	ObserverRemoteEndpoint = "/run/heliopause-observer/gvisor-remote.sock"
	supervisorLockEndpoint = "/run/heliopause-observer/.heliopause-supervisor.lock"
	supervisorStopTimeout  = 5 * time.Second
)

// ObserverSupervisor owns the one trusted receiver and helper process used by
// all Linux dynamic adapters in a single helox process. It deliberately does
// not reuse a fixed endpoint owned by another process.
type ObserverSupervisor struct {
	observer   *SharedObserver
	helper     ObserverProcess
	remotePath string
	remoteID   os.FileInfo
	lockPath   string
	lockID     os.FileInfo

	mu        sync.Mutex
	closing   bool
	closeOnce sync.Once
	closeErr  error
}

// ObserverProcess is the supervisor's narrow helper lifecycle dependency.
// Its implementation belongs to the Host tooling adapter, not Sandbox.
type ObserverProcess interface {
	Done() <-chan struct{}
	ExitError() error
	Stop(context.Context) error
}

// ObserverLauncher starts one helper only after its explicit readiness signal.
// It is a function seam so Sandbox never imports a concrete Host-tool package.
type ObserverLauncher func(context.Context, string, string) (ObserverProcess, error)

// NewObserverSupervisor creates the fixed, process-scoped observation
// lifecycle. The caller must Close it and propagate any cleanup uncertainty.
func NewObserverSupervisor(ctx context.Context, launcher ObserverLauncher) (*ObserverSupervisor, error) {
	return newObserverSupervisor(ctx, launcher, ObserverRemoteEndpoint, ObserverOutputEndpoint, supervisorLockEndpoint)
}

func newObserverSupervisor(ctx context.Context, launcher ObserverLauncher, remotePath, outputPath, lockPath string) (*ObserverSupervisor, error) {
	if ctx == nil || launcher == nil || !sameRuntimeDirectory(remotePath, outputPath, lockPath) {
		return nil, errors.New("observer supervisor configuration is invalid")
	}
	if err := verifyObserverRuntimeDirectory(filepath.Dir(remotePath)); err != nil {
		return nil, err
	}
	lockID, err := acquireSupervisorLock(lockPath)
	if err != nil {
		return nil, err
	}
	supervisor := &ObserverSupervisor{remotePath: remotePath, lockPath: lockPath, lockID: lockID}
	observer, err := newExclusiveSharedObserver(outputPath)
	if err != nil {
		_ = supervisor.releaseLock()
		return nil, err
	}
	supervisor.observer = observer
	helper, err := launcher(ctx, remotePath, outputPath)
	if err != nil {
		return nil, errors.Join(errors.New("start observer supervisor helper"), err, supervisor.Close())
	}
	supervisor.helper = helper
	remoteID, err := verifyOwnedObserverSocket(remotePath)
	if err != nil {
		return nil, errors.Join(errors.New("verify observer helper readiness endpoint"), err, supervisor.Close())
	}
	outputEndpoint, outputID := observer.endpointIdentity()
	if outputEndpoint != outputPath || !sameFile(outputID, mustObserverSocket(outputPath)) {
		return nil, errors.Join(errors.New("verify observer receiver readiness endpoint"), supervisor.Close())
	}
	supervisor.remoteID = remoteID
	go supervisor.watchHelper()
	return supervisor, nil
}

// Observer is the trusted TraceObserver shared by every dynamic adapter in the
// process. Helper death is converted to a global observer fault by watchHelper.
func (s *ObserverSupervisor) Observer() TraceObserver {
	if s == nil {
		return nil
	}
	return s.observer
}

func (s *ObserverSupervisor) watchHelper() {
	<-s.helper.Done()
	s.mu.Lock()
	closing := s.closing
	s.mu.Unlock()
	if !closing {
		s.observer.Fail(errors.New("observer helper exited unexpectedly"))
	}
}

// Close terminates the helper and removes only socket and lock identities this
// supervisor created. If identity or cleanup is uncertain, the lock is kept so
// the next process fails closed instead of assuming ownership.
func (s *ObserverSupervisor) Close() error {
	if s == nil {
		return nil
	}
	s.closeOnce.Do(func() { s.closeErr = s.close() })
	return s.closeErr
}

func (s *ObserverSupervisor) close() error {
	s.mu.Lock()
	s.closing = true
	s.mu.Unlock()
	var cleanupErr error
	if s.helper != nil {
		stopContext, cancel := context.WithTimeout(context.Background(), supervisorStopTimeout)
		if err := s.helper.Stop(stopContext); err != nil {
			cleanupErr = errors.Join(cleanupErr, errors.New("observer helper cleanup failed"))
		}
		cancel()
	}
	if s.remoteID != nil {
		if err := removeOwnedObserverSocket(s.remotePath, s.remoteID); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	if s.observer != nil {
		if err := s.observer.Close(); err != nil {
			cleanupErr = errors.Join(cleanupErr, errors.New("observer receiver cleanup failed"))
		}
		endpoint, identity := s.observer.endpointIdentity()
		if err := removeOwnedObserverSocket(endpoint, identity); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	if cleanupErr != nil {
		return cleanupErr
	}
	return s.releaseLock()
}

func sameRuntimeDirectory(paths ...string) bool {
	if len(paths) == 0 {
		return false
	}
	directory := filepath.Dir(paths[0])
	for _, path := range paths {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || filepath.Dir(path) != directory {
			return false
		}
	}
	return true
}

func verifyObserverRuntimeDirectory(directory string) error {
	for current := filepath.Clean(directory); current != filepath.Dir(current); current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode()&0o022 != 0 {
			return errors.New("observer runtime directory is not protected")
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != 0 && int(stat.Uid) != os.Geteuid() {
			return errors.New("observer runtime directory owner is not trusted")
		}
	}
	return nil
}

func acquireSupervisorLock(path string) (os.FileInfo, error) {
	if _, err := os.Lstat(path); err == nil {
		return nil, errors.New("observer supervisor endpoint is already owned")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("inspect observer supervisor endpoint")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, errors.New("acquire observer supervisor endpoint")
	}
	info, statErr := file.Stat()
	closeErr := file.Close()
	if statErr != nil || closeErr != nil {
		return nil, errors.New("capture observer supervisor endpoint identity")
	}
	return info, nil
}

func verifyOwnedObserverSocket(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSocket == 0 || info.Mode()&os.ModeSymlink != 0 || info.Mode()&0o022 != 0 {
		return nil, errors.New("observer endpoint is unavailable or unsafe")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Geteuid() {
		return nil, errors.New("observer endpoint owner is not trusted")
	}
	return info, nil
}

func mustObserverSocket(path string) os.FileInfo {
	info, err := verifyOwnedObserverSocket(path)
	if err != nil {
		return nil
	}
	return info
}

func sameFile(left, right os.FileInfo) bool {
	return left != nil && right != nil && os.SameFile(left, right)
}

func removeOwnedObserverSocket(path string, expected os.FileInfo) error {
	if path == "" || expected == nil {
		return errors.New("observer endpoint cleanup identity is unavailable")
	}
	current, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || !sameFile(expected, current) || current.Mode()&os.ModeSocket == 0 {
		return errors.New("observer endpoint cleanup identity changed")
	}
	if err := os.Remove(path); err != nil {
		return errors.New("remove observer endpoint")
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		return errors.New("verify observer endpoint cleanup")
	}
	return nil
}

func (s *ObserverSupervisor) releaseLock() error {
	if s.lockPath == "" || s.lockID == nil {
		return errors.New("observer supervisor lock identity is unavailable")
	}
	current, err := os.Lstat(s.lockPath)
	if err != nil || !sameFile(s.lockID, current) {
		return errors.New("observer supervisor lock identity changed")
	}
	if err := os.Remove(s.lockPath); err != nil {
		return fmt.Errorf("remove observer supervisor lock: %w", err)
	}
	if _, err := os.Lstat(s.lockPath); !errors.Is(err, os.ErrNotExist) {
		return errors.New("verify observer supervisor lock cleanup")
	}
	return nil
}
