//go:build linux

package releaseinstall

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/testutil/releasefixture"
)

// The operator supplies already materialized bytes. This tests protected copy,
// update and rollback mechanics with an injected verifier, never authenticity.
func TestSemgrepSystemSupplyInDisposableNamespace(t *testing.T) {
	input := os.Getenv("HELOX_SEMGREP_RELEASE_SUPPLY")
	if input == "" {
		t.Skip("explicit selected supply and disposable namespace required")
	}
	if os.Geteuid() != 0 {
		t.Fatal("root required inside private namespace")
	}
	for _, kind := range []string{"mnt", "net"} {
		self, err := os.Readlink("/proc/self/ns/" + kind)
		if err != nil {
			t.Fatal(err)
		}
		initial, err := os.Readlink("/proc/1/ns/" + kind)
		if err != nil || self == initial {
			t.Fatal("refusing Host namespace")
		}
	}
	for _, path := range []string{"/usr/lib", "/var/lib"} {
		if err := syscall.Mount("tmpfs", path, "tmpfs", syscall.MS_NODEV|syscall.MS_NOSUID, "mode=0755,size=3221225472"); err != nil {
			t.Fatal(err)
		}
	}
	installer, err := NewSystemInstaller(allowVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	pin := runtimeidentity.SelectedSemgrepSupply()
	for _, version := range []string{"v0.1.0", "v0.1.1"} {
		bundle := systemBundleFixture(t, version, "semgrep-"+version)
		body, err := os.ReadFile(filepath.Join(bundle, manifestName))
		if err != nil {
			t.Fatal(err)
		}
		var value manifest
		if err := json.Unmarshal(body, &value); err != nil {
			t.Fatal(err)
		}
		for _, selected := range runtimeidentity.SemgrepReleaseAssets(pin) {
			in, err := os.Open(filepath.Join(input, selected.Path))
			if err != nil {
				t.Fatal(err)
			}
			out, err := os.OpenFile(filepath.Join(bundle, selected.Path), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				_ = in.Close()
				t.Fatal(err)
			}
			count, copyErr := io.CopyN(out, in, selected.Size)
			closeErr := out.Close()
			inputErr := in.Close()
			if copyErr != nil || closeErr != nil || inputErr != nil || count != selected.Size {
				t.Fatal("selected supply copy failed", selected.Path, copyErr, closeErr, inputErr)
			}
			role := "semgrep-data"
			if selected.Path == pin.Tool.Path {
				role = "semgrep"
			}
			value.Assets = append(value.Assets, asset{selected.Path, "linux/amd64", role, selected.SHA256, selected.Size})
		}
		for _, extra := range []struct{ name, role, path string }{{runtimeidentity.PipxRuntimeAssetName, "pipx-runtime", os.Getenv("HELOX_PIPX_RUNTIME_ZIP")}, {"pipx-NOTICE.txt", "release-metadata", filepath.Join("..", "..", "tools", "pipx", "NOTICE")}} {
			data, err := os.ReadFile(extra.path)
			if err != nil {
				t.Fatal(err)
			}
			if err := writeNew(filepath.Join(bundle, extra.name), data, 0600); err != nil {
				t.Fatal(err)
			}
			value.Assets = append(value.Assets, asset{extra.name, "linux/amd64", extra.role, digest(data), int64(len(data))})
		}
		responder := []byte("unsigned copy-only responder")
		if err := writeNew(filepath.Join(bundle, runtimeidentity.SandboxResponderAssetName), responder, 0600); err != nil {
			t.Fatal(err)
		}
		value.Assets = append(value.Assets, asset{runtimeidentity.SandboxResponderAssetName, "linux/amd64", "sandbox-responder", digest(responder), int64(len(responder))})
		body, err = json.Marshal(value)
		if err != nil {
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
		materials, err := releasefixture.BindCurrentRuntime(body, sbom, images)
		if err != nil {
			t.Fatal(err)
		}
		for name, data := range map[string][]byte{manifestName: materials.Manifest, sbomName: materials.SBOM, runtimeImagesName: materials.Images} {
			if err := os.WriteFile(filepath.Join(bundle, name), data, 0600); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := installer.Install(context.Background(), Request{bundle, SystemInstallRoot}); err != nil {
			t.Fatal(err)
		}
		for _, selected := range runtimeidentity.SemgrepReleaseAssets(pin) {
			mode := os.FileMode(0444)
			if selected.Path == pin.Tool.Path {
				mode = 0555
			}
			path := filepath.Join(SystemInstallRoot, "versions", version, selected.Path)
			if err := verifyRecord(path, record{selected.Path, selected.SHA256, selected.Size}); err != nil {
				t.Fatal(err)
			}
			info, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
			owner, ok := info.Sys().(*syscall.Stat_t)
			if !ok || owner.Uid != 0 || owner.Nlink != 1 || info.Mode().Perm() != mode {
				t.Fatal("protected role lost", selected.Path)
			}
		}
		if !CheckInstallation(SystemInstallRoot).Healthy {
			t.Fatal("doctor rejected selected supply")
		}
		if err := os.RemoveAll(bundle); err != nil {
			t.Fatal(err)
		}
	}
	old := filepath.Join(SystemInstallRoot, "versions", "v0.1.0")
	if _, err := installer.ActivateExisting(context.Background(), Request{old, SystemInstallRoot}); err != nil {
		t.Fatal(err)
	}
	if target, err := os.Readlink(filepath.Join(SystemInstallRoot, "current")); err != nil || target != "versions/v0.1.0" {
		t.Fatal("rollback pointer", target, err)
	}
	if err := os.Chmod(filepath.Join(old, runtimeidentity.SemgrepRulesAssetName), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, runtimeidentity.SemgrepRulesAssetName), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if CheckInstallation(SystemInstallRoot).Healthy {
		t.Fatal("doctor accepted changed selected supply")
	}
	if _, err := installer.ActivateExisting(context.Background(), Request{old, SystemInstallRoot}); err == nil {
		t.Fatal("rollback accepted changed supply")
	}
	t.Log("selected core/source/notices/rules: two versions copied/hash checked; root-owned single-link0555/0444, doctor, rollback and mutation refusal; unsigned copy-only, no artifact execution")
}
