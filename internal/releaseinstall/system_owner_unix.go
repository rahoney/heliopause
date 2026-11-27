//go:build linux || darwin

package releaseinstall

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

func verifySystemDirectory(path string, allowMissing bool) error {
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			if !allowMissing || !errors.Is(err, os.ErrNotExist) {
				return errors.New("protected release directory is unavailable")
			}
		} else {
			owner, ok := info.Sys().(*syscall.Stat_t)
			if !ok || owner.Uid != 0 || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0022 != 0 {
				return errors.New("protected release directory or ancestor is untrusted")
			}
		}
		if current == filepath.Dir(current) {
			return nil
		}
	}
}

func verifySystemRegularFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0022 != 0 {
		return errors.New("protected release member is untrusted")
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != 0 || owner.Nlink != 1 || info.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 {
		return errors.New("protected release member owner or link identity is untrusted")
	}
	return nil
}
