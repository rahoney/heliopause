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

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestGoBuildCacheSnapshotUsesRetainedApprovedBytes(t *testing.T) {
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
	defer guard.Close()
	snapshot, cache, err := guard.OpenBuildInputs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := goCacheInventory(context.Background(), cache)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := guard.SnapshotBuildCache(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Identity().Source().String() != "project-local" || artifact.Identity().Variant() != "go-cache" || artifact.Identity().Version() != snapshot.GraphDigest().String() {
		t.Fatal("cache snapshot lost independent graph binding")
	}
	parts := strings.Split(artifact.ContentHandle(), ":")
	if len(parts) != 3 || parts[0] != "intake" || parts[2] != "go-cache" {
		t.Fatal("cache snapshot handle is not opaque intake")
	}
	if _, err := domain.ParseRunID(parts[1]); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(p.cache.intakeRoot, parts[1], "go-cache.tar")
	info, err := os.Lstat(file)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o400 {
		t.Fatal("cache snapshot is not private readonly data")
	}
	body, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(body)
	if uint64(len(body)) != artifact.SizeBytes() || hex.EncodeToString(hash[:]) != artifact.Digest().String() {
		t.Fatal("cache snapshot digest differs from consumed bytes")
	}
	want := map[string]goCacheFile{}
	for _, entry := range inventory {
		if entry.Path != "." {
			want[entry.Path] = entry
		}
	}
	seen := map[string]bool{}
	reader := tar.NewReader(bytes.NewReader(body))
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		name := strings.TrimSuffix(header.Name, "/")
		expected, ok := want[name]
		if !ok || seen[name] || header.Uid != 0 || header.Gid != 0 || header.Linkname != "" || header.Uname != "" || header.Gname != "" {
			t.Fatal("cache snapshot copied unauthenticated member or ownership")
		}
		seen[name] = true
		if expected.Kind == "directory" {
			if header.Typeflag != tar.TypeDir || header.Mode != 0o700 {
				t.Fatal("cache snapshot directory role differs")
			}
			continue
		}
		if header.Typeflag != tar.TypeReg || header.Mode != 0o400 || header.Size != expected.Size {
			t.Fatal("cache snapshot file role or size differs")
		}
		hash := sha256.New()
		n, err := io.Copy(hash, reader)
		if err != nil || n != expected.Size || hex.EncodeToString(hash.Sum(nil)) != expected.SHA256 {
			t.Fatal("cache snapshot copied bytes outside retained approval")
		}
	}
	if len(seen) != len(want) {
		t.Fatal("cache snapshot lost approved inventory")
	}
}

