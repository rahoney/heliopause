package npm

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

func TestResolveKeepsMissingIntegrityForVerification(t *testing.T) {
	t.Parallel()

	const registryURL = "https://registry.test/"
	resolver := newTestResolver(t, registryURL, roundTripFunc(func(*http.Request) (*http.Response, error) {
		body := strings.Replace(metadataJSON(registryURL, "tiny", "1.2.3", "latest"), `"integrity":"sha512-c2FmZQ=="`, `"integrity":""`, 1)
		return response(http.StatusOK, "application/json", body), nil
	}))
	resolved, err := resolver.Resolve(context.Background(), mustReference(t, "tiny"))
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.DeclaredIntegrity() != "" {
		t.Fatalf("DeclaredIntegrity() = %q", resolved.DeclaredIntegrity())
	}
}

func TestAcquireStreamsOnlyToRunLocalIntake(t *testing.T) {
	t.Parallel()

	const registryURL = "https://registry.test/"
	intakeRoot := canonicalTempDir(t)
	resolver := newTestResolver(t, registryURL, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.EscapedPath() {
		case "/tiny":
			return response(http.StatusOK, "application/json", metadataJSON(registryURL, "tiny", "1.2.3", "latest")), nil
		case "/tiny/-/tiny-1.2.3.tgz":
			return response(http.StatusOK, "application/octet-stream", "controlled tarball bytes"), nil
		default:
			return response(http.StatusNotFound, "text/plain", "missing"), nil
		}
	}))
	resolver.intakeRoot = intakeRoot
	resolved, err := resolver.Resolve(context.Background(), mustReference(t, "tiny"))
	if err != nil {
		t.Fatal(err)
	}
	runID := mustRunID(t)
	artifact, err := resolver.Acquire(context.Background(), runID, resolved)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if artifact.ContentHandle() != "intake:"+runID.String()+":tarball" || artifact.SizeBytes() != uint64(len("controlled tarball bytes")) || artifact.Digest().String() != "72a283223529044ee71f5518bb1b26ad5b0fb55e7a81549c65e895ad873eb2b1" {
		t.Fatalf("artifact = %#v", artifact)
	}
	if integrity, ok := artifact.DeclaredIntegrity(); !ok || integrity != "sha512-c2FmZQ==" {
		t.Fatalf("DeclaredIntegrity() = %q, %v", integrity, ok)
	}
	wantSHA512 := sha512.Sum512([]byte("controlled tarball bytes"))
	if integrity, ok := artifact.ObservedIntegrity(); !ok || integrity != "sha512-"+base64.StdEncoding.EncodeToString(wantSHA512[:]) {
		t.Fatalf("ObservedIntegrity() = %q, %v", integrity, ok)
	}
	info, err := os.Stat(filepath.Join(intakeRoot, runID.String(), "tarball.tgz"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("intake tarball info = %#v, error = %v", info, err)
	}
}

func TestAcquireRejectsUnsafeTarballAndRemovesPartialIntake(t *testing.T) {
	t.Parallel()

	const registryURL = "https://registry.test/"
	intakeRoot := canonicalTempDir(t)
	resolver := newTestResolver(t, registryURL, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.EscapedPath() {
		case "/tiny":
			return response(http.StatusOK, "application/json", metadataJSON(registryURL, "tiny", "1.2.3", "latest")), nil
		case "/tiny/-/tiny-1.2.3.tgz":
			return response(http.StatusOK, "text/plain", "not a tarball"), nil
		default:
			return response(http.StatusNotFound, "text/plain", "missing"), nil
		}
	}))
	resolver.intakeRoot = intakeRoot
	resolved, err := resolver.Resolve(context.Background(), mustReference(t, "tiny"))
	if err != nil {
		t.Fatal(err)
	}
	runID := mustRunID(t)
	if _, err := resolver.Acquire(context.Background(), runID, resolved); err == nil {
		t.Fatal("Acquire() error = nil")
	}
	if _, err := os.Stat(filepath.Join(intakeRoot, runID.String())); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial intake directory stat error = %v", err)
	}
}

func TestAcquireEnforcesTarballLimitAndRemovesPartialIntake(t *testing.T) {
	t.Parallel()

	const registryURL = "https://registry.test/"
	intakeRoot := canonicalTempDir(t)
	resolver := newTestResolver(t, registryURL, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.EscapedPath() {
		case "/tiny":
			return response(http.StatusOK, "application/json", metadataJSON(registryURL, "tiny", "1.2.3", "latest")), nil
		case "/tiny/-/tiny-1.2.3.tgz":
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/octet-stream"}}, Body: io.NopCloser(&repeatedReader{remaining: tarballLimit + 1})}, nil
		default:
			return response(http.StatusNotFound, "text/plain", "missing"), nil
		}
	}))
	resolver.intakeRoot = intakeRoot
	resolved, err := resolver.Resolve(context.Background(), mustReference(t, "tiny"))
	if err != nil {
		t.Fatal(err)
	}
	runID := mustRunID(t)
	if _, err := resolver.Acquire(context.Background(), runID, resolved); err == nil {
		t.Fatal("Acquire() error = nil")
	}
	if _, err := os.Stat(filepath.Join(intakeRoot, runID.String())); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial intake directory stat error = %v", err)
	}
}

func TestCreateRunDirectoryRejectsSymlinkAncestor(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	target := t.TempDir()
	if err := os.Symlink(target, filepath.Join(parent, "linked")); err != nil {
		t.Fatal(err)
	}
	if _, err := createRunDirectory(filepath.Join(parent, "linked", "intake"), mustRunID(t).String()); err == nil {
		t.Fatal("createRunDirectory() error = nil")
	}
}

func newTestResolver(t *testing.T, rawURL string, transport http.RoundTripper) *Resolver {
	t.Helper()
	registry, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := newResolver(registry, &http.Client{Transport: transport}, t.TempDir(), false)
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

func mustRunID(t *testing.T) domain.RunID {
	t.Helper()
	id, err := domain.ParseRunID("run_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func canonicalTempDir(t *testing.T) string {
	t.Helper()
	path, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return path
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

type repeatedReader struct{ remaining int64 }

func (r *repeatedReader) Read(buffer []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	count := int64(len(buffer))
	if count > r.remaining {
		count = r.remaining
	}
	for index := range buffer[:int(count)] {
		buffer[index] = 'x'
	}
	r.remaining -= count
	return int(count), nil
}
