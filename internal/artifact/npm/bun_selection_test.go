package npm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func bunMetadata(t *testing.T, pkg BunLockedPackage) map[string]any {
	t.Helper()
	fields := map[string]any{"name": pkg.Name, "version": pkg.Version, "dist": map[string]any{"integrity": pkg.Integrity, "tarball": publicRegistryURL + "independent/" + strings.ReplaceAll(pkg.Name, "/", "-") + "-" + pkg.Version + ".tgz"}}
	if len(pkg.Dependencies) != 0 {
		fields["dependencies"] = pkg.Dependencies
	}
	if len(pkg.PeerDependencies) != 0 {
		fields["peerDependencies"] = pkg.PeerDependencies
	}
	if len(pkg.Bins) != 0 {
		fields["bin"] = pkg.Bins
	}
	return fields
}

func bunMetadataJSON(t *testing.T, fields map[string]any) string {
	t.Helper()
	body, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func bunTestSelection(t *testing.T, manifest, body []byte, present bool) (BunSelection, []domain.ProjectControlFile) {
	t.Helper()
	lock, err := parseBunTestLock(t, manifest, body, present)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]BunLockedPackage{}
	for _, pkg := range lock.Packages() {
		byPath["/"+pkg.Name+"/"+pkg.Version] = pkg
	}
	calls := map[string]int{}
	resolver := newTestResolver(t, publicRegistryURL, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		pkg, exists := byPath[request.URL.Path]
		if !exists || request.URL.Host != "registry.npmjs.org" || request.URL.Scheme != "https" || request.Header.Get("Accept") != "application/json" {
			t.Fatal("native location replaced the exact public endpoint")
		}
		if strings.HasPrefix(pkg.Name, "@") && !strings.Contains(request.URL.EscapedPath(), "%2F") {
			t.Fatal("scoped package did not use an escaped exact endpoint")
		}
		calls[request.URL.Path]++
		return response(http.StatusOK, "application/json", bunMetadataJSON(t, bunMetadata(t, pkg))), nil
	}))
	selected, err := resolver.BindBunLock(context.Background(), lock)
	if err != nil || !selected.SourceBound() || len(selected.Artifacts()) != len(byPath) {
		t.Fatal("independent source binding failed", err)
	}
	if len(calls) != len(byPath) {
		t.Fatal("missing exact package lookup")
	}
	for _, count := range calls {
		if count != 1 {
			t.Fatal("shared archive was fetched for each native instance")
		}
	}
	return selected, []domain.ProjectControlFile{bunTestControl(t, "package.json", manifest, true), bunTestControl(t, "bun.lock", body, present)}
}

func TestBunBindingPreservesEveryInstanceAndFetchesUniqueArchives(t *testing.T) {
	roots, rows := bunScopedRows(true)
	manifest, body := bunSyntheticLock(t, roots, rows)
	selected, _ := bunTestSelection(t, manifest, body, true)
	if len(selected.Lock().Packages()) != 4 || len(selected.Artifacts()) != 3 {
		t.Fatal("instance tree was flattened or shared archive duplicated")
	}
	for _, artifact := range selected.Artifacts() {
		if !strings.Contains(artifact.AcquisitionLocator(), "/independent/") {
			t.Fatal("native default registry string was used to derive a URL")
		}
	}
	copy := selected.Artifacts()
	copy[0] = domain.ResolvedArtifact{}
	if selected.Artifacts()[0].Identity().Name() == "" {
		t.Fatal("selection exposed mutable artifact storage")
	}
	if (BunSelection{}).SourceBound() {
		t.Fatal("zero value acquired source authority")
	}
}

