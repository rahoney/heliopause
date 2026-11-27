package main

import (
	"crypto/rand"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// publishNativeAsset reuses the no-replace PNPM publisher for verified native
// data. It does not execute or register an installed/runtime role.
func publishNativeAsset(output string, body []byte, mode os.FileMode) error {
	if mode != 0555 && mode != 0444 || len(body) == 0 {
		return errors.New("native asset mode or body is invalid")
	}
	parent := filepath.Dir(output)
	canonical, err := filepath.EvalSymlinks(parent)
	if err != nil || canonical != parent {
		return errors.New("native asset parent is unavailable or not canonical")
	}
	directory, err := os.OpenRoot(parent)
	if err != nil {
		return err
	}
	defer directory.Close()
	before, err := directory.Stat(".")
	if err != nil || !nativeOutputDirectoryOwned(before) {
		return errors.New("native asset parent is not an owned protected directory")
	}
	if _, err := directory.Lstat(filepath.Base(output)); !errors.Is(err, os.ErrNotExist) {
		return errors.New("native native output already exists or cannot be checked")
	}
	temporary := ".native-" + rand.Text()
	file, err := directory.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	created, err := file.Stat()
	if err != nil {
		return err
	}
	defer func() {
		// A foreign replacement is never removed, even on failure.
		if info, err := directory.Lstat(temporary); err == nil && os.SameFile(created, info) {
			_ = directory.Remove(temporary)
		}
	}()
	if count, err := file.Write(body); err != nil || count != len(body) {
		return errors.Join(errors.New("native native asset write was incomplete"), err)
	}
	if err := errors.Join(file.Chmod(mode), file.Sync()); err != nil {
		return err
	}
	readback, err := directory.Open(temporary)
	if err != nil {
		return err
	}
	readInfo, statErr := readback.Stat()
	readBody, readErr := io.ReadAll(io.LimitReader(readback, int64(len(body))+1))
	closeErr := readback.Close()
	if statErr != nil || readErr != nil || closeErr != nil || !os.SameFile(created, readInfo) || len(readBody) != len(body) || hash(readBody) != hash(body) {
		return errors.New("native native asset readback differs from the authenticated bytes")
	}
	current, err := os.Lstat(parent)
	if err != nil || !os.SameFile(before, current) || !nativeOutputDirectoryOwned(current) {
		return errors.New("native asset parent changed before publication")
	}
	staged, err := directory.Lstat(temporary)
	if err != nil || !staged.Mode().IsRegular() || !os.SameFile(created, staged) {
		return errors.New("native temporary native asset changed")
	}
	// Link is atomic and refuses any existing destination. Both operands stay
	// within the opened parent, including if its pathname is renamed.
	if err := directory.Link(temporary, filepath.Base(output)); err != nil {
		return err
	}
	published, err := directory.Lstat(filepath.Base(output))
	if err != nil || !os.SameFile(created, published) || published.Mode().Perm() != mode || published.Size() != int64(len(body)) {
		return errors.New("native native asset publication changed")
	}
	if err := directory.Remove(temporary); err != nil {
		return err
	}
	parentFile, err := directory.Open(".")
	if err != nil {
		return err
	}
	if err := errors.Join(parentFile.Sync(), parentFile.Close()); err != nil {
		return err
	}
	current, err = os.Lstat(parent)
	if err != nil || !os.SameFile(before, current) || !nativeOutputDirectoryOwned(current) {
		return errors.New("native asset parent changed after publication")
	}
	return nil
}

func nativeOutputDirectoryOwned(info os.FileInfo) bool {
	owner, ok := info.Sys().(*syscall.Stat_t)
	return ok && owner.Uid == uint32(os.Geteuid()) && info.IsDir() && info.Mode().Perm()&0022 == 0
}
