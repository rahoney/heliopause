package npm

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func bunTestControl(t *testing.T, name string, body []byte, present bool) domain.ProjectControlFile {
	t.Helper()
	control, err := domain.NewProjectControlFile(name, body, present)
	if err != nil {
		t.Fatal(err)
	}
	return control
}

func parseBunTestLock(t *testing.T, manifest, body []byte, present bool) (BunLock, error) {
	t.Helper()
	return ParseBunLock(manifest, bunTestControl(t, "bun.lock", body, present), "1.4.4")
}

func bunNativeFixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile("testdata/bun-lock/" + name + ".bun.lock")
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestBunLockActualNativeSelectionAndExplicitAbsence(t *testing.T) {
	for _, test := range []struct {
		name, dependencies string
		packages, roots    int
	}{
		{"single", `{"is-number":"6.0.0"}`, 1, 1},
		{"transitive", `{"is-odd":"3.0.1"}`, 2, 1},
		{"peer", `{"react":"18.3.1","use-sync-external-store":"1.2.0"}`, 4, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			manifest := []byte(`{"name":"haa-bun-` + test.name + `","dependencies":` + test.dependencies + `}`)
			lock, err := parseBunTestLock(t, manifest, bunNativeFixture(t, test.name), true)
			if err != nil || !lock.valid || lock.DependencyFree() || len(lock.Packages()) != test.packages || len(lock.Roots()) != test.roots {
				t.Fatal("actual native selection differs", err)
			}
			for _, pkg := range lock.Packages() {
				if pkg.Name == "is-odd" && pkg.Targets["is-number"] != "is-number" || pkg.Name == "use-sync-external-store" && pkg.Targets["react"] != "react" || pkg.Name == "loose-envify" && pkg.Bins["loose-envify"] != "cli.js" {
					t.Fatal("native dependency/peer/bin data was lost")
				}
			}
		})
	}
	lock, err := parseBunTestLock(t, []byte(`{"dependencies":{}}`), nil, false)
	if err != nil || !lock.DependencyFree() || len(lock.Packages()) != 0 || len(lock.Roots()) != 0 || (BunLock{}).DependencyFree() {
		t.Fatal("explicit empty selection differs", err)
	}
	for _, test := range []struct {
		manifest string
		present  bool
	}{
		{`{"dependencies":{}}`, true},
		{`{"dependencies":{"is-number":"6.0.0"}}`, false},
	} {
		if selected, err := parseBunTestLock(t, []byte(test.manifest), nil, test.present); err == nil || selected.valid {
			t.Fatal("present empty file or missing nonempty lock was accepted")
		}
	}
	if selected, err := ParseBunLock([]byte(`{}`), domain.ProjectControlFile{}, "1.4.4"); err == nil || selected.valid {
		t.Fatal("unknown control became an absent lock")
	}
}

func bunSyntheticRow(name, version string, info map[string]any) []any {
	return []any{name + "@" + version, "", info, "sha512-" + base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{9}, 64))}
}

func bunSyntheticLock(t *testing.T, roots map[string]string, rows map[string][]any) ([]byte, []byte) {
	t.Helper()
	manifest, err := json.Marshal(map[string]any{"name": "fixture", "dependencies": roots})
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{"lockfileVersion": 2, "configVersion": 1, "workspaces": map[string]any{"": map[string]any{"name": "fixture", "dependencies": roots}}, "packages": rows})
	if err != nil {
		t.Fatal(err)
	}
	return manifest, body
}

func bunScopedRows(duplicateVersion bool) (map[string]string, map[string][]any) {
	version := "1.1.0"
	if duplicateVersion {
		version = "1.0.0"
	}
	return map[string]string{"@one/root": "1.0.0", "@two/lib": "1.0.0"}, map[string][]any{
		"@one/root":          bunSyntheticRow("@one/root", "1.0.0", map[string]any{"dependencies": map[string]string{"@two/lib": "^1.0.0", "consumer": "1.0.0"}}),
		"@two/lib":           bunSyntheticRow("@two/lib", "1.0.0", map[string]any{}),
		"@one/root/@two/lib": bunSyntheticRow("@two/lib", version, map[string]any{}),
		"@one/root/consumer": bunSyntheticRow("consumer", "1.0.0", map[string]any{"dependencies": map[string]string{"@two/lib": "^1.0.0"}}),
	}
}

