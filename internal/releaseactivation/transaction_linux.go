//go:build linux

package releaseactivation

import (
	"context"
	"errors"
	"os"
	"sync"
	"syscall"
)

const (
	updateLockPath    = stateRoot + "/update.lock"
	installerLockPath = stateRoot + "/installer.lock"
)

var ErrBusy = errors.New("SELF_UPDATE_BUSY")

// UpdateOwner excludes another update before resolution. It is coordination,
// never a replacement for the independently authenticated installer boundary.
type UpdateOwner struct {
	mu       sync.Mutex
	file     *os.File
	closed   bool
	closeErr error
}

func AcquireUpdateOwner(ctx context.Context, lease *RuntimeLease) (*UpdateOwner, error) {
	if lease != nil && lease.borrowed {
		return nil, ErrTransition
	}
	installed, err := lease.Installed()
	if err != nil {
		return nil, err
	}
	if err := CheckRuntimeController(installed); err != nil {
		return nil, err
	}
	return lockPersistentFile(ctx, updateLockPath, false, false)
}

func acquireInstallerOwner(ctx context.Context) (*UpdateOwner, error) {
	if ctx == nil || os.Geteuid() != 0 {
		return nil, ErrCaller
	}
	if err := protectedDirectory(stateRoot); err != nil {
		return nil, err
	}
	return lockPersistentFile(ctx, installerLockPath, true, false)
}

func acquireExclusiveRuntime(ctx context.Context) (*UpdateOwner, error) {
	if ctx == nil || os.Geteuid() != 0 {
		return nil, ErrCaller
	}
	return lockPersistentFile(ctx, runtimeLockPath, false, true)
}

func lockPersistentFile(ctx context.Context, path string, create, wait bool) (*UpdateOwner, error) {
	if ctx == nil {
		return nil, ErrUntrusted
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if create {
		if err := protectedDirectory(stateRoot); err != nil {
			return nil, err
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
		if err == nil {
			modeErr, syncErr, closeErr := file.Chmod(0600), file.Sync(), file.Close()
			if modeErr != nil || syncErr != nil || closeErr != nil {
				return nil, ErrUntrusted
			}
			if err := syncProtectedDirectory(stateRoot); err != nil {
				return nil, err
			}
		} else if !errors.Is(err, os.ErrExist) {
			return nil, ErrUntrusted
		}
	}
	file, identity, err := openProtectedFile(path)
	if err != nil {
		return nil, err
	}
	if wait {
		err = waitFileLock(ctx, file, syscall.LOCK_EX)
	} else {
		err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			err = ErrBusy
		}
	}
	owner := &UpdateOwner{file: file}
	if err != nil {
		return nil, errors.Join(err, owner.Close())
	}
	current, err := os.Lstat(path)
	if err != nil || !os.SameFile(identity, current) {
		return nil, errors.Join(ErrUntrusted, owner.Close())
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(err, owner.Close())
	}
	return owner, nil
}

func (o *UpdateOwner) Close() error {
	if o == nil {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return o.closeErr
	}
	o.closed = true
	if o.file != nil {
		unlockErr, closeErr := syscall.Flock(int(o.file.Fd()), syscall.LOCK_UN), o.file.Close()
		if unlockErr != nil || closeErr != nil {
			o.closeErr = ErrUntrusted
		}
	}
	return o.closeErr
}