func TestBunBindingRejectsMetadataDriftWithoutPartialSelection(t *testing.T) {
	manifest := []byte(`{"name":"haa-bun-single","dependencies":{"is-number":"6.0.0"}}`)
	lock, err := parseBunTestLock(t, manifest, bunNativeFixture(t, "single"), true)
	if err != nil {
		t.Fatal(err)
	}
	pkg := lock.Packages()[0]
	for _, fault := range []string{"name", "version", "integrity", "origin", "dependency", "peer", "bin", "optional", "peer-meta", "bundled", "os", "cpu", "libc", "bad-optional-type", "bad-peer-meta-type", "bad-bundle-type", "bad-os-type", "bad-cpu-type", "bad-libc-type", "duplicate", "response-type", "status", "oversize"} {
		t.Run(fault, func(t *testing.T) {
			resolver := newTestResolver(t, publicRegistryURL, roundTripFunc(func(*http.Request) (*http.Response, error) {
				metadata := bunMetadata(t, pkg)
				switch fault {
				case "name", "version":
					metadata[fault] = "foreign"
				case "integrity":
					metadata["dist"].(map[string]any)["integrity"] = "sha512-drift"
				case "origin":
					metadata["dist"].(map[string]any)["tarball"] = "https://foreign.test/a.tgz"
				case "dependency":
					metadata["dependencies"] = map[string]string{"missing": "1.0.0"}
				case "peer":
					metadata["peerDependencies"] = map[string]string{"missing": "1.0.0"}
				case "bin":
					metadata["bin"] = "cli.js"
				case "optional":
					metadata["optionalDependencies"] = map[string]string{"missing": "1.0.0"}
				case "peer-meta":
					metadata["peerDependenciesMeta"] = map[string]any{"missing": map[string]bool{"optional": true}}
				case "bundled":
					metadata["bundledDependencies"] = []string{"missing"}
				case "os", "cpu", "libc":
					metadata[fault] = []string{"unsupported"}
				case "bad-optional-type":
					metadata["optionalDependencies"] = []any{}
				case "bad-peer-meta-type":
					metadata["peerDependenciesMeta"] = []any{}
				case "bad-bundle-type":
					metadata["bundledDependencies"] = map[string]any{}
				case "bad-os-type", "bad-cpu-type", "bad-libc-type":
					metadata[strings.TrimSuffix(strings.TrimPrefix(fault, "bad-"), "-type")] = map[string]any{}
				case "duplicate":
					return response(http.StatusOK, "application/json", `{"name":"foreign",`+bunMetadataJSON(t, metadata)[1:]), nil
				case "response-type":
					return response(http.StatusOK, "text/plain", bunMetadataJSON(t, metadata)), nil
				case "status":
					return response(http.StatusServiceUnavailable, "application/json", `{}`), nil
				case "oversize":
					metadata["description"] = strings.Repeat("x", metadataLimit)
				}
				return response(http.StatusOK, "application/json", bunMetadataJSON(t, metadata)), nil
			}))
			selected, err := resolver.BindBunLock(context.Background(), lock)
			if err == nil || !errors.Is(err, errBunBinding) || selected.SourceBound() || len(selected.Artifacts()) != 0 || selected.Lock().valid {
				t.Fatal("metadata drift exposed partial source authority", err)
			}
		})
	}
}

func TestBunBindingCancellationDiscardsEarlierBindings(t *testing.T) {
	manifest := []byte(`{"name":"haa-bun-transitive","dependencies":{"is-odd":"3.0.1"}}`)
	lock, err := parseBunTestLock(t, manifest, bunNativeFixture(t, "transitive"), true)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]BunLockedPackage{}
	for _, pkg := range lock.Packages() {
		byPath["/"+pkg.Name+"/"+pkg.Version] = pkg
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	resolver := newTestResolver(t, publicRegistryURL, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if calls == 2 {
			cancel()
		}
		return response(http.StatusOK, "application/json", bunMetadataJSON(t, bunMetadata(t, byPath[request.URL.Path]))), nil
	}))
	selected, err := resolver.BindBunLock(ctx, lock)
	if !errors.Is(err, context.Canceled) || calls != 2 || selected.SourceBound() || len(selected.Artifacts()) != 0 {
		t.Fatal("cancellation lost original cause or retained earlier bindings", err)
	}
	for _, origin := range []string{"http://registry.npmjs.org/", "https://foreign.test/"} {
		r := newTestResolver(t, origin, roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("foreign origin reached network")
			return nil, nil
		}))
		if _, err := r.BindBunLock(context.Background(), lock); err == nil {
			t.Fatal("foreign resolver accepted")
		}
	}
}

func TestBunBindingKeepsFormattedEmptySupportedFields(t *testing.T) {
	manifest := []byte(`{"name":"haa-bun-single","dependencies":{"is-number":"6.0.0"}}`)
	lock, err := parseBunTestLock(t, manifest, bunNativeFixture(t, "single"), true)
	if err != nil {
		t.Fatal(err)
	}
	pkg := lock.Packages()[0]
	r := newTestResolver(t, publicRegistryURL, roundTripFunc(func(*http.Request) (*http.Response, error) {
		prefix := `{"optionalDependencies":{ },"peerDependenciesMeta":{ },"bundledDependencies":[ ],"os":[ ],"cpu":[ ],"libc":[ ],`
		return response(http.StatusOK, "application/json", prefix+bunMetadataJSON(t, bunMetadata(t, pkg))[1:]), nil
	}))
	if selected, err := r.BindBunLock(context.Background(), lock); err != nil || !selected.SourceBound() {
		t.Fatal("valid metadata whitespace changed empty field semantics", err)
	}
}
