//go:build linux || darwin

package releaseinstall

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestSystemCopyPermissionsIgnorePrivateUmask(t *testing.T) {
	if os.Getenv("HELOX_INSTALL_PERMISSION_CHILD") != "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSystemCopyPermissionsIgnorePrivateUmask$", "-test.count=1")
		command.Env = append(os.Environ(), "HELOX_INSTALL_PERMISSION_CHILD=1")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("private-umask copy test: %v\n%s", err, output)
		}
		return
	}
	previous := syscall.Umask(0077)
	defer syscall.Umask(previous)
	base := t.TempDir()
	bundle := releaseBundle(t, base, "v1.2.3", "permissions")
	value, body, err := readManifest(bundle)
	if err != nil {
		t.Fatal(err)
	}
	files, err := selectFiles(value, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "copy-mechanics")
	if err := ensureDirectoryMode(root, 0755); err != nil {
		t.Fatal(err)
	}
	// This exercises the shared copy primitive's actual kernel permissions,
	// not protected ownership, provenance or the privileged activation flow.
	result, err := activate(context.Background(), root, value.Version, body, files, bundle, true)
	if err != nil {
		t.Fatal(err)
	}
	for path, mode := range map[string]os.FileMode{
		root:                            0755,
		filepath.Join(root, "versions"): 0755,
		filepath.Join(root, "versions", result.Version):                      0755,
		filepath.Join(root, "versions", result.Version, "helox-linux-amd64"): 0555,
		filepath.Join(root, "versions", result.Version, manifestName):        0444,
		filepath.Join(root, "versions", result.Version, installRecordName):   0444,
	} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != mode {
			t.Fatalf("system copy mode %s=%v error=%v", path, info, err)
		}
	}
}