func TestGoBuildCacheSnapshotRejectsPoisonedInputsAndRemovesFailedSpool(t *testing.T) {
	for _, kind := range []string{"extra", "content", "mode", "hardlink", "symlink", "receipt", "evidence", "cancelled", "closed"} {
		t.Run(kind, func(t *testing.T) {
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
			defer guard.Close()
			_, cache, err := guard.OpenBuildInputs(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			inventory, err := goCacheInventory(context.Background(), cache)
			if err != nil {
				t.Fatal(err)
			}
			member := ""
			for _, entry := range inventory {
				if entry.Kind == "file" {
					member = filepath.Join(cache, filepath.FromSlash(entry.Path))
					break
				}
			}
			if member == "" {
				t.Fatal("fixture lacks approved cache data")
			}
			ctx := context.Background()
			switch kind {
			case "extra":
				err = os.WriteFile(filepath.Join(cache, "undeclared"), []byte("poison"), 0o444)
			case "content":
				if err = os.Chmod(member, 0o600); err == nil {
					err = os.WriteFile(member, []byte("poison"), 0o600)
				}
			case "mode":
				err = os.Chmod(member, 0o644)
			case "hardlink":
				err = os.Link(member, filepath.Join(t.TempDir(), "alias"))
			case "symlink":
				if err = os.Remove(member); err == nil {
					err = os.Symlink("/dev/null", member)
				}
			case "receipt":
				receipt := filepath.Join(filepath.Dir(cache), goCacheReceipt)
				if err = os.Chmod(receipt, 0o600); err == nil {
					err = os.WriteFile(receipt, []byte("{}"), 0o600)
				}
			case "evidence":
				err = os.RemoveAll(p.cache.evidenceRoot)
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "closed":
				err = guard.Close()
			}
			if err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadDir(p.cache.intakeRoot)
			if err != nil {
				t.Fatal(err)
			}
			artifact, err := guard.SnapshotBuildCache(ctx)
			if err == nil || artifact.ContentHandle() != "" || artifact.Digest().String() != "" {
				t.Fatal("failed cache snapshot returned usable build input")
			}
			after, err := os.ReadDir(p.cache.intakeRoot)
			if err != nil {
				t.Fatal(err)
			}
			if len(before) != len(after) {
				t.Fatal("failed cache snapshot retained partial spool")
			}
			for i := range before {
				if before[i].Name() != after[i].Name() {
					t.Fatal("failed cache snapshot changed intake inventory")
				}
			}
		})
	}
}

func TestGoBuildCacheArchiveBoundRejectsOverflowBeforeWriting(t *testing.T) {
	var body bytes.Buffer
	writer := &goBuildCacheArchiveWriter{writer: &body, remaining: 3}
	if n, err := writer.Write([]byte("abc")); n != 3 || err != nil {
		t.Fatal("exact cache archive boundary rejected")
	}
	if n, err := writer.Write([]byte("d")); n != 0 || err == nil || body.String() != "abc" {
		t.Fatal("cache archive overflow wrote beyond bound")
	}
}

func TestGoBuildCacheSnapshotRejectsUnmanagedApproval(t *testing.T) {
	_, guard, _, _ := approvedGoFixture(t)
	defer guard.Close()
	if artifact, err := guard.SnapshotBuildCache(context.Background()); err == nil || artifact.ContentHandle() != "" {
		t.Fatal("unmanaged cache recovered build approval")
	}
}

type goBuildCacheMutationWriter struct {
	body   bytes.Buffer
	mutate func([]byte)
}

func (w *goBuildCacheMutationWriter) Write(body []byte) (int, error) {
	if w.mutate != nil {
		w.mutate(body)
	}
	return w.body.Write(body)
}

func TestGoBuildCacheCopyRehashesConsumedBytesDespiteRestoredMetadata(t *testing.T) {
	p, guard, _, staged := approvedGoFixture(t)
	defer guard.Close()
	cache, err := p.cache.OpenProjectCache(context.Background(), staged)
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := goCacheInventory(context.Background(), cache)
	if err != nil {
		t.Fatal(err)
	}
	var target goCacheFile
	for _, entry := range inventory {
		if entry.Kind == "file" && entry.Size > 0 {
			target = entry
			break
		}
	}
	if target.Path == "" {
		t.Fatal("fixture lacks nonempty authenticated content")
	}
	name := filepath.Join(cache, filepath.FromSlash(target.Path))
	original, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(cache)
	if err != nil {
		t.Fatal(err)
	}
	roots := []*os.Root{root}
	defer func() {
		for i := len(roots) - 1; i >= 0; i-- {
			_ = roots[i].Close()
		}
	}()
	changed := false
	output := &goBuildCacheMutationWriter{}
	output.mutate = func(header []byte) {
		if changed || len(header) != 512 || header[156] != '0' || strings.TrimRight(string(header[:100]), "\x00") != target.Path {
			return
		}
		changed = true
		body, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		body[0] ^= 1
		if err := os.Chmod(name, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, body, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(name, original.Mode().Perm()); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(name, original.ModTime(), original.ModTime()); err != nil {
			t.Fatal(err)
		}
		current, err := os.Stat(name)
		if err != nil || !os.SameFile(original, current) || current.Size() != original.Size() || current.Mode() != original.Mode() || !current.ModTime().Equal(original.ModTime()) {
			t.Fatal("mutation fixture failed to preserve file metadata")
		}
	}
	writer := tar.NewWriter(output)
	err = copyApprovedGoBuildCache(context.Background(), root, inventory, writer, &roots)
	if !changed || err == nil || !strings.Contains(err.Error(), "consumed bytes differ") {
		t.Fatal("restored metadata hid changed consumed cache bytes")
	}
	if _, err := p.cache.OpenProjectCache(context.Background(), staged); err == nil {
		t.Fatal("poisoned cache retained approval")
	}
}
