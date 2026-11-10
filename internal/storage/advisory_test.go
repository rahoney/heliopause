package storage

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

func TestAdvisoryStorageConcurrentRefreshKeepsOneCompleteGeneration(t *testing.T) {
	ctx := advisoryStorageContext(t)
	path := storageTestRoot(t)
	store, err := NewLocal(ctx, path, storageFixturePolicy(t, 10))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key := advisoryStorageKey(t, "a")
	var workers sync.WaitGroup
	for n := range 8 {
		workers.Go(func() {
			body := bytes.Repeat([]byte{byte('a' + n)}, 128)
			if stored, err := store.WriteAdvisory(ctx, key, body); err != nil || !stored {
				t.Error("concurrent complete write failed", err)
				return
			}
			got, hit, err := store.ReadAdvisory(ctx, key, 4096)
			if err != nil || !hit || len(got) != len(body) || !bytes.Equal(got, bytes.Repeat(got[:1], len(got))) {
				t.Error("reader saw an unsealed or mixed generation", err)
			}
		})
	}
	workers.Wait()
	entries, err := os.ReadDir(filepath.Join(path, "advisory-cache"))
	if err != nil || len(entries) != 1 || store.pending != 0 || store.cacheCharged != 128 {
		t.Fatal("concurrent refresh leaked copies or accounting", err, len(entries), store.pending, store.cacheCharged)
	}
}

func TestAdvisoryStorageSweepPreservesActiveReservation(t *testing.T) {
	ctx := advisoryStorageContext(t)
	path := storageTestRoot(t)
	store, err := NewLocal(ctx, path, storageFixturePolicy(t, 10))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.withLock(ctx, store.openAdvisoryRoot); err != nil {
		t.Fatal(err)
	}
	run, _ := domain.NewRunID()
	temporary := ".advisory-" + run.String()
	if err := store.advisory.Mkdir(temporary, 0700); err != nil {
		t.Fatal(err)
	}
	lease, err := store.Reserve(ctx, ports.StorageRequest{Operation: ports.StorageOperationFromContext(ctx), Run: run, Kind: "advisory", Temporary: temporary, Target: run.String(), Bytes: 10 << 10, Files: 1, CacheKey: advisoryStorageKey(t, "a")})
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	future := time.Now().Add(48 * time.Hour)
	store.now = func() time.Time { return future }
	if stored, err := store.WriteAdvisory(ctx, advisoryStorageKey(t, "b"), bytes.Repeat([]byte("b"), 10<<10)); err != nil || stored {
		t.Fatal("pressure sweep adopted an active reservation", err)
	}
	if _, err := store.advisory.Lstat(temporary); err != nil || store.pending != 10<<10 || store.cacheCharged != 10<<10 {
		t.Fatal("active allocation or accounting was removed", err)
	}
	// Only the original writer, after removing its exact empty allocation, may
	// abort the reservation. Expiry or descriptor close alone is insufficient.
	if err := store.advisory.Remove(temporary); err != nil {
		t.Fatal(err)
	}
	if err := lease.Abort(ctx); err != nil {
		t.Fatal(err)
	}
}

