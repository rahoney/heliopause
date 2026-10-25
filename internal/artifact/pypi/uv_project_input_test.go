package pypi

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestUVProjectInputAcceptsPinnedNativeEmptyVirtualProject(t *testing.T) {
	body, err := os.ReadFile("testdata/uv-0.13.4-empty-project.pyproject.toml")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := os.ReadFile("testdata/uv-0.13.4-empty-project.uv.lock")
	if err != nil {
		t.Fatal(err)
	}
	input, err := ParseUVProjectInput(body, lock, true, UVProjectLockUpgrade, "")
	if err != nil || input.Project() != "haa-empty-project" || len(input.Requirements()) != 0 {
		t.Fatal("pinned native virtual project input rejected", err)
	}
	controls, err := input.Controls()
	if err != nil || !bytes.Equal(controls[0].Body(), body) || !bytes.Equal(controls[1].Body(), lock) {
		t.Fatal("native original input bytes changed", err)
	}
}

func uvVirtualProjectTestInput(dependencies string) []byte {
	return []byte("[project]\nname = \"haa-example\"\nversion = \"0.1.0\"\nrequires-python = \"==" + runtimeidentity.PythonVersion + "\"\ndependencies = " + dependencies + "\n")
}

func TestUVProjectInputFreezesVirtualInputAndDistinctLockAbsence(t *testing.T) {
	body := uvVirtualProjectTestInput(`["packaging>=24,<26", "idna==3.10"]`)
	for _, intent := range []UVProjectIntent{UVProjectAdd, UVProjectSync, UVProjectLockUpgrade} {
		t.Run(string(intent), func(t *testing.T) {
			selector := ""
			if intent == UVProjectAdd {
				selector = "Charset_Normalizer>=3,<4"
			}
			lock := []byte("version = 1\nrevision = 5\n")
			input, err := ParseUVProjectInput(body, lock, true, intent, selector)
			if err != nil || !input.Valid() || input.Project() != "haa-example" || input.Version() != "0.1.0" || input.Intent() != intent || len(input.Requirements()) != 2 {
				t.Fatal("valid virtual project rejected", input, err)
			}
			if intent == UVProjectAdd && input.Selector() != "charset-normalizer>=3,<4" {
				t.Fatal("add selection did not retain normalized bounded requirement")
			}
			controls, err := input.Controls()
			if err != nil || len(controls) != 3 || !controls[1].Present() || !bytes.Equal(controls[0].Body(), body) {
				t.Fatal("frozen control identity lost", err)
			}
			controls[0], controls[2] = controls[2], controls[0]
			replay, err := ParseUVProjectControls(controls)
			if err != nil || !bytes.Equal(replay.Body(), body) || replay.Intent() != intent {
				t.Fatal("control-order independent replay failed", err)
			}
			input.Body()[0] = '!'
			copied, _ := input.NativeLock()
			copied[0] = '!'
			input.Requirements()[0] = UpgradeRequirement{}
			copied, _ = input.NativeLock()
			if !bytes.Equal(input.Body(), body) || !bytes.Equal(copied, lock) || input.Requirements()[0].Project() != "packaging" {
				t.Fatal("returned slice mutated frozen input")
			}
		})
	}
	empty := uvVirtualProjectTestInput(`[]`)
	empty = append(empty, []byte("[tool.uv]\npackage = false\n")...)
	input, err := ParseUVProjectInput(empty, nil, false, UVProjectSync, "")
	if err != nil || len(input.Requirements()) != 0 {
		t.Fatal("explicit empty virtual project rejected", err)
	}
	controls, err := input.Controls()
	if err != nil || controls[1].Present() || len(controls[1].Body()) != 0 {
		t.Fatal("absent original lock fabricated", err)
	}
	if _, err := ParseUVProjectControls(controls); err != nil {
		t.Fatal("explicit empty input replay failed", err)
	}
}

