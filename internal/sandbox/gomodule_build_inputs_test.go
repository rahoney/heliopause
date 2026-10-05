package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type goBuildTarFixtureMember struct {
	name, body string
	kind       byte
	mode       int64
}

func goBuildInputFixture(t *testing.T, root, prefix string, members []goBuildTarFixtureMember) domain.AcquiredArtifact {
	t.Helper()
	// testing.TempDir's numbered child uses the process umask. The fixture
	// models the private intake, whose canonical owner prepares mode 0700.
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	writer := tar.NewWriter(&body)
	for _, member := range members {
		header := &tar.Header{Name: member.name, Typeflag: member.kind, Mode: member.mode, Size: int64(len(member.body))}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(member.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	run, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	variant := map[string]string{"project": "go-source", "cache": "go-cache"}[prefix]
	if err := os.Mkdir(filepath.Join(root, run.String()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, run.String(), variant+".tar"), body.Bytes(), 0o400); err != nil {
		t.Fatal(err)
	}
	source, _ := domain.NewSourceID("project-local")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "fixture", "snapshot", variant)
	hash := sha256.Sum256(body.Bytes())
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(hash[:]))
	artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+run.String()+":"+variant, uint64(body.Len()), "sha256:"+digest.String())
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func TestGoBuildInputsConsumeExactDataAndReconstructGuestMetadata(t *testing.T) {
	root := t.TempDir()
	mod := "module example.com/input\ngo 1.26.0\n"
	members := []goBuildTarFixtureMember{{"go.mod", mod, tar.TypeReg, 0o400}, {"go.sum", "", tar.TypeReg, 0o400}, {"testdata", "", tar.TypeDir, 0o700}, {"testdata/bad.go", "invalid Go parser fixture", tar.TypeReg, 0o400}}
	artifact := goBuildInputFixture(t, root, "project", members)
	input, err := openGoBuildInput(root, artifact, "project")
	if err != nil {
		t.Fatal(err)
	}
	defer input.close()
	manifest := goBuildInputManifest{members: map[string]goBuildInputMember{}, controls: map[string][]byte{}}
	if err := input.scan(context.Background(), &manifest, nil); err != nil {
		t.Fatal(err)
	}
	if len(manifest.members) != 4 || string(manifest.controls["go.mod"]) != mod || len(manifest.controls) != 2 {
		t.Fatal("input data/controls were changed or omitted")
	}
	var copied bytes.Buffer
	writer := tar.NewWriter(&copied)
	if err := input.scan(context.Background(), &manifest, writer); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	reader := tar.NewReader(&copied)
	for _, member := range members {
		header, err := reader.Next()
		if err != nil {
			t.Fatal(err)
		}
		mode := int64(0o400)
		if member.kind == tar.TypeDir {
			mode = 0o500
		}
		if header.Name != "project/"+member.name || header.Mode != mode || header.Typeflag != member.kind || header.Uid != 1000 || header.Gid != 1000 {
			t.Fatal("guest metadata does not match fixed input contract")
		}
		body, err := io.ReadAll(reader)
		if err != nil || string(body) != member.body {
			t.Fatal("input copy rewrote data")
		}
	}
	if _, err := reader.Next(); !errors.Is(err, io.EOF) {
		t.Fatal("input copy added a member")
	}
	cache := goBuildInputFixture(t, root, "cache", nil)
	empty, err := openGoBuildInput(root, cache, "cache")
	if err != nil {
		t.Fatal(err)
	}
	defer empty.close()
	if err := empty.scan(context.Background(), &manifest, nil); err != nil {
		t.Fatal("explicit empty cache rejected:", err)
	}
}

func TestGoBuildInputRejectsUnsafeMembersBeforeIntroduction(t *testing.T) {
	for _, member := range []goBuildTarFixtureMember{
		{"../escape", "x", tar.TypeReg, 0o400}, {"/absolute", "x", tar.TypeReg, 0o400}, {"dir/missing-parent", "x", tar.TypeReg, 0o400},
		{"symlink", "", tar.TypeSymlink, 0o400}, {"hardlink", "", tar.TypeLink, 0o400}, {"fifo", "", tar.TypeFifo, 0o400},
		{"writable", "x", tar.TypeReg, 0o600}, {"executable", "x", tar.TypeReg, 0o500}, {"special-mode", "x", tar.TypeReg, 0o4400},
		{".metadata", "x", tar.TypeReg, 0o400}, {"invalid:name", "x", tar.TypeReg, 0o400},
	} {
		t.Run(member.name, func(t *testing.T) {
			root := t.TempDir()
			artifact := goBuildInputFixture(t, root, "project", []goBuildTarFixtureMember{member})
			input, err := openGoBuildInput(root, artifact, "project")
			if err != nil {
				t.Fatal(err)
			}
			defer input.close()
			manifest := goBuildInputManifest{members: map[string]goBuildInputMember{}, controls: map[string][]byte{}}
			if err := input.scan(context.Background(), &manifest, nil); err == nil {
				t.Fatal("unsafe input accepted")
			}
		})
	}
	root := t.TempDir()
	artifact := goBuildInputFixture(t, root, "cache", []goBuildTarFixtureMember{{"same", "x", tar.TypeReg, 0o400}, {"same", "x", tar.TypeReg, 0o400}})
	input, err := openGoBuildInput(root, artifact, "cache")
	if err != nil {
		t.Fatal(err)
	}
	defer input.close()
	manifest := goBuildInputManifest{members: map[string]goBuildInputMember{}, controls: map[string][]byte{}}
	if err := input.scan(context.Background(), &manifest, nil); err == nil {
		t.Fatal("duplicate member accepted")
	}
}

