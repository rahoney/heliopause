package pypi

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestPipxPreQueryGateBindsSameWholeOriginalGraphWithoutReceipt(t *testing.T) {
	lock, inputs, pages := toolTestSelection(t, toolTestWheel(t, "haa-tool-fixture", "child>=1", toolTestEntrypoints), toolTestWheel(t, "child", "", ""))
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	ordinary, err := lock.BindWheels(inputs, pages, target, DefaultWheelLimits())
	if err != nil {
		t.Fatal(err)
	}
	checked, err := lock.BindPipxQueryWheels(inputs, []SimpleProject{pages[1], pages[0]}, target, DefaultWheelLimits())
	if err != nil || !reflect.DeepEqual(checked, ordinary) || len(checked.Nodes()) != 2 {
		t.Fatal("pre-query gate changed authenticated original graph", err)
	}
}

func TestPipxPreQueryGateRejectsStartupAcrossRootAndDependencyBeforeReceipt(t *testing.T) {
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	for _, project := range []string{"haa-tool-fixture", "child"} {
		for _, name := range []string{"startup.pth", "sitecustomize.py", "usercustomize.py", "sitecustomize/__init__.py", "child-1.0.0.data/purelib/sitecustomize.py"} {
			t.Run(project+"/"+name, func(t *testing.T) {
				var rootExtra, childExtra []wheelTestEntry
				if project == "haa-tool-fixture" {
					name = strings.Replace(name, "child-1.0.0.data", "haa_tool_fixture-1.0.0.data", 1)
					rootExtra = []wheelTestEntry{{name: name, body: []byte("import os\n")}}
				} else {
					childExtra = []wheelTestEntry{{name: name, body: []byte("import os\n")}}
				}
				lock, inputs, pages := toolTestSelection(t, toolTestWheel(t, "haa-tool-fixture", "child>=1", toolTestEntrypoints, rootExtra...), toolTestWheel(t, "child", "", "", childExtra...))
				if _, err := lock.BindWheels(inputs, pages, target, DefaultWheelLimits()); err != nil {
					t.Fatal("pipx gate changed ordinary UV binding", err)
				}
				if result, err := lock.BindPipxQueryWheels(inputs, pages, target, DefaultWheelLimits()); err == nil || len(result.Nodes()) != 0 {
					t.Fatal("startup surface accepted before native receipt exists")
				}
			})
		}
	}
	child := toolTestWheel(t, "child", "", "", wheelTestEntry{name: "child/sitecustomize.py", body: []byte("# ordinary nested module\n")})
	lock, inputs, pages := toolTestSelection(t, toolTestWheel(t, "haa-tool-fixture", "child>=1", toolTestEntrypoints), child)
	if _, err := lock.BindPipxQueryWheels(inputs, pages, target, DefaultWheelLimits()); err != nil {
		t.Fatal("ordinary nested module mistaken for startup", err)
	}
}

func TestPipxPreQueryGateRetainsOriginalAuthenticationSourceAndBounds(t *testing.T) {
	for _, fault := range []string{"missing input", "duplicate input", "missing page", "duplicate page", "changed bytes", "size", "source hash", "yanked", "requires Python", "root without dependency", "wrong dependency version", "expansion bound", "target"} {
		t.Run(fault, func(t *testing.T) {
			requirement := "child>=1"
			if fault == "root without dependency" {
				requirement = ""
			}
			if fault == "wrong dependency version" {
				requirement = "child>=2"
			}
			lock, inputs, pages := toolTestSelection(t, toolTestWheel(t, "haa-tool-fixture", requirement, toolTestEntrypoints), toolTestWheel(t, "child", "", ""))
			target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
			limits := DefaultWheelLimits()
			switch fault {
			case "missing input":
				inputs = inputs[:1]
			case "duplicate input":
				inputs[1] = inputs[0]
			case "missing page":
				pages = pages[:1]
			case "duplicate page":
				pages[1] = pages[0]
			case "changed bytes":
				inputs[1].Reader = bytes.NewReader(bytes.Repeat([]byte("!"), int(inputs[1].Size)))
			case "size":
				inputs[1].Size--
			case "source hash":
				pages[1].files[0].sha256 = strings.Repeat("0", 64)
			case "yanked":
				pages[1].files[0].yanked = true
			case "requires Python":
				pages[1].files[0].requiresPython = ">=4.0"
			case "expansion bound":
				limits.MaxFiles = 1
			case "target":
				target.Python = "cp313"
			}
			if result, err := lock.BindPipxQueryWheels(inputs, pages, target, limits); err == nil || len(result.Nodes()) != 0 {
				t.Fatal("incomplete/changed/unbound input accepted")
			}
		})
	}
	if result, err := (UVLock{}).BindPipxQueryWheels(nil, nil, WheelTarget{}, DefaultWheelLimits()); err == nil || len(result.Nodes()) != 0 {
		t.Fatal("zero lock accepted")
	}
}
