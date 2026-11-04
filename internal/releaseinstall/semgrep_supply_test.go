package releaseinstall

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func semgrepManifestFixture(t *testing.T) manifest {
	t.Helper()
	value := pipxDataManifestFixture(t, true)
	value.Assets = append(value.Assets, asset{runtimeidentity.SandboxResponderAssetName, "linux/amd64", "sandbox-responder", strings.Repeat("a", 64), 123})
	pin := runtimeidentity.SelectedSemgrepSupply()
	for _, item := range runtimeidentity.SemgrepReleaseAssets(pin) {
		role := "semgrep-data"
		if item.Path == pin.Tool.Path {
			role = "semgrep"
		}
		value.Assets = append(value.Assets, asset{item.Path, "linux/amd64", role, item.SHA256, item.Size})
	}
	return value
}

func TestSemgrepCompleteSetSelectionAndPassiveModes(t *testing.T) {
	value := semgrepManifestFixture(t)
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	m, err := ParseReleaseManifest(body)
	if err != nil || !m.HasSemgrep() || !m.HasPipxRuntime() {
		t.Fatal("complete set rejected", err)
	}
	native, err := m.NativeUpdateFiles()
	if err != nil || len(native) != 8 {
		t.Fatal("native role set", len(native), err)
	}
	payloads, err := m.NativeUpdatePayloads()
	if err != nil || len(payloads) != 17 {
		t.Fatal("payload set", len(payloads), err)
	}
	selected, err := selectFiles(value, "linux", "amd64")
	if err != nil || len(selected) != 15 {
		t.Fatal("installer selection", len(selected), err)
	}
	data := 0
	for _, file := range selected {
		if runtimeidentity.SemgrepAssetMaximum(file.Name) == 0 {
			continue
		}
		passive := file.Name != runtimeidentity.SelectedSemgrepSupply().Tool.Path
		if ReleaseFile(file).IsRuntimeData() != passive {
			t.Fatal("execution role confused", file.Name)
		}
		if passive {
			data++
		}
		if mode := assetCopyMode(file, 0555, 0444); passive && mode != 0444 || !passive && mode != 0555 {
			t.Fatalf("wrong mode %s %o", file.Name, mode)
		}
	}
	if data != 3 {
		t.Fatal("passive set incomplete")
	}
}

func TestSemgrepReleaseRejectsIncompleteMixedAndExcessiveAssets(t *testing.T) {
	for _, mode := range []string{"missing", "duplicate", "role", "platform", "source-limit", "missing-responder", "missing-pipx"} {
		t.Run(mode, func(t *testing.T) {
			value := semgrepManifestFixture(t)
			last := len(value.Assets) - 1
			switch mode {
			case "missing":
				value.Assets = value.Assets[:last]
			case "duplicate":
				value.Assets = append(value.Assets, value.Assets[last])
			case "role":
				value.Assets[last].Role = "semgrep"
			case "platform":
				value.Assets[last].Platform = "linux/arm64"
			case "source-limit":
				value.Assets[last-2].Size = 512<<20 + 1
			case "missing-responder":
				value.Assets = append(value.Assets[:15], value.Assets[16:]...)
			case "missing-pipx":
				value.Assets = append(value.Assets[:13], value.Assets[15:]...)
			}
			body, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ParseReleaseManifest(body); err == nil {
				t.Fatal("invalid supply accepted")
			}
		})
	}
}

