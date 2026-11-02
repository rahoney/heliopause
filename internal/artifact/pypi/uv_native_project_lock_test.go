package pypi

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func uvNativeProjectFixture(t *testing.T) (UVProjectInput, []byte, UVProjectLock) {
	t.Helper()
	input, body, export, _, _ := uvNativeProjectFixtureWithArchives(t)
	return input, body, export
}

func uvNativeProjectFixtureWithArchives(t *testing.T) (UVProjectInput, []byte, UVProjectLock, []UVWheelInput, []SimpleProject) {
	t.Helper()
	selected, inputs, pages := uvTestSelection(t, "child>=1.0")
	export, err := ParseUVProjectLock(uvProjectBody(selected.Packages()), runtimeidentity.PythonVersion)
	if err != nil {
		t.Fatal(err)
	}
	input, err := ParseUVProjectInput(uvVirtualProjectTestInput(`["parent>=1.0,<2"]`), nil, false, UVProjectSync, "")
	if err != nil {
		t.Fatal(err)
	}
	// Structural source/SRI fixture, not a claim of native public acquisition.
	body := "version=1\nrevision=5\nrequires-python='==" + runtimeidentity.PythonVersion + "'\n"
	for _, item := range selected.Packages() {
		body += fmt.Sprintf("\n[[package]]\nname=%q\nversion=%q\nsource={registry='https://pypi.org/simple'}\n", item.Name, item.Version)
		if item.Name == "parent" {
			body += "dependencies=[{name='child'}]\n"
		}
		body += "wheels=["
		for _, wheel := range item.Wheels {
			body += fmt.Sprintf("{url=%q,hash=%q,size=%d},", wheel.URL, "sha256:"+wheel.SHA256, wheel.Size)
		}
		body += "]\n"
	}
	body += "\n[[package]]\nname='haa-example'\nversion='0.1.0'\nsource={virtual='.'}\ndependencies=[{name='parent'}]\n[package.metadata]\nrequires-dist=[{name='parent',specifier='<2,>=1.0'}]\n"
	return input, []byte(body), export, inputs, pages
}

func TestUVNativeProjectLockLinksCompleteWheelExport(t *testing.T) {
	input, body, export := uvNativeProjectFixture(t)
	lock, err := ParseUVNativeProjectLock(input, body)
	if err != nil || lock.ValidateTargetExport(export) != nil {
		t.Fatal("native project identity/closure and complete export rejected", err)
	}
	control, err := lock.Control()
	if err != nil || control.Name() != UVNativeLockControl || !control.Present() || !bytes.Equal(control.Body(), body) {
		t.Fatal("native original control bytes lost", err)
	}
	body[0] = '!'
	if current, err := lock.Control(); err != nil || current.Body()[0] == '!' {
		t.Fatal("caller mutated native lock bytes", err)
	}
	// A validated public sdist declaration is inert metadata, never selected.
	_, body, _ = uvNativeProjectFixture(t)
	body = bytes.Replace(body, []byte("wheels=["), []byte("sdist={url='https://files.pythonhosted.org/packages/child-1.0.tar.gz',hash='sha256:"+strings.Repeat("a", 64)+"',size=100,upload-time='2026-10-25T00:00:00Z'}\nwheels=["), 1)
	if _, err := ParseUVNativeProjectLock(input, body); err != nil {
		t.Fatal("inert matching source alternative rejected", err)
	}
}

func TestUVNativeProjectLockAcceptsActualEmptyVirtualRootAndExport(t *testing.T) {
	body, err := os.ReadFile("testdata/uv-0.13.4-empty-project.pyproject.toml")
	if err != nil {
		t.Fatal(err)
	}
	// Project the historical format onto the current runtime; keep captured testdata unchanged.
	body = bytes.ReplaceAll(body, []byte("3.14.7"), []byte(runtimeidentity.PythonVersion))
	native, err := os.ReadFile("testdata/uv-0.13.4-empty-project.uv.lock")
	if err != nil {
		t.Fatal(err)
	}
	// Project the historical format onto the current runtime; keep captured testdata unchanged.
	native = bytes.ReplaceAll(native, []byte("3.14.7"), []byte(runtimeidentity.PythonVersion))
	input, err := ParseUVProjectInput(body, nil, false, UVProjectSync, "")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := ParseUVNativeProjectLock(input, native)
	if err != nil {
		t.Fatal(err)
	}
	export, err := ParseUVProjectLock(uvProjectBody(nil), runtimeidentity.PythonVersion)
	if err != nil || lock.ValidateTargetExport(export) != nil {
		t.Fatal("explicit empty native root/export combination rejected", err)
	}
	if _, err := ParseUVNativeProjectLock(input, []byte("version=1\nrevision=5\nrequires-python='=="+runtimeidentity.PythonVersion+"'\npackage=[]\n")); err == nil {
		t.Fatal("missing virtual root treated as an empty project")
	}
}

