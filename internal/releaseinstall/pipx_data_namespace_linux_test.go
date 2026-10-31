//go:build linux

package releaseinstall

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/testutil/releasefixture"
)

func TestPipxSystemDataInDisposableNamespace(t *testing.T) {
	if os.Getenv("HELOX_PIPX_DATA_COPY_NAMESPACE") != "1" {
		t.Skip("explicit disposable DATA copy namespace")
	}
	if os.Geteuid() != 0 {
		t.Fatal("copy fixture requires root only inside private namespace")
	}
	for _, kind := range []string{"mnt", "net"} {
		self, err := os.Readlink("/proc/self/ns/" + kind)
		if err != nil {
			t.Fatal(err)
		}
		initial, err := os.Readlink("/proc/1/ns/" + kind)
		if err != nil || self == initial {
			t.Fatal("refusing host namespace copy")
		}
	}
	body, err := os.ReadFile(os.Getenv("HELOX_PIPX_RUNTIME_ZIP"))
	if err != nil || int64(len(body)) != runtimeidentity.PipxToolSize || digest(body) != runtimeidentity.PipxToolSHA256 {
		t.Fatal("selected immutable ZIP missing", err)
	}
	if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/usr/lib", "/var/lib"} {
		if err := syscall.Mount("tmpfs", path, "tmpfs", syscall.MS_NODEV|syscall.MS_NOSUID, "mode=0755,size=67108864"); err != nil {
			t.Fatal(err)
		}
	}
	bundle := systemBundleFixture(t, "v0.1.0", "pipx-data")
	manifestPath := filepath.Join(bundle, manifestName)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var value manifest
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	notices, err := os.ReadFile(filepath.Join("..", "..", "tools", "pipx", "NOTICE"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []struct {
		name, role string
		body       []byte
	}{{runtimeidentity.PipxRuntimeAssetName, "pipx-runtime", body}, {"pipx-NOTICE.txt", "release-metadata", notices}} {
		if err := writeNew(filepath.Join(bundle, file.name), file.body, 0600); err != nil {
			t.Fatal(err)
		}
		value.Assets = append(value.Assets, asset{file.name, "linux/amd64", file.role, digest(file.body), int64(len(file.body))})
	}
	raw, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	sbom, err := os.ReadFile(filepath.Join(bundle, sbomName))
	if err != nil {
		t.Fatal(err)
	}
	images, err := os.ReadFile(filepath.Join(bundle, runtimeImagesName))
	if err != nil {
		t.Fatal(err)
	}
	materials, err := releasefixture.BindCurrentRuntime(raw, sbom, images)
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string][]byte{manifestName: materials.Manifest, sbomName: materials.SBOM, runtimeImagesName: materials.Images} {
		if err := os.WriteFile(filepath.Join(bundle, name), body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	installer, err := NewSystemInstaller(allowVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := installer.Install(context.Background(), Request{bundle, SystemInstallRoot}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{runtimeidentity.PipxRuntimeAssetName, "pipx-NOTICE.txt"} {
		path := filepath.Join(SystemInstallRoot, "versions", "v0.1.0", name)
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		owner, ok := info.Sys().(*syscall.Stat_t)
		if !ok || owner.Uid != 0 || owner.Nlink != 1 || info.Mode().Perm() != 0444 {
			t.Fatal("DATA is executable or unprotected", name)
		}
	}
	copied, err := os.ReadFile(filepath.Join(SystemInstallRoot, "versions", "v0.1.0", runtimeidentity.PipxRuntimeAssetName))
	if err != nil || digest(copied) != runtimeidentity.PipxToolSHA256 {
		t.Fatal("selected DATA bytes changed", err)
	}
	t.Log("unsigned copy-only fixture: native6 retained; selected ZIP/NOTICE root-owned single-link0444; no artifact execution or authenticity qualification")
}
