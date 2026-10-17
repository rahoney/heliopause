//go:build linux || darwin

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func lockState(state string) (func(), error) {
	f, err := os.OpenFile(filepath.Join(state, "operation.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("another update operation owns state: %w", err)
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }, nil
}
