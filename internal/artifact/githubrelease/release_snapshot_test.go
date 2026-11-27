package githubrelease

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func snapshotFixture() releaseResponse {
	return releaseResponse{ID: 41, TagName: "v1.2.3", Immutable: true, PublishedAt: "2026-08-20T01:02:03Z", Assets: []releaseAssetResponse{
		{ID: 7, Name: "manifest.json", State: "uploaded", Size: 3, Digest: "sha256:" + fmt.Sprintf("%x", sha256.Sum256([]byte("one")))},
		{ID: 8, Name: "tool-linux-amd64", State: "uploaded", Size: 3, Digest: "sha256:" + fmt.Sprintf("%x", sha256.Sum256([]byte("two")))},
	}}
}

func TestReleaseSnapshotSelectsOnceAndAcquiresExactFrozenID(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	var changed atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/releases/assets/8" {
			_, _ = w.Write([]byte("two"))
			return
		}
		if r.URL.Path != "/repos/owner/repo/releases/tags/v1.2.3" {
			http.NotFound(w, r)
			return
		}
		requests.Add(1)
		value := snapshotFixture()
		if changed.Load() {
			value.Assets[1].ID = 99
			value.Assets[1].Digest = "sha256:" + strings.Repeat("f", 64)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(value)
	}))
	defer server.Close()
	api, _ := url.Parse(server.URL)
	client, err := newClient(api, server.Client(), t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := ParseReference("owner/repo@v1.2.3#manifest.json")
	snapshot, err := client.FreezeRelease(context.Background(), reference)
	if err != nil {
		t.Fatal(err)
	}
	changed.Store(true)
	if snapshot.ReleaseID() != 41 || !snapshot.Immutable() {
		t.Fatal("release identity lost")
	}
	names := snapshot.Names()
	if len(names) != 2 {
		t.Fatal(names)
	}
	names[0] = "changed"
	ref, resolved, err := snapshot.ResolveAsset("tool-linux-amd64")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Locator() != "owner/repo@v1.2.3#tool-linux-amd64" || resolved.DeclaredIntegrity() != snapshotFixture().Assets[1].Digest {
		t.Fatal("selection drifted")
	}
	run, _ := domain.NewRunID()
	acquired, err := client.Acquire(context.Background(), run, resolved)
	if err != nil {
		t.Fatal(err)
	}
	if acquired.Digest().String() != strings.TrimPrefix(snapshotFixture().Assets[1].Digest, "sha256:") || requests.Load() != 1 {
		t.Fatalf("re-resolved selection: requests=%d", requests.Load())
	}
	if _, _, err := snapshot.ResolveAsset("changed"); err == nil {
		t.Fatal("caller changed frozen map")
	}
	if _, _, err := (ReleaseSnapshot{}).ResolveAsset("manifest.json"); err == nil {
		t.Fatal("zero snapshot accepted")
	}
}

func TestReleaseSnapshotRejectsIncompleteOrAmbiguousDiscovery(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(*releaseResponse)
	}{
		{"draft", func(r *releaseResponse) { r.Draft = true }},
		{"wrong-tag", func(r *releaseResponse) { r.TagName = "v1.2.4" }},
		{"unpublished", func(r *releaseResponse) { r.PublishedAt = "" }},
		{"duplicate-name", func(r *releaseResponse) { a := r.Assets[1]; a.ID = 9; r.Assets = append(r.Assets, a) }},
		{"duplicate-id", func(r *releaseResponse) { r.Assets[1].ID = r.Assets[0].ID }},
		{"missing-id", func(r *releaseResponse) { r.Assets[1].ID = 0 }},
		{"missing-integrity", func(r *releaseResponse) { r.Assets[1].Digest = "" }},
		{"incomplete-upload", func(r *releaseResponse) { r.Assets[1].State = "new" }},
		{"unsafe-name", func(r *releaseResponse) { r.Assets[1].Name = "../outside" }},
		{"excess-assets", func(r *releaseResponse) {
			for len(r.Assets) <= maximumFrozenAssets {
				a := r.Assets[1]
				a.ID = uint64(100 + len(r.Assets))
				a.Name = fmt.Sprintf("file-%d", a.ID)
				r.Assets = append(r.Assets, a)
			}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			value := snapshotFixture()
			tt.change(&value)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(value)
			}))
			defer server.Close()
			api, _ := url.Parse(server.URL)
			client, err := newClient(api, server.Client(), t.TempDir(), false)
			if err != nil {
				t.Fatal(err)
			}
			reference, _ := ParseReference("owner/repo@v1.2.3#manifest.json")
			snapshot, err := client.FreezeRelease(context.Background(), reference)
			if err == nil || snapshot.ReleaseID() != 0 || len(snapshot.Names()) != 0 {
				t.Fatalf("invalid discovery produced a snapshot: %v", err)
			}
		})
	}
}
