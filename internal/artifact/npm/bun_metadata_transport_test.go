package npm

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestBunBindingRetriesHTTP2HeaderTimeoutOnce(t *testing.T) {
	manifest, err := os.ReadFile("testdata/bun-lock/peer.package.json")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := parseBunTestLock(t, manifest, bunNativeFixture(t, "peer"), true)
	if err != nil {
		t.Fatal(err)
	}
	const target = "/use-sync-external-store/1.2.0"
	metadata := make(map[string]string)
	for _, pkg := range lock.Packages() {
		metadata["/"+pkg.Name+"/"+pkg.Version] = bunMetadataJSON(t, bunMetadata(t, pkg))
	}

	for _, tc := range []struct {
		name     string
		stalls   int32
		status   int
		requests int32
		bound    bool
	}{
		{name: "recovers", stalls: 1, status: http.StatusOK, requests: 2, bound: true},
		{name: "stops after one retry", stalls: 2, status: http.StatusOK, requests: 2},
		{name: "HTTP refusal is not retried", status: http.StatusServiceUnavailable, requests: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path == target {
					if requests.Add(1) <= tc.stalls {
						<-req.Context().Done()
						return
					}
					if tc.status != http.StatusOK {
						w.WriteHeader(tc.status)
						return
					}
				}
				body, present := metadata[req.URL.Path]
				if !present {
					http.NotFound(w, req)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(body))
			}))
			server.EnableHTTP2 = true
			server.StartTLS()
			defer server.Close()

			resolver, err := NewPublicResolver(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			transport := resolver.client.Transport.(*http.Transport)
			defer transport.CloseIdleConnections()
			transport.ResponseHeaderTimeout = 200 * time.Millisecond
			transport.TLSClientConfig = server.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
			transport.TLSClientConfig.ServerName = "127.0.0.1"
			addr := server.Listener.Addr().String()
			transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, addr)
			}

			selected, err := resolver.BindBunLock(context.Background(), lock)
			if requests.Load() != tc.requests {
				t.Errorf("target requests = %d, want %d (binding error: %v)", requests.Load(), tc.requests, err)
			}
			if tc.bound {
				if err != nil || !selected.SourceBound() || len(selected.Artifacts()) != len(metadata) {
					t.Fatalf("binding = %v, artifacts = %d, error = %v", selected.SourceBound(), len(selected.Artifacts()), err)
				}
				return
			}
			if !errors.Is(err, errBunBinding) || selected.SourceBound() || len(selected.Artifacts()) != 0 {
				t.Fatalf("rejected binding = %v, artifacts = %d, error = %v", selected.SourceBound(), len(selected.Artifacts()), err)
			}
		})
	}
}
