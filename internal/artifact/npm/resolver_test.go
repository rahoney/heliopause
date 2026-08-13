package npm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestParseReference(t *testing.T) {
	t.Parallel()

	valid := []string{"tiny", "tiny@1.2.3", "tiny@latest", "@scope/tiny", "@scope/tiny@next"}
	for _, locator := range valid {
		locator := locator
		t.Run(locator, func(t *testing.T) {
			t.Parallel()
			reference, err := ParseReference(locator)
			if err != nil || reference.Source().String() != "npm" || reference.Locator() != locator {
				t.Fatalf("ParseReference() = %#v, %v", reference, err)
			}
		})
	}
	invalid := []string{"", " package", "tiny@^1.2.3", "tiny@*", "git+https://example.test/x", "https://example.test/x", "@scope", "@scope//tiny"}
	for _, locator := range invalid {
		locator := locator
		t.Run("invalid "+locator, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseReference(locator); err == nil {
				t.Fatal("ParseReference() error = nil")
			}
		})
	}
}

func TestResolveExactVersionAndTag(t *testing.T) {
	t.Parallel()

	const registryURL = "https://registry.test/"
	resolver := newTestResolver(t, registryURL, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Accept") != metadataAccept {
			t.Errorf("Accept = %q", request.Header.Get("Accept"))
		}
		if request.URL.EscapedPath() != "/@scope%2Ftiny" {
			t.Errorf("path = %q", request.URL.EscapedPath())
		}
		return response(http.StatusOK, "application/json; charset=utf-8", metadataJSON(registryURL, "@scope/tiny", "1.2.3", "latest")), nil
	}))
	for _, locator := range []string{"@scope/tiny", "@scope/tiny@latest", "@scope/tiny@1.2.3"} {
		t.Run(locator, func(t *testing.T) {
			reference := mustReference(t, locator)
			resolved, err := resolver.Resolve(context.Background(), reference)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if got := resolved.Identity(); got.Name() != "@scope/tiny" || got.Version() != "1.2.3" || got.Variant() != "tarball" {
				t.Fatalf("Identity() = %#v", got)
			}
			if resolved.AcquisitionLocator() != strings.TrimSuffix(registryURL, "/")+"/@scope%2Ftiny/-/tiny-1.2.3.tgz" || resolved.DeclaredIntegrity() != "sha512-c2FmZQ==" {
				t.Fatalf("resolved target = %#v", resolved)
			}
		})
	}
}

func TestResolveRejectsUnsafeOrMalformedMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		content string
		body    string
	}{
		{name: "redirect", status: http.StatusFound, content: "application/json", body: "{}"},
		{name: "non JSON", status: http.StatusOK, content: "text/plain", body: "{}"},
		{name: "bad JSON", status: http.StatusOK, content: "application/json", body: "{"},
		{name: "missing selected version", status: http.StatusOK, content: "application/json", body: `{"name":"tiny","dist-tags":{"latest":"1.2.3"},"versions":{}}`},
		{name: "wrong tarball host", status: http.StatusOK, content: "application/json", body: metadataJSON("https://elsewhere.test", "tiny", "1.2.3", "latest")},
		{name: "oversized", status: http.StatusOK, content: "application/json", body: strings.Repeat("x", metadataLimit+1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			resolver := newTestResolver(t, "https://registry.test/", roundTripFunc(func(*http.Request) (*http.Response, error) {
				return response(test.status, test.content, test.body), nil
			}))
			if _, err := resolver.Resolve(context.Background(), mustReference(t, "tiny")); err == nil {
				t.Fatal("Resolve() error = nil")
			}
		})
	}
}

func TestResolvePreservesContextCancellation(t *testing.T) {
	t.Parallel()

	resolver := newTestResolver(t, "https://registry.test/", roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("transport must not be called")
	}))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := resolver.Resolve(ctx, mustReference(t, "tiny"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Resolve() error = %v", err)
	}
}

func newTestResolver(t *testing.T, rawURL string, transport http.RoundTripper) *Resolver {
	t.Helper()
	registry, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := newResolver(registry, &http.Client{Transport: transport}, false)
	if err != nil {
		t.Fatal(err)
	}
	return resolver
}

func mustReference(t *testing.T, locator string) domain.ArtifactReference {
	t.Helper()
	reference, err := ParseReference(locator)
	if err != nil {
		t.Fatal(err)
	}
	return reference
}

func metadataJSON(registryURL, name, version, tag string) string {
	return fmt.Sprintf(`{"name":%q,"dist-tags":{%q:%q},"versions":{%q:{"name":%q,"version":%q,"dist":{"tarball":%q,"integrity":"sha512-c2FmZQ=="}}}}`, name, tag, version, version, name, version, strings.TrimSuffix(registryURL, "/")+"/"+url.PathEscape(name)+"/-/tiny-"+version+".tgz")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func response(status int, contentType, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{contentType}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
