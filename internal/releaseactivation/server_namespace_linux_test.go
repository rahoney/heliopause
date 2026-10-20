//go:build linux

package releaseactivation

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestInstallerSocketInDisposableNamespace(t *testing.T) {
	if os.Getenv("HELOX_INSTALLER_SOCKET_NAMESPACE") != "1" {
		t.Skip("explicit disposable root installer socket qualification")
	}
	if os.Getenv("HELOX_INSTALLER_SOCKET_CLIENT_CHILD") == "1" {
		if os.Geteuid() != 1000 || os.Getegid() != 1000 {
			t.Fatal("ordinary client required")
		}
		client, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: installerSocketPath, Net: "unix"})
		if err != nil {
			t.Fatal(err)
		}
		if err := verifyInstallerEndpoint(client, 1000); err != nil {
			t.Fatal("ordinary client root endpoint verification failed", err)
		}
		if err := client.Close(); err != nil {
			t.Fatal(err)
		}
		return
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
	client, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: installerSocketPath, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyInstallerEndpoint(client, 1000); err != nil {
		t.Fatal("actual root endpoint rejected", err)
	}
	if err := verifyInstallerEndpoint(client, 1001); !errors.Is(err, ErrUntrusted) {
		t.Fatal("foreign socket group accepted", err)
	}
	if err := os.Chmod(installerSocketPath, 0666); err != nil {
		t.Fatal(err)
	}
	if err := verifyInstallerEndpoint(client, 1000); !errors.Is(err, ErrUntrusted) {
		t.Fatal("public-write socket accepted", err)
	}
	if err := os.Chmod(installerSocketPath, 0660); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(installerRunDirectory, 0777); err != nil {
		t.Fatal(err)
	}
	if err := verifyInstallerEndpoint(client, 1000); !errors.Is(err, ErrUntrusted) {
		t.Fatal("unprotected socket parent accepted", err)
	}
	if err := os.Chmod(installerRunDirectory, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chown(installerSocketPath, 1000, 1000); err != nil {
		t.Fatal(err)
	}
	if err := verifyInstallerEndpoint(client, 1000); !errors.Is(err, ErrUntrusted) {
		t.Fatal("ordinary-owned endpoint accepted", err)
	}
	if err := os.Chown(installerSocketPath, 0, 1000); err != nil {
		t.Fatal(err)
	}
	if err := verifyInstallerEndpoint(client, 1000); err != nil {
		t.Fatal("restored exact metadata rejected", err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	childContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	child := exec.CommandContext(childContext, os.Args[0], "-test.run=^TestInstallerSocketInDisposableNamespace$", "-test.count=1")
	child.Env = []string{"PATH=/usr/bin:/bin", "HELOX_INSTALLER_SOCKET_NAMESPACE=1", "HELOX_INSTALLER_SOCKET_CLIENT_CHILD=1"}
	child.Dir = "/"
	child.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 1000, Gid: 1000}}
	if output, err := child.CombinedOutput(); err != nil {
		t.Fatalf("actual ordinary endpoint verifier failed: %v %s", err, output)
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
	t.Log("PASS: client root SO_PEERCRED/parent/mode/UID/GID admission, live denial, strict stopped-node ECONNREFUSED recovery and unknown replacement retention; no activation")
}
