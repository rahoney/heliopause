//go:build linux

package releaseinstall

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"
)

type interruptedCopyContext struct {
	context.Context
	calls int
}

func (c *interruptedCopyContext) Err() error {
	c.calls++
	if c.calls == 3 {
		ready := os.NewFile(3, "copy-interruption-ready")
		if _, err := ready.Write([]byte{1}); err != nil {
			return err
		}
		if err := ready.Close(); err != nil {
			return err
		}
		// Test-only fault barrier after one metadata copy; the parent sends an
		// actual SIGKILL. No product hook or timing-based sleep changes the copier.
		select {}
	}
	return nil
}

func resetCopyNamespace(t *testing.T) {
	t.Helper()
	for _, path := range []string{"/usr/lib", "/var/lib"} {
		if err := syscall.Mount("tmpfs", path, "tmpfs", syscall.MS_NODEV|syscall.MS_NOSUID, "mode=0755,size=67108864"); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{SystemInstallRoot, filepath.Join(SystemInstallRoot, "versions"), SystemStagingRoot} {
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0755); err != nil {
			t.Fatal(err)
		}
	}
}

func pendingInventory(t *testing.T, path string) map[string]string {
	t.Helper()
	out := map[string]string{}
	var visit func(string)
	visit = func(p string) {
		info, err := os.Lstat(p)
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		owner, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			t.Fatal("missing kernel owner")
		}
		record, _ := json.Marshal(struct {
			Mode     os.FileMode
			UID, GID uint32
			Inode    uint64
			Links    uint64
			Data     string
		}{info.Mode(), owner.Uid, owner.Gid, owner.Ino, uint64(owner.Nlink), ""})
		detail := string(record)
		if info.Mode().IsRegular() {
			body, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(body)
			detail += hex.EncodeToString(sum[:])
		} else if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(p)
			if err != nil {
				t.Fatal(err)
			}
			detail += target
		}
		out[p] = detail
		if info.IsDir() {
			entries, err := os.ReadDir(p)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				visit(filepath.Join(p, entry.Name()))
			}
		}
	}
	visit(path)
	return out
}

