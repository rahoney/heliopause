//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/releaseinstall"
	"golang.org/x/sys/unix"
)

type enrollmentTestServices struct {
	fail                    string
	activeCalls, drainCalls int
	hook                    func()
}

func (s *enrollmentTestServices) active(_ context.Context, version string, gid uint32) (*serviceProcess, error) {
	s.activeCalls++
	if s.hook != nil {
		s.hook()
		s.hook = nil
	}
	if s.fail == "service" {
		return nil, ErrTransition
	}
	pid := 200
	if s.fail == "service-replaced" {
		pid += s.activeCalls
	}
	if s.fail == "service-unpinned" {
		return &serviceProcess{pid: pid, version: version, gid: gid}, nil
	}
	fd, err := unix.PidfdOpen(os.Getpid(), 0)
	if err != nil {
		return nil, err
	}
	return &serviceProcess{pidfd: os.NewFile(uintptr(fd), "mocked-service-pidfd"), pid: pid, version: version, gid: gid}, nil
}
func (s *enrollmentTestServices) resourcesDrained(context.Context, uint32, int, int) error {
	s.drainCalls++
	if s.fail == "drain" || s.fail == "late-drain" && s.drainCalls > 1 {
		return ErrBusy
	}
	return nil
}
func (s *enrollmentTestServices) runtimeHealthy(context.Context) error {
	if s.fail == "runtime" {
		return ErrHealth
	}
	return nil
}
func (s *enrollmentTestServices) Close() error { return nil }