func TestUVNativeProjectLockRefusesSourceSchemaGraphAndManifestSubstitution(t *testing.T) {
	input, body, _ := uvNativeProjectFixture(t)
	base := string(body)
	for name, change := range map[string][2]string{
		"future version":          {"version=1\n", "version=2\n"},
		"past revision":           {"revision=5", "revision=3"},
		"runtime":                 {"==" + runtimeidentity.PythonVersion, "==3.15.0"},
		"root name":               {"name='haa-example'", "name='foreign-root'"},
		"root version":            {"version='0.1.0'", "version='0.2.0'"},
		"local root":              {"virtual='.'", "virtual='../other'"},
		"executable root":         {"virtual='.'", "editable='.'"},
		"root mixed source":       {"virtual='.'", "virtual='.',registry=''"},
		"foreign registry":        {"https://pypi.org/simple", "https://example.org/simple"},
		"dependency mixed source": {"registry='https://pypi.org/simple'", "registry='https://pypi.org/simple',virtual=''"},
		"foreign wheel":           {"files.pythonhosted.org", "example.org"},
		"hash algorithm":          {"sha256:", "sha512:"},
		"missing transitive":      {"dependencies=[{name='child'}]", "dependencies=[{name='missing'}]"},
		"duplicate edge":          {"dependencies=[{name='child'}]", "dependencies=[{name='child'},{name='child'}]"},
		"self edge":               {"dependencies=[{name='child'}]", "dependencies=[{name='parent'}]"},
		"root edge":               {"dependencies=[{name='child'}]", "dependencies=[{name='haa-example'}]"},
		"disconnected":            {"dependencies=[{name='child'}]", "dependencies=[]"},
		"invalid marker type":     {"dependencies=[{name='child'}]", "dependencies=[{name='child',marker=true}]"},
		"wrong root constraint":   {"specifier='<2,>=1.0'", "specifier='>=2'"},
		"missing root metadata":   {"requires-dist=[{name='parent',specifier='<2,>=1.0'}]", "requires-dist=[]"},
		"duplicate metadata":      {"requires-dist=[{name='parent',specifier='<2,>=1.0'}]", "requires-dist=[{name='parent',specifier='<2,>=1.0'},{name='parent',specifier='<2,>=1.0'}]"},
	} {
		t.Run(name, func(t *testing.T) {
			changed := strings.ReplaceAll(base, change[0], change[1])
			if changed == base {
				t.Fatal("negative fixture did not change input")
			}
			if _, err := ParseUVNativeProjectLock(input, []byte(changed)); err == nil {
				t.Fatal("invalid native project lock accepted")
			}
		})
	}
	for name, changed := range map[string][]byte{
		"empty":                        nil,
		"oversize":                     bytes.Repeat([]byte(" "), maxReportBytes+1),
		"unknown field":                append([]byte("unreviewed=true\n"), body...),
		"duplicate package":            append(append([]byte(nil), body...), []byte("\n[[package]]\nname='child'\nversion='1.0'\nsource={registry='https://pypi.org/simple'}\n")...),
		"source alternative injection": bytes.Replace(body, []byte("wheels=["), []byte("sdist={url='file:///etc/passwd',hash='sha256:"+strings.Repeat("a", 64)+"',size=100}\nwheels=["), 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseUVNativeProjectLock(input, changed); err == nil {
				t.Fatal("invalid native bytes accepted")
			}
		})
	}
	other, err := ParseUVProjectInput(uvVirtualProjectTestInput(`["parent>=2"]`), nil, false, UVProjectSync, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseUVNativeProjectLock(other, body); err == nil {
		t.Fatal("lock reused for a different manifest")
	}
}

func TestUVNativeProjectLockRefusesTargetExportSubstitution(t *testing.T) {
	input, body, export := uvNativeProjectFixture(t)
	lock, err := ParseUVNativeProjectLock(input, body)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func([]UVLockedPackage) []UVLockedPackage{
		"missing package": func(p []UVLockedPackage) []UVLockedPackage { return p[1:] },
		"hash":            func(p []UVLockedPackage) []UVLockedPackage { p[0].Wheels[0].SHA256 = strings.Repeat("f", 64); return p },
		"size":            func(p []UVLockedPackage) []UVLockedPackage { p[0].Wheels[0].Size++; return p },
		"url": func(p []UVLockedPackage) []UVLockedPackage {
			p[0].Wheels[0].URL = strings.Replace(p[0].Wheels[0].URL, "/packages/", "/packages/other/", 1)
			return p
		},
	} {
		t.Run(name, func(t *testing.T) {
			changed, err := ParseUVProjectLock(uvProjectBody(mutate(export.Packages())), runtimeidentity.PythonVersion)
			if err != nil {
				t.Fatal("negative export must be a valid independent selection", err)
			}
			if lock.ValidateTargetExport(changed) == nil {
				t.Fatal("substituted export accepted")
			}
		})
	}
	if lock.ValidateTargetExport(UVProjectLock{}) == nil || (UVNativeProjectLock{}).ValidateTargetExport(export) == nil {
		t.Fatal("zero selection accepted")
	}
	if _, err := (UVNativeProjectLock{}).Control(); err == nil {
		t.Fatal("zero native lock produced a control")
	}
}

