package pypi

import (
	"bytes"
	"os"
	"reflect"
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
	// Project the historical format onto the current runtime; keep captured testdata unchanged.
	body = bytes.ReplaceAll(body, []byte("3.14.7"), []byte(runtimeidentity.PythonVersion))
	lock, err := os.ReadFile("testdata/uv-0.13.4-empty-project.uv.lock")
	if err != nil {
		t.Fatal(err)
	}
	// Project the historical format onto the current runtime; keep captured testdata unchanged.
	lock = bytes.ReplaceAll(lock, []byte("3.14.7"), []byte(runtimeidentity.PythonVersion))
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
		"invalid groups":        base + "[dependency-groups]\ndev = 1\n",
		"package true":          base + "[tool.uv]\npackage = true\n",
		"tool without package":  base + "[tool.uv]\n",
		"empty tool":            base + "[tool]\n",
		"wrong runtime":         strings.Replace(base, runtimeidentity.PythonVersion, "3.15.0", 1),
		"python name injection": strings.Replace(base, "=="+runtimeidentity.PythonVersion, "3.14", 1),
		"invalid version":       strings.Replace(base, "0.1.0", "latest", 1),
		"NUL":                   base + "\x00",
		"invalid UTF8":          base + "\xff",
	}
	for _, dependency := range []string{"packaging @ https://example.org/p.whl", "packaging@25.0", "-r requirements.txt", "../local", "git+https://example.org/a", "haa-example"} {
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

func TestUVProjectSelectedScopePreservesDefaultNestedGroupsAndExtras(t *testing.T) {
	body := append(uvVirtualProjectTestInput(`["packaging==25.0"]`), []byte(`
[project.optional-dependencies]
net=["idna==3.10"]
[dependency-groups]
dev=["six==1.17.0",{include-group="lint"}]
lint=["typing-extensions==4.15.0"]
`)...)
	cases := []struct {
		name      string
		selection UVProjectSelection
		want      []string
	}{
		{"default", UVProjectSelection{}, []string{"packaging==25.0", "six==1.17.0", "typing-extensions==4.15.0"}},
		{"no-dev", UVProjectSelection{NoGroups: []string{"dev"}}, []string{"packaging==25.0"}},
		{"only-lint", UVProjectSelection{OnlyGroups: []string{"lint"}}, []string{"typing-extensions==4.15.0"}},
		{"net", UVProjectSelection{Extras: []string{"net"}}, []string{"packaging==25.0", "idna==3.10", "six==1.17.0", "typing-extensions==4.15.0"}},
		{"all-extras", UVProjectSelection{AllExtras: true}, []string{"packaging==25.0", "idna==3.10", "six==1.17.0", "typing-extensions==4.15.0"}},
		{"included-group-not-removed", UVProjectSelection{NoGroups: []string{"lint"}}, []string{"packaging==25.0", "six==1.17.0", "typing-extensions==4.15.0"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			input, err := ParseUVProjectInput(body, nil, false, UVProjectSync, "", c.selection)
			if err != nil {
				t.Fatal(err)
			}
			got, err := input.SelectedRequirements()
			if err != nil || !reflect.DeepEqual(got, c.want) {
				t.Fatal("wrong scope", got, err)
			}
			controls, err := input.Controls()
			if err != nil {
				t.Fatal(err)
			}
			replay, err := ParseUVProjectControls(controls)
			if err != nil {
				t.Fatal(err)
			}
			got, err = replay.SelectedRequirements()
			if err != nil || !reflect.DeepEqual(got, c.want) {
				t.Fatal("scope intent not bound", got, err)
			}
		})
	}
}

func TestUVProjectScopeRejectsUnknownCyclesAndConflicts(t *testing.T) {
	base := uvVirtualProjectTestInput(`[]`)
	for _, tail := range []string{
		`[dependency-groups]
dev=[{include-group="missing"}]
`,
		`[dependency-groups]
dev=[{include-group="lint"}]
lint=[{include-group="dev"}]
`,
		`[dependency-groups]
dev=[{include-group="lint",other="value"}]
lint=[]
`,
		`[project.optional-dependencies]
net=["bad @ https://evil.invalid/a.whl"]
`,
	} {
		if _, err := ParseUVProjectInput(append(append([]byte{}, base...), []byte(tail)...), nil, false, UVProjectSync, ""); err == nil {
			t.Fatal("unsupported source/group admitted", tail)
		}
	}
	for _, selection := range []UVProjectSelection{{Groups: []string{"missing"}}, {Extras: []string{"missing"}}, {Locked: true}, {Frozen: true}, {Locked: true, Frozen: true}, {Groups: []string{"dev"}, OnlyGroups: []string{"dev"}}} {
		if _, err := ParseUVProjectInput(base, nil, false, UVProjectSync, "", selection); err == nil {
			t.Fatal("invalid scope admitted", selection)
		}
	}
}

func TestUVProjectGeneralRequirementsRemainFrozenForNativeSemantics(t *testing.T) {
	for _, dependencies := range []string{`["packaging[extra]"]`, `["packaging;python_version>'3'"]`, `["idna<3.9; python_version<'3.12'","idna>=3.10; python_version>='3.12'"]`} {
		body := uvVirtualProjectTestInput(dependencies)
		input, err := ParseUVProjectInput(body, nil, false, UVProjectSync, "")
		if err != nil {
			t.Fatal(err)
		}
		if !input.NeedsMetadataProjection() {
			t.Fatal("general semantics bypassed the target metadata owner")
		}
		got, err := input.SelectedRequirements()
		if err != nil || len(got) == 0 || !bytes.Equal(input.Body(), body) {
			t.Fatal("requirements lost frozen source", got, err)
		}
		controls, err := input.Controls()
		if err != nil {
			t.Fatal(err)
		}
		replay, err := ParseUVProjectControls(controls)
		if err != nil || !replay.NeedsMetadataProjection() {
			t.Fatal("general source replay lost scope", err)
		}
	}
}
