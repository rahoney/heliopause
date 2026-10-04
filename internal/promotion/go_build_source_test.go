package promotion

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
)

func retainedGoBuildFixture(t *testing.T) (*GoProjectPromotion, *approvedGoProjectGuard) {
	t.Helper()
	p, guard, update, staged := approvedGoFixture(t)
	if err := guard.Commit(context.Background(), update, staged); err != nil {
		t.Fatal(err)
	}
	if err := guard.Close(); err != nil {
		t.Fatal(err)
	}
	guard, err := p.Begin(context.Background(), update.Snapshot().Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if !guard.closed {
			if err := guard.Close(); err != nil {
				t.Error(err)
			}
		}
	})
	return p, guard
}

func writeGoBuildSource(t *testing.T, root, name, content string) {
	t.Helper()
	name = filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(name), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestGoBuildSourceCapturesDataWithoutCredentialNamespaces(t *testing.T) {
	p, guard := retainedGoBuildFixture(t)
	project := guard.plan.root
	writeGoBuildSource(t, project, "main.go", "package main\nfunc main() {}\n")
	writeGoBuildSource(t, project, "assets/input.txt", "ordinary source data")
	writeGoBuildSource(t, project, "testdata/broken.go", "package broken\nfunc !\n")
	writeGoBuildSource(t, project, ".env", "synthetic excluded value")
	writeGoBuildSource(t, project, ".git/config", "synthetic excluded config")
	writeGoBuildSource(t, project, "assets/.private/secret", "synthetic excluded nested value")
	if err := os.Mkdir(filepath.Join(project, "empty"), 0o700); err != nil {
		t.Fatal(err)
	}
	artifact, err := guard.SnapshotBuildSource(context.Background())
	if err != nil || artifact.ContentHandle() == "" || artifact.Identity().Source().String() != "project-local" || artifact.Identity().Variant() != "go-source" {
		t.Fatalf("local build source: %v", err)
	}
	if strings.Contains(artifact.Identity().Name(), project) {
		t.Fatal("project path leaked into public identity")
	}
	parts := strings.Split(artifact.ContentHandle(), ":")
	if len(parts) != 3 || parts[0] != "intake" || parts[2] != "go-source" {
		t.Fatal("unexpected source handle")
	}
	archivePath := filepath.Join(p.cache.intakeRoot, parts[1], "go-source.tar")
	archive, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(archive)
	if hex.EncodeToString(hash[:]) != artifact.Digest().String() || uint64(len(archive)) != artifact.SizeBytes() {
		t.Fatal("source bytes do not match exact artifact")
	}
	info, err := os.Lstat(archivePath)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o400 {
		t.Fatalf("source intake mode: %v", err)
	}
	want := map[string]string{
		"main.go":            "package main\nfunc main() {}\n",
		"assets/input.txt":   "ordinary source data",
		"testdata/broken.go": "package broken\nfunc !\n",
	}
	for _, control := range guard.Controls() {
		want[control.Name()] = string(control.Body())
	}
	dirs := map[string]bool{"assets": true, "testdata": true, "empty": true}
	reader := tar.NewReader(bytes.NewReader(archive))
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if header.Typeflag == tar.TypeDir {
			if !dirs[header.Name] || header.Mode != 0o700 {
				t.Fatal("unplanned source directory")
			}
			delete(dirs, header.Name)
			continue
		}
		body, err := io.ReadAll(reader)
		expected, exists := want[header.Name]
		if err != nil || !exists || string(body) != expected || header.Typeflag != tar.TypeReg || header.Mode != 0o400 {
			t.Fatalf("unplanned or changed source member %q", header.Name)
		}
		delete(want, header.Name)
	}
	if len(want) != 0 || len(dirs) != 0 {
		t.Fatal("source inventory is incomplete")
	}
	if err := guard.VerifyBuildSource(context.Background()); err != nil {
		t.Fatal(err)
	}
	writeGoBuildSource(t, project, ".env", "another synthetic excluded value")
	if err := guard.VerifyBuildSource(context.Background()); err != nil {
		t.Fatal("excluded namespace became source input")
	}
	if next, err := guard.SnapshotBuildSource(context.Background()); err == nil || next.ContentHandle() != "" {
		t.Fatal("source snapshot was replaced under one guard")
	}
}

