//go:build linux && realcorpus

package promotion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// A real bounded tmpfs write failure qualifies the existing state-publication
// rollback. Source/SRI are structural fixtures; no native package is executed.
func TestBunActualENOSPCInDisposableNamespace(t *testing.T) {
	if os.Getenv("HELOX_BUN_ENOSPC_NAMESPACE") != "1" || os.Geteuid() != 0 {
		t.Fatal("requires declared Root in disposable namespaces")
	}
	for _, kind := range []string{"mnt", "net"} {
		self, err := os.Readlink("/proc/self/ns/" + kind)
		host, other := os.Readlink("/proc/1/ns/" + kind)
		if err != nil || other != nil || self == host {
			t.Fatal("refusing mount outside disposable namespaces")
		}
	}
	f, owner, staged, update := bunTransactionFixture(t)
	if err := os.Mkdir(owner.engine.stateRoot, 0700); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mount("tmpfs", owner.engine.stateRoot, "tmpfs", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC, "size=1048576,nr_inodes=64,mode=0700"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := syscall.Unmount(owner.engine.stateRoot, 0); err != nil {
			t.Error("owned tmpfs unmount failed", err)
		}
	}()
	var filesystem syscall.Statfs_t
	if err := syscall.Statfs(owner.engine.stateRoot, &filesystem); err != nil || filesystem.Type != 0x01021994 || filesystem.Blocks*uint64(filesystem.Bsize) > 1<<20 {
		t.Fatal("not the bounded task tmpfs", err)
	}
	root := f.cache.frozen.Snapshot().Context().Target().String()
	original, err := os.Lstat(filepath.Join(root, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	filler := filepath.Join(owner.engine.stateRoot, "owned-space-pressure")
	reached := false
	var actualWriteErr error
	owner.engine.checkpoint = func(phase string) error {
		if phase != "before-state" {
			return nil
		}
		if reached {
			return errors.New("state checkpoint repeated")
		}
		reached = true
		file, err := os.OpenFile(filler, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		block := make([]byte, 4096)
		for total := 0; total < 2<<20; total += len(block) {
			if _, err := file.Write(block); err != nil {
				actualWriteErr = err
				break
			}
		}
		closeErr := file.Close()
		if !errors.Is(actualWriteErr, syscall.ENOSPC) {
			return errors.Join(errors.New("bounded task tmpfs did not report actual ENOSPC"), actualWriteErr, closeErr)
		}
		return closeErr
	}
	guard, err := owner.Begin(context.Background(), f.cache.frozen.Snapshot().Context())
	if err != nil {
		t.Fatal(err)
	}
	err = guard.Commit(context.Background(), update, staged)
	if !reached || !errors.Is(actualWriteErr, syscall.ENOSPC) || !errors.Is(err, syscall.ENOSPC) {
		t.Fatal("actual publication lost ENOSPC cause", err, actualWriteErr)
	}
	if err := guard.Close(); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil || string(body) != "{}" {
		t.Fatal("disk full lost original bytes", err)
	}
	after, err := os.Lstat(filepath.Join(root, "package.json"))
	if err != nil || !os.SameFile(original, after) || original.Mode() != after.Mode() {
		t.Fatal("disk full replaced original inode/mode", err)
	}
	for _, name := range []string{"bun.lock", "node_modules", ".heliopause", ".heliopause-npm-transaction.lock"} {
		if _, err := os.Lstat(filepath.Join(root, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("uncommitted member survived disk full", name, err)
		}
	}
	journals, err := filepath.Glob(filepath.Join(root, ".heliopause-npm-commit-*"))
	if err != nil || len(journals) != 0 {
		t.Fatal("known rollback journal survived", err, journals)
	}
	entries, err := os.ReadDir(owner.engine.stateRoot)
	if err != nil || len(entries) != 1 || entries[0].Name() != filepath.Base(filler) {
		t.Fatal("partial approval/temporary file survived", err, entries)
	}
	if _, _, _, ok := owner.PublishedRecords(); ok {
		t.Fatal("failed disk-full publication exposed records")
	}
	directory, err := os.Open("/proc/self/fd")
	if err != nil {
		t.Fatal(err)
	}
	names, readErr := directory.Readdirnames(-1)
	if readErr != nil {
		_ = directory.Close()
		t.Fatal(readErr)
	}
	for _, name := range names {
		link, err := os.Readlink("/proc/self/fd/" + name)
		if err != nil {
			_ = directory.Close()
			t.Fatal(err)
		}
		if strings.Contains(link, root) || strings.Contains(link, owner.engine.stateRoot) {
			_ = directory.Close()
			t.Fatal("transaction descriptor retained", link)
		}
	}
	if err := directory.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filler); err != nil {
		t.Fatal(err)
	}
	t.Log("actual_enospc_before_state: syscall_enospc=true cause_preserved=true original_inode_mode_restored=true partial_approval_absent=true task_fds=0 publication=false")
}
