package npm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func npmWholeBindingFixture(t *testing.T) (NPMProjectLock, []domain.ProjectControlFile) {
	t.Helper()
	manifest := `{"dependencies":{"alpha":"^1","beta":"^1"}}`
	body := npmWholeLockFixture(t, manifest, map[string]map[string]any{
		"node_modules/alpha":                     {"dependencies": map[string]string{"shared": "^1"}, "bin": map[string]string{"alpha-cli": "cli.js"}},
		"node_modules/beta":                      {"dependencies": map[string]string{"shared": "^1"}},
		"node_modules/shared":                    {},
		"node_modules/alpha/node_modules/shared": {},
	})
	lock, err := ParseNPMProjectLock([]byte(manifest), body, "11.0.0")
	if err != nil {
		t.Fatal(err)
	}
	return lock, npmProjectInputControls(t, []byte(manifest), body, true)
}

func npmWholeSourceMetadata(t *testing.T, pkg NPMProjectPackage) map[string]any {
	t.Helper()
	meta := map[string]any{"name": pkg.Name, "version": pkg.Version, "dist": map[string]any{"integrity": pkg.Integrity, "tarball": pkg.Registry}}
	if len(pkg.Dependencies) != 0 {
		meta["dependencies"] = pkg.Dependencies
	}
	if len(pkg.OptionalDependencies) != 0 {
		meta["optionalDependencies"] = pkg.OptionalDependencies
	}
	if len(pkg.PeerDependencies) != 0 {
		meta["peerDependencies"] = pkg.PeerDependencies
	}
	if len(pkg.OptionalPeers) != 0 {
		flags := map[string]any{}
		for name, optional := range pkg.OptionalPeers {
			flags[name] = map[string]bool{"optional": optional}
		}
		meta["peerDependenciesMeta"] = flags
	}
	if len(pkg.Bins) != 0 {
		meta["bin"] = pkg.Bins
	}
	return meta
}

func TestNPMProjectBindingChecksAllPlacementsWithUniqueExactQueries(t *testing.T) {
	lock, controls := npmWholeBindingFixture(t)
	byPath := map[string]NPMProjectPackage{}
	for _, pkg := range lock.Packages() {
		byPath["/"+pkg.Name+"/"+pkg.Version] = pkg
	}
	calls := map[string]int{}
	resolver := newTestResolver(t, publicRegistryURL, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		pkg, exists := byPath[request.URL.Path]
		if !exists || request.URL.Scheme != "https" || request.URL.Host != "registry.npmjs.org" || request.Header.Get("Accept") != "application/json" {
			t.Fatal("query did not use independent exact endpoint")
		}
		calls[request.URL.Path]++
		meta := npmWholeSourceMetadata(t, pkg)
		// Published development inputs do not belong to the install closure.
		meta["devDependencies"] = map[string]string{"dev-only": "file:../tools"}
		body, err := json.Marshal(meta)
		if err != nil {
			t.Fatal(err)
		}
		return response(http.StatusOK, "application/json", string(body)), nil
	}))
	selected, err := resolver.BindNPMProjectLock(context.Background(), lock)
	if err != nil || !selected.SourceBound() || len(selected.Artifacts()) != 3 || len(selected.Lock().Packages()) != 4 || len(calls) != 3 {
		t.Fatal("complete selection was flattened or binding failed", err)
	}
	for _, count := range calls {
		if count != 1 {
			t.Fatal("registry metadata queried per placement")
		}
	}
	target, _ := domain.NewInstallTarget("/tmp/npm-whole-selection")
	install, _ := domain.NewNPMProjectInstallContext(target)
	snapshot, err := selected.ProjectSnapshot(install, controls, "pinned-test-runtime")
	if err != nil || !snapshot.Valid() || len(snapshot.Dependencies()) != 3 {
		t.Fatal("whole snapshot unavailable", err)
	}
	if snapshot.GraphDigest().String() == "" || len(snapshot.ControlDigests()) != 2 || snapshot.Context() != install {
		t.Fatal("snapshot lost source controls/context")
	}
	if _, err := selected.ProjectSnapshot(install, controls, "unsafe runtime"); err == nil {
		t.Fatal("untrusted runtime identity admitted")
	}
	changed, _ := domain.NewProjectControlFile("package-lock.json", append(controls[1].Body(), '\n'), true)
	if _, err := selected.ProjectSnapshot(install, []domain.ProjectControlFile{controls[0], changed}, "pinned-test-runtime"); err == nil {
		t.Fatal("changed exact selection controls admitted")
	}
	copy := selected.Artifacts()
	copy[0] = domain.ResolvedArtifact{}
	if selected.Artifacts()[0].Identity().Name() == "" {
		t.Fatal("archive slice was mutable")
	}
}

