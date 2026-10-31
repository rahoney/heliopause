package promotion

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestUVProjectControlOnlyCacheKeepsApprovedContextAndDoesNotInstall(t *testing.T) {
	f := newUVProjectPublicationFixture(t)
	manifest, err := os.ReadFile(filepath.Join(f.root, "pyproject.toml"))
	if err != nil {
		t.Fatal(err)
	}
	lock, err := os.ReadFile("../artifact/pypi/testdata/uv-0.13.4-empty-project.uv.lock")
	if err != nil {
		t.Fatal(err)
	}
	// Project the historical format onto the current runtime; keep captured testdata unchanged.
	lock = bytes.ReplaceAll(lock, []byte("3.14.7"), []byte(runtimeidentity.PythonVersion))
	input, err := artifactpypi.ParseUVProjectInput(manifest, lock, true, artifactpypi.UVProjectLockUpgrade, "")
	if err != nil {
		t.Fatal(err)
	}
	original, err := sandbox.NewUVProjectOriginalControls(input)
	if err != nil {
		t.Fatal(err)
	}
	target, err := domain.NewInstallTarget(f.root)
	if err != nil {
		t.Fatal(err)
	}
	controls, err := domain.NewPythonProjectControlContext(target)
	if err != nil {
		t.Fatal(err)
	}
	update, err := f.resolver.ResolveProjectLockUpdate(context.Background(), controls, original, "")
	if err != nil {
		t.Fatal(err)
	}
	cache, err := NewUVProjectCache(f.publication.cache, update)
	if err != nil {
		t.Fatal(err)
	}
	approved := uvProjectSyntheticApproval(t, update.Snapshot())
	staged, err := cache.StageProject(context.Background(), approved)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cache.OpenProjectCache(context.Background(), staged); err != nil {
		t.Fatal(err)
	}
	if staged.Set().Inspected().Snapshot().Context() != controls || controls.Mode() == domain.InstallPythonVenv {
		t.Fatal("control-only approval became installed venv")
	}
	venv, err := domain.NewPythonVenvInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	other, err := domain.NewDependencyFreeProjectSnapshot(venv, update.Snapshot().Source(), update.Snapshot().ControlDigests(), update.Snapshot().GraphDigest())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cache.StageProject(context.Background(), uvProjectSyntheticApproval(t, other)); err == nil {
		t.Fatal("venv approval substituted control-only context")
	}
	for _, name := range []string{"uv.lock", ".venv", ".heliopause"} {
		if _, err := os.Lstat(filepath.Join(f.root, name)); !os.IsNotExist(err) {
			t.Fatal("cache changed project or installed state", name, err)
		}
	}
}
