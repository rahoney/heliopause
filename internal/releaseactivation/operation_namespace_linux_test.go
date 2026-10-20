//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/rahoney/heliopause/internal/releaseinstall"
	"golang.org/x/sys/unix"
)

type operationVerifier struct{}

func (operationVerifier) Verify(context.Context, string) error { return nil }

type operationServices struct {
	events        []string
	version, fail string
	drains        int
}

func (s *operationServices) active(_ context.Context, version string, _ uint32) (*serviceProcess, error) {
	s.events = append(s.events, "active:"+version)
	if s.version != version {
		return nil, ErrTransition
	}
	return &serviceProcess{pid: 200, version: version}, nil
}
func (s *operationServices) drained(context.Context, *controllerPeer, *serviceProcess) error {
	s.drains++
	s.events = append(s.events, "drained")
	if s.fail == "self-report" && s.drains > 1 {
		return ErrBusy
	}
	return nil
}
func (s *operationServices) resourcesDrained(context.Context, uint32, int, int) error {
	s.events = append(s.events, "resources-drained")
	if s.fail == "self-report" && s.drains > 1 {
		return ErrBusy
	}
	return nil
}
func (s *operationServices) quiesce(context.Context, updateJournal, InstalledRelease, *InstalledRelease) error {
	s.events = append(s.events, "quiesce")
	if s.fail == "stop-unknown" {
		return ErrTransition
	}
	s.version = ""
	return nil
}
func (s *operationServices) start(_ context.Context, version string, _ uint32) (*serviceProcess, error) {
	s.events = append(s.events, "start:"+version)
	if s.fail == "start" && version == "v1.0.1" {
		return nil, ErrHealth
	}
	s.version = version
	return &serviceProcess{pid: 200, version: version}, nil
}
func (s *operationServices) runtimeHealthy(context.Context) error {
	s.events = append(s.events, "runtime-healthy")
	return nil
}
func (s *operationServices) Close() error { return nil }

func operationReleaseFixture(t *testing.T, version string) (wireRelease, map[string][]byte) {
	t.Helper()
	release, files := wireFixture(t)
	var value map[string]any
	if err := json.Unmarshal(release.body, &value); err != nil {
		t.Fatal(err)
	}
	value["version"] = version
	var sbom map[string]any
	if err := json.Unmarshal(files["helox-release-sbom.cdx.json"], &sbom); err != nil {
		t.Fatal(err)
	}
	sbom["metadata"].(map[string]any)["component"].(map[string]any)["version"] = version
	body, err := json.Marshal(sbom)
	if err != nil {
		t.Fatal(err)
	}
	files["helox-release-sbom.cdx.json"] = body
	metadata := value["sbom"].(map[string]any)
	metadata["sha256"], metadata["size"] = bodySHA256(body), len(body)
	for _, raw := range value["assets"].([]any) {
		asset := raw.(map[string]any)
		name := asset["name"].(string)
		if original, found := files[name]; found && asset["platform"] == "linux/amd64" {
			payload := append(append([]byte(nil), original...), []byte("-"+version)...)
			files[name] = payload
			asset["sha256"], asset["size"] = bodySHA256(payload), len(payload)
		}
	}
	body, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	release, err = newWireRelease(body, release.proofs)
	if err != nil {
		t.Fatal(err)
	}
	return release, files
}

func writeOperationStage(t *testing.T, release wireRelease, files map[string][]byte) *preparedRelease {
	t.Helper()
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
	if err := os.WriteFile(filepath.Join(directory, "helox-release-manifest.json"), release.body, 0600); err != nil {
		t.Fatal(err)
	}
	for _, file := range release.files {
		if err := os.WriteFile(filepath.Join(directory, file.Name), files[file.Name], 0600); err != nil {
			t.Fatal(err)
		}
	}
	identity, err := os.Lstat(directory)
	if err != nil {
		t.Fatal(err)
	}
	return &preparedRelease{directory: directory, identity: identity, release: release, proof: []byte("mocked-proof-only")}
}