func TestGoBuildInputRehashesCopiedContentAndRejectsCancellation(t *testing.T) {
	root := t.TempDir()
	artifact := goBuildInputFixture(t, root, "cache", []goBuildTarFixtureMember{{"data", "approved", tar.TypeReg, 0o400}})
	input, err := openGoBuildInput(root, artifact, "cache")
	if err != nil {
		t.Fatal(err)
	}
	defer input.close()
	manifest := goBuildInputManifest{members: map[string]goBuildInputMember{}, controls: map[string][]byte{}}
	if err := input.scan(context.Background(), &manifest, nil); err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(artifact.ContentHandle(), ":")
	file := filepath.Join(root, parts[1], "go-cache.tar")
	body, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	body[512] ^= 1
	if err := os.Chmod(file, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(file, 0o400); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(file, input.before.ModTime(), input.before.ModTime()); err != nil {
		t.Fatal(err)
	}
	var copied bytes.Buffer
	if err := input.scan(context.Background(), &manifest, tar.NewWriter(&copied)); err == nil {
		t.Fatal("restored metadata hid changed copied bytes")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := input.scan(ctx, &manifest, nil); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled copy continued")
	}
}

func TestGoBuildInputHandleAndOwnedFileBoundary(t *testing.T) {
	for _, kind := range []string{"hardlink", "symlink", "mode", "wrong-role", "run-alias"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			artifact := goBuildInputFixture(t, root, "cache", nil)
			parts := strings.Split(artifact.ContentHandle(), ":")
			file := filepath.Join(root, parts[1], "go-cache.tar")
			prefix := "cache"
			var err error
			switch kind {
			case "hardlink":
				err = os.Link(file, filepath.Join(root, "alias"))
			case "symlink":
				if err = os.Remove(file); err == nil {
					err = os.Symlink("/dev/null", file)
				}
			case "mode":
				err = os.Chmod(file, 0o600)
			case "wrong-role":
				prefix = "project"
			case "run-alias":
				original := filepath.Join(root, parts[1])
				moved := original + "-moved"
				if err = os.Rename(original, moved); err == nil {
					err = os.Symlink(moved, original)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if input, err := openGoBuildInput(root, artifact, prefix); err == nil {
				_ = input.close()
				t.Fatal("unsafe intake identity accepted")
			}
		})
	}
}

func TestGoBuildInputBindingRequiresExactProjectControlsAndGraph(t *testing.T) {
	target, _ := domain.NewInstallTarget(t.TempDir())
	install, _ := domain.NewInstallContext(target)
	mod := []byte("module example.com/input\ngo 1.26.0\n")
	snapshot, err := artifactgo.BuildProjectSnapshot(install, nil, []byte("example.com/input go@1.26.0\ngo@1.26.0 toolchain@go1.26.0\n"), mod, nil)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	source := goBuildInputFixture(t, root, "project", []goBuildTarFixtureMember{{"go.mod", string(mod), tar.TypeReg, 0o400}, {"go.sum", "", tar.TypeReg, 0o400}})
	cache := goBuildInputFixture(t, root, "cache", nil)
	key := sha256.Sum256([]byte(target.String()))
	local, _ := domain.NewSourceID("project-local")
	sourceID, _ := domain.NewResolvedArtifactIdentity(local, "project-"+hex.EncodeToString(key[:]), "snapshot", "go-source")
	cacheID, _ := domain.NewResolvedArtifactIdentity(local, "cache-"+hex.EncodeToString(key[:]), snapshot.GraphDigest().String(), "go-cache")
	source, _ = domain.NewAcquiredArtifactWithDeclaredIntegrity(sourceID, source.Digest(), source.ContentHandle(), source.SizeBytes(), "sha256:"+source.Digest().String())
	cache, _ = domain.NewAcquiredArtifactWithDeclaredIntegrity(cacheID, cache.Digest(), cache.ContentHandle(), cache.SizeBytes(), "sha256:"+cache.Digest().String())
	manifest := goBuildInputManifest{controls: map[string][]byte{"go.mod": mod, "go.sum": nil}}
	if err := validateGoBuildInputBinding(snapshot, source, cache, manifest); err != nil {
		t.Fatal(err)
	}
	manifest.controls["go.mod"] = append(append([]byte(nil), mod...), []byte("// drift\n")...)
	if err := validateGoBuildInputBinding(snapshot, source, cache, manifest); err == nil {
		t.Fatal("changed controls accepted")
	}
	if err := validateGoBuildInputBinding(domain.ProjectDependencySnapshot{}, source, cache, manifest); err == nil {
		t.Fatal("missing complete snapshot accepted")
	}
}
