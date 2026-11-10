package githubrelease

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauspost/compress/snappy"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
)

type attestationRoundTripper func(*http.Request) (*http.Response, error)

func (f attestationRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func attestationHTTPResponse(body []byte, kind string, status int) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{kind}}, Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body))}
}

func TestAttestationTransportErrorsDoNotExposeSignedURL(t *testing.T) {
	client, err := NewPublicClient(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient.Transport = attestationRoundTripper(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "api.github.com" {
			return attestationHTTPResponse([]byte(`{"attestations":[{"repository_id":1331904030,"bundle_url":"https://tmaproduction.blob.core.windows.net/attestations/1331904030/fixture.json.sn?sig=fixture"}]}`), "application/json", 200), nil
		}
		return nil, errors.New("transport failure at " + r.URL.String())
	})
	selector, _ := ParseSelector("rahoney/heliopause@v0.1.0#manifest.json")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	resolved, err := resolvedArtifactForAsset(mustSource(t), ResolvedAsset{selector: selector, releaseID: 41, assetID: 7, digest: digest, size: 3})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.readAttestations(context.Background(), resolved); err == nil || strings.Contains(err.Error(), "sig=") || strings.Contains(err.Error(), "https://") {
		t.Fatalf("unsafe transport diagnostic: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.readAttestations(ctx, resolved); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel cause lost: %v", err)
	}
}

