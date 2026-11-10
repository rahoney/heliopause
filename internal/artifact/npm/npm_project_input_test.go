package npm

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func npmProjectInputControls(t *testing.T, manifest, lock []byte, present bool) []domain.ProjectControlFile {
	t.Helper()
	a, err := domain.NewProjectControlFile("package.json", manifest, true)
	if err != nil {
		t.Fatal(err)
	}
	b, err := domain.NewProjectControlFile("package-lock.json", lock, present)
	if err != nil {
		t.Fatal(err)
	}
	return []domain.ProjectControlFile{a, b}
}

func npmProjectInputFixture(t *testing.T) ([]byte, []byte) {
	t.Helper()
	manifest := []byte(`{"name":"app","private":true,"dependencies":{"alpha":"^1"},"devDependencies":{"beta":"~2"},"optionalDependencies":{"alpha":"^2","optional":"*"},"peerDependencies":{"peer":"^3"},"peerDependenciesMeta":{"peer":{"optional":true}},"scripts":{"test":"node test.js","postinstall":"node setup.js"}}`)
	var root map[string]any
	if err := json.Unmarshal(manifest, &root); err != nil {
		t.Fatal(err)
	}
	packages := map[string]any{"": root}
	for _, name := range []string{"alpha", "beta", "optional", "peer"} {
		packages["node_modules/"+name] = map[string]any{"version": "2.0.0", "resolved": "https://registry.npmjs.org/" + name + "/-/" + name + "-2.0.0.tgz", "integrity": "sha512-" + base64.StdEncoding.EncodeToString(make([]byte, 64))}
	}
	body, err := json.Marshal(map[string]any{"lockfileVersion": 3, "packages": packages})
	if err != nil {
		t.Fatal(err)
	}
	return manifest, body
}

func TestNPMProjectInputPreservesWholeGroupsAndIntent(t *testing.T) {
	manifest, lock := npmProjectInputFixture(t)
	for _, intent := range []NPMProjectIntent{NPMProjectInstall, NPMProjectCI} {
		t.Run(string(intent), func(t *testing.T) {
			controls := npmProjectInputControls(t, manifest, lock, true)
			input, err := NewNPMProjectInput(intent, controls, "11.0.0")
			if err != nil || input.Intent() != intent || input.ClientVersion() != "11.0.0" {
				t.Fatal(input, err)
			}
			selected, err := input.ValidateOutput(manifest, lock)
			if err != nil || len(selected) != 2 || !bytes.Equal(selected[0].Body(), manifest) || !bytes.Equal(selected[1].Body(), lock) {
				t.Fatal("exact native controls were not preserved", err)
			}
			copy := input.Controls()
			copy[0] = domain.ProjectControlFile{}
			body := input.Controls()[0].Body()
			body[0] = '!'
			if !bytes.Equal(input.Controls()[0].Body(), manifest) {
				t.Fatal("caller mutated frozen controls")
			}
		})
	}
}