func TestNPMProjectBindingRejectsSourceChangesAndNoPartialResult(t *testing.T) {
	lock, _ := npmWholeBindingFixture(t)
	pkg := lock.Packages()[0]
	for _, fault := range []string{"identity", "version", "integrity", "url", "dependencies", "optional", "peer", "bin", "platform", "cancel"} {
		t.Run(fault, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			resolver := newTestResolver(t, publicRegistryURL, roundTripFunc(func(*http.Request) (*http.Response, error) {
				meta := npmWholeSourceMetadata(t, pkg)
				switch fault {
				case "identity":
					meta["name"] = "other"
				case "version":
					meta["version"] = "9.0.0"
				case "integrity":
					meta["dist"].(map[string]any)["integrity"] = "sha512-different"
				case "url":
					meta["dist"].(map[string]any)["tarball"] = publicRegistryURL + "another/source.tgz"
				case "dependencies":
					meta["dependencies"] = map[string]string{"different": "^1"}
				case "optional":
					meta["optionalDependencies"] = map[string]string{"different": "^1"}
				case "peer":
					meta["peerDependencies"] = map[string]string{"different": "^1"}
				case "bin":
					meta["bin"] = map[string]string{"command": "cli.js"}
				case "platform":
					meta["os"] = []string{"darwin"}
				case "cancel":
					cancel()
				}
				body, _ := json.Marshal(meta)
				return response(http.StatusOK, "application/json", string(body)), nil
			}))
			selected, err := resolver.BindNPMProjectLock(ctx, lock)
			if err == nil || selected.SourceBound() || len(selected.Artifacts()) != 0 {
				t.Fatal("drift/partial selection admitted", err)
			}
			if fault == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal("first cancellation lost", err)
			}
		})
	}
}

func TestNPMProjectBindingExplicitEmptyAndCancelledInput(t *testing.T) {
	manifest := []byte(`{}`)
	body := npmWholeLockFixture(t, string(manifest), nil)
	lock, err := ParseNPMProjectLock(manifest, body, "11.0.0")
	if err != nil {
		t.Fatal(err)
	}
	resolver := newTestResolver(t, publicRegistryURL, roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("dependency-free binding queried a package")
		return nil, nil
	}))
	selected, err := resolver.BindNPMProjectLock(context.Background(), lock)
	if err != nil || !selected.SourceBound() || !selected.Lock().DependencyFree() {
		t.Fatal(selected, err)
	}
	target, _ := domain.NewInstallTarget("/tmp/npm-empty-selection")
	install, _ := domain.NewNPMProjectInstallContext(target)
	snapshot, err := selected.ProjectSnapshot(install, npmProjectInputControls(t, manifest, body, true), "pinned-test-runtime")
	if err != nil || !snapshot.DependencyFree() {
		t.Fatal(snapshot, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if selected, err := resolver.BindNPMProjectLock(ctx, lock); !errors.Is(err, context.Canceled) || selected.SourceBound() {
		t.Fatal("cancelled empty selection admitted", err)
	}
	if _, err := resolver.BindNPMProjectLock(context.Background(), NPMProjectLock{}); err == nil {
		t.Fatal("zero-value selection admitted")
	}
	if (NPMProjectSelection{}).SourceBound() {
		t.Fatal("invalid source/snapshot state")
	}
}