func TestBunTreeKeepsScopedAncestryAndRepeatedArchiveInstances(t *testing.T) {
	for _, repeated := range []bool{false, true} {
		roots, rows := bunScopedRows(repeated)
		manifest, body := bunSyntheticLock(t, roots, rows)
		lock, err := parseBunTestLock(t, manifest, body, true)
		if err != nil || len(lock.Packages()) != 4 {
			t.Fatal("scoped/repeated instances were lost", err)
		}
		for _, pkg := range lock.Packages() {
			if pkg.Key == "@one/root" || pkg.Key == "@one/root/consumer" {
				if pkg.Targets["@two/lib"] != "@one/root/@two/lib" {
					t.Fatal("ancestor walk escaped the scoped package unit")
				}
				pkg.Dependencies["foreign"] = "*"
				pkg.Targets["@two/lib"] = "foreign"
			}
		}
		for _, pkg := range lock.Packages() {
			if pkg.Dependencies["foreign"] != "" || pkg.Targets["@two/lib"] == "foreign" {
				t.Fatal("selection exposed mutable instance maps")
			}
		}
		copy := lock.Roots()
		copy[0].Instance = "foreign"
		if lock.Roots()[0].Instance == "foreign" {
			t.Fatal("selection exposed mutable roots")
		}
	}
}

func TestBunPeerVersionAmbiguityIsNotGuessed(t *testing.T) {
	roots, rows := bunScopedRows(false)
	rows["@one/root/consumer"][2] = map[string]any{"peerDependencies": map[string]string{"@two/lib": "^1.0.0"}}
	manifest, body := bunSyntheticLock(t, roots, rows)
	if selected, err := parseBunTestLock(t, manifest, body, true); err == nil || selected.valid || len(selected.Packages()) != 0 {
		t.Fatal("multiple matching peer versions were guessed")
	}
	rows["@two/lib"] = bunSyntheticRow("@two/lib", "2.0.0", map[string]any{})
	roots["@two/lib"] = "2.0.0"
	manifest, body = bunSyntheticLock(t, roots, rows)
	if selected, err := parseBunTestLock(t, manifest, body, true); err != nil || len(selected.Packages()) != 4 {
		t.Fatal("single matching peer version in its actual ancestor was lost", err)
	}
}

func TestBunLockRejectsUnsupportedAndInconsistentData(t *testing.T) {
	base := string(bunNativeFixture(t, "single"))
	manifest := []byte(`{"name":"haa-bun-single","dependencies":{"is-number":"6.0.0"}}`)
	cases := map[string]string{
		"schema":       strings.Replace(base, `"lockfileVersion": 2`, `"lockfileVersion": 3`, 1),
		"config":       strings.Replace(base, `"configVersion": 1`, `"configVersion": 2`, 1),
		"float-schema": strings.Replace(base, `"lockfileVersion": 2`, `"lockfileVersion": 2.0`, 1),
		"root-name":    strings.Replace(base, "haa-bun-single", "foreign", 1),
		"root-spec":    strings.Replace(base, `"is-number": "6.0.0"`, `"is-number": "7.0.0"`, 1),
		"name":         strings.Replace(base, "is-number@6.0.0", "foreign@6.0.0", 1),
		"version":      strings.Replace(base, "is-number@6.0.0", "is-number@7.0.0", 1),
		"leading-zero": strings.Replace(base, "is-number@6.0.0", "is-number@06.0.0", 1),
		"source":       strings.Replace(base, `, "", {}`, `, "https://foreign.test/a.tgz", {}`, 1),
		"optional":     strings.Replace(base, `, "", {}`, `, "", {"optionalDependencies":{"x":"1.0.0"}}`, 1),
		"os":           strings.Replace(base, `, "", {}`, `, "", {"os":"linux"}`, 1),
		"null-info":    strings.Replace(base, `, "", {}`, `, "", null`, 1),
		"null-flag":    strings.Replace(base, `, "", {}`, `, "", {"hasInstallScript":null}`, 1),
		"null-deps":    strings.Replace(base, `, "", {}`, `, "", {"dependencies":null}`, 1),
		"escape-bin":   strings.Replace(base, `, "", {}`, `, "", {"bin":{"x":"../escape.js"}}`, 1),
		"foreign-lock": strings.Replace(base, `"configVersion": 1,`, `"configVersion": 1,"overrides":{},`, 1),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if body == base {
				t.Fatal("mutation did not alter the fixture")
			}
			selected, err := parseBunTestLock(t, manifest, []byte(body), true)
			if err == nil || selected.valid || selected.DependencyFree() || len(selected.Roots()) != 0 || len(selected.Packages()) != 0 {
				t.Fatal("unsupported selection emitted a partial result")
			}
		})
	}
	for name, info := range map[string]map[string]any{
		"missing-dependency": {"dependencies": map[string]string{"missing": "1.0.0"}},
		"self-dependency":    {"dependencies": map[string]string{"root": "1.0.0"}},
		"bad-script":         {"hasInstallScript": "true"},
		"non-string-bin":     {"bin": map[string]any{"x": 42}},
	} {
		t.Run(name, func(t *testing.T) {
			m, b := bunSyntheticLock(t, map[string]string{"root": "1.0.0"}, map[string][]any{"root": bunSyntheticRow("root", "1.0.0", info)})
			if selected, err := parseBunTestLock(t, m, b, true); err == nil || selected.valid {
				t.Fatal("inconsistent package accepted")
			}
		})
	}
}

