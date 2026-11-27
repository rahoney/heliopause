//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"

	"github.com/rahoney/heliopause/internal/releaseinstall"
)

func bootstrapFixture(t *testing.T) nativeBootstrapSet {
	t.Helper()
	resetOperationNamespace(t)
	if err := os.Remove(runtimeLockPath); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mount("tmpfs", "/usr/libexec", "tmpfs", syscall.MS_NODEV|syscall.MS_NOSUID, "mode=0755,size=16777216"); err != nil {
		t.Fatal(err)
	}
	release, files := operationReleaseFixture(t, "v1.0.0")
	if err := os.MkdirAll(bootstrapDirectory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(bootstrapDirectory, 0700); err != nil {
		t.Fatal(err)
	}
	for _, file := range release.files {
		mode := os.FileMode(0500)
		if file.Role == "release-metadata" {
			mode = 0600
		}
		enrollmentWrite(t, filepath.Join(bootstrapDirectory, file.Name), files[file.Name], mode)
	}
	enrollmentWrite(t, filepath.Join(bootstrapDirectory, "helox-release-manifest.json"), release.body, 0600)
	if err := storeCheckedProof(context.Background(), release.manifest.SHA256(), []byte("mock-initial-proof")); err != nil {
		t.Fatal(err)
	}
	configs := runtimeConfigFixture(t).Configs
	for index, path := range configPaths() {
		enrollmentWrite(t, path, configs[index].Body, os.FileMode(configs[index].Mode))
	}
	record, err := nativeRegistration(release.manifest, configs)
	if err != nil {
		t.Fatal(err)
	}
	return nativeBootstrapSet{record, release, []byte("mock-initial-proof"), configs}
}

func bootstrapChecksFixture(t *testing.T, set nativeBootstrapSet) bootstrapChecks {
	t.Helper()
	return bootstrapChecks{
		read: func(ctx context.Context) (nativeBootstrapSet, error) {
			configs, err := readNativeConfigSnapshots(ctx)
			if err != nil {
				return nativeBootstrapSet{}, err
			}
			record, err := nativeRegistration(set.release.manifest, configs)
			return nativeBootstrapSet{record, set.release, set.proof, configs}, err
		},
		self: func(nativeBootstrapSet) error { return nil },
		installed: func(_ context.Context, record registration) (InstalledRelease, error) {
			return InstalledRelease{record, set.release.manifest, set.proof}, nil
		},
		installer: func(nativeBootstrapSet) (*releaseinstall.Installer, error) {
			return releaseinstall.NewSystemInstaller(operationVerifier{})
		},
		consumers: func(context.Context, int, int) error { return nil },
	}
}

func bootstrapSourceInventory(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	entries, err := os.ReadDir(bootstrapDirectory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		path := filepath.Join(bootstrapDirectory, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		owner, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			t.Fatal("kernel owner missing")
		}
		body := []byte("directory")
		if info.Mode().IsRegular() {
			body, err = os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
		}
		out[entry.Name()] = fmt.Sprintf("%s:%v:%d:%d:%d:%d", bodySHA256(body), info.Mode(), owner.Uid, owner.Gid, owner.Ino, owner.Nlink)
	}
	return out
}

func TestInitialNativeFilesInDisposableNamespace(t *testing.T) {
	if os.Getenv("HELOX_BOOTSTRAP_NAMESPACE") != "1" {
		t.Skip("explicit disposable initial bootstrap qualification")
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
	cases := []string{"fresh", "canceled", "complete-retry", "partial-copy", "unsigned", "wrong-executable", "legacy", "os-package", "foreign-root", "foreign-version", "registration", "journal", "history", "foreign-lock", "changed-intent", "malformed-intent", "intent-mode", "current", "busy", "wrong-verifier", "changed-config", "tampered", "cleanup-full", "cleanup-subset", "cleanup-empty", "cleanup-unknown", "cleanup-owner", "cleanup-mode", "cleanup-link", "cleanup-tamper", "cleanup-cancel"}
	for _, kind := range cases {
		t.Run(kind, func(t *testing.T) {
			set := bootstrapFixture(t)
			checks := bootstrapChecksFixture(t, set)
			ctx := context.Background()
			saveIntent := func(record registration) {
				t.Helper()
				body, err := json.Marshal(record)
				if err != nil {
					t.Fatal(err)
				}
				enrollmentWrite(t, bootstrapIntentPath, body, 0600)
			}
			switch kind {
			case "canceled":
				canceled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = canceled
			case "partial-copy", "foreign-version", "current":
				saveIntent(set.record)
				versions := filepath.Join(releaseinstall.SystemInstallRoot, "versions")
				if err := os.MkdirAll(versions, 0755); err != nil {
					t.Fatal(err)
				}
				for _, path := range []string{releaseinstall.SystemInstallRoot, versions} {
					if err := os.Chmod(path, 0755); err != nil {
						t.Fatal(err)
					}
				}
				if kind == "partial-copy" {
					name, err := releaseinstall.SystemCopyStageName(set.record.ManifestSHA256)
					if err != nil {
						t.Fatal(err)
					}
					path := filepath.Join(versions, name)
					if err := os.Mkdir(path, 0700); err != nil {
						t.Fatal(err)
					}
					enrollmentWrite(t, filepath.Join(path, "helox-release-manifest.json"), set.release.body[:len(set.release.body)/2], 0444)
				}
				if kind == "foreign-version" {
					if err := os.Mkdir(filepath.Join(versions, "v9.9.9"), 0755); err != nil {
						t.Fatal(err)
					}
				}
				if kind == "current" {
					if err := os.Symlink("versions/v9.9.9", filepath.Join(releaseinstall.SystemInstallRoot, "current")); err != nil {
						t.Fatal(err)
					}
				}
			case "unsigned":
				checks.read = readBootstrapSet
			case "wrong-executable":
				checks.self = verifyBootstrapExecutable
			case "legacy":
				if err := os.Mkdir("/usr/libexec/heliopause", 0755); err != nil {
					t.Fatal(err)
				}
			case "os-package":
				if err := os.MkdirAll(filepath.Join(releaseinstall.SystemInstallRoot, "releases"), 0755); err != nil {
					t.Fatal(err)
				}
			case "foreign-root":
				if err := os.MkdirAll(releaseinstall.SystemInstallRoot, 0755); err != nil {
					t.Fatal(err)
				}
				enrollmentWrite(t, filepath.Join(releaseinstall.SystemInstallRoot, "foreign"), []byte("retain"), 0600)
			case "registration":
				body, err := json.Marshal(set.record)
				if err != nil {
					t.Fatal(err)
				}
				enrollmentWrite(t, registrationPath, body, 0644)
			case "journal":
				enrollmentWrite(t, journalPath, []byte("retain"), 0600)
			case "history":
				enrollmentWrite(t, historyPath, []byte("retain"), 0600)
			case "foreign-lock":
				enrollmentWrite(t, runtimeLockPath, nil, 0444)
			case "changed-intent":
				record := set.record
				record.ClientUID++
				saveIntent(record)
			case "malformed-intent":
				enrollmentWrite(t, bootstrapIntentPath, []byte("unknown"), 0600)
			case "intent-mode":
				saveIntent(set.record)
				if err := os.Chmod(bootstrapIntentPath, 0644); err != nil {
					t.Fatal(err)
				}
			case "busy":
				checks.consumers = func(context.Context, int, int) error { return ErrBusy }
			case "wrong-verifier":
				checks.installer = func(nativeBootstrapSet) (*releaseinstall.Installer, error) {
					return releaseinstall.NewSystemInstaller(deniedBootstrapVerifier{})
				}
			case "changed-config":
				calls := 0
				base := checks.self
				checks.self = func(set nativeBootstrapSet) error {
					calls++
					if calls == 1 {
						body, err := os.ReadFile(policyConfigPath)
						if err != nil {
							t.Fatal(err)
						}
						body = bytes.Replace(body, []byte(`"client_uid":1000`), []byte(`"client_uid":1001`), 1)
						enrollmentWrite(t, policyConfigPath, body, 0644)
					}
					return base(set)
				}
			case "tampered":
				enrollmentWrite(t, filepath.Join(bootstrapDirectory, "helox-linux-amd64"), []byte("wrong bytes"), 0500)
			}
			result := prepareBootstrapFiles(ctx, checks)
			positive := kind == "fresh" || kind == "complete-retry" || kind == "partial-copy" || len(kind) > 8 && kind[:8] == "cleanup-"
			if !positive {
				if result == nil {
					t.Fatal("unsafe initial preparation accepted")
				}
				if kind == "canceled" && !errors.Is(result, context.Canceled) {
					t.Fatal("cancellation lost", result)
				}
				if kind == "unsigned" || kind == "wrong-executable" {
					if _, err := os.Lstat(bootstrapIntentPath); !errors.Is(err, os.ErrNotExist) {
						t.Fatal("unverified helper wrote initial intent", err)
					}
				}
				return
			}
			if result != nil {
				t.Fatal(result)
			}
			if _, err := os.Lstat(registrationPath); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("file preparation published healthy registration", err)
			}
			if _, err := AcquireRuntimeLease(ctx); !errors.Is(err, ErrTransition) {
				t.Fatal("unregistered files enabled ordinary runtime", err)
			}
			if err := prepareBootstrapFiles(ctx, checks); err != nil {
				t.Fatal("exact complete retry failed", err)
			}
			if _, err := readBootstrapSet(ctx); err == nil {
				t.Fatal("synthetic proof passed initial production SDK")
			}
			if len(kind) <= 8 || kind[:8] != "cleanup-" {
				return
			}
			installed := InstalledRelease{set.record, set.release.manifest, set.proof}
			if err := cleanupBootstrapSource(ctx, installed); err == nil {
				t.Fatal("fabricated installed set passed production cleanup SDK")
			}
			member := filepath.Join(bootstrapDirectory, "helox-linux-amd64")
			switch kind {
			case "cleanup-subset":
				if err := os.Remove(filepath.Join(bootstrapDirectory, "helox-release-manifest.json")); err != nil {
					t.Fatal(err)
				}
			case "cleanup-empty":
				entries, err := os.ReadDir(bootstrapDirectory)
				if err != nil {
					t.Fatal(err)
				}
				for _, entry := range entries {
					if err := os.Remove(filepath.Join(bootstrapDirectory, entry.Name())); err != nil {
						t.Fatal(err)
					}
				}
			case "cleanup-unknown":
				enrollmentWrite(t, filepath.Join(bootstrapDirectory, "foreign"), []byte("retain"), 0600)
			case "cleanup-owner":
				if err := os.Chown(member, 1000, 1000); err != nil {
					t.Fatal(err)
				}
			case "cleanup-mode":
				if err := os.Chmod(member, 0600); err != nil {
					t.Fatal(err)
				}
			case "cleanup-link":
				if err := os.Link(member, filepath.Join(stateRoot, "foreign-link")); err != nil {
					t.Fatal(err)
				}
			case "cleanup-tamper":
				enrollmentWrite(t, member, []byte("unknown"), 0500)
			case "cleanup-cancel":
				canceled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = canceled
			}
			before := bootstrapSourceInventory(t)
			err := discardBootstrapSource(ctx, installed)
			cleanupPositive := kind == "cleanup-full" || kind == "cleanup-subset" || kind == "cleanup-empty"
			if cleanupPositive {
				if err != nil {
					t.Fatal(err)
				}
				if _, err := os.Lstat(bootstrapDirectory); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("completed bootstrap source retained", err)
				}
				if err := cleanupBootstrapSource(ctx, installed); err != nil {
					t.Fatal("empty cleanup retry failed", err)
				}
			} else {
				if err == nil {
					t.Fatal("unknown initial source removed")
				}
				if after := bootstrapSourceInventory(t); !reflect.DeepEqual(before, after) {
					t.Fatal("failed cleanup changed bootstrap source")
				}
			}
		})
	}
}

type deniedBootstrapVerifier struct{}

func (deniedBootstrapVerifier) Verify(context.Context, string) error { return ErrUntrusted }
