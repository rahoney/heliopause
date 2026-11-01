package npm

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
)

func npmWholeLockFixture(t *testing.T, manifest string, packages map[string]map[string]any) []byte {
	t.Helper()
	var root map[string]any
	if err := json.Unmarshal([]byte(manifest), &root); err != nil {
		t.Fatal(err)
	}
	values := map[string]any{"": root}
	for location, item := range packages {
		name, valid := packageNameFromLockPath(location)
		if !valid {
			t.Fatal("invalid fixture location")
		}
		copy := map[string]any{"version": "1.0.0", "resolved": "https://registry.npmjs.org/" + name + "/-/" + name + "-1.0.0.tgz", "integrity": "sha512-" + base64.StdEncoding.EncodeToString(make([]byte, 64))}
		for key, value := range item {
			copy[key] = value
		}
		values[location] = copy
	}
	body, err := json.Marshal(map[string]any{"lockfileVersion": 3, "packages": values})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestNPMProjectLockRetainsWholeSharedClosureAndPeerPlacement(t *testing.T) {
	manifest := `{"dependencies":{"alpha":"^1","beta":"^1"},"devDependencies":{"dev":"^1"}}`
	packages := map[string]map[string]any{
		"node_modules/alpha":                   {"dependencies": map[string]string{"shared": "^1"}},
		"node_modules/beta":                    {"dependencies": map[string]string{"child": "^1", "shared": "^1"}},
		"node_modules/dev":                     {"dev": true},
		"node_modules/shared":                  {},
		"node_modules/beta/node_modules/child": {"peerDependencies": map[string]string{"shared": "^1"}},
		// Peers resolve beside the parent, independently of child-owned packages.
		"node_modules/beta/node_modules/shared": {},
	}
	lock, err := ParseNPMProjectLock([]byte(manifest), npmWholeLockFixture(t, manifest, packages), "11.0.0")
	if err != nil || lock.DependencyFree() || len(lock.Roots()) != 3 || len(lock.Packages()) != 6 {
		t.Fatal(lock, err)
	}
	byLocation := map[string]NPMProjectPackage{}
	for _, pkg := range lock.Packages() {
		byLocation[pkg.Location] = pkg
	}
	if byLocation["node_modules/alpha"].Targets["shared"] != "node_modules/shared" || byLocation["node_modules/beta"].Targets["shared"] != "node_modules/beta/node_modules/shared" || byLocation["node_modules/beta/node_modules/child"].Targets["shared"] != "node_modules/beta/node_modules/shared" {
		t.Fatal("shared/peer resolution lost native placement", byLocation)
	}
	copy := lock.Packages()
	for i := range copy {
		copy[i].Targets["foreign"] = "outside"
	}
	for _, pkg := range lock.Packages() {
		if _, exists := pkg.Targets["foreign"]; exists {
			t.Fatal("caller changed frozen wiring")
		}
	}
}

func TestNPMProjectLockPlatformOptionalAndMissingPeers(t *testing.T) {
	manifest := `{"dependencies":{"alpha":"^1"},"optionalDependencies":{"darwin-only":"^1","unavailable":"*"},"peerDependencies":{"not-installed":"^1"},"peerDependenciesMeta":{"not-installed":{"optional":true}}}`
	packages := map[string]map[string]any{
		"node_modules/alpha":       {"optionalDependencies": map[string]string{"darwin-only": "^1"}},
		"node_modules/darwin-only": {"optional": true, "os": []string{"darwin"}},
	}
	lock, err := ParseNPMProjectLock([]byte(manifest), npmWholeLockFixture(t, manifest, packages), "11.0.0")
	if err != nil || len(lock.Packages()) != 1 || lock.Packages()[0].Name != "alpha" || len(lock.Packages()[0].Targets) != 0 {
		t.Fatal("optional platform selection differs", lock, err)
	}
	// Missing required peers and required platform mismatch remain failures.
	for _, changed := range []string{
		`{"dependencies":{"alpha":"^1"},"peerDependencies":{"not-installed":"^1"}}`,
		`{"dependencies":{"darwin-only":"^1"},"optionalDependencies":{"alpha":"^1"}}`,
	} {
		if _, err := ParseNPMProjectLock([]byte(changed), npmWholeLockFixture(t, changed, packages), "11.0.0"); err == nil {
			t.Fatal("required missing/platform dependency accepted")
		}
	}
}

func TestNPMProjectLockRejectsIncompleteForeignAndMalformedOptionalInputs(t *testing.T) {
	manifest := `{"dependencies":{"alpha":"^1"}}`
	for name, packages := range map[string]map[string]map[string]any{
		"missing root":               {},
		"missing dependency":         {"node_modules/alpha": {"dependencies": map[string]string{"missing": "^1"}}},
		"orphan":                     {"node_modules/alpha": {}, "node_modules/foreign": {}},
		"missing parent":             {"node_modules/alpha": {"optionalDependencies": map[string]string{"x": "*"}}, "node_modules/absent/node_modules/x": {}},
		"invalid peer metadata name": {"node_modules/alpha": {"peerDependenciesMeta": map[string]any{"../other": map[string]bool{"optional": true}}}},
		"bad bin escape":             {"node_modules/alpha": {"bin": map[string]string{"cmd": "../outside.js"}}},
		"malformed optional os":      {"node_modules/alpha": {"optionalDependencies": map[string]string{"optional": "*"}}, "node_modules/optional": {"os": []string{"!"}, "optional": true}},
		"malformed optional cpu":     {"node_modules/alpha": {"optionalDependencies": map[string]string{"optional": "*"}}, "node_modules/optional": {"cpu": []string{" x64"}, "optional": true}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseNPMProjectLock([]byte(manifest), npmWholeLockFixture(t, manifest, packages), "11.0.0"); err == nil {
				t.Fatal("invalid whole lock accepted")
			}
		})
	}
}

