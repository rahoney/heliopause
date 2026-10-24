package promotion

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func requireNoCargoCachePublication(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil && !os.IsNotExist(err) || len(entries) != 0 {
		t.Fatalf("failed cache retained private/published data: %v %v", entries, err)
	}
}

func TestCargoCacheSyncBarrierFailureDoesNotPublish(t *testing.T) {
	cache, root := newCargoCacheForTest(t)
	set, intake := cargoCacheFixture(t, cache, filepath.Join(root, "project"), nil)
	before, err := os.ReadFile(intake)
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("private cache sync fixture")
	calls := 0
	cache.beforeTreeSync = func(string) error { calls++; return cause }
	staged, err := cache.StageProject(context.Background(), set)
	if !errors.Is(err, cause) || staged.Valid() || calls != 1 {
		t.Fatalf("sync failure published/lost original cause: valid=%v calls=%d err=%v", staged.Valid(), calls, err)
	}
	requireNoCargoCachePublication(t, cache.cacheRoot)
	after, err := os.ReadFile(intake)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("sync failure changed approved intake", err)
	}
}

func TestCargoCacheSyncBarrierSeesSealedUnpublishedTree(t *testing.T) {
	cache, root := newCargoCacheForTest(t)
	set, _ := cargoCacheFixture(t, cache, filepath.Join(root, "project"), nil)
	calls := 0
	cache.beforeTreeSync = func(temporary string) error {
		calls++
		if filepath.Dir(temporary) != cache.cacheRoot || !strings.HasPrefix(filepath.Base(temporary), ".cargo-stage-") {
			t.Fatal("durability barrier does not own an unpublished tree", temporary)
		}
		entries, err := os.ReadDir(cache.cacheRoot)
		if err != nil || len(entries) != 1 || entries[0].Name() != filepath.Base(temporary) {
			t.Fatal("cache published before final tree sync", entries, err)
		}
		if body, err := os.ReadFile(filepath.Join(temporary, cargoCacheReceipt)); err != nil || len(body) == 0 {
			t.Fatal("final tree sync precedes receipt", err)
		}
		return filepath.WalkDir(filepath.Join(temporary, "vendor"), func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			info, err := entry.Info()
			if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o222 != 0 {
				t.Fatal("unsealed member reached final sync", path, err)
			}
			return nil
		})
	}
	staged, err := cache.StageProject(context.Background(), set)
	if err != nil || !staged.Valid() || calls != 1 {
		t.Fatal("final private-tree barrier missing", calls, err)
	}
	if _, err := cache.OpenProjectCache(context.Background(), staged); err != nil {
		t.Fatal("synced cache cannot be independently reopened", err)
	}
}

func TestCargoCacheSyncBarrierRejectsLateSymlink(t *testing.T) {
	cache, root := newCargoCacheForTest(t)
	set, _ := cargoCacheFixture(t, cache, filepath.Join(root, "project"), nil)
	outside := filepath.Join(root, "outside")
	const original = "foreign bytes"
	if err := os.WriteFile(outside, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(outside)
	if err != nil {
		t.Fatal(err)
	}
	cache.beforeTreeSync = func(temporary string) error {
		member := filepath.Join(temporary, "vendor", "itoa-1.0.17", "Cargo.toml")
		if err := os.Remove(member); err != nil {
			return err
		}
		return os.Symlink(outside, member)
	}
	staged, err := cache.StageProject(context.Background(), set)
	if err == nil || staged.Valid() {
		t.Fatal("final real sync was bypassed by a successful observer")
	}
	requireNoCargoCachePublication(t, cache.cacheRoot)
	if body, err := os.ReadFile(outside); err != nil || string(body) != original {
		t.Fatal("cache cleanup followed foreign symlink", err)
	}
	if after, err := os.Stat(outside); err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() {
		t.Fatal("cache cleanup changed foreign identity/mode", err)
	}
}

func TestPrivateCacheRecordKeepsExclusiveCreation(t *testing.T) {
	root := t.TempDir()
	if err := writePrivateRecord(root, "member", []byte("original")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "member")
	before, err := os.Stat(path)
	if err != nil || before.Mode().Perm() != 0o600 {
		t.Fatal("private member creation mode changed", err)
	}
	if err := writePrivateRecord(root, "member", []byte("replacement")); err == nil {
		t.Fatal("private writer replaced an existing member")
	}
	after, err := os.Stat(path)
	if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() {
		t.Fatal("private writer changed original identity", err)
	}
	if body, err := os.ReadFile(path); err != nil || string(body) != "original" {
		t.Fatal("private writer changed original bytes", err)
	}
}
