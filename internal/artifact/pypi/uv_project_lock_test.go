package pypi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func uvProjectBody(packages []UVLockedPackage) []byte {
	body := "lock-version='1.0'\ncreated-by='uv'\nrequires-python='>=3.14.7'\n"
	if len(packages) == 0 {
		return []byte(body + "packages=[]\n")
	}
	for _, project := range packages {
		body += fmt.Sprintf("\n[[packages]]\nname=%q\nversion=%q\nwheels=[", project.Name, project.Version)
		for index, wheel := range project.Wheels {
			if index != 0 {
				body += ","
			}
			body += fmt.Sprintf("{url=%q,size=%d,hashes={sha256=%q}}", wheel.URL, wheel.Size, wheel.SHA256)
		}
		body += "]\n"
	}
	return []byte(body)
}

func uvProjectContext(t *testing.T, path string) domain.InstallContext {
	t.Helper()
	target, err := domain.NewInstallTarget(path)
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewPythonVenvInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	return install
}

func uvProjectControls(t *testing.T, lock UVProjectLock) []domain.ProjectControlFile {
	t.Helper()
	native, err := lock.Control()
	if err != nil {
		t.Fatal(err)
	}
	intent, err := domain.NewProjectControlFile("uv-sync-intent.json", []byte(`{"source_kind":"requirements","allow_empty_requirements":false}`), true)
	if err != nil {
		t.Fatal(err)
	}
	return []domain.ProjectControlFile{native, intent}
}

