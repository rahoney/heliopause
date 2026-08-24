//go:build unix

package hosttool

import (
	"errors"
	"os"
	"syscall"
)

func verifyTrustedOwner(info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 {
		return errors.New("host tool path is not owned by root")
	}
	return nil
}

func verifyEndpointOwner(info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 && int(stat.Uid) != os.Geteuid() {
		return errors.New("docker endpoint owner is not trusted")
	}
	return nil
}
