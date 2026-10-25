package pypi

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

func uvNativeProjectFixture(t *testing.T) (UVProjectInput, []byte, UVProjectLock) {
	t.Helper()
	input, body, export, _, _ := uvNativeProjectFixtureWithArchives(t)
	return input, body, export
}

func uvNativeProjectFixtureWithArchives(t *testing.T) (UVProjectInput, []byte, UVProjectLock, []UVWheelInput, []SimpleProject) {
	t.Helper()
	selected, inputs, pages := uvTestSelection(t, "child>=1.0")
	export, err := ParseUVProjectLock(uvProjectBody(selected.Packages()), "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	input, err := ParseUVProjectInput(uvVirtualProjectTestInput(`["parent>=1.0,<2"]`), nil, false, UVProjectSync, "")
	if err != nil {
		t.Fatal(err)
	}
	// Structural source/SRI fixture, not a claim of native public acquisition.
	body := "version=1\nrevision=5\nrequires-python='==3.14.7'\n"
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
	native, err := os.ReadFile("testdata/uv-0.13.4-empty-project.uv.lock")
	if err != nil {
		t.Fatal(err)
	}
	input, err := ParseUVProjectInput(body, nil, false, UVProjectSync, "")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := ParseUVNativeProjectLock(input, native)
	if err != nil {
		t.Fatal(err)
	}
	export, err := ParseUVProjectLock(uvProjectBody(nil), "3.14.7")
	if err != nil || lock.ValidateTargetExport(export) != nil {
		t.Fatal("explicit empty native root/export combination rejected", err)
	}
	if _, err := ParseUVNativeProjectLock(input, []byte("version=1\nrevision=5\nrequires-python='==3.14.7'\npackage=[]\n")); err == nil {
		t.Fatal("missing virtual root treated as an empty project")
	}
}

func TestUVNativeProjectLockRefusesSourceSchemaGraphAndManifestSubstitution(t *testing.T) {
	input, body, _ := uvNativeProjectFixture(t)
	base := string(body)
	for name, change := range map[string][2]string{
		"future version":          {"version=1\n", "version=2\n"},
		"past revision":           {"revision=5", "revision=3"},
		"runtime":                 {"==3.14.7", "==3.15.0"},
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
		"marker":                  {"dependencies=[{name='child'}]", "dependencies=[{name='child',marker=\"python_version>'3'\"}]"},
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
			changed, err := ParseUVProjectLock(uvProjectBody(mutate(export.Packages())), "3.14.7")
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
