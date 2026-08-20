package githubrelease

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestClientResolveAndAcquireBindsAPIAsset(t *testing.T) {
	t.Parallel()
	content := []byte("verified GitHub Release bytes")
	digest := fmt.Sprintf("%x", sha256.Sum256(content))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/repos/owner/repo/releases/tags/v1.2.3":
			if request.Header.Get("Accept") != "application/vnd.github+json" || request.Header.Get("Authorization") != "" {
				t.Errorf("unexpected release request headers: %#v", request.Header)
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(writer, `{"id":41,"tag_name":"v1.2.3","draft":false,"published_at":"2026-08-20T01:02:03Z","assets":[{"id":7,"name":"tool-linux-amd64","state":"uploaded","size":%d,"digest":"sha256:%s"}]}`, len(content), digest)
		case "/repos/owner/repo/releases/assets/7":
			if request.Header.Get("Accept") != "application/octet-stream" || request.Header.Get("Authorization") != "" {
				t.Errorf("unexpected asset request headers: %#v", request.Header)
			}
			writer.Header().Set("Content-Length", fmt.Sprint(len(content)))
			_, _ = writer.Write(content)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	api, _ := url.Parse(server.URL)
	client, err := newClient(api, server.Client(), t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := ParseReference("owner/repo@v1.2.3#tool-linux-amd64")
	resolved, err := client.Resolve(context.Background(), reference)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.DeclaredIntegrity() != "sha256:"+digest || resolved.Identity().Name() != "owner-repo" {
		t.Fatalf("resolved = %#v", resolved)
	}
	runID, _ := domain.NewRunID()
	acquired, err := client.Acquire(context.Background(), runID, resolved)
	if err != nil {
		t.Fatal(err)
	}
	if acquired.Digest().String() != digest || acquired.SizeBytes() != uint64(len(content)) {
		t.Fatalf("acquired = %#v", acquired)
	}
}

func TestClientAcquireFailsClosedOnDigestOrSizeMismatch(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Length", "3")
		_, _ = writer.Write([]byte("bad"))
	}))
	defer server.Close()
	api, _ := url.Parse(server.URL)
	root := t.TempDir()
	client, _ := newClient(api, server.Client(), root, false)
	selector, _ := ParseSelector("owner/repo@v1#asset")
	digest, _ := domain.NewSHA256Digest("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	asset := ResolvedAsset{releaseID: 1, assetID: 2, selector: selector, digest: digest, size: 3}
	identity, _ := domain.NewResolvedArtifactIdentity(mustSource(t), "owner-repo", "v1", "asset")
	resolved, _ := domain.NewResolvedArtifact(identity, assetLocator(asset), "sha256:"+digest.String())
	runID, _ := domain.NewRunID()
	if _, err := client.Acquire(context.Background(), runID, resolved); err == nil {
		t.Fatal("Acquire() error = nil")
	}
}

func mustSource(t *testing.T) domain.SourceID {
	t.Helper()
	source, err := domain.NewSourceID(SourceName)
	if err != nil {
		t.Fatal(err)
	}
	return source
}