func advisoryStorageContext(t *testing.T) context.Context {
	t.Helper()
	operation, err := domain.NewOperationID()
	if err != nil {
		t.Fatal(err)
	}
	return ports.WithStorageOperation(context.Background(), operation)
}
func advisoryStorageKey(t *testing.T, value string) domain.ContentDigest {
	t.Helper()
	key, err := domain.NewSHA256Digest(strings.Repeat(value, 64))
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestAdvisoryStoragePersistsOnlySealedBytesAndExpiresWithAccounting(t *testing.T) {
	ctx := advisoryStorageContext(t)
	path := storageTestRoot(t)
	policy := storageFixturePolicy(t, 10)
	store, err := NewLocal(ctx, path, policy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("cache composition was not inert")
	}
	key := advisoryStorageKey(t, "a")
	body := []byte(`{"complete":"opaque public advisory DATA"}`)
	if stored, err := store.WriteAdvisory(ctx, key, body); err != nil || !stored {
		t.Fatal("complete cache write failed", err)
	}
	if got, hit, err := store.ReadAdvisory(ctx, key, 4096); err != nil || !hit || !bytes.Equal(got, body) || store.cacheCharged != uint64(len(body)) {
		t.Fatal("sealed cache read or accounting failed", hit, err)
	}
	if _, hit, err := store.ReadAdvisory(ctx, advisoryStorageKey(t, "b"), 4096); err != nil || hit {
		t.Fatal("other query key reused the response", err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = NewLocal(ctx, path, policy)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if got, hit, err := store.ReadAdvisory(ctx, key, 4096); err != nil || !hit || !bytes.Equal(got, body) {
		t.Fatal("durable cache index did not reopen", err)
	}
	now := time.Now().Add(time.Duration(policy.Spec().AdvisoryTTLSeconds+1) * time.Second)
	store.now = func() time.Time { return now }
	if _, hit, err := store.ReadAdvisory(ctx, key, 4096); err != nil || hit || store.cacheCharged != 0 || store.charged != 0 {
		t.Fatal("expired DATA was reused or remained charged after confirmed removal", hit, err)
	}
	entries, err := os.ReadDir(filepath.Join(path, "advisory-cache"))
	if err != nil || len(entries) != 0 {
		t.Fatal("expired cache payload remained", err)
	}
}

func TestAdvisoryStorageRefusesChangedOrUnknownPayloadBeforeReuseOrDeletion(t *testing.T) {
	for _, mode := range []string{"changed", "missing", "hardlink", "root-replaced"} {
		t.Run(mode, func(t *testing.T) {
			ctx := advisoryStorageContext(t)
			path := storageTestRoot(t)
			store, err := NewLocal(ctx, path, storageFixturePolicy(t, 10))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			key := advisoryStorageKey(t, "a")
			body := []byte(`{"complete":true}`)
			if stored, err := store.WriteAdvisory(ctx, key, body); err != nil || !stored {
				t.Fatal(err)
			}
			run := store.advisoryIndex[key.String()]
			payload := filepath.Join(path, "advisory-cache", run, "response.json")
			charged := store.cacheCharged
			switch mode {
			case "changed":
				if err := os.WriteFile(payload, []byte(`{"complete":false}`), 0600); err != nil {
					t.Fatal(err)
				}
			case "missing":
				if err := os.Remove(payload); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(payload, filepath.Join(t.TempDir(), "foreign.json")); err != nil {
					t.Fatal(err)
				}
			case "root-replaced":
				if err := os.Rename(filepath.Join(path, "advisory-cache"), filepath.Join(path, "original-cache")); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(filepath.Join(path, "advisory-cache"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			if _, hit, err := store.ReadAdvisory(ctx, key, 4096); err == nil || hit {
				t.Fatal("changed cache was reused")
			}
			future := time.Now().Add(48 * time.Hour)
			store.now = func() time.Time { return future }
			if _, hit, err := store.ReadAdvisory(ctx, key, 4096); err == nil || hit || store.cacheCharged != charged {
				t.Fatal("expiry adopted or deleted unknown cache DATA", hit, err)
			}
		})
	}
}

func TestAdvisoryStorageCapacityMissAndCancellationDoNotCreateReusableCopies(t *testing.T) {
	ctx := advisoryStorageContext(t)
	path := storageTestRoot(t)
	policy := storageFixturePolicy(t, 10)
	store, err := NewLocal(ctx, path, policy)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	body := bytes.Repeat([]byte("a"), 10<<10)
	if stored, err := store.WriteAdvisory(ctx, advisoryStorageKey(t, "a"), body); err != nil || !stored {
		t.Fatal(err)
	}
	if stored, err := store.WriteAdvisory(ctx, advisoryStorageKey(t, "b"), bytes.Repeat([]byte("b"), 20<<10)); err != nil || stored {
		t.Fatal("optional capacity miss became an operational failure or exceeded quota", err)
	}
	if store.cacheCharged != uint64(len(body)) || store.pending != 0 {
		t.Fatal("capacity miss leaked a reservation")
	}
	entries, err := os.ReadDir(filepath.Join(path, "advisory-cache"))
	if err != nil || len(entries) != 1 {
		t.Fatal("capacity miss left a temporary copy", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if stored, err := store.WriteAdvisory(cancelled, advisoryStorageKey(t, "c"), []byte("cancelled")); !errors.Is(err, context.Canceled) || stored {
		t.Fatal("cancelled cache write succeeded or lost its cause", err)
	}
	if _, hit, err := store.ReadAdvisory(ctx, advisoryStorageKey(t, "c"), 4096); err != nil || hit {
		t.Fatal("cancelled cache write was reusable", err)
	}
}

func TestAdvisoryStorageReclaimsSupersededAndUnrequestedExpiredDATA(t *testing.T) {
	ctx := advisoryStorageContext(t)
	path := storageTestRoot(t)
	policy := storageFixturePolicy(t, 10)
	store, err := NewLocal(ctx, path, policy)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key := advisoryStorageKey(t, "a")
	for _, body := range [][]byte{[]byte(`{"revision":1}`), []byte(`{"revision":2}`)} {
		if stored, err := store.WriteAdvisory(ctx, key, body); err != nil || !stored {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(filepath.Join(path, "advisory-cache"))
	if err != nil || len(entries) != 1 {
		t.Fatal("superseded response copies accumulated", err)
	}
	future := time.Now().Add(48 * time.Hour)
	store.now = func() time.Time { return future }
	if _, hit, err := store.ReadAdvisory(advisoryStorageContext(t), advisoryStorageKey(t, "b"), 4096); err != nil || hit || store.cacheCharged != 0 {
		t.Fatal("unrequested expired DATA was not swept", hit, err)
	}
}

func TestAdvisoryStoragePressureEvictsOnlyRetainedDATA(t *testing.T) {
	ctx := advisoryStorageContext(t)
	path := storageTestRoot(t)
	store, err := NewLocal(ctx, path, storageFixturePolicy(t, 10))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	body := bytes.Repeat([]byte("a"), 10<<10)
	for _, value := range []string{"a", "b"} {
		if stored, err := store.WriteAdvisory(ctx, advisoryStorageKey(t, value), body); err != nil || !stored {
			t.Fatal("retained DATA could not make bounded cache space", err)
		}
	}
	if _, hit, err := store.ReadAdvisory(ctx, advisoryStorageKey(t, "a"), 20<<10); err != nil || hit {
		t.Fatal("pressure did not retire the prior unreferenced DATA", err)
	}
	if got, hit, err := store.ReadAdvisory(ctx, advisoryStorageKey(t, "b"), 20<<10); err != nil || !hit || !bytes.Equal(got, body) || store.cacheCharged != uint64(len(body)) {
		t.Fatal("pressure lost current DATA or accounting", err)
	}
}

func TestAdvisoryStorageReplaysDeleteIntentWithoutAdoptingUnknownState(t *testing.T) {
	for _, mode := range []string{"body", "empty-directory", "absent", "foreign-member", "replaced-ancestor"} {
		t.Run(mode, func(t *testing.T) {
			ctx := advisoryStorageContext(t)
			path := storageTestRoot(t)
			policy := storageFixturePolicy(t, 10)
			store, err := NewLocal(ctx, path, policy)
			if err != nil {
				t.Fatal(err)
			}
			key := advisoryStorageKey(t, "a")
			body := []byte(`{"complete":true}`)
			if stored, err := store.WriteAdvisory(ctx, key, body); err != nil || !stored {
				t.Fatal(err)
			}
			run := store.advisoryIndex[key.String()]
			directory := filepath.Join(path, "advisory-cache", run)
			if err := store.withLock(ctx, func() error {
				record := store.entries[resourceKey("advisory", run)]
				record.Phase = "DELETING"
				return store.append(journalEvent{"deleting", record})
			}); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "empty-directory", "absent":
				if err := os.Remove(filepath.Join(directory, "response.json")); err != nil {
					t.Fatal(err)
				}
				if mode == "absent" {
					if err := os.Remove(directory); err != nil {
						t.Fatal(err)
					}
				}
			case "foreign-member":
				if err := os.WriteFile(filepath.Join(directory, "foreign.txt"), []byte("preserve"), 0600); err != nil {
					t.Fatal(err)
				}
			case "replaced-ancestor":
				if err := os.Rename(filepath.Join(path, "advisory-cache"), filepath.Join(path, "original-cache")); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(filepath.Join(path, "advisory-cache"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			store, err = NewLocal(ctx, path, policy)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			_, hit, readErr := store.ReadAdvisory(ctx, key, 4096)
			if mode == "foreign-member" || mode == "replaced-ancestor" {
				if readErr == nil || hit || store.cacheCharged != uint64(len(body)) {
					t.Fatal("delete recovery adopted unknown ownership", readErr)
				}
				return
			}
			if readErr != nil || hit || store.cacheCharged != 0 {
				t.Fatal("confirmed delete intent did not recover", readErr, hit, store.cacheCharged)
			}
		})
	}
}
