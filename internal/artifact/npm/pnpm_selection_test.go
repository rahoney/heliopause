package npm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func pnpmMetadataBody(t *testing.T, pkg PNPMLockedPackage, tarball string) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"name": pkg.Name, "version": pkg.Version,
		"dist": map[string]string{"integrity": pkg.Integrity, "tarball": tarball},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestPNPMBindingUsesIndependentRegistryAndPreservesPeerWiring(t *testing.T) {
	lock, err := parsePNPMTestLock([]byte(pnpmPeerManifest), pnpmFixture(t, "peer"))
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]PNPMLockedPackage{}
	for _, pkg := range lock.Packages() {
		byName[pkg.Name] = pkg
	}
	calls := 0
	resolver := newTestResolver(t, publicRegistryURL, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		pkg, ok := byName[strings.TrimPrefix(request.URL.Path[:strings.LastIndexByte(request.URL.Path, '/')], "/")]
		if !ok || request.URL.Host != "registry.npmjs.org" || request.URL.Scheme != "https" {
			t.Fatal("binding queried a native-selected origin")
		}
		if !strings.HasSuffix(request.URL.Path, "/"+pkg.Version) {
			t.Fatal("binding queried the complete package history")
		}
		// Deliberately non-derived metadata URL: binding must retain this exact
		// source locator instead of assembling a conventional name-version URL.
		return response(http.StatusOK, "application/json", pnpmMetadataBody(t, pkg, publicRegistryURL+"selected/"+pkg.Name+".tgz")), nil
	}))
	selected, err := resolver.BindPNPMLock(context.Background(), lock)
	if err != nil || !selected.SourceBound() || calls != 4 || len(selected.Artifacts()) != 4 || len(selected.Lock().Snapshots()) != 4 {
		t.Fatal("source binding or wiring lost", err)
	}
	if selected.Artifacts()[0].AcquisitionLocator() != publicRegistryURL+"selected/js-tokens.tgz" {
		t.Fatal("independent locator was reconstructed")
	}
	copy := selected.Artifacts()
	copy[0] = copy[len(copy)-1]
	if selected.Artifacts()[0] == copy[0] {
		t.Fatal("artifact getter exposed mutable storage")
	}

	empty, err := parsePNPMTestLock([]byte(`{}`), []byte(pnpmEmptyLock))
	if err != nil {
		t.Fatal(err)
	}
	selected, err = resolver.BindPNPMLock(context.Background(), empty)
	if err != nil || !selected.SourceBound() || !selected.Lock().DependencyFree() || len(selected.Artifacts()) != 0 || calls != 4 {
		t.Fatal("explicit empty selection changed", err)
	}
	if (PNPMSelection{}).SourceBound() {
		t.Fatal("zero value claimed source binding")
	}
}

func TestPNPMBindingRejectsMetadataDriftAndReturnsNoPartialSelection(t *testing.T) {
	lock, err := parsePNPMTestLock([]byte(pnpmSingleManifest), pnpmFixture(t, "single"))
	if err != nil {
		t.Fatal(err)
	}
	pkg := lock.Packages()[0]
	for _, kind := range []string{"name", "version", "integrity", "foreign-url", "missing", "cancel-after-response", "explicit-url"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			candidate := lock
			if kind == "explicit-url" {
				body := strings.Replace(string(pnpmFixture(t, "single")), "resolution: {integrity:", "resolution: {tarball: 'https://registry.npmjs.org/other.tgz', integrity:", 1)
				candidate, err = parsePNPMTestLock([]byte(pnpmSingleManifest), []byte(body))
				if err != nil {
					t.Fatal(err)
				}
			}
			resolver := newTestResolver(t, publicRegistryURL, roundTripFunc(func(*http.Request) (*http.Response, error) {
				value := pkg
				url := publicRegistryURL + "selected.tgz"
				switch kind {
				case "name":
					value.Name = "other"
				case "version":
					value.Version = "7.0.0"
				case "integrity":
					value.Integrity = strings.Replace(pkg.Integrity, "Wu1", "Au1", 1)
				case "foreign-url":
					url = "https://foreign.invalid/package.tgz"
				case "missing":
					return response(http.StatusNotFound, "application/json", `{}`), nil
				case "cancel-after-response":
					cancel()
				}
				return response(http.StatusOK, "application/json", pnpmMetadataBody(t, value, url)), nil
			}))
			selected, err := resolver.BindPNPMLock(ctx, candidate)
			if err == nil || selected.SourceBound() || len(selected.Artifacts()) != 0 || len(selected.Lock().Snapshots()) != 0 {
				t.Fatal("failed metadata emitted a partial selection")
			}
			if kind == "cancel-after-response" && !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation cause lost", err)
			}
		})
	}
	for _, origin := range []string{publicRegistryURL, "https://foreign.invalid/", "http://registry.npmjs.org/"} {
		calls := 0
		resolver := newTestResolver(t, origin, roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			t.Fatal("rejected input made a request")
			return nil, errors.New("unexpected")
		}))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := resolver.BindPNPMLock(ctx, lock); err == nil {
			t.Fatal("cancelled/foreign binding accepted")
		}
		if _, err := resolver.BindPNPMLock(context.Background(), PNPMLock{}); err == nil {
			t.Fatal("unparsed lock accepted")
		}
		if origin != publicRegistryURL {
			if _, err := resolver.BindPNPMLock(context.Background(), lock); err == nil {
				t.Fatal("foreign source bound a valid lock")
			}
		}
		if calls != 0 {
			t.Fatal("rejected binding contacted registry")
		}
	}
}

