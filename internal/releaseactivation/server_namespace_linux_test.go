//go:build linux

package releaseactivation

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

func TestInstallerSocketInDisposableNamespace(t *testing.T) {
	if os.Getenv("HELOX_INSTALLER_SOCKET_NAMESPACE") != "1" {
		t.Skip("explicit disposable root installer socket qualification")
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
	if err := syscall.Mount("tmpfs", "/run", "tmpfs", syscall.MS_NODEV|syscall.MS_NOSUID, "mode=0755,size=16777216"); err != nil {
		t.Fatal(err)
	}
	listener, identity, err := listenInstaller(1000)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := listenInstaller(1000); !errors.Is(err, ErrBusy) {
		t.Fatal("live/unknown listener replaced", err)
	}
	if err := closeInstallerListener(listener, identity); err != nil {
		t.Fatal(err)
	}
	fd, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := unix.Bind(fd, &unix.SockaddrUnix{Name: installerSocketPath}); err != nil {
		_ = unix.Close(fd)
		t.Fatal(err)
	}
	if err := os.Chmod(installerSocketPath, 0660); err != nil {
		t.Fatal(err)
	}
	if err := os.Chown(installerSocketPath, 0, 1000); err != nil {
		t.Fatal(err)
	}
	if err := unix.Close(fd); err != nil {
		t.Fatal(err)
	}
	listener, identity, err = listenInstaller(1000)
	if err != nil {
		t.Fatal("confirmed stopped root-reserved endpoint not reclaimed", err)
	}
	if err := os.Rename(installerSocketPath, filepath.Join(installerRunDirectory, "retained-owned.sock")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installerSocketPath, []byte("unknown replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := closeInstallerListener(listener, identity); !errors.Is(err, ErrCleanup) {
		t.Fatal("unknown replacement treated as owned", err)
	}
	body, err := os.ReadFile(installerSocketPath)
	if err != nil || string(body) != "unknown replacement" {
		t.Fatal("unknown replacement deleted")
	}
	t.Log("PASS: actual root socket live denial, strict stopped-node ECONNREFUSED recovery and unknown replacement retention; no activation")
}
