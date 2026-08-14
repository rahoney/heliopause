//go:build linux && amd64

package promotion

import (
	"syscall"
	"unsafe"
)

const (
	renameNoReplaceFlag = 1
	amd64Renameat2      = 316
)

func renameNoReplace(oldPath, newPath string) error {
	oldPointer, err := syscall.BytePtrFromString(oldPath)
	if err != nil {
		return err
	}
	newPointer, err := syscall.BytePtrFromString(newPath)
	if err != nil {
		return err
	}
	atFDCWD := -100
	_, _, errno := syscall.Syscall6(amd64Renameat2, uintptr(atFDCWD), uintptr(unsafe.Pointer(oldPointer)), uintptr(atFDCWD), uintptr(unsafe.Pointer(newPointer)), renameNoReplaceFlag, 0)
	if errno != 0 {
		return errno
	}
	return nil
}