func TestNPMProjectInstallReconcilesLockWhileCIPreservesBytes(t *testing.T) {
	manifest, lock := npmProjectInputFixture(t)
	oldLock := bytes.ReplaceAll(lock, []byte(`"alpha":"^1"`), []byte(`"alpha":"^0"`))
	if bytes.Equal(oldLock, lock) {
		t.Fatal("stale fixture missed its root")
	}
	for _, original := range []struct {
		name    string
		body    []byte
		present bool
	}{{"initial", nil, false}, {"stale", oldLock, true}, {"existing", lock, true}} {
		t.Run(original.name, func(t *testing.T) {
			controls := npmProjectInputControls(t, manifest, original.body, original.present)
			input, err := NewNPMProjectInput(NPMProjectInstall, controls, "11.0.0")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := input.ValidateOutput(manifest, lock); err != nil {
				t.Fatal("install did not accept reconciled selection", err)
			}
			ci, err := NewNPMProjectInput(NPMProjectCI, controls, "11.0.0")
			if original.name != "existing" {
				if err == nil {
					t.Fatal("ci accepted missing/stale lock")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			// Whitespace is semantically equivalent, but ci must preserve bytes.
			if _, err := ci.ValidateOutput(manifest, append(lock, '\n')); err == nil {
				t.Fatal("ci accepted rewritten lock")
			}
		})
	}
}

func TestNPMProjectInputRejectsUnsafeSourcesBeforeNativeExecution(t *testing.T) {
	manifest, lock := npmProjectInputFixture(t)
	for name, changed := range map[string][]byte{
		"url":               bytes.ReplaceAll(manifest, []byte(`"alpha":"^1"`), []byte(`"alpha":"https://example.test/a.tgz"`)),
		"file":              bytes.ReplaceAll(manifest, []byte(`"alpha":"^1"`), []byte(`"alpha":"file:../other"`)),
		"alias":             bytes.ReplaceAll(manifest, []byte(`"alpha":"^1"`), []byte(`"alpha":"npm:other@1"`)),
		"workspace":         []byte(`{"workspaces":["packages/*"]}`),
		"duplicates":        []byte(`{"dependencies":{},"dependencies":{"alpha":"^1"}}`),
		"wrong client":      []byte(`{"packageManager":"pnpm@11.0.0"}`),
		"wrong pin":         []byte(`{"packageManager":"npm@11.0.1"}`),
		"null group":        []byte(`{"dependencies":null}`),
		"null scripts":      []byte(`{"scripts":null}`),
		"bad script":        []byte(`{"scripts":{"postinstall":true}}`),
		"invalid meta name": []byte(`{"peerDependenciesMeta":{"../absent":{"optional":true}}}`),
		"unknown meta":      []byte(`{"peerDependencies":{"peer":"*"},"peerDependenciesMeta":{"peer":{"optional":true,"other":true}}}`),
		"oversized":         []byte(strings.Repeat(" ", ProjectControlLimit)),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewNPMProjectInput(NPMProjectInstall, npmProjectInputControls(t, changed, nil, false), "11.0.0"); err == nil {
				t.Fatal("unsupported manifest accepted")
			}
		})
	}
	for name, changed := range map[string][]byte{
		"transitive URL": bytes.ReplaceAll(lock, []byte("https://registry.npmjs.org/beta/-/beta-2.0.0.tgz"), []byte("https://example.test/beta.tgz")),
		"credential":     bytes.ReplaceAll(lock, []byte("https://registry.npmjs.org/"), []byte("https://user:secret@registry.npmjs.org/")),
		"missing SRI":    bytes.ReplaceAll(lock, []byte("sha512-"), []byte("sha1-")),
		"path escape":    bytes.ReplaceAll(lock, []byte("node_modules/beta"), []byte("../beta")),
		"root source":    bytes.ReplaceAll(lock, []byte(`"alpha":"^1"`), []byte(`"alpha":"git+https://example.test/repo"`)),
		"old format":     bytes.ReplaceAll(lock, []byte(`"lockfileVersion":3`), []byte(`"lockfileVersion":2`)),
		"duplicate key":  []byte(`{"lockfileVersion":3,"lockfileVersion":3,"packages":{"":{}}}`),
		"empty present":  {},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewNPMProjectInput(NPMProjectInstall, npmProjectInputControls(t, manifest, changed, true), "11.0.0"); err == nil {
				t.Fatal("unsafe original lock accepted")
			}
		})
	}
}

func TestNPMProjectOutputRefusesChangedManifestAndStaleOrMalformedLock(t *testing.T) {
	manifest, lock := npmProjectInputFixture(t)
	input, err := NewNPMProjectInput(NPMProjectInstall, npmProjectInputControls(t, manifest, nil, false), "11.0.0")
	if err != nil {
		t.Fatal(err)
	}
	for _, changed := range [][]byte{append(bytes.Clone(manifest), '\n'), bytes.ReplaceAll(manifest, []byte("node setup.js"), []byte("node changed.js"))} {
		if _, err := input.ValidateOutput(changed, lock); err == nil {
			t.Fatal("changed manifest accepted")
		}
	}
	for _, changed := range [][]byte{nil, []byte(`{`), bytes.ReplaceAll(lock, []byte(`"alpha":"^1"`), []byte(`"alpha":"^0"`))} {
		if _, err := input.ValidateOutput(manifest, changed); err == nil {
			t.Fatal("missing/malformed/stale selection accepted")
		}
	}
	if _, err := (NPMProjectInput{}).ValidateOutput(manifest, lock); err == nil {
		t.Fatal("unbound input accepted output")
	}
	if _, err := NewNPMProjectInput("update", input.Controls(), "11.0.0"); err == nil {
		t.Fatal("unsupported whole-project intent accepted")
	}
}