func TestSemgrepRuntimeMetadataBindsTupleAndReleaseAssets(t *testing.T) {
	legacy, _, metadata, _ := pipxRuntimeMetadataFixture(t)
	metadata["schema"] = "helox.runtime-image-manifest/v4"
	metadata["semgrep"] = runtimeidentity.SelectedSemgrepSupply()
	body, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	value := semgrepManifestFixture(t)
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	current, err := ParseReleaseManifest(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateRuntimeImagesForRelease(body, current); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRuntimeImagesForRelease(body, legacy); err == nil {
		t.Fatal("legacy gained Semgrep membership")
	}
	for index := len(value.Assets) - 4; index < len(value.Assets); index++ {
		original := value.Assets[index]
		value.Assets[index].SHA256 = strings.Repeat("f", 64)
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		changed, err := ParseReleaseManifest(encoded)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateRuntimeImagesForRelease(body, changed); err == nil {
			t.Fatal("foreign asset accepted", original.Name)
		}
		value.Assets[index] = original
	}
	pin := runtimeidentity.SelectedSemgrepSupply()
	pin.SourceCommit = strings.Repeat("f", 40)
	metadata["semgrep"] = pin
	body, err = json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateRuntimeImagesForRelease(body, current); err == nil {
		t.Fatal("foreign source tuple accepted")
	}
}

// Copy mechanics use synthetic payloads and an injected verifier; no system
// installation, native execution or provenance qualification is claimed.
func TestSemgrepInstallUpdateDoctorAndRollbackRetainAllAssets(t *testing.T) {
	root := t.TempDir()
	installer, err := newInstaller(allowVerifier{}, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	installRoot := filepath.Join(root, "installed")
	for _, version := range []string{"v1.2.3", "v1.2.4"} {
		bundle := releaseBundle(t, root, version, "semgrep-copy-"+version)
		body, err := os.ReadFile(filepath.Join(bundle, manifestName))
		if err != nil {
			t.Fatal(err)
		}
		var value manifest
		if err := json.Unmarshal(body, &value); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"haa-release-installer-linux-amd64", "pnpm-linux-amd64", "pnpm-NOTICE.txt", "bun-linux-amd64", "bun-NOTICE.txt", runtimeidentity.PipxRuntimeAssetName, "pipx-NOTICE.txt", runtimeidentity.SandboxResponderAssetName, "semgrep-core-linux-amd64", "semgrep-SOURCES.tar", "semgrep-NOTICES.tar", runtimeidentity.SemgrepRulesAssetName} {
			role := "release-metadata"
			switch name {
			case "haa-release-installer-linux-amd64":
				role = "release-installer"
			case "pnpm-linux-amd64":
				role = "pnpm"
			case "bun-linux-amd64":
				role = "bun"
			case runtimeidentity.PipxRuntimeAssetName:
				role = "pipx-runtime"
			case runtimeidentity.SandboxResponderAssetName:
				role = "sandbox-responder"
			case "semgrep-core-linux-amd64":
				role = "semgrep"
			case "semgrep-SOURCES.tar", "semgrep-NOTICES.tar", runtimeidentity.SemgrepRulesAssetName:
				role = "semgrep-data"
			}
			payload := []byte(version + name)
			if err := os.WriteFile(filepath.Join(bundle, name), payload, 0600); err != nil {
				t.Fatal(err)
			}
			value.Assets = append(value.Assets, asset{name, "linux/amd64", role, digest(payload), int64(len(payload))})
		}
		for _, item := range []struct{ name, platform, role string }{{"helox-linux-arm64", "linux/arm64", "helox"}, {"helox-darwin-amd64", "darwin/amd64", "helox"}, {"helox-darwin-arm64", "darwin/arm64", "helox"}, {"helox-linux-amd64.deb", "linux/amd64", "os-package-deb"}, {"helox-linux-amd64.rpm", "linux/amd64", "os-package-rpm"}} {
			payload := []byte(item.name)
			value.Assets = append(value.Assets, asset{item.name, item.platform, item.role, digest(payload), int64(len(payload))})
		}
		body, err = json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(bundle, manifestName), body, 0600); err != nil {
			t.Fatal(err)
		}
		result, err := installer.Install(context.Background(), Request{bundle, installRoot})
		if err != nil {
			t.Fatal(err)
		}
		if !checkInstallation(installRoot, "linux", "amd64").Healthy {
			t.Fatal("doctor rejected complete install")
		}
		for _, asset := range runtimeidentity.SemgrepReleaseAssets(runtimeidentity.SelectedSemgrepSupply()) {
			path := filepath.Join(result.InstallRoot, "versions", version, asset.Path)
			info, err := os.Stat(path)
			mode := os.FileMode(0600)
			if asset.Path == runtimeidentity.SelectedSemgrepSupply().Tool.Path {
				mode = 0700
			}
			if err != nil || info.Mode().Perm() != mode {
				t.Fatal("copy role lost", asset.Path, err)
			}
		}
	}
	if _, err := installer.ActivateExisting(context.Background(), Request{filepath.Join(installRoot, "versions", "v1.2.3"), installRoot}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(installRoot, "versions", "v1.2.3", runtimeidentity.SemgrepRulesAssetName)
	if err := os.WriteFile(path, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if checkInstallation(installRoot, "linux", "amd64").Healthy {
		t.Fatal("doctor accepted changed rules")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if checkInstallation(installRoot, "linux", "amd64").Healthy {
		t.Fatal("doctor accepted absent rules")
	}
}