func enrollmentFixture(t *testing.T) InstalledRelease {
	t.Helper()
	resetOperationNamespace(t)
	if err := os.Remove(runtimeLockPath); err != nil {
		t.Fatal(err)
	}
	release, files := operationReleaseFixture(t, "v1.0.0")
	stage := writeOperationStage(t, release, files)
	installer, err := releaseinstall.NewSystemInstaller(operationVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := installer.Install(context.Background(), releaseinstall.Request{BundleDirectory: stage.directory, InstallRoot: releaseinstall.SystemInstallRoot}); err != nil {
		t.Fatal(err)
	}
	if err := stage.Close(); err != nil {
		t.Fatal(err)
	}
	if err := storeCheckedProof(context.Background(), release.manifest.SHA256(), []byte("mocked-enrollment-proof")); err != nil {
		t.Fatal(err)
	}
	for index, path := range configPaths() {
		enrollmentWrite(t, path, runtimeConfigFixture(t).Configs[index].Body, 0644)
	}
	return InstalledRelease{record: registration{registrationSchema, "NATIVE", release.manifest.Version(), release.manifest.SHA256(), 1000, 1000}, manifest: release.manifest, proof: []byte("mocked-enrollment-proof")}
}
func enrollmentWrite(t *testing.T, path string, body []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, body, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func TestNativeEnrollmentInDisposableNamespace(t *testing.T) {
	if os.Getenv("HELOX_ENROLLMENT_NAMESPACE") != "1" {
		t.Skip("explicit disposable root enrollment qualification")
	}
	if os.Getenv("HELOX_ENROLLMENT_READ_CHILD") == "1" {
		if os.Geteuid() != 1000 {
			t.Fatal("ordinary UID required")
		}
		body, err := os.ReadFile(registrationPath)
		if err != nil {
			t.Fatal(err)
		}
		record, err := parseRegistration(body)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := readStoredProofs(context.Background(), record.ManifestSHA256); err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{runtimeLockPath, updateLockPath} {
			f, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := syscall.Flock(int(f.Fd()), syscall.LOCK_SH|syscall.LOCK_NB); err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
			if f, err := os.OpenFile(path, os.O_WRONLY, 0); err == nil {
				_ = f.Close()
				t.Fatal("ordinary UID can write lock")
			}
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
	kinds := []string{"new", "repeat", "history-committed", "history-rolled-back", "registered-missing-locks", "replaced-lock", "retry-locks", "unsigned", "wrong-controller", "different-record", "journal", "history", "bad-history", "lock-content", "lock-mode", "lock-owner", "lock-group", "lock-link", "lock-symlink", "busy-update", "busy-runtime", "private-dir", "private-proof", "private-config", "config-uid", "config-client", "config-unit", "current", "tamper", "service", "service-unpinned", "service-replaced", "runtime", "drain", "late-drain", "changed-config"}
	for _, kind := range kinds {
		t.Run(kind, func(t *testing.T) {
			installed := enrollmentFixture(t)
			services := &enrollmentTestServices{fail: kind}
			ctx := context.Background()
			expected, err := json.Marshal(installed.record)
			if err != nil {
				t.Fatal(err)
			}
			var previous []byte
			switch kind {
			case "repeat", "history-committed", "history-rolled-back", "registered-missing-locks":
				if kind != "registered-missing-locks" {
					for _, path := range []string{runtimeLockPath, updateLockPath} {
						if err := prepareEnrollmentLock(path); err != nil {
							t.Fatal(err)
						}
					}
				}
				previous = expected
				enrollmentWrite(t, registrationPath, previous, 0644)
				if kind == "history-committed" || kind == "history-rolled-back" {
					history := runtimeConfigFixture(t)
					history.Old = installed.record
					history.Version = installed.Version()
					history.ManifestSHA256 = installed.record.ManifestSHA256
					history.State = "COMMITTED"
					if kind == "history-rolled-back" {
						history.State = "ROLLED_BACK"
					}
					body, err := json.Marshal(history)
					if err != nil {
						t.Fatal(err)
					}
					enrollmentWrite(t, historyPath, body, 0600)
				}

			case "retry-locks":
				for _, path := range []string{runtimeLockPath, updateLockPath} {
					if err := prepareEnrollmentLock(path); err != nil {
						t.Fatal(err)
					}
				}
			case "different-record":
				record := installed.record
				record.ClientUID++
				previous, err = json.Marshal(record)
				if err != nil {
					t.Fatal(err)
				}
				enrollmentWrite(t, registrationPath, previous, 0644)
			case "journal":
				enrollmentWrite(t, journalPath, []byte("pending"), 0600)
			case "history", "bad-history":
				enrollmentWrite(t, historyPath, []byte("unknown"), 0600)
				if kind == "bad-history" {
					previous = expected
					enrollmentWrite(t, registrationPath, previous, 0644)
				}
			case "lock-content":
				enrollmentWrite(t, runtimeLockPath, []byte("foreign"), 0444)
			case "lock-mode":
				enrollmentWrite(t, runtimeLockPath, nil, 0644)
			case "lock-owner", "lock-group":
				enrollmentWrite(t, runtimeLockPath, nil, 0444)
				uid, gid := 0, 1000
				if kind == "lock-owner" {
					uid = 1000
					gid = 0
				}
				if err := os.Chown(runtimeLockPath, uid, gid); err != nil {
					t.Fatal(err)
				}
			case "lock-link":
				enrollmentWrite(t, updateLockPath, nil, 0444)
				if err := os.Link(updateLockPath, runtimeLockPath); err != nil {
					t.Fatal(err)
				}
			case "lock-symlink":
				if err := os.Symlink("foreign.lock", runtimeLockPath); err != nil {
					t.Fatal(err)
				}
			case "busy-update", "busy-runtime":
				for _, path := range []string{runtimeLockPath, updateLockPath} {
					if err := prepareEnrollmentLock(path); err != nil {
						t.Fatal(err)
					}
				}
				path := runtimeLockPath
				if kind == "busy-update" {
					path = updateLockPath
				}
				held, err := lockPersistentFile(ctx, path, false, false)
				if err != nil {
					t.Fatal(err)
				}
				defer held.Close()
				if kind == "busy-runtime" {
					bounded, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
					defer cancel()
					ctx = bounded
				}
			case "private-dir":
				if err := os.Chmod(filepath.Dir(installed.ClientPath()), 0700); err != nil {
					t.Fatal(err)
				}
			case "private-proof":
				if err := os.Chmod(filepath.Join(proofRoot, installed.manifest.SHA256(), bodySHA256(installed.proof)+".json"), 0400); err != nil {
					t.Fatal(err)
				}
			case "private-config":
				if err := os.Chmod(hostConfigPath, 0600); err != nil {
					t.Fatal(err)
				}
			case "config-uid", "config-client", "config-unit":
				path := policyConfigPath
				body, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				switch kind {
				case "config-uid":
					body = bytes.Replace(body, []byte(`"client_uid":1000`), []byte(`"client_uid":0`), 1)
				case "config-client":
					body = bytes.Replace(body, []byte("helox-linux-amd64"), []byte("foreign"), 1)
				case "config-unit":
					path = policyUnitPath
					body = []byte("foreign unit")
				}
				enrollmentWrite(t, path, body, 0644)
			case "current":
				path := filepath.Join(releaseinstall.SystemInstallRoot, "current")
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("releases/v1.0.0", path); err != nil {
					t.Fatal(err)
				}
			case "tamper":
				enrollmentWrite(t, installed.ClientPath(), []byte("tampered"), 0555)
			case "replaced-lock":
				services.hook = func() {
					if err := os.Remove(updateLockPath); err != nil {
						t.Fatal(err)
					}
					enrollmentWrite(t, updateLockPath, nil, 0444)
				}
			case "changed-config":
				services.hook = func() { enrollmentWrite(t, hostConfigPath, []byte("foreign"), 0644) }
			}
			existing, statErr := os.Lstat(registrationPath)
			if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
				t.Fatal(statErr)
			}
			recorded := func(_ context.Context, record registration) (InstalledRelease, error) {
				if kind == "unsigned" || kind == "tamper" {
					return readRecordedRelease(ctx, record)
				}
				return installed, nil
			}
			controller := func(InstalledRelease) error {
				if kind == "wrong-controller" {
					return ErrCaller
				}
				return nil
			}
			serviceCalls := 0
			run := func() error {
				return registerNative(ctx, recorded, controller, func(context.Context) (enrollmentServices, error) { serviceCalls++; return services, nil })
			}
			result := run()
			positive := kind == "new" || kind == "repeat" || kind == "retry-locks" || kind == "history-committed" || kind == "history-rolled-back"
			if positive {
				if result != nil {
					t.Fatal(result)
				}
				body, err := os.ReadFile(registrationPath)
				if err != nil || !bytes.Equal(body, expected) {
					t.Fatal("registration differs", err)
				}
				before, err := os.Lstat(registrationPath)
				if err != nil {
					t.Fatal(err)
				}
				if err := run(); err != nil {
					t.Fatal("repeat failed", err)
				}
				after, err := os.Lstat(registrationPath)
				if err != nil || !os.SameFile(before, after) {
					t.Fatal("repeat replaced registration", err)
				}
				if kind == "repeat" && !os.SameFile(existing, after) {
					t.Fatal("existing registration replaced")
				}
				// This is mocked root mechanics, never an SDK or quarantine positive.

				if err := verifyInstallerController(installed); err == nil {
					t.Fatal("test executable became installed signed installer")
				}
				if _, err := ReadInstalledRelease(ctx); err == nil {
					t.Fatal("fabricated signed set accepted by production SDK")
				}
				if kind == "new" {
					cmd := exec.Command("/proc/self/exe", "-test.run=^TestNativeEnrollmentInDisposableNamespace$", "-test.v")
					cmd.Env = append(os.Environ(), "HELOX_ENROLLMENT_READ_CHILD=1")
					cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 1000, Gid: 1000}}
					if out, err := cmd.CombinedOutput(); err != nil {
						t.Fatalf("ordinary reader: %v %s", err, out)
					}
				}
			} else {
				if result == nil {
					t.Fatal("unsafe enrollment accepted")
				}
				body, err := os.ReadFile(registrationPath)
				if len(previous) == 0 {
					if !errors.Is(err, os.ErrNotExist) {
						t.Fatal("failed enrollment published state", err)
					}
				} else if err != nil || !bytes.Equal(body, previous) {
					t.Fatal("existing record changed", err)
				}
				if (kind == "unsigned" || kind == "wrong-controller") && serviceCalls != 0 {
					t.Fatal("unverified controller reached runtime tools")
				}
			}
		})
	}
}
