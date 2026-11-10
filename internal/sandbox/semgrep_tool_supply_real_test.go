//go:build linux && realcorpus

package sandbox

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// Only fixed /bin/sleep keepers execute. Semgrep and artifact source are DATA
// here; this closes supply lifetime, not actual scanner/gVisor qualification.
func TestSemgrepActualReadonlySupplyLifecycle(t *testing.T) {
	input := os.Getenv("HELOX_SEMGREP_SUPPLY_LIFECYCLE")
	if input == "" {
		t.Skip("explicit selected supply and prepared private namespace required")
	}
	if os.Geteuid() != 0 {
		t.Fatal("root inside private namespace required")
	}
	self, err := os.Readlink("/proc/self/ns/mnt")
	if err != nil {
		t.Fatal(err)
	}
	initial, err := os.Readlink("/proc/1/ns/mnt")
	if err != nil || self == initial {
		t.Fatal("refusing Host mount namespace")
	}
	root := t.TempDir()
	if err := os.Chmod(root, 0755); err != nil {
		t.Fatal(err)
	}
	version := filepath.Join(root, "versions", "v1.0.0-s04-supply")
	if err := os.MkdirAll(version, 0755); err != nil {
		t.Fatal(err)
	}
	pin := runtimeidentity.SelectedSemgrepSupply()
	for _, asset := range runtimeidentity.SemgrepReleaseAssets(pin) {
		if asset.Path != pin.Tool.Path && asset.Path != runtimeidentity.SemgrepRulesAssetName {
			continue
		}
		in, err := os.Open(filepath.Join(input, asset.Path))
		if err != nil {
			t.Fatal(err)
		}
		out, err := os.OpenFile(filepath.Join(version, asset.Path), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			_ = in.Close()
			t.Fatal(err)
		}
		count, copyErr := io.CopyN(out, in, asset.Size)
		closeErr, inputErr := out.Close(), in.Close()
		if copyErr != nil || closeErr != nil || inputErr != nil || count != asset.Size {
			t.Fatal("selected source copy", copyErr, closeErr, inputErr)
		}
		mode := os.FileMode(0444)
		if asset.Path == pin.Tool.Path {
			mode = 0555
		}
		if err := os.Chmod(filepath.Join(version, asset.Path), mode); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("versions/v1.0.0-s04-supply", filepath.Join(root, "current")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	tool, err := readInstalledSemgrepTool(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	transaction, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	volume, err := createSemgrepSupplyVolume(ctx, executor, "s04-actual-"+transaction.String())
	if err != nil {
		t.Fatal(err)
	}
	// Reuse existing fixed keeper primitives in this test only. The production
	// scanner will own a separate traced execution and target source lifecycle.
	keepers := &pipxNativeSession{runner: executor}
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 30*time.Second)
		defer done()
		for index := len(keepers.keepers) - 1; index >= 0; index-- {
			if err := keepers.removeKeeper(cleanup, keepers.keepers[index]); err != nil {
				t.Error(err)
			}
		}
		if err := volume.close(cleanup, executor); err != nil {
			t.Error(err)
		}
	}()
	rw, err := volume.preparationMount()
	if err != nil {
		t.Fatal(err)
	}
	writer, err := keepers.dataKeeper(ctx, "0:0", "--mount", rw)
	if err != nil {
		t.Fatal(err)
	}
	if err := volume.seed(ctx, executor, tool, writer.id); err != nil {
		t.Fatal(err)
	}
	ro, err := volume.readerMount()
	if err != nil {
		t.Fatal(err)
	}
	anchor, err := keepers.dataKeeper(ctx, "1000:1000", "--mount", ro)
	if err != nil {
		t.Fatal(err)
	}
	if err := volume.bindAnchor(ctx, executor, anchor.id); err != nil {
		t.Fatal(err)
	}
	if err := volume.seal(ctx, executor); err == nil {
		t.Fatal("attached writer was treated as absent")
	}
	if err := keepers.removeKeeper(ctx, writer); err != nil {
		t.Fatal(err)
	}
	if err := volume.seal(ctx, executor); err != nil {
		t.Fatal(err)
	}
	if err := volume.verifyReader(ctx, executor, anchor.id); err != nil {
		t.Fatal(err)
	}
	competitor, err := keepers.dataKeeper(ctx, "0:0", "--mount", rw)
	if err != nil {
		t.Fatal(err)
	}
	if err := volume.verifyReader(ctx, executor, anchor.id); err == nil {
		t.Fatal("competing writer admitted")
	}
	if err := keepers.removeKeeper(ctx, competitor); err != nil {
		t.Fatal(err)
	}
	if err := volume.verifyReader(ctx, executor, anchor.id); err != nil {
		t.Fatal(err)
	}
	t.Log("selected core/rules exact protected reader and Docker tar readback; readonly seal, writer-removal prerequisite and competing-writer refusal; only fixed sleep ran, no scanner/artifact execution")
}
