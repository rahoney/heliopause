package releaseinstall

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type installVerifierFunc func(context.Context, string) error

func (f installVerifierFunc) Verify(ctx context.Context, path string) error { return f(ctx, path) }

func TestExistingVersionRollbackReverifiesAndPreservesCurrentOnFailure(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "installed")
	installer, err := newInstaller(allowVerifier{}, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	first, err := installer.Install(context.Background(), Request{releaseBundle(t, base, "v1.2.3", "old"), root})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := installer.Install(context.Background(), Request{releaseBundle(t, base, "v1.2.4", "new"), root}); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(root, "versions", "v1.2.3")
	calls := 0
	denied := errors.New("unapproved previous version")
	installer.verifier = installVerifierFunc(func(context.Context, string) error { calls++; return denied })
	if _, err := installer.ActivateExisting(context.Background(), Request{old, root}); !errors.Is(err, denied) || calls != 1 {
		t.Fatal("rollback lost mandatory verifier or its cause")
	}
	current := func() string {
		target, err := os.Readlink(filepath.Join(root, "current"))
		if err != nil {
			t.Fatal(err)
		}
		return target
	}
	if current() != "versions/v1.2.4" {
		t.Fatal("unverified rollback changed current")
	}
	installer.verifier = allowVerifier{}
	result, err := installer.ActivateExisting(context.Background(), Request{old, root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != first.Version || result.ManifestSHA256 != first.ManifestSHA256 || current() != "versions/v1.2.3" {
		t.Fatal("rollback did not select exact verified installed bytes")
	}
	if _, err := installer.ActivateExisting(context.Background(), Request{filepath.Join(root, "versions", "v1.2.4"), root}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "helox-linux-amd64"), []byte("changed"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := installer.ActivateExisting(context.Background(), Request{old, root}); err == nil || current() != "versions/v1.2.4" {
		t.Fatal("tampered prior version changed current")
	}
	outside := releaseBundle(t, base, "v1.2.2", "outside")
	if _, err := installer.ActivateExisting(context.Background(), Request{outside, root}); err == nil || current() != "versions/v1.2.4" {
		t.Fatal("external bundle was used as an existing version")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := installer.ActivateExisting(ctx, Request{old, root}); !errors.Is(err, context.Canceled) || current() != "versions/v1.2.4" {
		t.Fatal("cancelled rollback lost cause or mutated current")
	}
}

func TestInstallerSelectsOptionalFourthRoleWithoutChangingUserPermissions(t *testing.T) {
	base := t.TempDir()
	bundle := releaseBundle(t, base, "v1.2.3", "four-roles")
	body, err := os.ReadFile(filepath.Join(bundle, manifestName))
	if err != nil {
		t.Fatal(err)
	}
	var value manifest
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatal(err)
	}
	payload := []byte("separate installer")
	name := "haa-release-installer-linux-amd64"
	if err := os.WriteFile(filepath.Join(bundle, name), payload, 0600); err != nil {
		t.Fatal(err)
	}
	value.Assets = append(value.Assets, asset{Name: name, Platform: "linux/amd64", Role: "release-installer", SHA256: digest(payload), Size: int64(len(payload))})
	body, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, manifestName), body, 0600); err != nil {
		t.Fatal(err)
	}
	installer, err := newInstaller(allowVerifier{}, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "installed")
	result, err := installer.Install(context.Background(), Request{bundle, root})
	if err != nil {
		t.Fatal(err)
	}
	for path, mode := range map[string]os.FileMode{root: 0700, filepath.Join(root, "versions"): 0700, filepath.Join(root, "versions", result.Version, name): 0700, filepath.Join(root, "versions", result.Version, manifestName): 0600} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != mode {
			t.Fatalf("user mode %s=%v error=%v", path, info, err)
		}
	}
	if check := checkInstallation(root, "linux", "amd64"); !check.Healthy {
		t.Fatalf("four-role doctor=%v", check)
	}
	value.Assets[len(value.Assets)-1].Role = "helox"
	body, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, manifestName), body, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := installer.Install(context.Background(), Request{bundle, filepath.Join(base, "other")}); err == nil {
		t.Fatal("fourth role identity was ignored")
	}
}

func TestSystemInstallerRejectsArbitraryTargetBeforeVerifierOrMutation(t *testing.T) {
	base := t.TempDir()
	bundle := releaseBundle(t, base, "v1.2.3", "invalid-system-target")
	calls := 0
	installer := &Installer{verifier: installVerifierFunc(func(context.Context, string) error { calls++; return nil }), goos: "linux", goarch: "amd64", system: true}
	target := filepath.Join(base, "foreign")
	if _, err := installer.Install(context.Background(), Request{bundle, target}); err == nil || calls != 0 {
		t.Fatal("protected primitive accepted arbitrary caller target")
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("rejected request created target")
	}
}

func TestVerifierCancellationPreventsAnyInstallMutation(t *testing.T) {
	base := t.TempDir()
	bundle := releaseBundle(t, base, "v1.2.3", "cancelled-verifier")
	target := filepath.Join(base, "not-created")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	installer, err := newInstaller(installVerifierFunc(func(context.Context, string) error { cancel(); return nil }), "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := installer.Install(ctx, Request{bundle, target}); !errors.Is(err, context.Canceled) {
		t.Fatal("verifier-time cancellation lost its cause")
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("cancelled verification created install state")
	}
}