func TestUVCompleteProjectBindsDisconnectedRootsWithoutPrimary(t *testing.T) {
	single, inputs, pages := uvTestSelection(t, "")
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	if _, err := single.BindWheels(inputs, pages, target, DefaultWheelLimits()); err == nil {
		t.Fatal("single-root connected graph invariant was relaxed")
	}
	project, err := ParseUVProjectLock(uvProjectBody(single.Packages()), "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	selected, err := project.SelectInstalledWheelControls(uvControlFixture(), target)
	if err != nil || len(selected) != 2 {
		t.Fatal("complete native census rejected", err)
	}
	install := uvProjectContext(t, "/tmp/uv-complete-fixture")
	result, err := project.BindProject(install, uvProjectControls(t, project), inputs, pages, target, DefaultWheelLimits())
	if err != nil || !result.Valid() || result.DependencyFree() || result.Source() != PublicPyPIProfile().Source() || len(result.Dependencies()) != 2 || result.Context() != install {
		t.Fatal("complete snapshot did not retain both roots", err)
	}
	if result.Dependencies()[0].Identity().Name() != "child" || result.Dependencies()[1].Identity().Name() != "parent" {
		t.Fatal("complete project lost exact selected entries")
	}
}

func TestUVCompleteProjectRequiresFullIndependentClosure(t *testing.T) {
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	for _, requirement := range []string{"child>=1.0", "child>=2.0", "missing>=1.0", "parent>=1.0"} {
		t.Run(requirement, func(t *testing.T) {
			single, inputs, pages := uvTestSelection(t, requirement)
			project, err := ParseUVProjectLock(uvProjectBody(single.Packages()), "3.14.7")
			if err != nil {
				t.Fatal(err)
			}
			_, err = project.BindProject(uvProjectContext(t, "/tmp/uv-complete-fixture"), uvProjectControls(t, project), inputs, pages, target, DefaultWheelLimits())
			if (requirement == "child>=1.0") != (err == nil) {
				t.Fatal("unsupported/incomplete/self closure was accepted", requirement, err)
			}
		})
	}
	single, inputs, pages := uvTestSelection(t, "child>=1.0")
	project, _ := ParseUVProjectLock(uvProjectBody(single.Packages()), "3.14.7")
	install := uvProjectContext(t, "/tmp/uv-complete-fixture")
	controls := uvProjectControls(t, project)
	if _, err := project.BindProject(install, controls, inputs[:1], pages, target, DefaultWheelLimits()); err == nil {
		t.Fatal("partial selected archives accepted")
	}
	inputs[0].Reader = bytes.NewReader(bytes.Repeat([]byte("x"), int(inputs[0].Size)))
	if _, err := project.BindProject(install, controls, inputs, pages, target, DefaultWheelLimits()); err == nil {
		t.Fatal("native metadata replaced archive authentication")
	}
}

func TestUVProjectControlContextRetainsIndependentWheelBinding(t *testing.T) {
	for _, tamper := range []bool{false, true} {
		single, inputs, pages := uvTestSelection(t, "")
		project, err := ParseUVProjectLock(uvProjectBody(single.Packages()), "3.14.7")
		if err != nil {
			t.Fatal(err)
		}
		target, err := domain.NewInstallTarget("/tmp/control-only-project")
		if err != nil {
			t.Fatal(err)
		}
		controls, err := domain.NewPythonProjectControlContext(target)
		if err != nil {
			t.Fatal(err)
		}
		if tamper {
			inputs[0].Reader = bytes.NewReader(bytes.Repeat([]byte("x"), int(inputs[0].Size)))
		}
		result, err := project.BindProject(controls, uvProjectControls(t, project), inputs, pages, WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits())
		if tamper {
			if err == nil {
				t.Fatal("control-only operation bypassed whole archive authentication")
			}
			continue
		}
		if err != nil || result.Context() != controls || len(result.Dependencies()) != 2 {
			t.Fatal("control-only selection lost context or closure", err)
		}
	}
}

func TestUVCompleteEmptyIsExplicitAndDoesNotRelaxSingleRoot(t *testing.T) {
	body := uvProjectBody(nil)
	project, err := ParseUVProjectLock(body, "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	install := uvProjectContext(t, "/tmp/uv-empty-fixture")
	controls := uvProjectControls(t, project)
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	result, err := project.BindProject(install, controls, nil, nil, target, DefaultWheelLimits())
	if err != nil || !result.Valid() || !result.DependencyFree() || len(result.Dependencies()) != 0 || len(result.ControlDigests()) != 2 {
		t.Fatal("explicit empty snapshot rejected", err)
	}
	if _, err := project.SelectInstalledWheelControls([]UVInstalledWheelControls{}, target); err != nil {
		t.Fatal(err)
	}
	if _, err := project.SelectInstalledWheelControls(nil, target); err == nil {
		t.Fatal("missing native census became empty success")
	}
	if _, err := project.SelectInstalledWheelControls(uvControlFixture(), target); err == nil {
		t.Fatal("extra original output became empty success")
	}
	reference, _ := ParseReference("parent@1.0")
	if _, err := ParseUVLock(reference, body, "3.14.7"); err == nil {
		t.Fatal("single-root empty input accepted")
	}
	if _, err := (UVLock{}).BindWheels(nil, nil, target, DefaultWheelLimits()); err == nil {
		t.Fatal("single-root zero input accepted")
	}
	if _, err := (UVProjectLock{}).BindProject(install, controls, nil, nil, target, DefaultWheelLimits()); err == nil {
		t.Fatal("zero project lock became explicit empty")
	}
	for name, input := range map[string][]byte{
		"missing packages": bytes.Replace(body, []byte("packages=[]\n"), nil, 1),
		"null packages":    bytes.Replace(body, []byte("packages=[]"), []byte("packages=null"), 1),
		"future":           bytes.Replace(body, []byte("1.0"), []byte("2.0"), 1),
		"creator":          bytes.Replace(body, []byte("'uv'"), []byte("'other'"), 1),
		"target":           bytes.Replace(body, []byte(">=3.14.7"), []byte(">=4.0"), 1),
		"unknown":          append(append([]byte(nil), body...), []byte("[unknown]\nvalue=1\n")...),
		"empty":            nil,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseUVProjectLock(input, "3.14.7"); err == nil {
				t.Fatal("ambiguous empty selection accepted")
			}
		})
	}
	if _, err := project.BindProject(install, controls, nil, nil, WheelTarget{"cp313", "cp313", target.Platform}, DefaultWheelLimits()); err == nil {
		t.Fatal("empty binding bypassed target verification")
	}
	limits := DefaultWheelLimits()
	limits.MaxCompressed++
	if _, err := project.BindProject(install, controls, nil, nil, target, limits); err == nil {
		t.Fatal("empty binding weakened source resource limits")
	}
	for _, limits := range []WheelLimits{{}, {-1, 1, 1, 1}, {1, 0, 1, 1}, {1, 1, 0, 1}, {1, 1, 1, 0}} {
		if _, err := project.BindProject(install, controls, nil, nil, target, limits); err == nil {
			t.Fatal("empty binding accepted invalid resource limits", limits)
		}
	}
}

