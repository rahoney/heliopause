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

func pipxDataManifestFixture(t *testing.T, withPipx bool) manifest {
	t.Helper()
	var value manifest
	if err := json.Unmarshal(publishedManifest(t), &value); err != nil {
		t.Fatal(err)
	}
	value.Assets = append(value.Assets, asset{"haa-release-installer-linux-amd64", "linux/amd64", "release-installer", strings.Repeat("a", 64), 123}, asset{"pnpm-linux-amd64", "linux/amd64", "pnpm", strings.Repeat("b", 64), 768}, asset{"pnpm-NOTICE.txt", "linux/amd64", "release-metadata", strings.Repeat("c", 64), 1024}, asset{"bun-linux-amd64", "linux/amd64", "bun", strings.Repeat("d", 64), 768}, asset{"bun-NOTICE.txt", "linux/amd64", "release-metadata", strings.Repeat("e", 64), 1024})
	if withPipx {
		value.Assets = append(value.Assets, asset{runtimeidentity.PipxRuntimeAssetName, "linux/amd64", "pipx-runtime", runtimeidentity.PipxToolSHA256, runtimeidentity.PipxToolSize}, asset{"pipx-NOTICE.txt", "linux/amd64", "release-metadata", runtimeidentity.PipxNoticesSHA256, runtimeidentity.PipxNoticesSize})
	}
	return value
}

func TestPipxDataPairPreservesNativeRolesAndExactCompletePayloadSet(t *testing.T) {
	for _, withPipx := range []bool{false, true} {
		value := pipxDataManifestFixture(t, withPipx)
		body, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := ParseReleaseManifest(body)
		if err != nil {
			t.Fatal(err)
		}
		native, err := parsed.NativeUpdateFiles()
		if err != nil || len(native) != 6 {
			t.Fatal("native6 changed", err)
		}
		for _, file := range native {
			if file.IsRuntimeData() || file.Name == runtimeidentity.PipxRuntimeAssetName {
				t.Fatal("ZIP became native executable")
			}
		}
		payloads, err := parsed.NativeUpdatePayloads()
		want := 10
		if withPipx {
			want = 12
		}
		if err != nil || len(payloads) != want || parsed.HasPipxRuntime() != withPipx {
			t.Fatal("incomplete payload set", err)
		}
		seen := map[string]bool{}
		data := 0
		for _, file := range payloads {
			if seen[file.Name] {
				t.Fatal("duplicate payload")
			}
			seen[file.Name] = true
			if file.IsRuntimeData() {
				data++
				if file.Name != runtimeidentity.PipxRuntimeAssetName || file.SHA256 != runtimeidentity.PipxToolSHA256 {
					t.Fatal("DATA identity lost")
				}
			}
		}
		if withPipx && (data != 1 || !seen["pipx-NOTICE.txt"]) {
			t.Fatal("DATA pair omitted")
		}
		selected, err := selectFiles(value, "linux", "amd64")
		if err != nil || len(selected) != want-2 {
			t.Fatal("installer and payload selection differ", len(selected), err)
		}
	}
}

func TestPipxDataRejectsMissingSubstitutedAndExcessiveRoles(t *testing.T) {
	for _, mode := range []string{"missing-zip", "missing-notice", "duplicate", "zip-role", "notice-role", "platform", "foreign-name", "zip-limit", "notice-limit", "requires-bun"} {
		t.Run(mode, func(t *testing.T) {
			value := pipxDataManifestFixture(t, true)
			zip := len(value.Assets) - 2
			notice := zip + 1
			switch mode {
			case "missing-zip":
				value.Assets = append(value.Assets[:zip], value.Assets[notice])
			case "missing-notice":
				value.Assets = value.Assets[:notice]
			case "duplicate":
				value.Assets = append(value.Assets, value.Assets[zip])
			case "zip-role":
				value.Assets[zip].Role = "bun"
			case "notice-role":
				value.Assets[notice].Role = "pipx-runtime"
			case "platform":
				value.Assets[zip].Platform = "linux/arm64"
			case "foreign-name":
				value.Assets[zip].Name = "foreign.zip"
			case "zip-limit":
				value.Assets[zip].Size = 8<<20 + 1
			case "notice-limit":
				value.Assets[notice].Size = 64<<10 + 1
			case "requires-bun":
				value.Assets = append(value.Assets[:zip-2], value.Assets[zip:]...)
			}
			body, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ParseReleaseManifest(body); err == nil {
				t.Fatal("invalid DATA declaration accepted")
			}
		})
	}
	for _, file := range []ReleaseFile{{Name: runtimeidentity.PipxRuntimeAssetName, Role: "helox", Platform: "linux/amd64"}, {Name: "foreign.zip", Role: "pipx-runtime", Platform: "linux/amd64"}, {Name: runtimeidentity.PipxRuntimeAssetName, Role: "pipx-runtime", Platform: "linux/arm64"}} {
		if file.IsRuntimeData() {
			t.Fatal("foreign declaration acquired DATA role")
		}
	}
}

func TestPipxUserInstallRetainsNonexecutableDataAndPriorMetadataModes(t *testing.T) {
	root := t.TempDir()
	bundle := releaseBundle(t, root, "v1.2.3", "pipx-data")
	path := filepath.Join(bundle, manifestName)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value manifest
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"haa-release-installer-linux-amd64": "release-installer", "pnpm-linux-amd64": "pnpm", "pnpm-NOTICE.txt": "release-metadata", "bun-linux-amd64": "bun", "bun-NOTICE.txt": "release-metadata"} {
		body := []byte("copy-only-" + name)
		if err := os.WriteFile(filepath.Join(bundle, name), body, 0600); err != nil {
			t.Fatal(err)
		}
		value.Assets = append(value.Assets, asset{name, "linux/amd64", role, digest(body), int64(len(body))})
	}
	for _, file := range []struct {
		name, role string
		body       []byte
	}{{runtimeidentity.PipxRuntimeAssetName, "pipx-runtime", []byte("DATA-only copy fixture; never imported")}, {"pipx-NOTICE.txt", "release-metadata", []byte("copy-only notices")}} {
		if err := os.WriteFile(filepath.Join(bundle, file.name), file.body, 0600); err != nil {
			t.Fatal(err)
		}
		value.Assets = append(value.Assets, asset{file.name, "linux/amd64", file.role, digest(file.body), int64(len(file.body))})
	}
	body, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}
	installer, err := newInstaller(allowVerifier{}, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	result, err := installer.Install(context.Background(), Request{bundle, filepath.Join(root, "installed")})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{runtimeidentity.PipxRuntimeAssetName, "pipx-NOTICE.txt", manifestName} {
		info, err := os.Lstat(filepath.Join(result.InstallRoot, "versions", "v1.2.3", name))
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatal("DATA became executable or user metadata mode changed", name, err)
		}
	}
}

func TestPipxNoticesBindExactSelectedSourceBytes(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "tools", "pipx", "NOTICE"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePipxNotices(body); err != nil {
		t.Fatal(err)
	}
	body[len(body)-1] ^= 1
	if err := ValidatePipxNotices(body); err == nil {
		t.Fatal("changed notices accepted")
	}
}