func resetOperationNamespace(t *testing.T) {
	t.Helper()
	for _, path := range []string{"/usr/lib", "/var/lib", "/etc"} {
		if err := syscall.Mount("tmpfs", path, "tmpfs", syscall.MS_NODEV|syscall.MS_NOSUID, "mode=0755,size=16777216"); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{stateRoot, "/etc/heliopause", "/etc/systemd/system"} {
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
		for current := path; current != filepath.Dir(current); current = filepath.Dir(current) {
			if err := os.Chmod(current, 0755); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := os.WriteFile(runtimeLockPath, []byte("persistent"), 0444); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(runtimeLockPath, 0444); err != nil {
		t.Fatal(err)
	}
}

func operationFixture(t *testing.T, kind string) (*rootActivation, *operationServices) {
	t.Helper()
	ctx := context.Background()
	old, oldFiles := operationReleaseFixture(t, "v1.0.0")
	version := "v1.0.1"
	if kind == "same-version" {
		version = "v1.0.0"
	}
	next, nextFiles := operationReleaseFixture(t, version)
	oldStage := writeOperationStage(t, old, oldFiles)
	installer, err := releaseinstall.NewSystemInstaller(operationVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := installer.Install(ctx, releaseinstall.Request{BundleDirectory: oldStage.directory, InstallRoot: releaseinstall.SystemInstallRoot}); err != nil {
		t.Fatal(err)
	}
	if err := oldStage.Close(); err != nil {
		t.Fatal(err)
	}
	oldView := InstalledRelease{registration{registrationSchema, "NATIVE", old.manifest.Version(), old.manifest.SHA256(), 1000, 1000}, old.manifest, []byte("mocked-old-proof")}
	nextView := InstalledRelease{registration{registrationSchema, "NATIVE", next.manifest.Version(), next.manifest.SHA256(), 1000, 1000}, next.manifest, []byte("mocked-new-proof")}
	record, err := json.Marshal(oldView.record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(registrationPath, record, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(registrationPath, 0644); err != nil {
		t.Fatal(err)
	}
	if err := storeCheckedProof(ctx, old.manifest.SHA256(), old.proofs[0]); err != nil {
		t.Fatal(err)
	}
	// Real production SDK must still reject these fabricated native payloads.
	if _, err := ReadInstalledRelease(ctx); err == nil {
		t.Fatal("synthetic namespace view passed production SDK")
	}
	stage := writeOperationStage(t, next, nextFiles)
	if kind == "existing-candidate" {
		if _, err := installer.Install(ctx, releaseinstall.Request{BundleDirectory: stage.directory, InstallRoot: releaseinstall.SystemInstallRoot}); err != nil {
			t.Fatal(err)
		}
		if _, err := installer.ActivateExisting(ctx, releaseinstall.Request{BundleDirectory: filepath.Dir(oldView.ClientPath()), InstallRoot: releaseinstall.SystemInstallRoot}); err != nil {
			t.Fatal(err)
		}
	}
	value := runtimeConfigFixture(t)
	value.Old = oldView.record
	value.Version = nextView.Version()
	value.ManifestSHA256 = nextView.manifest.SHA256()
	value.Stage = filepath.Base(stage.directory)
	for index, path := range configPaths() {
		if err := os.WriteFile(path, value.Configs[index].Body, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0644); err != nil {
			t.Fatal(err)
		}
	}
	journal, err := persistPreparedJournal(ctx, value)
	if err != nil {
		t.Fatal(err)
	}
	fd, err := unix.PidfdOpen(os.Getpid(), 0)
	if err != nil {
		t.Fatal(err)
	}
	peer := &controllerPeer{pidfd: os.NewFile(uintptr(fd), "mock-old-peer"), pid: int32(os.Getpid()), uid: 1000, gid: 1000}
	t.Cleanup(func() { _ = peer.Close() })
	services := &operationServices{version: oldView.Version(), fail: kind}
	views := map[string]InstalledRelease{oldView.Version(): oldView, nextView.Version(): nextView}
	checks := activationChecks{
		installed: func(ctx context.Context) (InstalledRelease, error) {
			body, err := readProtected(ctx, registrationPath, 16<<10)
			if err != nil {
				return InstalledRelease{}, err
			}
			record, err := parseRegistration(body)
			if err != nil {
				return InstalledRelease{}, err
			}
			view, found := views[record.Version]
			if !found || view.record != record {
				return InstalledRelease{}, ErrUntrusted
			}
			link, err := os.Readlink(filepath.Join(releaseinstall.SystemInstallRoot, "current"))
			if err != nil || link != filepath.Join("versions", view.Version()) {
				return InstalledRelease{}, ErrUntrusted
			}
			return view, nil
		},
		recorded: func(_ context.Context, record registration) (InstalledRelease, error) {
			view, found := views[record.Version]
			if !found || view.record != record {
				return InstalledRelease{}, ErrUntrusted
			}
			return view, nil
		},
		health: func(_ context.Context, _, current InstalledRelease, _ *controllerPeer, stage string, full bool) (*healthWorkspace, error) {
			services.events = append(services.events, "health")
			workspace, err := createHealthWorkspace(stage, current)
			if err != nil {
				return workspace, err
			}
			if err := os.WriteFile(filepath.Join(workspace.path, "payload"), []byte("owned health fixture"), 0600); err != nil {
				return workspace, err
			}
			if kind == "health" && full {
				return workspace, ErrHealth
			}
			if kind == "unknown-config" {
				if err := os.WriteFile(hostConfigPath, []byte("unknown root change"), 0644); err != nil {
					return workspace, err
				}
				return workspace, ErrHealth
			}
			return workspace, nil
		},
	}
	return &rootActivation{old: oldView, stage: stage, peer: peer, journal: journal, services: services, checks: checks, install: installer, restore: installer}, services
}

func TestRootActivationTransactionInDisposableNamespace(t *testing.T) {
	if os.Getenv("HELOX_OPERATION_NAMESPACE") != "1" {
		t.Skip("explicit disposable root activation transaction qualification")
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
	for _, kind := range []string{"success", "same-version", "existing-candidate", "health", "start", "stop-unknown", "unknown-config", "self-report"} {
		t.Run(kind, func(t *testing.T) {
			resetOperationNamespace(t)
			activation, services := operationFixture(t, kind)
			defer activation.Close()
			err := activation.activate(context.Background())
			success := kind == "success" || kind == "same-version" || kind == "existing-candidate"
			if (err == nil) != success {
				t.Fatal("transaction result differs", kind, err, services.events)
			}
			link, queryErr := os.Readlink(filepath.Join(releaseinstall.SystemInstallRoot, "current"))
			if queryErr != nil {
				t.Fatal(queryErr)
			}
			wantVersion := "v1.0.0"
			if success && kind != "same-version" || kind == "unknown-config" || kind == "self-report" {
				wantVersion = "v1.0.1"
			}
			if link != filepath.Join("versions", wantVersion) {
				t.Fatal("unexpected active pointer", kind, link, err, services.events)
			}
			uncertain := kind == "stop-unknown" || kind == "unknown-config" || kind == "self-report"
			if uncertain {
				journal, queryErr := readDurableJournal(context.Background())
				if queryErr != nil || journal.value.State != "RECOVERY_REQUIRED" {
					t.Fatal("uncertain transition lost marker", queryErr, err)
				}
				if activation.retired {
					t.Fatal("unknown outcome retired admission gate")
				}
				if kind == "unknown-config" {
					body, err := os.ReadFile(hostConfigPath)
					if err != nil || !bytes.Equal(body, []byte("unknown root change")) {
						t.Fatal("unknown config overwritten")
					}
				}
			} else {
				if !journalConfirmedAbsent() || !activation.retired {
					t.Fatal("verified terminal transaction left marker", err)
				}
				history, queryErr := readProtected(context.Background(), historyPath, 128<<10)
				if queryErr != nil {
					t.Fatal(queryErr)
				}
				journal, queryErr := parseJournal(history)
				if queryErr != nil {
					t.Fatal(queryErr)
				}
				state := "ROLLED_BACK"
				if success {
					state = "COMMITTED"
				}
				if journal.State != state {
					t.Fatal("wrong terminal history", journal.State)
				}
				if _, err := os.Lstat(activation.stage.directory); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("completed stage not removed")
				}
			}
			if strings.Contains(strings.Join(services.events, ","), "health") && kind == "self-report" && !errors.Is(err, ErrBusy) {
				t.Fatal("child self-report bypassed independent drain")
			}
		})
	}
	t.Log("PASS scope: real root copy/pointer/config/registration/journal/locks and cleanup; crypto, old peer, systemd and compatibility health mocked; no signed native4 activation")
}
