package npm

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"go.yaml.in/yaml/v3"
)

const pnpmSingleManifest = `{"dependencies":{"is-number":"6.0.0"}}`
const pnpmPeerManifest = `{"dependencies":{"react":"18.3.1","use-sync-external-store":"1.2.0"}}`
const pnpmEmptyLock = "lockfileVersion: '9.0'\nsettings:\n  autoInstallPeers: true\n  excludeLinksFromLockfile: false\nimporters:\n  .: {}\n"

func parsePNPMTestLock(manifest, body []byte) (PNPMLock, error) {
	return ParsePNPMLock(manifest, body, runtimeidentity.PNPMVersion)
}

func pnpmFixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile("testdata/pnpm-v9-" + name + ".synthetic.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestPNPMLockPreservesRootsPeerInstancesAndMetadata(t *testing.T) {
	for _, test := range []struct {
		name, manifest             string
		body                       []byte
		roots, packages, snapshots int
	}{
		{"empty", `{}`, []byte(pnpmEmptyLock), 0, 0, 0},
		{"single", pnpmSingleManifest, pnpmFixture(t, "single"), 1, 1, 1},
		{"multiple-root-peer", pnpmPeerManifest, pnpmFixture(t, "peer"), 2, 4, 4},
	} {
		t.Run(test.name, func(t *testing.T) {
			lock, err := parsePNPMTestLock([]byte(test.manifest), test.body)
			if err != nil || len(lock.Roots()) != test.roots || len(lock.Packages()) != test.packages || len(lock.Snapshots()) != test.snapshots || lock.DependencyFree() != (test.roots == 0) {
				t.Fatalf("selection differs: roots=%d packages=%d snapshots=%d error=%v", len(lock.Roots()), len(lock.Packages()), len(lock.Snapshots()), err)
			}
		})
	}
	if (PNPMLock{}).DependencyFree() {
		t.Fatal("zero value is not an explicitly parsed empty graph")
	}
	lock, err := parsePNPMTestLock([]byte(pnpmPeerManifest), pnpmFixture(t, "peer"))
	if err != nil {
		t.Fatal(err)
	}
	var peer PNPMLockedSnapshot
	for _, snapshot := range lock.Snapshots() {
		if snapshot.Package == "use-sync-external-store@1.2.0" {
			peer = snapshot
		}
	}
	if peer.Key != "use-sync-external-store@1.2.0(react@18.3.1)" || peer.Dependencies["react"] != "react@18.3.1" {
		t.Fatal("peer snapshot or its real edge was flattened")
	}

	// Additional, explicitly synthetic graph: the same tarball has two peer
	// instances reached through different roots. This is not source authenticity.
	var wire pnpmLockWire
	if yaml.Unmarshal(pnpmFixture(t, "peer"), &wire) != nil {
		t.Fatal("cannot construct synthetic peer graph")
	}
	wire.Importers["."].Dependencies["bridge"] = pnpmRootWire{"1.0.0", "1.0.0"}
	wire.Packages["react@17.0.2"] = wire.Packages["react@18.3.1"]
	wire.Packages["bridge@1.0.0"] = wire.Packages["react@18.3.1"]
	wire.Snapshots["react@17.0.2"] = pnpmSnapshotWire{}
	wire.Snapshots["bridge@1.0.0"] = pnpmSnapshotWire{Dependencies: map[string]string{"use-sync-external-store": "1.2.0(react@17.0.2)"}}
	wire.Snapshots["use-sync-external-store@1.2.0(react@17.0.2)"] = pnpmSnapshotWire{Dependencies: map[string]string{"react": "17.0.2"}}
	body, err := yaml.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	lock, err = parsePNPMTestLock([]byte(`{"dependencies":{"react":"18.3.1","use-sync-external-store":"1.2.0","bridge":"1.0.0"}}`), body)
	if err != nil || len(lock.Packages()) != 6 || len(lock.Snapshots()) != 7 {
		t.Fatalf("peer instances collapsed: packages=%d snapshots=%d error=%v", len(lock.Packages()), len(lock.Snapshots()), err)
	}
}