func TestPublicAttestationReadForCommittedSubject(t *testing.T) {
	if os.Getenv("HELOX_PUBLIC_RELEASE_ATTESTATION_TEST") != "1" {
		t.Skip("explicit public-network attestation qualification")
	}
	_, manifest, digest := attestationFixture(t)
	client, err := NewPublicClient(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	selector, err := ParseSelector("rahoney/heliopause@v0.1.0#helox-release-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	subject, err := domain.NewSHA256Digest(digest)
	if err != nil {
		t.Fatal(err)
	}
	// Only the committed signed subject is exercised here. Discovery IDs are
	// deliberately not used by the attestation endpoint; this does not claim
	// that v0.1.0 has a public GitHub Release or qualify payload activation.
	resolved, err := resolvedArtifactForAsset(mustSource(t), ResolvedAsset{selector: selector, releaseID: 1, assetID: 1, digest: subject, size: uint64(len(manifest))})
	if err != nil {
		t.Fatal(err)
	}
	bundles, err := client.readAttestations(context.Background(), resolved)
	if err != nil {
		t.Fatal(err)
	}
	identity := releaseprovenance.Identity{Version: "v0.1.0", SourceCommit: "ef5ef0e0e0437a9083133674ef8b9e6d693b83a9", WorkflowRun: "39036210335"}
	valid := 0
	for _, body := range bundles {
		if _, err := releaseprovenance.Verify(context.Background(), body, identity, []releaseprovenance.Subject{{Name: "helox-release-manifest.json", SHA256: digest}}); err == nil {
			valid++
		}
	}
	if valid == 0 {
		t.Fatal("public acquisition returned no authenticated official subject")
	}
	t.Logf("official attestation verified: bundles=%d valid=%d; no credentials or payload activation", len(bundles), valid)
}

func attestationFixture(t *testing.T) ([]byte, []byte, string) {
	t.Helper()
	directory := filepath.Join("..", "..", "..", "docs", "planning", "evidence", "m13-003-release-build", "v0.1.0")
	bundle, err := os.ReadFile(filepath.Join(directory, "actual-signed-bundle.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(directory, "helox-release-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(manifest)
	return bundle, manifest, hex.EncodeToString(sum[:])
}

func TestFrozenAttestationAcquisitionDecodesAndVerifiesActualSignedFixture(t *testing.T) {
	bundle, manifest, digest := attestationFixture(t)
	const location = "https://tmaproduction.blob.core.windows.net/attestations/1331904030/2026/10/17/fixture.json.sn?sig=fixture"
	calls := make(map[string]int)
	client, err := NewPublicClient(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient.Transport = attestationRoundTripper(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("Proxy-Authorization") != "" {
			t.Fatal("credentials reached public acquisition")
		}
		calls[r.URL.Host+r.URL.Path]++
		switch r.URL.Host + r.URL.Path {
		case "api.github.com/repos/rahoney/heliopause/releases/tags/v0.1.0":
			value := releaseResponse{ID: 41, TagName: "v0.1.0", Immutable: true, PublishedAt: "2026-10-17T13:29:00Z", Assets: []releaseAssetResponse{{ID: 7, Name: "helox-release-manifest.json", State: "uploaded", Size: int64(len(manifest)), Digest: "sha256:" + digest}}}
			body, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			return attestationHTTPResponse(body, "application/json", 200), nil
		case "api.github.com/repos/rahoney/heliopause/attestations/sha256:" + digest:
			if r.URL.Query().Get("per_page") != "16" || r.Header.Get("X-GitHub-Api-Version") != APIVersion {
				t.Fatal("unbounded/unversioned attestation query")
			}
			body, err := json.Marshal(map[string]any{"attestations": []any{map[string]any{"repository_id": 1331904030, "bundle_url": location}}})
			if err != nil {
				t.Fatal(err)
			}
			return attestationHTTPResponse(body, "application/json", 200), nil
		case "tmaproduction.blob.core.windows.net/attestations/1331904030/2026/10/17/fixture.json.sn":
			return attestationHTTPResponse(snappy.Encode(nil, bundle), "application/octet-stream", 200), nil
		default:
			return nil, errors.New("unexpected acquisition request")
		}
	})
	reference, _ := ParseReference("rahoney/heliopause@v0.1.0#helox-release-manifest.json")
	snapshot, err := client.FreezeRelease(context.Background(), reference)
	if err != nil {
		t.Fatal(err)
	}
	proofs, err := snapshot.Attestations(context.Background(), "helox-release-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(proofs) != 1 || !bytes.Equal(proofs[0], bundle) || len(calls) != 3 {
		t.Fatal("wrong proof or acquisition sequence")
	}
	for _, count := range calls {
		if count != 1 {
			t.Fatal("acquisition repeated a request")
		}
	}
	identity := releaseprovenance.Identity{Version: "v0.1.0", SourceCommit: "ef5ef0e0e0437a9083133674ef8b9e6d693b83a9", WorkflowRun: "39036210335"}
	if _, err := releaseprovenance.Verify(context.Background(), proofs[0], identity, []releaseprovenance.Subject{{Name: "helox-release-manifest.json", SHA256: digest}}); err != nil {
		t.Fatal(err)
	}
}

func TestInlineAttestationsRemainBounded(t *testing.T) {
	bundle, manifest, digest := attestationFixture(t)
	for _, kind := range []string{"valid", "excess-rows", "excess-inline"} {
		t.Run(kind, func(t *testing.T) {
			row := map[string]any{"repository_id": 1331904030, "bundle": json.RawMessage(bundle)}
			rows := []any{row}
			if kind == "excess-rows" {
				for len(rows) <= maximumAttestations {
					rows = append(rows, row)
				}
			}
			if kind == "excess-inline" {
				row["bundle"] = map[string]string{"padding": strings.Repeat("x", maximumAttestationBundle)}
			}
			body, err := json.Marshal(map[string]any{"attestations": rows})
			if err != nil {
				t.Fatal(err)
			}
			client, err := NewPublicClient(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			requests := 0
			client.httpClient.Transport = attestationRoundTripper(func(r *http.Request) (*http.Response, error) {
				requests++
				if r.URL.Host != "api.github.com" {
					t.Fatal("inline bundle triggered blob request")
				}
				return attestationHTTPResponse(body, "application/json", 200), nil
			})
			selector, _ := ParseSelector("rahoney/heliopause@v0.1.0#helox-release-manifest.json")
			d, _ := domain.NewSHA256Digest(digest)
			resolved, err := resolvedArtifactForAsset(mustSource(t), ResolvedAsset{selector: selector, releaseID: 41, assetID: 7, digest: d, size: uint64(len(manifest))})
			if err != nil {
				t.Fatal(err)
			}
			proofs, err := client.readAttestations(context.Background(), resolved)
			if kind != "valid" {
				if err == nil || len(proofs) != 0 {
					t.Fatal("unbounded inline proof accepted")
				}
				return
			}
			if err != nil || len(proofs) != 1 || requests != 1 {
				t.Fatalf("inline acquisition failed: %v", err)
			}
			identity := releaseprovenance.Identity{Version: "v0.1.0", SourceCommit: "ef5ef0e0e0437a9083133674ef8b9e6d693b83a9", WorkflowRun: "39036210335"}
			if _, err := releaseprovenance.Verify(context.Background(), proofs[0], identity, []releaseprovenance.Subject{{Name: "helox-release-manifest.json", SHA256: digest}}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAttestationAcquisitionRejectsUnboundedOrRedirectedInput(t *testing.T) {
	bundle, _, _ := attestationFixture(t)
	const validURL = "https://tmaproduction.blob.core.windows.net/attestations/1331904030/2026/10/17/fixture.json.sn?sig=fixture"
	for _, tt := range []struct {
		name, url   string
		blob        []byte
		apiOverride []byte
		status      int
		link        string
	}{
		{name: "wrong-host", url: "https://example.invalid/attestations/1331904030/fixture.json.sn?sig=fixture"},
		{name: "cleartext", url: strings.Replace(validURL, "https:", "http:", 1)},
		{name: "explicit-port", url: strings.Replace(validURL, ".net/", ".net:443/", 1)},
		{name: "userinfo", url: strings.Replace(validURL, "https://", "https://fixture@", 1)},
		{name: "fragment", url: validURL + "#fragment"},
		{name: "wrong-repository-path", url: strings.Replace(validURL, "/1331904030/", "/2/", 1)},
		{name: "path-traversal", url: strings.Replace(validURL, "/2026/", "/../2026/", 1)},
		{name: "encoded-path", url: strings.Replace(validURL, "/2026/", "/%32%30%32%36/", 1)},
		{name: "corrupt-block", url: validURL, blob: []byte{0xff}},
		{name: "size-bomb", url: validURL, blob: binary.AppendUvarint(nil, uint64(maximumAttestationBundle+1))},
		{name: "empty-json", url: validURL, blob: snappy.Encode(nil, []byte(" "))},
		{name: "invalid-json", url: validURL, blob: snappy.Encode(nil, []byte("not-json"))},
		{name: "blob-redirect", url: validURL, blob: snappy.Encode(nil, bundle), status: 302},
		{name: "api-oversize", apiOverride: bytes.Repeat([]byte(" "), maximumAttestationResponse+1)},
		{name: "trailing-json", apiOverride: []byte(`{"attestations":[]} {}`)},
		{name: "missing-repository-id", apiOverride: []byte(`{"attestations":[{"repository_id":0,"bundle":{}}]}`)},
		{name: "paginated-api", link: `<https://example.invalid/next>; rel="next"`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewPublicClient(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			requests := 0
			client.httpClient.Transport = attestationRoundTripper(func(r *http.Request) (*http.Response, error) {
				requests++
				if r.URL.Host == "api.github.com" {
					body := tt.apiOverride
					if body == nil {
						body, _ = json.Marshal(map[string]any{"attestations": []any{map[string]any{"repository_id": 1331904030, "bundle_url": tt.url}}})
					}
					response := attestationHTTPResponse(body, "application/json", 200)
					if tt.link != "" {
						response.Header.Set("Link", tt.link)
					}
					return response, nil
				}
				if r.URL.Host != attestationBlobHost {
					t.Fatal("untrusted host was requested")
				}
				status := tt.status
				if status == 0 {
					status = 200
				}
				response := attestationHTTPResponse(tt.blob, "application/octet-stream", status)
				if status == 302 {
					response.Header.Set("Location", "https://example.invalid/stolen?sig=fixture")
				}
				return response, nil
			})
			asset := snapshotFixture().Assets[0]
			selector, _ := ParseSelector("rahoney/heliopause@v0.1.0#manifest.json")
			d, _ := domain.NewSHA256Digest(strings.TrimPrefix(asset.Digest, "sha256:"))
			resolved, err := resolvedArtifactForAsset(mustSource(t), ResolvedAsset{selector: selector, releaseID: 41, assetID: 7, digest: d, size: 3})
			if err != nil {
				t.Fatal(err)
			}
			result, err := client.readAttestations(context.Background(), resolved)
			if err == nil || len(result) != 0 {
				t.Fatal("untrusted proof accepted")
			}
			if strings.Contains(err.Error(), "sig=") || strings.Contains(err.Error(), "https://") {
				t.Fatalf("remote URL leaked in error: %v", err)
			}
			if requests > 2 {
				t.Fatal("redirect followed")
			}
		})
	}
}
