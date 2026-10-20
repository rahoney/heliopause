//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/releaseinstall"
	"golang.org/x/sys/unix"
)

func TestProtectedJournalAndLocksInDisposableNamespace(t *testing.T) {
	if os.Getenv("HELOX_JOURNAL_NAMESPACE") != "1" {
		t.Skip("explicit disposable root journal qualification")
	}
	if os.Geteuid() != 0 {
		t.Fatal("root namespace required")
	}
	for _, kind := range []string{"mnt", "net"} {
		self, err := os.Readlink("/proc/self/ns/" + kind)
		if err != nil {
			t.Fatal(err)
		}
		initial, err := os.Readlink("/proc/1/ns/" + kind)
		if err != nil || self == initial {
			t.Fatal("separate namespaces required")
		}
	}
	if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/var/lib", "/etc"} {
		if err := syscall.Mount("tmpfs", path, "tmpfs", syscall.MS_NODEV|syscall.MS_NOSUID, "mode=0755,size=16777216"); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{stateRoot, "/etc/heliopause", "/etc/systemd", "/etc/systemd/system"} {
		if err := os.Mkdir(path, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0755); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	for _, path := range []string{runtimeLockPath, updateLockPath} {
		if err := os.WriteFile(path, []byte("persistent"), 0444); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0444); err != nil {
			t.Fatal(err)
		}
	}
	owner, err := acquireInstallerOwner(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acquireInstallerOwner(ctx); !errors.Is(err, ErrBusy) {
		t.Fatal("duplicate root installer was admitted", err)
	}
	if err := owner.Close(); err != nil {
		t.Fatal(err)
	}
	if err := owner.Close(); err != nil {
		t.Fatal("owner Close is not idempotent")
	}
	owner, err = acquireInstallerOwner(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	file, _, err := openProtectedFile(runtimeLockPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_SH|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	wait, cancel := context.WithTimeout(ctx, 40*time.Millisecond)
	if _, err := acquireExclusiveRuntime(wait); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("exclusive handoff ignored active shared caller", err)
	}
	cancel()
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	exclusive, err := acquireExclusiveRuntime(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if flags, err := unix.FcntlInt(exclusive.file.Fd(), unix.F_GETFD, 0); err != nil || flags&unix.FD_CLOEXEC == 0 {
		t.Fatal("exclusive lease leaks across exec", err)
	}
	if err := exclusive.Close(); err != nil {
		t.Fatal(err)
	}
	update, err := lockPersistentFile(ctx, updateLockPath, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lockPersistentFile(ctx, updateLockPath, false, false); !errors.Is(err, ErrBusy) {
		t.Fatal("duplicate ordinary update was admitted", err)
	}
	if err := update.Close(); err != nil {
		t.Fatal(err)
	}
	value := journalFixture()
	for index, path := range configPaths() {
		if err := os.WriteFile(path, value.Configs[index].Body, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0644); err != nil {
			t.Fatal(err)
		}
		snapshot, err := snapshotConfig(ctx, path)
		if err != nil || !bytes.Equal(snapshot.Body, value.Configs[index].Body) {
			t.Fatal("snapshot differs", err)
		}
	}
	// Exercise the actual PREPARED writer with owned stage identity and fixed
	// snapshots. Release authenticity is deliberately mocked for this mechanic.
	release, _ := wireFixture(t)
	if err := ensureStagingDirectory(); err != nil {
		t.Fatal(err)
	}
	directory, err := os.MkdirTemp(releaseinstall.SystemStagingRoot, ".prepared-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(directory, 0700); err != nil {
		t.Fatal(err)
	}
	stageIdentity, err := os.Lstat(directory)
	if err != nil {
		t.Fatal(err)
	}
	stage := &preparedRelease{directory: directory, identity: stageIdentity, release: release, proof: []byte("mocked-signed-stage")}
	current := InstalledRelease{record: registration{registrationSchema, "NATIVE", release.manifest.Version(), release.manifest.SHA256(), 1000, 1000}, manifest: release.manifest, proof: []byte("mocked-current")}
	prepared, err := createPreparedJournal(ctx, current, stage)
	if err != nil || prepared.value.State != "PREPARED" {
		t.Fatal("actual PREPARED writer failed", err)
	}
	readback, err := readDurableJournal(ctx)
	if err != nil || !bytes.Equal(prepared.body, readback.body) {
		t.Fatal("PREPARED durable readback differs", err)
	}
	if err := os.Remove(journalPath); err != nil {
		t.Fatal(err)
	}
	if err := stage.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := createPreparedJournal(ctx, current, stage); err == nil {
		t.Fatal("closed stage issued a PREPARED journal")
	}
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := replaceProtected(ctx, journalPath, nil, body, 0600, 0); err != nil {
		t.Fatal(err)
	}
	if err := replaceProtected(ctx, journalPath, nil, body, 0600, 0); err == nil {
		t.Fatal("create overwrote existing journal")
	}
	journal, err := readDurableJournal(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.transition(ctx, "COMMITTED"); !errors.Is(err, ErrTransition) {
		t.Fatal("PREPARED became COMMITTED without switch/verify")
	}
	for _, state := range []string{"SWITCHING", "VERIFYING", "COMMITTED"} {
		if err := journal.transition(ctx, state); err != nil {
			t.Fatal(err)
		}
	}
	info, err := os.Lstat(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	identity, ok := info.Sys().(*syscall.Stat_t)
	if !ok || identity.Uid != 0 || identity.Nlink != 1 || info.Mode().Perm() != 0600 {
		t.Fatal("journal protection differs")
	}
	if _, err := managedStatePresent(); !errors.Is(err, ErrTransition) {
		t.Fatal("terminal marker was treated as healthy absence")
	}
	if err := replaceProtected(ctx, journalPath, []byte("wrong old bytes"), body, 0600, 0); err == nil {
		t.Fatal("unknown old bytes overwritten")
	}
	for _, kind := range []string{"foreign", "hardlink", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			path := filepath.Join(stateRoot, "protected-fixture")
			if err := os.WriteFile(path, []byte("expected"), 0600); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "foreign":
				if err := os.Chown(path, 1000, 1000); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(path, path+"-link"); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Rename(path, path+"-target"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path+"-target", path); err != nil {
					t.Fatal(err)
				}
			}
			if err := replaceProtected(ctx, path, []byte("expected"), []byte("new"), 0600, 0); err == nil {
				t.Fatal("foreign file overwritten")
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Log("PASS: root journal snapshots/durable transitions/foreign-state denial, lock exclusion and canceled handoff; service/crypto/power-loss not qualified")
}
