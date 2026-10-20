//go:build linux

package releaseactivation

import (
	"context"
	"errors"
	"os"
	"sync"
	"syscall"
	"time"
)

// RuntimeLease protects one managed command's exact runtime until its actual
// cleanup is complete. It is an advisory coordination primitive, not authority.
type RuntimeLease struct {
	mu        sync.Mutex
	file      *os.File
	installed InstalledRelease
	closed    bool
	closeErr  error
}

// AcquireRuntimeLease returns nil only for confirmed absence of managed state.
// Existing partial/foreign/unreadable state never falls back to legacy mode.
func AcquireRuntimeLease(ctx context.Context) (*RuntimeLease, error) {
	if ctx == nil {
		return nil, ErrUntrusted
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	managed, err := managedStatePresent()
	if err != nil || !managed {
		return nil, err
	}
	file, info, err := openProtectedFile(runtimeLockPath)
	if err != nil {
		return nil, err
	}
	fail := func(cause error) (*RuntimeLease, error) {
		unlockErr := syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		closeErr := file.Close()
		if unlockErr != nil || closeErr != nil {
			cause = errors.Join(cause, ErrUntrusted)
		}
		return nil, cause
	}
	wait, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := waitFileLock(wait, file, syscall.LOCK_SH); err != nil {
		return fail(err)
	}
	current, err := os.Lstat(runtimeLockPath)
	if err != nil || !os.SameFile(info, current) {
		return fail(ErrUntrusted)
	}
	managed, err = managedStatePresent()
	if err != nil {
		return fail(err)
	}
	if !managed {
		return fail(ErrUntrusted)
	}
	installed, err := ReadInstalledRelease(ctx)
	if err != nil {
		return fail(err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	return &RuntimeLease{file: file, installed: installed}, nil
}

func managedStatePresent() (bool, error) {
	if _, err := os.Lstat(stateRoot); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, ErrUntrusted
	}
	if err := protectedDirectory(stateRoot); err != nil {
		return false, err
	}
	present := map[string]bool{}
	for _, path := range []string{registrationPath, journalPath, runtimeLockPath} {
		_, err := os.Lstat(path)
		if err == nil {
			present[path] = true
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			return false, ErrUntrusted
		}
	}
	if !present[registrationPath] && !present[journalPath] && !present[runtimeLockPath] {
		return false, nil
	}
	if !present[registrationPath] || !present[runtimeLockPath] || present[journalPath] {
		return false, ErrTransition
	}
	return true, nil
}

func waitFileLock(ctx context.Context, file *os.File, operation int) error {
	for {
		if err := ctx.Err(); err != nil {
			return errors.Join(ErrTransition, err)
		}
		err := syscall.Flock(int(file.Fd()), operation|syscall.LOCK_NB)
		if err == nil {
			return nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			return ErrUntrusted
		}
		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errors.Join(ErrTransition, ctx.Err())
		case <-timer.C:
		}
	}
}

func (l *RuntimeLease) Installed() (InstalledRelease, error) {
	if l == nil {
		return InstalledRelease{}, ErrUnmanaged
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed || l.file == nil {
		return InstalledRelease{}, ErrUntrusted
	}
	return l.installed, nil
}

// Close is also the explicit handoff after old command-owned runtime cleanup.
// Exclusive installation must not wait while holding this shared FD.
func (l *RuntimeLease) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return l.closeErr
	}
	l.closed = true
	if l.file != nil {
		unlockErr := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
		closeErr := l.file.Close()
		if unlockErr != nil || closeErr != nil {
			l.closeErr = ErrUntrusted
		}
		l.file = nil
	}
	return l.closeErr
}
