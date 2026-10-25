package pypi

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func uvActualProjectExportReplay(t *testing.T) (UVNativeProjectLock, []byte) {
	t.Helper()
	read := func(suffix string) []byte {
		t.Helper()
		body, err := os.ReadFile("testdata/uv-0.13.4-project-export-replay." + suffix)
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	input, err := ParseUVProjectInput(read("pyproject.toml"), nil, false, UVProjectSync, "")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := ParseUVNativeProjectLock(input, read("uv.lock"))
	if err != nil {
		t.Fatal(err)
	}
	return lock, read("pylock.toml")
}

func TestUVProjectExportPreservesActualNativeReplayAndStrictP1Reader(t *testing.T) {
	lock, body := uvActualProjectExportReplay(t)
	if _, err := ParseUVProjectLock(body, "3.14.7"); err == nil {
		t.Fatal("existing wheel-only reader unexpectedly accepted project metadata")
	}
	export, err := lock.ParseTargetExport(body)
	if err != nil || len(export.Packages()) != 1 || export.Packages()[0].Name != "certifi" {
		t.Fatal("actual fixed UV export format rejected", err)
	}
	controls, err := export.Controls()
	if err != nil || len(controls) != 4 || controls[2].Name() != UVProjectExportControl || !bytes.Equal(controls[2].Body(), body) || controls[3].Name() != UVSelectionLockControlName || bytes.Equal(controls[3].Body(), body) {
		t.Fatal("actual export confused with derived validation view", err)
	}
	if _, err := ParseUVProjectLock(controls[3].Body(), "3.14.7"); err != nil {
		t.Fatal("derived view did not use unchanged strict parser", err)
	}
	body[0] = '!'
	export.Packages()[0].Wheels[0].URL = "changed"
	again, err := export.Controls()
	if err != nil || again[2].Body()[0] == '!' || export.Packages()[0].Wheels[0].URL == "changed" {
		t.Fatal("mutable returned data changed frozen provenance", err)
	}
}

func TestUVProjectPublicIndexAcceptsOnlyExistingProfileRepresentations(t *testing.T) {
	for _, value := range []string{"https://pypi.org/simple", "https://pypi.org/simple/"} {
		if !uvPublicProjectIndex(value) {
			t.Fatal("existing public index representation rejected", value)
		}
	}
	for _, value := range []string{"https://pypi.org/simple//", "http://pypi.org/simple/", "https://pypi.org/simple/?q=x", "https://pypi.org/simple/#x", "https://user@pypi.org/simple/", "https://pypi.org:443/simple/", "https://example.org/simple/"} {
		if uvPublicProjectIndex(value) {
			t.Fatal("foreign or ambiguous index accepted", value)
		}
	}
	lock, body := uvActualProjectExportReplay(t)
	inputBody, err := os.ReadFile("testdata/uv-0.13.4-project-export-replay.pyproject.toml")
	if err != nil {
		t.Fatal(err)
	}
	input, err := ParseUVProjectInput(inputBody, nil, false, UVProjectSync, "")
	if err != nil {
		t.Fatal(err)
	}
	control, err := lock.Control()
	if err != nil {
		t.Fatal(err)
	}
	native, err := ParseUVNativeProjectLock(input, bytes.ReplaceAll(control.Body(), []byte("https://pypi.org/simple"), []byte("https://pypi.org/simple/")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := native.ParseTargetExport(bytes.ReplaceAll(body, []byte("https://pypi.org/simple"), []byte("https://pypi.org/simple/"))); err != nil {
		t.Fatal("configured public native/export representation rejected", err)
	}
}

func uvProjectExportBindingFixture(t *testing.T) (UVProjectExport, []UVWheelInput, []SimpleProject) {
	t.Helper()
	input, body, originalExport, inputs, pages := uvNativeProjectFixtureWithArchives(t)
	native, err := ParseUVNativeProjectLock(input, body)
	if err != nil {
		t.Fatal(err)
	}
	exportBody := uvProjectBody(originalExport.Packages())
	exportBody = bytes.ReplaceAll(exportBody, []byte("requires-python='>=3.14.7'"), []byte("requires-python='==3.14.7'"))
	exportBody = bytes.ReplaceAll(exportBody, []byte("[[packages]]\n"), []byte("[[packages]]\nindex='https://pypi.org/simple'\n"))
	export, err := native.ParseTargetExport(exportBody)
	if err != nil {
		t.Fatal(err)
	}
	return export, inputs, pages
}

func TestUVProjectExportReusesIndependentWholeArchiveBinding(t *testing.T) {
	export, inputs, pages := uvProjectExportBindingFixture(t)
	controls, err := export.Controls()
	if err != nil {
		t.Fatal(err)
	}
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	wheels, err := export.SelectInstalledWheelControls(uvControlFixture(), target)
	if err != nil || len(wheels) != 2 {
		t.Fatal("existing installed census selection failed", err)
	}
	install := uvProjectContext(t, "/tmp/uv-native-project-binding")
	snapshot, err := export.BindProject(install, controls, inputs, pages, target, DefaultWheelLimits())
	if err != nil || !snapshot.Valid() || len(snapshot.Dependencies()) != 2 || len(snapshot.ControlDigests()) != 4 || snapshot.Context() != install {
		t.Fatal("whole independently authenticated wheel selection failed", err)
	}
	for i := range controls {
		changed := append([]domain.ProjectControlFile(nil), controls...)
		body := changed[i].Body()
		body = append(body, '!')
		changed[i], err = domain.NewProjectControlFile(changed[i].Name(), body, true)
		if err != nil {
			t.Fatal(err)
		}
		if result, err := export.BindProject(install, changed, inputs, pages, target, DefaultWheelLimits()); err == nil || result.Valid() {
			t.Fatal("substituted provenance accepted", controls[i].Name())
		}
		missing := append([]domain.ProjectControlFile(nil), controls[:i]...)
		missing = append(missing, controls[i+1:]...)
		if _, err := export.BindProject(install, missing, inputs, pages, target, DefaultWheelLimits()); err == nil {
			t.Fatal("missing provenance accepted", controls[i].Name())
		}
	}
	inputs[0].Reader = bytes.NewReader(bytes.Repeat([]byte("x"), int(inputs[0].Size)))
	if result, err := export.BindProject(install, controls, inputs, pages, target, DefaultWheelLimits()); err == nil || result.Valid() {
		t.Fatal("native graph replaced independent whole archive authentication")
	}
}

func TestUVProjectExportRejectsMetadataSourceAndSelectionTampering(t *testing.T) {
	lock, body := uvActualProjectExportReplay(t)
	base := string(body)
	for name, change := range map[string][2]string{
		"index":         {`index = "https://pypi.org/simple"`, `index = "https://example.org/simple"`},
		"missing index": {`index = "https://pypi.org/simple"`, ""},
		"runtime":       {`requires-python = "==3.14.7"`, `requires-python = ">=3.14.7"`},
		"creator":       {`created-by = "uv"`, `created-by = "other"`},
		"schema":        {`lock-version = "1.0"`, `lock-version = "2.0"`},
		"sdist hash":    {strings.Repeat("a", 64), strings.Repeat("b", 64)},
		"sdist size":    {`size = 100,`, `size = 101,`},
		"sdist source":  {`/packages/certifi-`, `/packages/other/certifi-`},
		"wheel hash":    {`42d35340cc5b9ba78e818ddbff0c5b1c2b779de448a340fd1b989f5fa8670e1b`, strings.Repeat("f", 64)},
		"unknown field": {`index = `, `unreviewed = true` + "\nindex = "},
	} {
		t.Run(name, func(t *testing.T) {
			changed := strings.ReplaceAll(base, change[0], change[1])
			if changed == base {
				t.Fatal("negative fixture unchanged")
			}
			if _, err := lock.ParseTargetExport([]byte(changed)); err == nil {
				t.Fatal("tampered project export accepted")
			}
		})
	}
	if _, err := lock.ParseTargetExport(nil); err == nil {
		t.Fatal("missing export accepted")
	}
	if _, err := (UVNativeProjectLock{}).ParseTargetExport(body); err == nil {
		t.Fatal("zero native lock accepted")
	}
	if _, err := (UVProjectExport{}).Controls(); err == nil {
		t.Fatal("zero export produced controls")
	}
	if _, err := (UVProjectExport{}).SelectInstalledWheelControls(nil, WheelTarget{}); err == nil {
		t.Fatal("zero export selected installed census")
	}
}