func TestNPMProjectLockEmptyAndOptionalOverride(t *testing.T) {
	for _, manifest := range []string{`{}`, `{"dependencies":{},"scripts":{"test":"node test.js"}}`} {
		lock, err := ParseNPMProjectLock([]byte(manifest), npmWholeLockFixture(t, manifest, nil), "11.0.0")
		if err != nil || !lock.DependencyFree() {
			t.Fatal("explicit dependency-free project rejected", err)
		}
	}
	manifest := `{"dependencies":{"alpha":"^1"},"optionalDependencies":{"alpha":"^2"}}`
	lock, err := ParseNPMProjectLock([]byte(manifest), npmWholeLockFixture(t, manifest, map[string]map[string]any{"node_modules/alpha": {"version": "2.0.0"}}), "11.0.0")
	if err != nil || len(lock.Packages()) != 1 || len(lock.Roots()) != 2 {
		t.Fatal("optional override/declared groups lost", err)
	}
}

// These exact public controls were captured from the pinned native npm after
// the first real parser failure. Metadata without a declared peer is an
// annotation, not an implicit dependency or inspection root.
func TestNPMProjectLockMetadataAnnotationsDoNotCreatePeerEdges(t *testing.T) {
	manifest, err := os.ReadFile("testdata/npm-project-native/debug.package.json")
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile("testdata/npm-project-native/debug.package-lock.json")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := ParseNPMProjectLock(manifest, body, "11.0.0")
	if err != nil || len(lock.Roots()) != 2 || len(lock.Packages()) != 3 {
		t.Fatal("native metadata-only annotation rejected", err)
	}
	for _, pkg := range lock.Packages() {
		if pkg.Name == "debug" {
			if len(pkg.PeerDependencies) != 0 || !pkg.OptionalPeers["supports-color"] || len(pkg.Targets) != 1 || pkg.Targets["ms"] != "node_modules/ms" {
				t.Fatal("annotation manufactured a peer edge or lost metadata", pkg)
			}
		}
	}
}