func TestGoBuildSourceRejectsDriftBeforePublication(t *testing.T) {
	for _, kind := range []string{"content", "inode", "mode", "new-file", "removed-file", "new-directory", "directory-inode", "directory-alias", "controls"} {
		t.Run(kind, func(t *testing.T) {
			_, guard := retainedGoBuildFixture(t)
			project := guard.plan.root
			writeGoBuildSource(t, project, "pkg/source.go", "package pkg\n")
			if _, err := guard.SnapshotBuildSource(context.Background()); err != nil {
				t.Fatal(err)
			}
			name := filepath.Join(project, "pkg", "source.go")
			var err error
			switch kind {
			case "content":
				err = os.WriteFile(name, []byte("package changed\n"), 0o600)
			case "inode":
				writeGoBuildSource(t, project, ".replacement", "package pkg\n")
				err = os.Rename(filepath.Join(project, ".replacement"), name)
			case "mode":
				err = os.Chmod(name, 0o700)
			case "new-file":
				writeGoBuildSource(t, project, "pkg/added.go", "package pkg\n")
			case "removed-file":
				err = os.Remove(name)
			case "new-directory":
				err = os.Mkdir(filepath.Join(project, "additional"), 0o700)
			case "directory-inode", "directory-alias":
				err = os.Rename(filepath.Join(project, "pkg"), filepath.Join(project, ".old-pkg"))
				if err == nil && kind == "directory-alias" {
					err = os.Symlink(".old-pkg", filepath.Join(project, "pkg"))
				} else if err == nil {
					writeGoBuildSource(t, project, "pkg/source.go", "package pkg\n")
				}
			case "controls":
				err = os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.com/drift\n"), 0o600)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := guard.VerifyBuildSource(context.Background()); err == nil {
				t.Fatalf("changed %s source remained publishable", kind)
			}
		})
	}
}

func TestGoBuildSourceRejectsLinksAndOversizeBeforeIntake(t *testing.T) {
	for _, kind := range []string{"outside-symlink", "hidden-symlink", "directory-symlink", "hardlink", "oversize", "cancelled", "closed", "unmanaged"} {
		t.Run(kind, func(t *testing.T) {
			p, guard := retainedGoBuildFixture(t)
			project := guard.plan.root
			writeGoBuildSource(t, project, ".private/token", "synthetic excluded value")
			ctx := context.Background()
			var err error
			switch kind {
			case "outside-symlink":
				err = os.Symlink(t.TempDir(), filepath.Join(project, "visible"))
			case "hidden-symlink":
				err = os.Symlink(".private/token", filepath.Join(project, "visible"))
			case "directory-symlink":
				err = os.Symlink(".private", filepath.Join(project, "visible"))
			case "hardlink":
				err = os.Link(filepath.Join(project, ".private/token"), filepath.Join(project, "visible"))
			case "oversize":
				writeGoBuildSource(t, project, "visible", "")
				err = os.Truncate(filepath.Join(project, "visible"), artifactgo.MaxModuleFileBytes+1)
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "closed":
				err = guard.Close()
			case "unmanaged":
				guard.originalState = nil
			}
			if err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadDir(p.cache.intakeRoot)
			if err != nil {
				t.Fatal(err)
			}
			if artifact, err := guard.SnapshotBuildSource(ctx); err == nil || artifact.ContentHandle() != "" || guard.buildSource != nil {
				t.Fatalf("invalid %s source was frozen", kind)
			}
			after, err := os.ReadDir(p.cache.intakeRoot)
			if err != nil || len(before) != len(after) {
				t.Fatal("rejected source introduced partial intake")
			}
		})
	}
}
