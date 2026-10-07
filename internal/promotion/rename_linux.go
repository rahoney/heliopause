//go:build linux && amd64

package promotion

import (
	"os"
	"syscall"
	"unsafe"
)

const (
	renameNoReplaceFlag = 1
	amd64Renameat2      = 316
)

func renameNoReplace(oldPath, newPath string) error {
	return renameNoReplaceAt(-100, oldPath, newPath)
}

func renameRootNoReplace(root *os.Root, oldPath, newPath string) error {
	directory, err := root.Open(".")
	if err != nil {
		return err
	}
	defer directory.Close()
	return renameNoReplaceAt(int(directory.Fd()), oldPath, newPath)
}

func renameNoReplaceAt(directoryFD int, oldPath, newPath string) error {
	oldPointer, err := syscall.BytePtrFromString(oldPath)
	if err != nil {
		return err
	}
	newPointer, err := syscall.BytePtrFromString(newPath)
	if err != nil {
		return err
	}
	_, _, errno := syscall.Syscall6(amd64Renameat2, uintptr(directoryFD), uintptr(unsafe.Pointer(oldPointer)), uintptr(directoryFD), uintptr(unsafe.Pointer(newPointer)), renameNoReplaceFlag, 0)
	if errno != 0 {
		return errno
	}
	return nil
}