func TestInterruptedSystemCopyInDisposableNamespace(t *testing.T) {
	if os.Getenv("HELOX_COPY_INTERRUPTION_NAMESPACE") != "1" {
		t.Skip("explicit disposable root copy interruption qualification")
	}
	if os.Getenv("HELOX_COPY_INTERRUPTION_CHILD") == "1" {
		bundle := filepath.Join(SystemStagingRoot, "transaction-killed")
		value, body, err := readManifest(bundle)
		if err != nil {
			t.Fatal(err)
		}
		files, err := selectFiles(value, "linux", "amd64")
		if err != nil {
			t.Fatal(err)
		}
		_, err = activate(&interruptedCopyContext{Context: context.Background()}, SystemInstallRoot, value.Version, body, files, bundle, true)
		t.Fatal("interruption child unexpectedly returned", err)
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
	kinds := []string{"empty", "prefix", "complete", "umask", "installed-source", "unrelated-pending", "unknown", "bad-prefix", "oversized", "bad-mode", "hardlink", "symlink", "foreign-owner", "foreign-group", "foreign-dir", "setgid-dir", "directory-member", "symlink-stage", "wrong-verifier", "canceled", "non-system", "wrong-root", "reservation-collision", "sigkill"}
	for _, kind := range kinds {
		t.Run(kind, func(t *testing.T) {
			resetCopyNamespace(t)
			ctx := context.Background()
			suffix := kind
			if kind == "sigkill" {
				suffix = "killed"
			}
			bundle := systemBundleFixture(t, "v0.1.0", suffix)
			value, body, err := readManifest(bundle)
			if err != nil {
				t.Fatal(err)
			}
			installer, err := NewSystemInstaller(allowVerifier{})
			if err != nil {
				t.Fatal(err)
			}
			request := Request{BundleDirectory: bundle, InstallRoot: SystemInstallRoot}
			sum := sha256.Sum256(body)
			name, err := SystemCopyStageName(hex.EncodeToString(sum[:]))
			if err != nil {
				t.Fatal(err)
			}
			directory := filepath.Join(SystemInstallRoot, "versions", name)
			if kind == "installed-source" {
				if _, err := installer.Install(ctx, request); err != nil {
					t.Fatal(err)
				}
				request.BundleDirectory = filepath.Join(SystemInstallRoot, "versions", value.Version)
			}
			if kind == "sigkill" {
				read, write, err := os.Pipe()
				if err != nil {
					t.Fatal(err)
				}
				defer read.Close()
				bounded, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()
				command := exec.CommandContext(bounded, "/proc/self/exe", "-test.run=^TestInterruptedSystemCopyInDisposableNamespace$")
				command.Env = append(os.Environ(), "HELOX_COPY_INTERRUPTION_CHILD=1")
				command.ExtraFiles = []*os.File{write}
				var diagnostic bytes.Buffer
				command.Stdout = &diagnostic
				command.Stderr = &diagnostic
				if err := command.Start(); err != nil {
					_ = write.Close()
					t.Fatal(err)
				}
				if err := write.Close(); err != nil {
					_ = command.Process.Kill()
					_ = command.Wait()
					t.Fatal(err)
				}
				if err := read.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
					_ = command.Process.Kill()
					_ = command.Wait()
					t.Fatal(err)
				}
				var ready [1]byte
				if _, err := io.ReadFull(read, ready[:]); err != nil {
					_ = command.Process.Kill()
					_ = command.Wait()
					t.Fatalf("copy barrier failed: %v %s", err, diagnostic.String())
				}
				killErr := command.Process.Kill()
				waitErr := command.Wait()
				var exit *exec.ExitError
				if killErr != nil || !errors.As(waitErr, &exit) {
					t.Fatal("SIGKILL not confirmed", killErr, waitErr)
				}
				status, ok := exit.Sys().(syscall.WaitStatus)
				if !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
					t.Fatal("unexpected child termination", waitErr)
				}
				entries, err := os.ReadDir(directory)
				if err != nil || len(entries) != 1 || entries[0].Name() != manifestName {
					t.Fatal("copy was not interrupted in reserved stage", entries, err)
				}
				if _, err := os.Lstat(filepath.Join(SystemInstallRoot, "current")); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("killed copier published current", err)
				}
			} else {
				if err := os.Mkdir(directory, 0700); err != nil {
					t.Fatal(err)
				}
				if kind != "empty" {
					if kind == "complete" {
						names := []string{manifestName, sbomName, runtimeImagesName}
						files, err := selectFiles(value, "linux", "amd64")
						if err != nil {
							t.Fatal(err)
						}
						for _, file := range files {
							names = append(names, file.Name)
						}
						for _, name := range names {
							mode := os.FileMode(0555)
							if name == manifestName || name == sbomName || name == runtimeImagesName {
								mode = 0444
							}
							payload, err := os.ReadFile(filepath.Join(bundle, name))
							if err != nil {
								t.Fatal(err)
							}
							if err := writeNew(filepath.Join(directory, name), payload, mode); err != nil {
								t.Fatal(err)
							}
						}
						record, err := installationRecord(value.Version, body)
						if err != nil {
							t.Fatal(err)
						}
						if err := writeNew(filepath.Join(directory, installRecordName), record, 0444); err != nil {
							t.Fatal(err)
						}
						if err := os.Chmod(directory, 0755); err != nil {
							t.Fatal(err)
						}
					} else {
						payload, err := os.ReadFile(filepath.Join(bundle, "helox-linux-amd64"))
						if err != nil {
							t.Fatal(err)
						}
						partial := payload[:len(payload)/2]
						if kind == "bad-prefix" {
							partial = []byte("mismatch")
						}
						if kind == "oversized" {
							partial = append(payload, 'x')
						}
						if err := writeNew(filepath.Join(directory, "helox-linux-amd64"), partial, 0555); err != nil {
							t.Fatal(err)
						}
					}
				}
			}
			member := filepath.Join(directory, "helox-linux-amd64")
			switch kind {
			case "umask":
				if err := os.Chmod(member, 0500); err != nil {
					t.Fatal(err)
				}
			case "unrelated-pending":
				if err := os.Mkdir(filepath.Join(SystemInstallRoot, "versions", ".pending-foreign"), 0700); err != nil {
					t.Fatal(err)
				}
			case "unknown":
				if err := writeNew(filepath.Join(directory, "foreign.txt"), []byte("retain"), 0400); err != nil {
					t.Fatal(err)
				}
			case "bad-mode":
				if err := os.Chmod(member, 0666); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(member, filepath.Join(SystemInstallRoot, "other-link")); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Remove(member); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(bundle, "helox-linux-amd64"), member); err != nil {
					t.Fatal(err)
				}
			case "foreign-owner":
				if err := os.Chown(member, 1000, 0); err != nil {
					t.Fatal(err)
				}
			case "foreign-group":
				if err := os.Chown(member, 0, 1000); err != nil {
					t.Fatal(err)
				}
			case "foreign-dir":
				if err := os.Chown(directory, 1000, 0); err != nil {
					t.Fatal(err)
				}
			case "setgid-dir":
				if err := os.Chmod(directory, 0700|os.ModeSetgid); err != nil {
					t.Fatal(err)
				}
			case "directory-member":
				if err := os.Remove(member); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(member, 0700); err != nil {
					t.Fatal(err)
				}
			case "symlink-stage":
				if err := os.Remove(member); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(directory); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(bundle, directory); err != nil {
					t.Fatal(err)
				}
			case "wrong-verifier":
				installer.verifier = denyVerifier{}
			case "canceled":
				canceled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = canceled
			case "non-system":
				installer, err = New(allowVerifier{})
				if err != nil {
					t.Fatal(err)
				}
			case "wrong-root":
				request.InstallRoot = "/tmp/foreign"
			}
			before := pendingInventory(t, directory)
			if kind == "reservation-collision" {
				if _, err := installer.Install(ctx, request); err == nil {
					t.Fatal("existing copy directory adopted")
				}
				if after := pendingInventory(t, directory); !reflect.DeepEqual(before, after) {
					t.Fatal("existing copy directory changed")
				}
				return
			}
			result := installer.DiscardSystemCopy(ctx, request)
			positive := kind == "empty" || kind == "prefix" || kind == "complete" || kind == "umask" || kind == "installed-source" || kind == "unrelated-pending" || kind == "sigkill"
			if positive {
				if result != nil {
					t.Fatal(result)
				}
				if _, err := os.Lstat(directory); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("owned copy retained", err)
				}
				if err := installer.DiscardSystemCopy(ctx, request); err != nil {
					t.Fatal("repeat cleanup failed", err)
				}
				if kind == "unrelated-pending" {
					if _, err := os.Stat(filepath.Join(SystemInstallRoot, "versions", ".pending-foreign")); err != nil {
						t.Fatal("unrelated copy removed", err)
					}
				}
			} else {
				if result == nil {
					t.Fatal("unsafe cleanup accepted")
				}
				if after := pendingInventory(t, directory); !reflect.DeepEqual(before, after) {
					t.Fatal("failed cleanup changed owned/unknown set")
				}
				if kind == "canceled" && !errors.Is(result, context.Canceled) {
					t.Fatal("cancellation lost", result)
				}
			}
		})
	}
}