func TestUVProjectInputRefusesUnsupportedSchemaAndDependencies(t *testing.T) {
	base := string(uvVirtualProjectTestInput(`[]`))
	cases := map[string]string{
		"missing project":       "",
		"missing dependencies":  strings.Replace(base, "dependencies = []\n", "", 1),
		"wrong dependency type": strings.Replace(base, "[]", `"packaging"`, 1),
		"duplicate key":         base + "dependencies = []\n",
		"unknown root":          base + "[unknown]\nvalue = true\n",
		"workspace":             base + "[tool.uv.workspace]\nmembers = []\n",
		"source override":       base + "[tool.uv.sources]\npackaging = { path = \"/host\" }\n",
		"custom index":          base + "[[tool.uv.index]]\nurl = \"https://example.org\"\n",
		"build system":          base + "[build-system]\nrequires = []\n",
		"scripts":               base + "[project.scripts]\nprobe = \"probe:main\"\n",
		"groups":                base + "[dependency-groups]\ndev = []\n",
		"package true":          base + "[tool.uv]\npackage = true\n",
		"tool without package":  base + "[tool.uv]\n",
		"empty tool":            base + "[tool]\n",
		"wrong runtime":         strings.Replace(base, runtimeidentity.PythonVersion, "3.15.0", 1),
		"python name injection": strings.Replace(base, "=="+runtimeidentity.PythonVersion, "3.14", 1),
		"invalid version":       strings.Replace(base, "0.1.0", "latest", 1),
		"NUL":                   base + "\x00",
		"invalid UTF8":          base + "\xff",
	}
	for _, dependency := range []string{"packaging @ https://example.org/p.whl", "packaging@25.0", "packaging[extra]", "packaging;python_version>'3'", "-r requirements.txt", "../local", "git+https://example.org/a", "haa-example"} {
		cases[dependency] = string(uvVirtualProjectTestInput(`["` + dependency + `"]`))
	}
	cases["normalized duplicate"] = string(uvVirtualProjectTestInput(`["charset_normalizer==3.4.0", "charset-normalizer==3.4.0"]`))
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseUVProjectInput([]byte(body), nil, false, UVProjectSync, ""); err == nil {
				t.Fatal("unsupported project accepted")
			}
		})
	}
}

func TestUVProjectIntentAndFrozenControlsRejectTampering(t *testing.T) {
	body := uvVirtualProjectTestInput(`[]`)
	for _, test := range []struct {
		intent   UVProjectIntent
		selector string
		lock     []byte
		present  bool
	}{
		{UVProjectSync, "packaging", nil, false},
		{UVProjectLockUpgrade, "", nil, false},
		{UVProjectAdd, "", nil, false},
		{UVProjectAdd, "haa-example", nil, false},
		{UVProjectAdd, "idna@3.10", nil, false},
		{UVProjectSync, "", []byte("lock"), false},
		{UVProjectSync, "", nil, true},
		{"run", "", nil, false},
	} {
		if _, err := ParseUVProjectInput(body, test.lock, test.present, test.intent, test.selector); err == nil {
			t.Fatalf("invalid intent/presence accepted: %+v", test)
		}
	}
	input, err := ParseUVProjectInput(body, nil, false, UVProjectSync, "")
	if err != nil {
		t.Fatal(err)
	}
	controls, err := input.Controls()
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func([]domain.ProjectControlFile) []domain.ProjectControlFile{
		"missing":   func(c []domain.ProjectControlFile) []domain.ProjectControlFile { return c[:2] },
		"duplicate": func(c []domain.ProjectControlFile) []domain.ProjectControlFile { c[1] = c[0]; return c },
		"unknown": func(c []domain.ProjectControlFile) []domain.ProjectControlFile {
			c[1], _ = domain.NewProjectControlFile("uv.toml", nil, false)
			return c
		},
		"absent project": func(c []domain.ProjectControlFile) []domain.ProjectControlFile {
			c[0], _ = domain.NewProjectControlFile(UVProjectInputControl, nil, false)
			return c
		},
		"runtime": func(c []domain.ProjectControlFile) []domain.ProjectControlFile {
			c[2], _ = domain.NewProjectControlFile(UVProjectIntentControl, bytes.Replace(c[2].Body(), []byte(runtimeidentity.PythonVersion), []byte("3.15.0"), 1), true)
			return c
		},
		"extra JSON": func(c []domain.ProjectControlFile) []domain.ProjectControlFile {
			b := c[2].Body()
			b = append(b[:len(b)-1], []byte(`,"unknown":true}`)...)
			c[2], _ = domain.NewProjectControlFile(UVProjectIntentControl, b, true)
			return c
		},
		"duplicate JSON": func(c []domain.ProjectControlFile) []domain.ProjectControlFile {
			b := c[2].Body()
			b = append(b[:len(b)-1], []byte(`,"version":1}`)...)
			c[2], _ = domain.NewProjectControlFile(UVProjectIntentControl, b, true)
			return c
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseUVProjectControls(mutate(append([]domain.ProjectControlFile(nil), controls...))); err == nil {
				t.Fatal("tampered controls accepted")
			}
		})
	}
	if _, err := (UVProjectInput{}).Controls(); err == nil {
		t.Fatal("zero input produced controls")
	}
}
