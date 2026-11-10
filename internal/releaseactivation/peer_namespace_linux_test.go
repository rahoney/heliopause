//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/releaseinstall"
)

// This snapshot is deliberately test-only. Kernel UID/inode/pidfd mechanics
// are tested here; ReadInstalledRelease remains the production crypto issuer.
func peerKernelFixture() InstalledRelease {
	return InstalledRelease{record: registration{Schema: registrationSchema, Version: "v1.2.3", ClientUID: 1000, ClientGID: 1000}, proof: []byte("kernel-test-only")}
}

func TestProtectedPeerInDisposableNamespace(t *testing.T) {
	if os.Getenv("HELOX_PEER_AUTH_NAMESPACE") != "1" {
		t.Skip("explicit disposable kernel peer qualification")
	}
	if mode := os.Getenv("HELOX_PEER_AUTH_CHILD"); mode != "" {
		err := CheckRuntimeController(peerKernelFixture())
		if (err == nil) != (mode == "good") {
			t.Fatal("current controller identity gate differs", err)
		}
		connection, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: filepath.Join(releaseinstall.SystemInstallRoot, "caller.sock"), Net: "unix"})
		if err != nil {
			t.Fatal(err)
		}
		defer connection.Close()
		if _, err := io.Copy(io.Discard, connection); err != nil {
			t.Fatal(err)
		}
		return
	}
	if os.Geteuid() != 0 {
		t.Fatal("kernel peer qualification requires root")
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
	if err := syscall.Mount("tmpfs", "/usr/lib", "tmpfs", syscall.MS_NODEV|syscall.MS_NOSUID, "mode=0755,size=134217728"); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1.2.3", "v1.2.4"} {
		path := filepath.Join(releaseinstall.SystemInstallRoot, "versions", version)
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
		for current := path; current != filepath.Dir(releaseinstall.SystemInstallRoot); current = filepath.Dir(current) {
			if err := os.Chmod(current, 0755); err != nil {
				t.Fatal(err)
			}
		}
	}
	body, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(releaseinstall.SystemInstallRoot, "versions", "v1.2.3", "helox-linux-amd64")
	bad := filepath.Join(releaseinstall.SystemInstallRoot, "versions", "v1.2.4", "helox-linux-amd64")
	for _, path := range []string{good, bad} {
		if err := os.WriteFile(path, body, 0555); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0555); err != nil {
			t.Fatal(err)
		}
	}
	for _, test := range []struct {
		name, path string
		uid, gid   uint32
	}{
		{"good", good, 1000, 1000},
		{"wrong-inode", bad, 1000, 1000},
		{"foreign-user", good, 1001, 1000},
		{"foreign-group", good, 1000, 1001},
		{"root-user", good, 0, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(releaseinstall.SystemInstallRoot, "caller.sock")
			listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			if err := os.Chown(path, 0, 1000); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(path, 0660); err != nil {
				t.Fatal(err)
			}
			if err := listener.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, test.path, "-test.run=^TestProtectedPeerInDisposableNamespace$", "-test.count=1")
			command.Dir = "/"
			command.Env = []string{"PATH=/usr/bin:/bin", "HELOX_PEER_AUTH_NAMESPACE=1", "HELOX_PEER_AUTH_CHILD=" + test.name}
			command.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: test.uid, Gid: test.gid, Groups: []uint32{998, 1000}}}
			var output bytes.Buffer
			command.Stdout = &output
			command.Stderr = &output
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			waited := false
			defer func() {
				if !waited {
					cancel()
					_ = command.Wait()
				}
			}()
			connection, err := listener.AcceptUnix()
			if err != nil {
				t.Fatal("accept controlled child", err)
			}
			defer connection.Close()
			peer, err := authorizeOldController(connection, peerKernelFixture())
			if test.name == "good" {
				if err != nil {
					_ = connection.Close()
					t.Fatal("protected ordinary caller rejected", err)
				}
				defer peer.Close()
				if peer.uid != 1000 || peer.gid != 1000 || len(peer.groups) != 2 || peer.groups[0] != 998 || peer.groups[1] != 1000 {
					t.Fatal("kernel identity/groups changed")
				}
				if err := peer.alive(); err != nil {
					t.Fatal(err)
				}
			} else if !errors.Is(err, ErrCaller) || peer != nil {
				_ = connection.Close()
				t.Fatal("foreign UID/GID/inode authorized", err)
			}
			if err := connection.Close(); err != nil {
				t.Fatal(err)
			}
			if err := command.Wait(); err != nil {
				t.Fatalf("controlled child %v\n%s", err, output.Bytes())
			}
			waited = true
			if peer != nil {
				if err := peer.alive(); !errors.Is(err, ErrCaller) {
					t.Fatal("dead caller remained authorized")
				}
			}
		})
	}
	t.Log("PASS: actual protected inode/ordinary UID/GID/groups and pidfd lifecycle; same basename wrong inode/root/foreign UID/GID rejected; crypto snapshot mocked; no activation")
}
