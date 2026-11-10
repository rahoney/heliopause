//go:build linux

package releaseinstall

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/testutil/releasefixture"
)

// The operator must enter new mount/network namespaces before starting this
// statically linked test binary. No host service or runtime config is changed.
func TestProtectedCopyInDisposableNamespace(t *testing.T) {
	if os.Getenv("HELOX_PROTECTED_COPY_NAMESPACE") != "1" {
		t.Skip("explicit disposable namespace qualification")
	}
	if os.Getenv("HELOX_PROTECTED_COPY_READER") == "1" {
		checkOrdinarySystemReader(t)
		return
	}
	if os.Geteuid() != 0 {
		t.Fatal("namespace qualification requires root")
	}
	for _, kind := range []string{"mnt", "net"} {
		self, err := os.Readlink("/proc/self/ns/" + kind)
		if err != nil {
			t.Fatal(err)
		}
		initial, err := os.Readlink("/proc/1/ns/" + kind)
		if err != nil || self == initial {
			t.Fatal("refusing qualification without separate mount/network namespaces")
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
	previous := syscall.Umask(0077)
	defer syscall.Umask(previous)
	installer, err := NewSystemInstaller(allowVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	first := systemBundleFixture(t, "v0.1.0", "first")
	if _, err := installer.Install(context.Background(), Request{first, SystemInstallRoot}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{SystemInstallRoot, filepath.Join(SystemInstallRoot, "versions"), filepath.Join(SystemInstallRoot, "versions", "v0.1.0")} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0755 {
			t.Fatalf("system directory mode=%v error=%v", info, err)
		}
	}
	for _, name := range []string{"helox-linux-amd64", "haa_gvisor_observer-linux-amd64", "haa-network-policy-helper-linux-amd64", "haa-release-installer-linux-amd64", "pnpm-linux-amd64", "pnpm-NOTICE.txt", "bun-linux-amd64", "bun-NOTICE.txt", manifestName, sbomName, runtimeImagesName, installRecordName} {
		path := filepath.Join(SystemInstallRoot, "versions", "v0.1.0", name)
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		owner, ok := info.Sys().(*syscall.Stat_t)
		if !ok || owner.Uid != 0 || owner.Nlink != 1 {
			t.Fatal("installed file is not exclusively root-owned")
		}
		want := os.FileMode(0555)
		if name == manifestName || name == sbomName || name == runtimeImagesName || name == installRecordName || (name == "pnpm-NOTICE.txt" || name == "bun-NOTICE.txt") {
			want = 0444
		}
		if info.Mode().Perm() != want || info.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 {
			t.Fatal("protected payload has wrong permissions")
		}
	}
	self, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	driver := filepath.Join(SystemInstallRoot, "qualification-driver")
	if err := writeNew(driver, self, 0555); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	reader := exec.CommandContext(ctx, driver, "-test.run=^TestProtectedCopyInDisposableNamespace$", "-test.count=1")
	reader.Dir = "/"
	reader.Env = []string{"PATH=/usr/bin:/bin", "HELOX_PROTECTED_COPY_NAMESPACE=1", "HELOX_PROTECTED_COPY_READER=1"}
	reader.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 1000, Gid: 1000, Groups: []uint32{1000}}}
	if output, err := reader.CombinedOutput(); err != nil {
		t.Fatalf("ordinary reader=%v\n%s", err, output)
	}
	second := systemBundleFixture(t, "v0.1.1", "second")
	if _, err := installer.Install(context.Background(), Request{second, SystemInstallRoot}); err != nil {
		t.Fatal(err)
	}
	current := func() string {
		value, err := os.Readlink(filepath.Join(SystemInstallRoot, "current"))
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	old := filepath.Join(SystemInstallRoot, "versions", "v0.1.0")
	if _, err := installer.ActivateExisting(context.Background(), Request{old, SystemInstallRoot}); err != nil || current() != "versions/v0.1.0" {
		t.Fatal("verified root rollback failed", err)
	}
	if _, err := installer.ActivateExisting(context.Background(), Request{filepath.Join(SystemInstallRoot, "versions", "v0.1.1"), SystemInstallRoot}); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"foreign-owner", "hardlink", "symlink", "writable-root"} {
		t.Run(kind, func(t *testing.T) {
			bundle := systemBundleFixture(t, "v0.1.2", kind)
			path := filepath.Join(bundle, "helox-linux-amd64")
			switch kind {
			case "foreign-owner":
				if err := os.Chown(path, 1000, 1000); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(path, path+"-alias"); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Rename(path, path+"-original"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path+"-original", path); err != nil {
					t.Fatal(err)
				}
			case "writable-root":
				if err := os.Chmod(SystemInstallRoot, 0777); err != nil {
					t.Fatal(err)
				}
				defer func() {
					if err := os.Chmod(SystemInstallRoot, 0755); err != nil {
						t.Error(err)
					}
				}()
			}
			if _, err := installer.Install(context.Background(), Request{bundle, SystemInstallRoot}); err == nil || current() != "versions/v0.1.1" {
				t.Fatal("untrusted system input changed active version")
			}
		})
	}
	t.Log("PASS: actual root ownership/modes, ordinary read and write denial, existing-version rollback, four rejected ownership/link/path states; provenance is mocked, no host activation")
}

func checkOrdinarySystemReader(t *testing.T) {
	t.Helper()
	if os.Geteuid() != 1000 {
		t.Fatal("reader was not dropped to an ordinary UID")
	}
	for _, name := range []string{"helox-linux-amd64", "haa_gvisor_observer-linux-amd64", "haa-network-policy-helper-linux-amd64", "haa-release-installer-linux-amd64", "pnpm-linux-amd64", "pnpm-NOTICE.txt", "bun-linux-amd64", "bun-NOTICE.txt", manifestName} {
		path := filepath.Join(SystemInstallRoot, "current", name)
		if body, err := os.ReadFile(path); err != nil || len(body) == 0 {
			t.Fatal("ordinary user cannot read protected payload", err)
		}
		if file, err := os.OpenFile(path, os.O_WRONLY, 0); err == nil {
			_ = file.Close()
			t.Fatal("ordinary user can write protected payload")
		}
	}
	if _, err := NewSystemInstaller(allowVerifier{}); err == nil {
		t.Fatal("ordinary user obtained a protected installer")
	}
	calls := 0
	forged := &Installer{verifier: installVerifierFunc(func(context.Context, string) error { calls++; return nil }), goos: "linux", goarch: "amd64", system: true}
	if _, err := forged.ActivateExisting(context.Background(), Request{filepath.Join(SystemInstallRoot, "versions", "v0.1.0"), SystemInstallRoot}); err == nil || calls != 0 {
		t.Fatal("ordinary caller entered protected activation")
	}
}

func systemBundleFixture(t *testing.T, version, suffix string) string {
	t.Helper()
	root := filepath.Join(SystemStagingRoot, "transaction-"+suffix)
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	var value manifest
	if err := json.Unmarshal(publishedManifest(t), &value); err != nil {
		t.Fatal(err)
	}
	value.Version = version
	sbom := publishedMetadata(t, sbomName)
	var sbomValue map[string]any
	if err := json.Unmarshal(sbom, &sbomValue); err != nil {
		t.Fatal(err)
	}
	sbomValue["metadata"].(map[string]any)["component"].(map[string]any)["version"] = version
	var err error
	sbom, err = json.Marshal(sbomValue)
	if err != nil {
		t.Fatal(err)
	}
	images := publishedMetadata(t, runtimeImagesName)
	files := map[string][]byte{sbomName: sbom, runtimeImagesName: images}
	value.SBOM = fileRecord(sbomName, sbom)
	value.RuntimeImages = fileRecord(runtimeImagesName, images)
	roles := map[string]string{"helox-linux-amd64": "helox", "haa_gvisor_observer-linux-amd64": "gvisor-observer", "haa-network-policy-helper-linux-amd64": "network-policy-helper", "haa-release-installer-linux-amd64": "release-installer", "pnpm-linux-amd64": "pnpm", "pnpm-NOTICE.txt": "release-metadata", "bun-linux-amd64": "bun", "bun-NOTICE.txt": "release-metadata"}
	for index, item := range value.Assets {
		if _, found := roles[item.Name]; found {
			body := []byte("copy-only-" + suffix + "-" + item.Name)
			files[item.Name] = body
			item.SHA256 = digest(body)
			item.Size = int64(len(body))
			value.Assets[index] = item
			delete(roles, item.Name)
		}
	}
	for name, role := range roles {
		body := []byte("copy-only-" + suffix + "-" + name)
		if name == "pnpm-NOTICE.txt" || name == "bun-NOTICE.txt" {
			var err error
			body, err = os.ReadFile(filepath.Join("..", "..", "tools", strings.TrimSuffix(name, "-NOTICE.txt"), "NOTICE"))
			if err != nil {
				t.Fatal(err)
			}
		}
		files[name] = body
		value.Assets = append(value.Assets, asset{Name: name, Platform: "linux/amd64", Role: role, SHA256: digest(body), Size: int64(len(body))})
	}
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	materials, err := releasefixture.BindCurrentRuntime(body, sbom, images)
	if err != nil {
		t.Fatal(err)
	}
	body = materials.Manifest
	files[sbomName], files[runtimeImagesName] = materials.SBOM, materials.Images
	files[manifestName] = body
	for name, body := range files {
		if err := writeNew(filepath.Join(root, name), body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ParseReleaseManifest(body); err != nil {
		t.Fatal(err)
	}
	return root
}