func TestPNPMBindingDiscardsEarlierBindingsAfterLaterDrift(t *testing.T) {
	lock, err := parsePNPMTestLock([]byte(pnpmPeerManifest), pnpmFixture(t, "peer"))
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]PNPMLockedPackage{}
	for _, pkg := range lock.Packages() {
		byName[pkg.Name] = pkg
	}
	calls := 0
	resolver := newTestResolver(t, publicRegistryURL, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		pkg := byName[strings.TrimPrefix(request.URL.Path[:strings.LastIndexByte(request.URL.Path, '/')], "/")]
		if calls == 2 {
			pkg.Integrity = "sha512-" + strings.Repeat("A", 86) + "=="
		}
		return response(http.StatusOK, "application/json", pnpmMetadataBody(t, pkg, publicRegistryURL+"selected.tgz")), nil
	}))
	selected, err := resolver.BindPNPMLock(context.Background(), lock)
	if err == nil || calls != 2 || selected.SourceBound() || len(selected.Artifacts()) != 0 || len(selected.Lock().Roots()) != 0 {
		t.Fatal("later drift leaked earlier binding", err)
	}
}

func TestPNPMExactVersionMetadataKeepsScopedEscapingAndBounds(t *testing.T) {
	lock, err := parsePNPMTestLock([]byte(pnpmSingleManifest), pnpmFixture(t, "single"))
	if err != nil {
		t.Fatal(err)
	}
	pkg := lock.Packages()[0]
	pkg.Name, pkg.Version = "@scope/tiny", "1.2.3"
	calls := 0
	resolver := newTestResolver(t, publicRegistryURL, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.URL.EscapedPath() != "/@scope%2Ftiny/1.2.3" || request.Method != http.MethodGet || request.Header.Get("Accept") != "application/json" {
			t.Fatal("exact metadata endpoint changed")
		}
		return response(http.StatusOK, "application/json", pnpmMetadataBody(t, pkg, publicRegistryURL+"tiny.tgz")), nil
	}))
	artifact, err := resolver.resolvePNPMVersion(context.Background(), mustReference(t, "@scope/tiny@1.2.3"))
	if err != nil || artifact.Identity().Name() != pkg.Name || calls != 1 {
		t.Fatal("scoped exact metadata failed", err)
	}
	if _, err := resolver.resolvePNPMVersion(context.Background(), mustReference(t, "@scope/tiny@latest")); err == nil || calls != 1 {
		t.Fatal("mutable tag reached exact endpoint")
	}
	resolver.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(http.StatusOK, "application/json", strings.Repeat("x", metadataLimit+1)), nil
	})
	if _, err := resolver.resolvePNPMVersion(context.Background(), mustReference(t, "@scope/tiny@1.2.3")); err == nil || !strings.Contains(err.Error(), "exceeds configured limit") {
		t.Fatal("exact endpoint bypassed ordinary metadata bound", err)
	}
}