func TestPNPMLockSelectionIsDeterministicAndCannotBeMutated(t *testing.T) {
	input := pnpmFixture(t, "peer")
	before, err := parsePNPMTestLock([]byte(pnpmPeerManifest), input)
	if err != nil {
		t.Fatal(err)
	}
	var node yaml.Node
	if yaml.Unmarshal(input, &node) != nil {
		t.Fatal("fixture YAML unavailable")
	}
	var reverse func(*yaml.Node)
	reverse = func(n *yaml.Node) {
		for _, child := range n.Content {
			reverse(child)
		}
		if n.Kind == yaml.MappingNode {
			for left, right := 0, len(n.Content)-2; left < right; left, right = left+2, right-2 {
				n.Content[left], n.Content[right] = n.Content[right], n.Content[left]
				n.Content[left+1], n.Content[right+1] = n.Content[right+1], n.Content[left+1]
			}
		}
	}
	reverse(&node)
	reordered, err := yaml.Marshal(&node)
	if err != nil {
		t.Fatal(err)
	}
	after, err := parsePNPMTestLock([]byte(pnpmPeerManifest), reordered)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("mapping order changed selection", err)
	}
	before.Roots()[0].Name = "mutated"
	for _, p := range before.Packages() {
		if p.Engines != nil {
			p.Engines["mutated"] = "bad"
		}
		if p.PeerDependencies != nil {
			p.PeerDependencies["mutated"] = "bad"
		}
	}
	for _, s := range before.Snapshots() {
		s.Dependencies["mutated"] = "bad"
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("getter exposed mutable selection storage")
	}
}

func TestPNPMLockRejectsAmbiguousYAMLAndUnsupportedGraph(t *testing.T) {
	base := string(pnpmFixture(t, "single"))
	for name, body := range map[string]string{
		"duplicate":        base + "\nlockfileVersion: '9.0'\n",
		"duplicate-nested": strings.Replace(base, "specifier: 6.0.0", "specifier: 6.0.0\n        specifier: 7.0.0", 1),
		"anchor":           strings.Replace(base, "settings:", "settings: &settings", 1),
		"alias":            base + "extra: *settings\n",
		"merge":            strings.Replace(base, "autoInstallPeers: true", "<<: {autoInstallPeers: true}", 1),
		"tag":              strings.Replace(base, "settings:", "settings: !custom", 1),
		"trailing":         base + "---\nextra: true\n",
		"unknown":          base + "unknownSelection: true\n",
		"future-version":   strings.Replace(base, "'9.0'", "'12.0'", 1),
		"tar-integrity":    strings.Replace(base, "autoInstallPeers: true", "autoInstallPeers: true\n  integrityTarget: tar", 1),
		"revision":         strings.Replace(base, "resolution: {integrity:", "resolution: {revision: 0, integrity:", 1),
		"http":             strings.Replace(base, "resolution: {integrity:", "resolution: {tarball: 'http://registry.npmjs.org/a.tgz', integrity:", 1),
		"foreign":          strings.Replace(base, "resolution: {integrity:", "resolution: {tarball: 'https://foreign.invalid/a.tgz', integrity:", 1),
		"weak-sri":         strings.Replace(base, "sha512-", "sha1-", 1),
		"missing-sri":      strings.Replace(base, "resolution: {integrity:", "resolution: {absent:", 1),
		"wrong-root":       strings.Replace(base, "specifier: 6.0.0", "specifier: ^6.0.0", 1),
		"wrong-importer":   strings.Replace(base, "  .:", "  ../escape:", 1),
		"alias-root":       strings.Replace(base, "version: 6.0.0", "version: alias@6.0.0", 1),
		"empty-peer":       strings.Replace(base, "version: 6.0.0", "version: 6.0.0()", 1),
		"dangling":         strings.Replace(base, "version: 6.0.0", "version: 7.0.0", 1),
		"no-snapshots":     strings.Split(base, "snapshots:")[0],
		"unreachable":      base + "\n  orphan@1.0.0: {}\n",
		"platform":         strings.Replace(base, "    engines:", "    os: [linux]\n    engines:", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if body == base {
				t.Fatal("negative fixture did not change its input")
			}
			if _, err := parsePNPMTestLock([]byte(pnpmSingleManifest), []byte(body)); err == nil {
				t.Fatal("unsupported selection accepted")
			}
		})
	}
	oversized := append([]byte(pnpmEmptyLock), bytes.Repeat([]byte(" "), pnpmLockLimit)...)
	if err := validatePNPMYAML(oversized); err != nil {
		t.Fatal("oversize fixture must otherwise remain valid YAML", err)
	}
	if _, err := parsePNPMTestLock([]byte(`{}`), oversized); err == nil {
		t.Fatal("oversized lock accepted")
	}
	for _, manifest := range []string{
		`{"dependencies":{"is-number":"6.0.0","is-number":"6.0.0"}}`,
		`{"dependencies":{"is-number":"file:../local"}}`,
		`{"dependencies":{"is-number":"6.0.0"},"devDependencies":{"is-number":"6.0.0"}}`,
		`{"dependencies":{"is-number":"6.0.0"},"workspaces":["../other"]}`,
		`{"dependencies":{"is-number":"6.0.0"},"packageManager":"pnpm@0.0.0"}`,
	} {
		if _, err := parsePNPMTestLock([]byte(manifest), []byte(base)); err == nil {
			t.Fatal("unsupported manifest accepted", manifest)
		}
	}
	manifest := `{"dependencies":{"is-number":"6.0.0"},"packageManager":"pnpm@` + runtimeidentity.PNPMVersion + `"}`
	if _, err := parsePNPMTestLock([]byte(manifest), []byte(base)); err != nil {
		t.Fatal("current pinned client refused", err)
	}
}

