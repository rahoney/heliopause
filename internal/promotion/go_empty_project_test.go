package promotion

import (
	"archive/tar"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/policy"
)

func TestDependencyFreeGoDownloadAdoptionAndCacheBinding(t *testing.T) {
	root := canonicalGoTestRoot(t)
	project := filepath.Join(root, "project")
	if err := os.Mkdir(project, 0o700); err != nil {
		t.Fatal(err)
	}
	mod := []byte("module example.com/app\ngo 1.26.0\n")
	if err := os.WriteFile(filepath.Join(project, "go.mod"), mod, 0o600); err != nil {
		t.Fatal(err)
	}
	cache, err := newGoCacheForTest(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := NewGoProjectPromotion(cache, filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(project)
	install, _ := domain.NewInstallContext(target)
	guard, err := p.Begin(context.Background(), install)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := artifactgo.BuildProjectSnapshot(install, nil, []byte("example.com/app go@1.26.0\ngo@1.26.0 toolchain@go1.26.0\n"), mod, nil)
	if err != nil {
		t.Fatal(err)
	}
	inspected, err := domain.NewInspectedProjectSet(snapshot, nil)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := (policy.M4{}).EvaluateProjectSet(inspected)
	if err != nil {
		t.Fatal(err)
	}
	set, err := domain.NewProjectVerifiedSet(inspected, decision)
	if err != nil {
		t.Fatal(err)
	}
	staged, err := cache.StageProject(context.Background(), set)
	if err != nil {
		t.Fatal(err)
	}
	if err := guard.CommitSnapshot(context.Background(), snapshot, staged); err != nil {
		t.Fatal(err)
	}
	if err := guard.Close(); err != nil {
		t.Fatal(err)
	}
	retained, err := p.Begin(context.Background(), install)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := retained.SnapshotBuildCache(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(artifact.ContentHandle(), ":")
	if len(parts) != 3 || parts[0] != "intake" || parts[2] != "go-cache" || artifact.Identity().Version() != snapshot.GraphDigest().String() {
		t.Fatal("empty build cache lost its approved graph binding")
	}
	archive, err := os.Open(filepath.Join(cache.intakeRoot, parts[1], "go-cache.tar"))
	if err != nil {
		t.Fatal(err)
	}
	_, entryErr := tar.NewReader(archive).Next()
	closeErr := archive.Close()
	if !errors.Is(entryErr, io.EOF) || closeErr != nil {
		t.Fatal("approved dependency-free build cache contained an entry")
	}
	if err := retained.Close(); err != nil {
		t.Fatal(err)
	}
	dir, err := cache.OpenProjectCache(context.Background(), staged)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "poison.go"), []byte("package poison\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.OpenProjectCache(context.Background(), staged); err == nil {
		t.Fatal("explicit empty cache accepted an extra dependency")
	}
	if next, err := p.Begin(context.Background(), install); err == nil || next != nil {
		t.Fatal("poisoned empty cache restored retained approval")
	}
}
