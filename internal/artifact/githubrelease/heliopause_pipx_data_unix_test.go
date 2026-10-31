//go:build linux || darwin

package githubrelease

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestHeliopausePipxDataGraphIncludesThirteenExactSubjects(t *testing.T) {
	zip := []byte("unsigned graph DATA only; never imported")
	notices, err := os.ReadFile(filepath.Join("..", "..", "..", "tools", "pipx", "NOTICE"))
	if err != nil {
		t.Fatal(err)
	}
	f := newReleaseSetFixture(t, func(metadata map[string]any) {
		assets := metadata["assets"].([]any)
		for name, role := range map[string]string{runtimeidentity.PipxRuntimeAssetName: "pipx-runtime", "pipx-NOTICE.txt": "release-metadata"} {
			data := zip
			if name == "pipx-NOTICE.txt" {
				data = notices
			}
			assets = append(assets, map[string]any{"name": name, "platform": "linux/amd64", "role": role, "size": len(data), "sha256": fmt.Sprintf("%x", sha256.Sum256(data))})
		}
		metadata["assets"] = assets
	})
	f.files[runtimeidentity.PipxRuntimeAssetName] = zip
	f.files["pipx-NOTICE.txt"] = notices
	resolution, err := f.set.ResolveDependencies(context.Background(), f.set.Reference(), f.install)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolution.Graph().Nodes()) != 13 || len(resolution.Graph().Edges()) != 12 {
		t.Fatal("incomplete DATA graph")
	}
	manifest, err := f.set.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	native, err := manifest.NativeUpdateFiles()
	if err != nil || len(native) != 6 {
		t.Fatal("native6 changed", err)
	}
	payloads, err := manifest.NativeUpdatePayloads()
	if err != nil || len(payloads) != 12 {
		t.Fatal("incomplete payload selection", err)
	}
	found := 0
	for _, node := range resolution.Graph().Nodes() {
		name := node.Artifact().Identity().Variant()
		if name != runtimeidentity.PipxRuntimeAssetName && name != "pipx-NOTICE.txt" {
			continue
		}
		found++
		run, err := domain.NewRunID()
		if err != nil {
			t.Fatal(err)
		}
		acquired, err := f.set.Acquire(context.Background(), run, node.Artifact())
		if err != nil {
			t.Fatal(err)
		}
		data := zip
		if name == "pipx-NOTICE.txt" {
			data = notices
		}
		if acquired.Identity().Source().String() != "github-release" || acquired.Identity().Name() != "rahoney-heliopause" || acquired.Digest().String() != fmt.Sprintf("%x", sha256.Sum256(data)) || acquired.SizeBytes() != uint64(len(data)) {
			t.Fatal("original HAA DATA binding changed")
		}
	}
	if found != 2 {
		t.Fatal("DATA pair absent")
	}
}