func TestUVNativeProjectConditionalVariantsAndGroupClosure(t *testing.T) {
	for _, name := range []string{"v1", "v2"} {
		t.Run(name, func(t *testing.T) {
			root := filepath.Join("testdata", "uv-project-options-native", name)
			manifest, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
			if err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(filepath.Join(root, "uv.lock"))
			if err != nil {
				t.Fatal(err)
			}
			// Captured data remains immutable. If a later central runtime leaves the
			// historical Python range, project only both test headers for this
			// structural test instead of adding another runtime version pin.
			var manifestWire uvProjectInputWire
			if err := toml.Unmarshal(manifest, &manifestWire); err != nil {
				t.Fatal(err)
			}
			requirement, err := ParseBoundedRequirement("python" + manifestWire.Project.Python)
			if err != nil {
				t.Fatal(err)
			}
			if !requirement.Satisfies(runtimeidentity.PythonVersion) {
				var lockWire uvNativeProjectLockWire
				if err := toml.Unmarshal(body, &lockWire); err != nil {
					t.Fatal(err)
				}
				manifestWire.Project.Python = "==" + runtimeidentity.PythonVersion
				lockWire.Python = manifestWire.Project.Python
				manifest, err = toml.Marshal(manifestWire)
				if err != nil {
					t.Fatal(err)
				}
				body, err = toml.Marshal(lockWire)
				if err != nil {
					t.Fatal(err)
				}
			}
			input, err := ParseUVProjectInput(manifest, nil, false, UVProjectSync, "")
			if err != nil {
				t.Fatal(err)
			}
			if name == "v2" {
				// The actual universal charset-normalizer lock has 91 alternatives,
				// beyond the existing 64-wheel bound. Retain that refusal unchanged.
				if _, err := ParseUVNativeProjectLock(input, body); err == nil {
					t.Fatal("actual over-budget universal lock accepted")
				}
				var wire uvNativeProjectLockWire
				if err := toml.Unmarshal(body, &wire); err != nil {
					t.Fatal(err)
				}
				// Structural variant/group test only; this projection is not native evidence.
				for i := range wire.Packages {
					if len(wire.Packages[i].Wheels) > maxUVWheelAlternatives {
						wire.Packages[i].Wheels = wire.Packages[i].Wheels[:maxUVWheelAlternatives]
					}
				}
				body, err = toml.Marshal(wire)
				if err != nil {
					t.Fatal(err)
				}
			}
			lock, err := ParseUVNativeProjectLock(input, body)
			if err != nil {
				t.Fatal("bounded conditional/group lock rejected", err)
			}
			if !lock.rich || len(lock.variants) == 0 || lock.root.Name != input.Project() {
				t.Fatal("universal identities were lost")
			}
			if name == "v2" {
				if lock.variants["idna@3.8"].Name == "" || lock.variants["idna@3.20"].Name == "" {
					t.Fatal("conditional versions collapsed")
				}
			}
		})
	}
}

func TestUVNativeProjectEmptyGroupMetadataDoesNotRequireAnEdgeList(t *testing.T) {
	input, body, _ := uvNativeProjectFixture(t)
	manifest := append(input.Body(), []byte("\n[dependency-groups]\nempty=[]\n[tool.uv]\npackage=false\ndefault-groups=['empty']\n")...)
	input, err := ParseUVProjectInput(manifest, nil, false, UVProjectSync, "")
	if err != nil {
		t.Fatal(err)
	}
	var wire uvNativeProjectLockWire
	if err := toml.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	for i := range wire.Packages {
		if wire.Packages[i].Name == input.Project() {
			wire.Packages[i].DefaultGroups = []string{"empty"}
			wire.Packages[i].Metadata.RequiresDev = map[string][]uvNativeRequirement{"empty": {}}
		}
	}
	changed, err := toml.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseUVNativeProjectLock(input, changed); err != nil {
		t.Fatal("native empty group metadata rejected", err)
	}
	for i := range wire.Packages {
		if wire.Packages[i].Name == input.Project() {
			wire.Packages[i].Metadata.RequiresDev["empty"] = []uvNativeRequirement{{Name: "parent", Specifier: ">=1.0"}}
		}
	}
	changed, err = toml.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseUVNativeProjectLock(input, changed); err == nil {
		t.Fatal("nonempty metadata group accepted without a locked scope")
	}
}