func TestUVCompleteApprovalSubjectBindsControlsTargetAndImmutableBytes(t *testing.T) {
	single, inputs, pages := uvTestSelection(t, "child>=1.0")
	body := uvProjectBody(single.Packages())
	project, err := ParseUVProjectLock(body, "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	controls := uvProjectControls(t, project)
	install := uvProjectContext(t, "/tmp/uv-complete-fixture")
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	baseline, err := project.BindProject(install, controls, inputs, pages, target, DefaultWheelLimits())
	if err != nil {
		t.Fatal(err)
	}
	reversed := []domain.ProjectControlFile{controls[1], controls[0]}
	same, err := project.BindProject(install, reversed, inputs, pages, target, DefaultWheelLimits())
	if err != nil || same.GraphDigest() != baseline.GraphDigest() {
		t.Fatal("control ordering changed the selected subject", err)
	}
	changed, _ := domain.NewProjectControlFile(controls[1].Name(), []byte(`{"source_kind":"requirements","allow_empty_requirements":true}`), true)
	different, err := project.BindProject(install, []domain.ProjectControlFile{controls[0], changed}, inputs, pages, target, DefaultWheelLimits())
	if err != nil || different.GraphDigest() == baseline.GraphDigest() {
		t.Fatal("different empty permission reused approval subject", err)
	}
	different, err = project.BindProject(uvProjectContext(t, "/tmp/other-uv-fixture"), controls, inputs, pages, target, DefaultWheelLimits())
	if err != nil || different.GraphDigest() == baseline.GraphDigest() {
		t.Fatal("different target reused approval subject", err)
	}
	absent, _ := domain.NewProjectControlFile("missing", nil, false)
	changedLock, _ := domain.NewProjectControlFile(controls[0].Name(), append(controls[0].Body(), []byte("\n# changed\n")...), true)
	for name, bad := range map[string][]domain.ProjectControlFile{"missing lock": controls[1:], "changed lock": {changedLock, controls[1]}, "duplicate": {controls[0], controls[0]}, "absent": {controls[0], absent}} {
		t.Run(name, func(t *testing.T) {
			if _, err := project.BindProject(install, bad, inputs, pages, target, DefaultWheelLimits()); err == nil {
				t.Fatal("control mismatch accepted")
			}
		})
	}
	before := controls[0].Body()
	body[0] = 'x'
	exported := project.Packages()
	exported[0].Wheels[0].URL = "changed"
	exported[0].Name = "changed"
	control, _ := project.Control()
	copy := control.Body()
	copy[0] = 'y'
	again, _ := project.Control()
	if !bytes.Equal(before, again.Body()) || project.Packages()[0].Name == "changed" || project.Packages()[0].Wheels[0].URL == "changed" {
		t.Fatal("mutable input/export changed frozen native subject")
	}
}

func TestUVCompleteSubjectSeparatesExactWheelAlternatives(t *testing.T) {
	single, inputs, pages := uvTestSelection(t, "child>=1.0")
	packages := single.Packages()
	var original, alternative UVLockedWheel
	for i := range packages {
		if packages[i].Name == "parent" {
			original = packages[i].Wheels[0]
			alternative = original
			alternative.Filename = "parent-1.0-1-py3-none-any.whl"
			alternative.URL = "https://files.pythonhosted.org/packages/" + alternative.Filename
			packages[i].Wheels = append(packages[i].Wheels, alternative)
		}
	}
	wire := map[string]any{"meta": map[string]string{"api-version": "1.0"}, "name": "parent", "files": []any{}}
	for _, wheel := range []UVLockedWheel{original, alternative} {
		wire["files"] = append(wire["files"].([]any), map[string]any{"filename": wheel.Filename, "url": wheel.URL, "hashes": map[string]string{"sha256": wheel.SHA256}, "requires-python": ">=3.8", "yanked": false, "size": wheel.Size})
	}
	encoded, _ := json.Marshal(wire)
	page, err := ParseSimpleProject("parent", encoded)
	if err != nil {
		t.Fatal(err)
	}
	for i := range pages {
		if pages[i].Project() == "parent" {
			pages[i] = page
		}
	}
	project, err := ParseUVProjectLock(uvProjectBody(packages), "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	controls := uvProjectControls(t, project)
	install := uvProjectContext(t, "/tmp/uv-alternatives")
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	first, err := project.BindProject(install, controls, inputs, pages, target, DefaultWheelLimits())
	if err != nil {
		t.Fatal(err)
	}
	secondInputs := append([]UVWheelInput(nil), inputs...)
	for i := range secondInputs {
		if secondInputs[i].URL == original.URL {
			secondInputs[i].URL = alternative.URL
		}
	}
	second, err := project.BindProject(install, controls, secondInputs, pages, target, DefaultWheelLimits())
	if err != nil || first.GraphDigest() == second.GraphDigest() {
		t.Fatal("different exact wheel reused the same lock approval subject", err)
	}
}

func TestUVCompleteNativeFixtureRejectsUnreviewedSourcesAndBudgets(t *testing.T) {
	body, err := os.ReadFile("testdata/uv-0.13.4-requests.pylock.toml")
	if err != nil {
		t.Fatal(err)
	}
	for name, input := range map[string][]byte{
		"foreign URL":     bytes.ReplaceAll(body, []byte("files.pythonhosted.org"), []byte("example.org")),
		"future fields":   append(append([]byte(nil), body...), []byte("\n[packages.directory]\npath='.'\n")...),
		"oversize":        bytes.Repeat([]byte(" "), maxReportBytes+1),
		"download budget": bytes.Replace(body, []byte("size = 136464"), []byte("size = 67108864"), 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseUVProjectLock(input, "3.14.7"); err == nil {
				t.Fatal("unsafe complete native format accepted")
			}
		})
	}
}
