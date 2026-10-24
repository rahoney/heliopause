//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/rahoney/heliopause/internal/releaseinstall"
	"golang.org/x/sys/unix"
)

func TestProtectedStageInDisposableNamespace(t *testing.T) {
	if os.Getenv("HELOX_STAGE_NAMESPACE") != "1" {
		t.Skip("explicit disposable root stage qualification")
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
	if err := syscall.Mount("tmpfs", "/var/lib", "tmpfs", syscall.MS_NODEV|syscall.MS_NOSUID, "mode=0755,size=67108864"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(stateRoot, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(stateRoot, 0755); err != nil {
		t.Fatal(err)
	}
	release, files := wireFixture(t)
	fd, err := unix.PidfdOpen(os.Getpid(), 0)
	if err != nil {
		t.Fatal(err)
	}
	// Auth/current snapshot are mocked only for isolated filesystem mechanics.
	peer := &controllerPeer{pidfd: os.NewFile(uintptr(fd), "stage-kernel-test"), uid: 1000, gid: 1000}
	defer peer.Close()
	current := InstalledRelease{record: registration{Schema: registrationSchema, Version: release.manifest.Version(), ClientUID: 1000, ClientGID: 1000}, manifest: release.manifest, proof: []byte("test-current")}
	ctx := context.Background()
	var preamble bytes.Buffer
	if err := writeWirePreamble(ctx, &preamble, release); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareWireRelease(ctx, &preamble, peer, current); !errors.Is(err, ErrUntrusted) {
		t.Fatal("production SDK approved an unsigned new native set", err)
	}
	if _, err := os.Lstat(releaseinstall.SystemStagingRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("signature rejection wrote root staging")
	}
	payload := func() []byte {
		var stream bytes.Buffer
		if err := writeWirePayloads(ctx, &stream, release, func(name string) (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(files[name])), nil }); err != nil {
			t.Fatal(err)
		}
		return stream.Bytes()
	}
	stage, err := stageCheckedWire(ctx, bytes.NewReader(payload()), peer, release, release.proofs[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range release.files {
		path := filepath.Join(stage.directory, file.Name)
		body, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(body, files[file.Name]) {
			t.Fatal("root copy differs", err)
		}
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		owner, ok := info.Sys().(*syscall.Stat_t)
		if !ok || owner.Uid != 0 || owner.Nlink != 1 || info.Mode().Perm() != 0600 {
			t.Fatal("root-private stage member identity differs")
		}
	}
	if err := stage.Close(); err != nil {
		t.Fatal(err)
	}
	if err := stage.Close(); err != nil {
		t.Fatal("stage Close not idempotent")
	}
	for _, kind := range []string{"digest", "truncated", "marker"} {
		t.Run(kind, func(t *testing.T) {
			body := payload()
			switch kind {
			case "digest":
				body[0] ^= 1
			case "truncated":
				body = body[:len(body)/2]
			case "marker":
				body[len(body)-1] = '!'
			}
			if _, err := stageCheckedWire(ctx, bytes.NewReader(body), peer, release, release.proofs[0]); err == nil {
				t.Fatal("bad payload became a prepared stage")
			}
			remaining, err := os.ReadDir(releaseinstall.SystemStagingRoot)
			if err != nil || len(remaining) != 0 {
				t.Fatal("partial root stage leaked", err)
			}
		})
	}
	stage, err = stageCheckedWire(ctx, bytes.NewReader(payload()), peer, release, release.proofs[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(stage.directory, stage.directory+"-owned"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(stage.directory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := stage.Close(); !errors.Is(err, ErrCleanup) {
		t.Fatal("unknown stage ownership was treated as cleaned")
	}
	if _, err := os.Stat(stage.directory); err != nil {
		t.Fatal("foreign replacement was deleted")
	}
	t.Log("PASS: production unsigned-set rejection before root writes; mocked-auth/crypto filesystem positive and size/digest/marker/cleanup negatives; no activation")
}
