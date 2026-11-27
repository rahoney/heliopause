//go:build linux

package releaseactivation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/rahoney/heliopause/internal/releaseinstall"
	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
)

func TestProtectedReaderInDisposableNamespace(t *testing.T) {
	if os.Getenv("HELOX_ACTIVATION_READ_NAMESPACE") != "1" {
		t.Skip("explicit disposable namespace qualification")
	}
	if os.Geteuid() != 0 {
		t.Fatal("root namespace is required")
	}
	for _, kind := range []string{"mnt", "net"} {
		self, err := os.Readlink("/proc/self/ns/" + kind)
		if err != nil {
			t.Fatal(err)
		}
		initial, err := os.Readlink("/proc/1/ns/" + kind)
		if err != nil || self == initial {
			t.Fatal("separate mount/network namespaces required")
		}
	}
	if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/usr/lib", "/var/lib"} {
		if err := syscall.Mount("tmpfs", path, "tmpfs", syscall.MS_NODEV|syscall.MS_NOSUID, "mode=0755,size=67108864"); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	if lease, err := AcquireRuntimeLease(ctx); err != nil || lease != nil {
		t.Fatal("confirmed absent state was not legacy", err)
	}
	if err := os.Mkdir(stateRoot, 0755); err != nil {
		t.Fatal(err)
	}
	write := func(path string, body []byte) {
		t.Helper()
		if err := os.WriteFile(path, body, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(runtimeLockPath, nil)
	if _, err := AcquireRuntimeLease(ctx); !errors.Is(err, ErrTransition) {
		t.Fatal("partial registration became legacy", err)
	}
	record := registration{registrationSchema, "NATIVE", "v0.1.0", "", 1000, 1000}
	base := filepath.Join("..", "..", "docs", "planning", "evidence", "m13-003-release-build", "v0.1.0")
	manifestBody, err := os.ReadFile(filepath.Join(base, "helox-release-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := releaseinstall.ParseReleaseManifest(manifestBody)
	if err != nil {
		t.Fatal(err)
	}
	record.ManifestSHA256 = manifest.SHA256()
	body, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	write(registrationPath, body)
	write(journalPath, []byte(`{"phase":"SWITCHING"}`))
	if _, err := AcquireRuntimeLease(ctx); !errors.Is(err, ErrTransition) {
		t.Fatal("journal state became normal runtime", err)
	}
	if err := os.Remove(journalPath); err != nil {
		t.Fatal(err)
	}
	// A genuine old signed release lacks the new installer role. It must not
	// become an eligible managed release merely through a root-owned record.
	directory := filepath.Join(releaseinstall.SystemInstallRoot, "versions", record.Version)
	if err := os.MkdirAll(directory, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("versions", record.Version), filepath.Join(releaseinstall.SystemInstallRoot, "current")); err != nil {
		t.Fatal(err)
	}
	write(filepath.Join(directory, "helox-release-manifest.json"), manifestBody)
	if _, err := AcquireRuntimeLease(ctx); !errors.Is(err, ErrUntrusted) {
		t.Fatal("old incomplete native release became eligible", err)
	}
	file, _, err := openProtectedFile(runtimeLockPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal("failed verification leaked the shared lock", err)
	}
	flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, file.Fd(), syscall.F_GETFD, 0)
	if errno != 0 || flags&syscall.FD_CLOEXEC == 0 {
		t.Fatal("protected descriptor can leak across exec")
	}
	if err := (&RuntimeLease{file: file}).Close(); err != nil {
		t.Fatal(err)
	}
	proof, err := os.ReadFile(filepath.Join(base, "actual-signed-bundle.json"))
	if err != nil {
		t.Fatal(err)
	}
	proofDir := filepath.Join(proofRoot, manifest.SHA256())
	if err := os.MkdirAll(proofDir, 0755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(proof)
	proofPath := filepath.Join(proofDir, hex.EncodeToString(sum[:])+".json")
	write(proofPath, proof)
	stored, err := readStoredProofs(ctx, manifest.SHA256())
	if err != nil || len(stored) != 1 {
		t.Fatal("protected proof storage rejected actual bundle", err)
	}
	if _, err := releaseprovenance.Verify(ctx, stored[0], releaseprovenance.Identity{Version: manifest.Version(), SourceCommit: manifest.SourceCommit(), WorkflowRun: manifest.WorkflowRun()}, []releaseprovenance.Subject{{Name: "helox-release-manifest.json", SHA256: manifest.SHA256()}}); err != nil {
		t.Fatal("actual offline stored provenance failed", err)
	}
	write(proofPath, []byte(`{"changed":true}`))
	if _, err := readStoredProofs(ctx, manifest.SHA256()); !errors.Is(err, ErrUntrusted) {
		t.Fatal("proof filename/digest substitution accepted", err)
	}
	write(proofPath, proof)
	if err := os.Chown(proofPath, 1000, 1000); err != nil {
		t.Fatal(err)
	}
	if _, err := readStoredProofs(ctx, manifest.SHA256()); !errors.Is(err, ErrUntrusted) {
		t.Fatal("foreign proof owner accepted", err)
	}
	if err := os.Chown(proofPath, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(proofPath, proofPath+".link"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := openProtectedFile(proofPath); !errors.Is(err, ErrUntrusted) {
		t.Fatal("linked protected proof accepted", err)
	}
	t.Log("PASS: absent/partial/journal gates, failed-verification lock release, CLOEXEC, actual offline stored signature, modified/foreign/linked proof rejection; new native signed activation is not qualified")
}
