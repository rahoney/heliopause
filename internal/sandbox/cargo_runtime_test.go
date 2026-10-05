package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCargoRuntimeLockMatchesPinnedIdentity(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "scripts", "runtimes.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		RustImage struct {
			Reference    string `json:"reference"`
			Version      string `json:"rust_version"`
			Architecture string `json:"architecture"`
			Target       string `json:"target"`
		} `json:"rust_image"`
	}
	if err := json.Unmarshal(body, &lock); err != nil {
		t.Fatal(err)
	}
	if got := PinnedCargoRuntime(); got != (CargoRuntime{lock.RustImage.Reference, lock.RustImage.Version, lock.RustImage.Architecture, lock.RustImage.Target}) {
		t.Fatalf("cargo runtime lock mismatch: %#v", got)
	}
}

func TestProbeCargoRejectsUnverifiedRuntimeOrImage(t *testing.T) {
	base := map[string]string{
		"docker version --format {{.Server.Version}}": "29.8.1",
		"runsc --version":      canonicalRunscVersionOutput,
		"runsc trace metadata": "Name: syscall/open_result\nName: sentry/mount_topology_snapshot\nName: sentry/mount_topology_mutation\n",
		"docker image inspect " + PinnedCargoRuntime().ImageReference + " --format {{.Id}} {{.Architecture}}": "sha256:" + strings.Repeat("a", 64) + " amd64",
	}
	tests := []struct {
		name, operatingSystem, architecture, image, limitation string
		available                                              bool
	}{
		{"native unsupported", "darwin", "amd64", "", "M12_CARGO_LINUX_AMD64_ONLY", false},
		{"architecture unsupported", "linux", "arm64", "", "M12_CARGO_LINUX_AMD64_ONLY", false},
		{"missing image", "linux", "amd64", "", "M12_CARGO_IMAGE_UNAVAILABLE", false},
		{"wrong image architecture", "linux", "amd64", "sha256:" + strings.Repeat("a", 64) + " arm64", "M12_CARGO_IMAGE_UNAVAILABLE", false},
		{"non digest identity", "linux", "amd64", "sha256:" + strings.Repeat("x", 64) + " amd64", "M12_CARGO_IMAGE_UNAVAILABLE", false},
		{"extra image fields", "linux", "amd64", "sha256:" + strings.Repeat("a", 64) + " amd64 untrusted", "M12_CARGO_IMAGE_UNAVAILABLE", false},
		{"pinned image", "linux", "amd64", base["docker image inspect "+PinnedCargoRuntime().ImageReference+" --format {{.Id}} {{.Architecture}}"], "", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outputs := make(map[string]string, len(base))
			for key, value := range base {
				outputs[key] = value
			}
			outputs["docker image inspect "+PinnedCargoRuntime().ImageReference+" --format {{.Id}} {{.Architecture}}"] = test.image
			got, err := probeCargo(context.Background(), test.operatingSystem, test.architecture, fakeExecutor{outputs: outputs})
			if err != nil || got.Available != test.available || got.LimitationCode != test.limitation || got.Runtime != PinnedCargoRuntime() {
				t.Fatalf("probeCargo = %#v, %v", got, err)
			}
		})
	}
	if _, err := probeCargo(absentCargoContext(), "linux", "amd64", fakeExecutor{}); err == nil {
		t.Fatal("missing context accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := probeCargo(ctx, "linux", "amd64", fakeExecutor{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled probe: %v", err)
	}
}

func absentCargoContext() context.Context { return nil }
