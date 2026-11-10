//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/rahoney/heliopause/internal/releaseinstall"
	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
)

func TestProtectedProofAndRegistrationInDisposableNamespace(t *testing.T) {
	if os.Getenv("HELOX_PROOF_NAMESPACE") != "1" {
		t.Skip("explicit disposable root proof qualification")
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
	if err := syscall.Mount("tmpfs", "/var/lib", "tmpfs", syscall.MS_NODEV|syscall.MS_NOSUID, "mode=0755,size=16777216"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(stateRoot, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(stateRoot, 0755); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	synthetic, _ := wireFixture(t)
	current := InstalledRelease{record: registration{registrationSchema, "NATIVE", synthetic.manifest.Version(), synthetic.manifest.SHA256(), 1000, 1000}, manifest: synthetic.manifest, proof: []byte("mock-current")}
	if err := storeReleaseProof(ctx, synthetic, current); !errors.Is(err, ErrUntrusted) {
		t.Fatal("production store approved unsigned native4", err)
	}
	if _, err := os.Lstat(proofRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("invalid proof wrote root state")
	}
	base := filepath.Join("..", "..", "docs", "planning", "evidence", "m13-003-release-build", "v0.1.0")
	body, err := os.ReadFile(filepath.Join(base, "helox-release-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := releaseinstall.ParseReleaseManifest(body)
	if err != nil {
		t.Fatal(err)
	}
	proof, err := os.ReadFile(filepath.Join(base, "actual-signed-bundle.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := storeCheckedProof(ctx, manifest.SHA256(), proof); err != nil {
		t.Fatal(err)
	}
	if err := storeCheckedProof(ctx, manifest.SHA256(), proof); err != nil {
		t.Fatal("proof storage not idempotent", err)
	}
	stored, err := readStoredProofs(ctx, manifest.SHA256())
	if err != nil || len(stored) != 1 || !bytes.Equal(stored[0], proof) {
		t.Fatal("proof bytes changed", err)
	}
	expected := releaseprovenance.Identity{Version: manifest.Version(), SourceCommit: manifest.SourceCommit(), WorkflowRun: manifest.WorkflowRun()}
	if _, err := releaseprovenance.Verify(ctx, stored[0], expected, []releaseprovenance.Subject{{Name: "helox-release-manifest.json", SHA256: manifest.SHA256()}}); err != nil {
		t.Fatal("stored actual offline proof no longer verifies", err)
	}
	path := filepath.Join(proofRoot, manifest.SHA256(), bodySHA256(proof)+".json")
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != 0 || owner.Nlink != 1 || info.Mode().Perm() != 0444 {
		t.Fatal("public proof identity differs")
	}
	// This is storage mechanics only; arbitrary bytes never pass the SDK gate.
	max := bytes.Repeat([]byte{'x'}, 1<<20)
	digest := bodySHA256([]byte("bounded storage fixture"))
	if err := storeCheckedProof(ctx, digest, max); err != nil {
		t.Fatal("existing one-MiB proof budget was narrowed", err)
	}
	if _, err := readStoredProofs(ctx, digest); err != nil {
		t.Fatal(err)
	}
	if err := storeCheckedProof(ctx, digest, append(max, 'x')); err == nil {
		t.Fatal("oversized proof stored")
	}
	foreign := bodySHA256([]byte("foreign root directory fixture"))
	directory := filepath.Join(proofRoot, foreign)
	if err := os.Mkdir(directory, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chown(directory, 1000, 1000); err != nil {
		t.Fatal(err)
	}
	if err := storeCheckedProof(ctx, foreign, proof); err == nil {
		t.Fatal("foreign directory was adopted")
	}
	journal := journalFixture()
	old, err := json.Marshal(journal.Old)
	if err != nil {
		t.Fatal(err)
	}
	if err := replaceProtected(ctx, registrationPath, nil, old, 0644, 0); err != nil {
		t.Fatal(err)
	}
	if err := switchRegistration(ctx, journal, false); err != nil {
		t.Fatal(err)
	}
	actual, err := readProtected(ctx, registrationPath, 16<<10)
	if err != nil {
		t.Fatal(err)
	}
	record, err := parseRegistration(actual)
	if err != nil || record != candidateRegistration(journal) {
		t.Fatal("registration switch differs", err)
	}
	if err := switchRegistration(ctx, journal, true); err != nil {
		t.Fatal(err)
	}
	actual, err = readProtected(ctx, registrationPath, 16<<10)
	if err != nil || !bytes.Equal(actual, old) {
		t.Fatal("registration restore differs", err)
	}
	unknown := []byte("unknown root record")
	if err := os.WriteFile(registrationPath, unknown, 0644); err != nil {
		t.Fatal(err)
	}
	if err := switchRegistration(ctx, journal, true); err == nil {
		t.Fatal("unknown registration restored by guessing")
	}
	actual, err = os.ReadFile(registrationPath)
	if err != nil || !bytes.Equal(actual, unknown) {
		t.Fatal("unknown registration overwritten")
	}
	t.Log("PASS: actual stored old release offline proof, one-MiB storage/identity bounds, exact registration switch/restore and unsigned native4 denial; no native4 activation")
}