func TestBunLockRootAndNodeBudgetsAndUnsupportedManifest(t *testing.T) {
	for _, field := range []string{`"packageManager":"pnpm@12.15.0"`, `"optionalDependencies":{"x":"1.0.0"}`, `"devEngines":{"runtime":{"name":"bun"}}`, `"dependencies":null`, `"dependencies":{"x":"latest"}`} {
		if selected, err := parseBunTestLock(t, []byte(`{`+field+`}`), nil, false); err == nil || selected.valid {
			t.Fatal("unsupported manifest became an empty selection", field)
		}
	}
	roots, rows := map[string]string{}, map[string][]any{}
	for i := 0; i <= 256; i++ {
		roots[fmt.Sprintf("p%d", i)] = "1.0.0"
	}
	m, b := bunSyntheticLock(t, roots, rows)
	if selected, err := parseBunTestLock(t, m, b, true); err == nil || selected.valid {
		t.Fatal("root budget bypassed")
	}
	for i := 0; i <= bunLockNodeLimit; i++ {
		name := fmt.Sprintf("p%d", i)
		rows[name] = bunSyntheticRow(name, "1.0.0", map[string]any{})
	}
	m, b = bunSyntheticLock(t, map[string]string{"p0": "1.0.0"}, rows)
	if selected, err := parseBunTestLock(t, m, b, true); err == nil || selected.valid {
		t.Fatal("node budget bypassed")
	}
}

func TestBunManifestRejectsMalformedEmptyUnsupportedFields(t *testing.T) {
	for _, field := range []string{`"optionalDependencies":[]`, `"peerDependencies":[]`, `"workspaces":{}`, `"bun":[]`, `"overrides":[]`, `"trustedDependencies":{}`, `"bundledDependencies":{}`, `"catalog":[]`} {
		t.Run("bad-"+strings.Split(field, ":")[0], func(t *testing.T) {
			if selected, err := parseBunTestLock(t, []byte(`{`+field+`}`), nil, false); err == nil || selected.valid {
				t.Fatal("malformed empty field became a dependency-free selection", field)
			}
		})
	}
}

func TestBunManifestKeepsFormattedEmptySupportedFields(t *testing.T) {
	body := []byte(`{"optionalDependencies":{ },"peerDependencies":{ },"workspaces":[ ],"bun":{ },"overrides":{ },"trustedDependencies":[ ],"bundledDependencies":[ ],"catalog":{ }}`)
	selected, err := parseBunTestLock(t, body, nil, false)
	if err != nil || !selected.DependencyFree() {
		t.Fatal("valid JSON whitespace changed empty field semantics", err)
	}
}
