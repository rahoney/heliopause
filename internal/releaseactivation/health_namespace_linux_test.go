//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestProtectedHealthChannelInDisposableNamespace(t *testing.T) {
	if os.Getenv("HELOX_HEALTH_NAMESPACE") != "1" {
		t.Skip("explicit disposable root health qualification")
	}
	if os.Getenv("HELOX_HEALTH_ROLE") == "issuer" {
		release, _ := wireFixture(t)
		old := InstalledRelease{record: registration{registrationSchema, "NATIVE", "v0.1.0", release.manifest.SHA256(), 1000, 1000}, manifest: release.manifest, proof: []byte("mock-current-view")}
		next := old
		next.record.Version = "v0.1.1"
		fd, err := unix.PidfdOpen(os.Getpid(), 0)
		if err != nil {
			t.Fatal(err)
		}
		peer := &controllerPeer{pidfd: os.NewFile(uintptr(fd), "mock-old-peer-liveness"), uid: 1000, gid: 1000, groups: []uint32{1000}}
		defer peer.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		workspace, err := runCompatibilityHealth(ctx, old, next, peer, ".prepared-kernel-health", true)
		if err == nil || !errors.Is(err, io.EOF) {
			t.Fatal("incomplete old signed view borrowed access, or retained sender FD masked child EOF", err)
		}
		if ctx.Err() != nil {
			t.Fatal("dead child was not detected before deadline")
		}
		if workspace == nil {
			t.Fatal("owned workspace missing")
		}
		if _, err := os.Lstat(filepath.Join(workspace.path, "unexpected-health-callback")); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("unverified channel ran health callback")
		}
		if err := workspace.Close(); err != nil {
			t.Fatal(err)
		}
		if err := workspace.Close(); err != nil {
			t.Fatal("health workspace Close is not idempotent")
		}
		workspace, err = createHealthWorkspace(".prepared-replaced-health", next)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(workspace.path, workspace.path+"-owned"); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(workspace.path, 0700); err != nil {
			t.Fatal(err)
		}
		if err := workspace.Close(); !errors.Is(err, ErrCleanup) {
			t.Fatal("unknown health replacement was deleted")
		}
		if _, err := os.Lstat(workspace.path); err != nil {
			t.Fatal("unknown replacement not retained")
		}
		t.Log("PASS: real root issuer/ordinary child FD transfer and rejected incomplete signed view before borrowing; actual EOF/wait and owned workspace cleanup; crypto/old peer snapshot mocked")
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
	for _, path := range []string{"/var/lib", "/usr/lib"} {
		if err := syscall.Mount("tmpfs", path, "tmpfs", syscall.MS_NODEV|syscall.MS_NOSUID, "mode=0755,size=134217728"); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(stateRoot, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(stateRoot, 0755); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	issuer := nativeVersionPath("v0.1.0", "haa-release-installer-linux-amd64")
	for _, path := range []string{issuer, nativeVersionPath("v0.1.1", "helox-linux-amd64")} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		for current := filepath.Dir(path); current != filepath.Dir(current); current = filepath.Dir(current) {
			if err := os.Chmod(current, 0755); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(path, body, 0555); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0555); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, issuer, "-test.run=^TestProtectedHealthChannelInDisposableNamespace$", "-test.v")
	child.Env = []string{"HELOX_HEALTH_NAMESPACE=1", "HELOX_HEALTH_ROLE=issuer"}
	var output bytes.Buffer
	child.Stdout = &output
	child.Stderr = &output
	if err := child.Run(); err != nil {
		t.Fatal("root issuer driver failed", err, output.String())
	}
	t.Log(output.String())
}