func TestPNPMYAMLRejectsStructuralFloodsBeforeMapDecoding(t *testing.T) {
	var deep strings.Builder
	for i := 0; i < 66; i++ {
		deep.WriteString(strings.Repeat("  ", i) + "nested:\n")
	}
	deep.WriteString(strings.Repeat("  ", 66) + "value: x\n")
	if validatePNPMYAML([]byte(deep.String())) == nil {
		t.Fatal("excessive YAML depth accepted")
	}
	flood := "items:\n" + strings.Repeat("  - a\n", 131072)
	if validatePNPMYAML([]byte(flood)) == nil {
		t.Fatal("excessive YAML node count accepted")
	}
	for _, value := range []string{"!!binary YWJj", "!!timestamp 2026-10-24", "null"} {
		if validatePNPMYAML([]byte("value: "+value+"\n")) == nil {
			t.Fatal("ambiguous scalar accepted", value)
		}
	}
	if _, err := ParsePNPMLock([]byte(`{}`), []byte(pnpmEmptyLock), ""); err == nil {
		t.Fatal("unknown producer version accepted")
	}
}

func TestPNPMLockKeepsScopedDevRootsAndRejectsMissingPeerEdges(t *testing.T) {
	var wire pnpmLockWire
	if yaml.Unmarshal(pnpmFixture(t, "single"), &wire) != nil {
		t.Fatal("fixture unavailable")
	}
	wire.Importers["."] = pnpmImporterWire{Dev: map[string]pnpmRootWire{"@scope/tiny": {"6.0.0", "6.0.0"}}}
	wire.Packages["@scope/tiny@6.0.0"] = wire.Packages["is-number@6.0.0"]
	delete(wire.Packages, "is-number@6.0.0")
	wire.Snapshots = map[string]pnpmSnapshotWire{"@scope/tiny@6.0.0": {}}
	body, err := yaml.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := parsePNPMTestLock([]byte(`{"devDependencies":{"@scope/tiny":"6.0.0"}}`), body)
	if err != nil || len(selected.Roots()) != 1 || selected.Roots()[0].Group != "devDependencies" || selected.Packages()[0].Name != "@scope/tiny" {
		t.Fatal("scoped dev selection lost", err)
	}
	peer := string(pnpmFixture(t, "peer"))
	peer = strings.Replace(peer, "      react: 18.3.1", "      absent: 18.3.1", 1)
	if _, err := parsePNPMTestLock([]byte(pnpmPeerManifest), []byte(peer)); err == nil {
		t.Fatal("missing mandatory peer binding accepted")
	}
}
