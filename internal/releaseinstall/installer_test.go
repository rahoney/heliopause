package releaseinstall

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallVerifiesBundleAndAtomicallyActivates(t *testing.T) {
	root := t.TempDir()
	bundle := releaseBundle(t, root, "v1.2.3", "first")
	installer, err := newInstaller(allowVerifier{}, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	result, err := installer.Install(context.Background(), Request{BundleDirectory: bundle, InstallRoot: filepath.Join(root, "installed")})
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != "v1.2.3" || result.ManifestSHA256 == "" {
		t.Fatalf("result=%#v", result)
	}
	current, err := os.Readlink(filepath.Join(result.InstallRoot, "current"))
	if err != nil || current != "versions/v1.2.3" {
		t.Fatalf("current=%q err=%v", current, err)
	}
	for _, name := range []string{"helox-linux-amd64", "haa_gvisor_observer-linux-amd64", "haa-network-policy-helper-linux-amd64", manifestName, sbomName, runtimeImagesName, installRecordName} {
		if _, err := os.Lstat(filepath.Join(result.InstallRoot, "versions", "v1.2.3", name)); err != nil {
			t.Fatalf("installed %s: %v", name, err)
		}
	}
	if check := checkInstallation(result.InstallRoot, "linux", "amd64"); !check.Healthy || check.Detail != "OK" {
		t.Fatalf("installation doctor check=%#v", check)
	}
}

func TestCheckInstallationFailsClosedForUnavailableOrTamperedState(t *testing.T) {
	if check := CheckInstallation(filepath.Join(t.TempDir(), "missing")); check.Healthy || check.Detail == "" {
		t.Fatalf("missing installation check=%#v", check)
	}
	root := t.TempDir()
	installer, _ := newInstaller(allowVerifier{}, "linux", "amd64")
	bundle := releaseBundle(t, root, "v1.2.3", "first")
	installRoot := filepath.Join(root, "installed")
	if _, err := installer.Install(context.Background(), Request{BundleDirectory: bundle, InstallRoot: installRoot}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installRoot, "versions", "v1.2.3", "helox-linux-amd64"), []byte("tampered"), 0o700); err != nil {
		t.Fatal(err)
	}
	if check := checkInstallation(installRoot, "linux", "amd64"); check.Healthy || check.Detail != "RELEASE_ASSET_MISMATCH" {
		t.Fatalf("tampered installation check=%#v", check)
	}
}

func TestInstallFailurePreservesExistingActiveVersion(t *testing.T) {
	root := t.TempDir()
	installRoot := filepath.Join(root, "installed")
	installer, _ := newInstaller(allowVerifier{}, "linux", "amd64")
	first := releaseBundle(t, root, "v1.2.3", "first")
	if _, err := installer.Install(context.Background(), Request{BundleDirectory: first, InstallRoot: installRoot}); err != nil {
		t.Fatal(err)
	}
	second := releaseBundle(t, root, "v1.2.4", "second")
	if err := os.Remove(filepath.Join(second, "helox-linux-amd64")); err != nil {
		t.Fatal(err)
	}
	if _, err := installer.Install(context.Background(), Request{BundleDirectory: second, InstallRoot: installRoot}); err == nil {
		t.Fatal("incomplete bundle was activated")
	}
	current, err := os.Readlink(filepath.Join(installRoot, "current"))
	if err != nil || current != "versions/v1.2.3" {
		t.Fatalf("current=%q err=%v", current, err)
	}
	entries, err := os.ReadDir(filepath.Join(installRoot, "versions"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".pending-") {
			t.Fatalf("failed install left pending stage %q", entry.Name())
		}
	}
}

func TestInstallRefusesUnverifiedBundleOrPointerReplacement(t *testing.T) {
	root := t.TempDir()
	bundle := releaseBundle(t, root, "v1.2.3", "first")
	denied, _ := newInstaller(denyVerifier{}, "linux", "amd64")
	if _, err := denied.Install(context.Background(), Request{BundleDirectory: bundle, InstallRoot: filepath.Join(root, "installed")}); err == nil {
		t.Fatal("unverified bundle was accepted")
	}
	installer, _ := newInstaller(allowVerifier{}, "linux", "amd64")
	installRoot := filepath.Join(root, "installed")
	if err := os.MkdirAll(installRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installRoot, "current"), []byte("not a symlink"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := installer.Install(context.Background(), Request{BundleDirectory: bundle, InstallRoot: installRoot}); err == nil {
		t.Fatal("non-symlink active pointer was replaced")
	}
}

type allowVerifier struct{}

func (allowVerifier) Verify(context.Context, string) error { return nil }

type denyVerifier struct{}

func (denyVerifier) Verify(context.Context, string) error { return errors.New("denied") }

func releaseBundle(t *testing.T, root, version, suffix string) string {
	t.Helper()
	bundle := filepath.Join(root, "bundle-"+suffix)
	if err := os.Mkdir(bundle, 0o700); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		sbomName:                                []byte("sbom"),
		runtimeImagesName:                       []byte(`{"schema":"helox.runtime-image-manifest/v1"}`),
		"helox-linux-amd64":                     []byte("helox-" + suffix),
		"haa_gvisor_observer-linux-amd64":       []byte("observer-" + suffix),
		"haa-network-policy-helper-linux-amd64": []byte("policy-" + suffix),
	}
	assets := make([]asset, 0, 3)
	for name, role := range map[string]string{
		"helox-linux-amd64": "helox", "haa_gvisor_observer-linux-amd64": "gvisor-observer", "haa-network-policy-helper-linux-amd64": "network-policy-helper",
	} {
		body := files[name]
		assets = append(assets, asset{Name: name, Platform: "linux/amd64", Role: role, SHA256: digest(body), Size: int64(len(body))})
	}
	manifestBody, err := json.Marshal(manifest{Schema: manifestSchema, Version: version, SourceCommit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", WorkflowRun: "https://github.com/rahoney/heliopause/actions/runs/1", RuntimeLock: runtimeLock{SHA256: digest([]byte("runtime-lock")), GVisorRelease: "release-20260810.0", GVisorCommit: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}, SBOM: fileRecord(sbomName, files[sbomName]), RuntimeImages: fileRecord(runtimeImagesName, files[runtimeImagesName]), Assets: assets})
	if err != nil {
		t.Fatal(err)
	}
	files[manifestName] = append(manifestBody, '\n')
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(bundle, name), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return bundle
}

func fileRecord(name string, body []byte) record {
	return record{Name: name, SHA256: digest(body), Size: int64(len(body))}
}
func digest(body []byte) string { sum := sha256.Sum256(body); return hex.EncodeToString(sum[:]) }
