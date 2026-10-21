package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const bundleVersion = "v1.2.3"
const bundleCommit = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
const bundleWorkflow = "https://github.com/rahoney/heliopause/actions/runs/123"

func bundleFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	binary := filepath.Join(root, "helox-linux-amd64")
	buildNativeHeloxTest(t, binary)
	assets := releaseAssets(t, root, binary)
	lock := filepath.Join(t.TempDir(), "runtimes.lock.json")
	if err := os.WriteFile(lock, []byte(`{"schema_version":2,"gvisor":{"release":"release-20260810.0","commit":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	images := filepath.Join(root, imageFileName)
	if err := os.WriteFile(images, []byte(`{"schema":"helox.runtime-image-manifest/v1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, manifestFileName)
	if err := buildRelease(bundleVersion, bundleCommit, bundleWorkflow, lock, filepath.Join(root, sbomFileName), images, manifest, assets); err != nil {
		t.Fatal(err)
	}
	if err := writeChecksums(manifest, filepath.Join(root, checksumFileName), bundleVersion, bundleCommit, bundleWorkflow); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestVerifyBundleBindsPackagesAndExactFileSet(t *testing.T) {
	root := bundleFixture(t)
	names, err := verifyBundle(root, bundleVersion, bundleCommit, bundleWorkflow)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 13 {
		t.Fatalf("complete bundle has %d files, want 13", len(names))
	}
	for _, name := range []string{"haa-release-installer-linux-amd64", "helox-linux-amd64.deb", "helox-linux-amd64.rpm", manifestFileName, checksumFileName, sbomFileName, imageFileName} {
		if !slices.Contains(names, name) {
			t.Fatalf("verified bundle omits %s", name)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != len(names) {
		t.Fatal("verified list does not match exact directory", err)
	}
	if err := writeChecksums(filepath.Join(root, manifestFileName), filepath.Join(root, checksumFileName), bundleVersion, bundleCommit, bundleWorkflow); err == nil {
		t.Fatal("pre-existing checksums overwritten")
	}
	body, err := os.ReadFile(filepath.Join(root, manifestFileName))
	if err != nil {
		t.Fatal(err)
	}
	var value releaseManifest
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct{ name, role string }{{"haa-release-installer-linux-amd64", "release-installer"}, {"helox-linux-amd64.deb", "os-package-deb"}, {"helox-linux-amd64.rpm", "os-package-rpm"}} {
		found := false
		for _, asset := range value.Assets {
			if asset.Name == want.name {
				found = asset.Role == want.role && asset.Platform == "linux/amd64"
			}
		}
		if !found {
			t.Fatalf("package identity missing: %s", want.name)
		}
	}
}

func TestVerifyBundleRejectsTamperingAndUnboundControlData(t *testing.T) {
	for _, name := range []string{"extra-file", "directory", "symlink-payload", "missing-package", "changed-package", "missing-installer", "changed-installer", "checksum-path", "wrong-source", "wrong-run", "wrong-version", "unknown-manifest-field", "unsafe-asset-name", "wrong-role", "bad-lock-digest"} {
		t.Run(name, func(t *testing.T) {
			root := bundleFixture(t)
			version, commit, workflow := bundleVersion, bundleCommit, bundleWorkflow
			switch name {
			case "extra-file":
				if err := os.WriteFile(filepath.Join(root, "unlisted"), []byte("extra"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(filepath.Join(root, "unlisted"), 0o700); err != nil {
					t.Fatal(err)
				}
			case "symlink-payload":
				p := filepath.Join(root, "helox-linux-amd64.deb")
				if err := os.Remove(p); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(root, "helox-linux-amd64.rpm"), p); err != nil {
					t.Fatal(err)
				}
			case "missing-package":
				if err := os.Remove(filepath.Join(root, "helox-linux-amd64.rpm")); err != nil {
					t.Fatal(err)
				}
			case "missing-installer":
				if err := os.Remove(filepath.Join(root, "haa-release-installer-linux-amd64")); err != nil {
					t.Fatal(err)
				}
			case "changed-installer":
				if err := os.WriteFile(filepath.Join(root, "haa-release-installer-linux-amd64"), []byte("changed installer"), 0600); err != nil {
					t.Fatal(err)
				}
			case "changed-package":
				if err := os.WriteFile(filepath.Join(root, "helox-linux-amd64.deb"), []byte("different"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "checksum-path":
				if err := os.WriteFile(filepath.Join(root, checksumFileName), []byte(strings.Repeat("a", 64)+"  ../outside\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "wrong-source":
				commit = strings.Repeat("c", 40)
			case "wrong-run":
				workflow = "https://github.com/rahoney/heliopause/actions/runs/124"
			case "wrong-version":
				version = "v1.2.4"
			default:
				path := filepath.Join(root, manifestFileName)
				body, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var value map[string]any
				if err := json.Unmarshal(body, &value); err != nil {
					t.Fatal(err)
				}
				switch name {
				case "unknown-manifest-field":
					value["unknown"] = true
				case "unsafe-asset-name":
					value["assets"].([]any)[0].(map[string]any)["name"] = "../outside"
				case "wrong-role":
					value["assets"].([]any)[0].(map[string]any)["role"] = "unknown"
				case "bad-lock-digest":
					value["runtime_lock"].(map[string]any)["sha256"] = strings.Repeat("z", 64)
				}
				body, err = json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, body, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := verifyBundle(root, version, commit, workflow); err == nil {
				t.Fatal("unsafe or unbound bundle accepted")
			}
		})
	}
}
