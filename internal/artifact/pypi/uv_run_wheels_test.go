package pypi

import (
	"bytes"
	"reflect"
	"testing"
)

func TestUVRunExtrasBindAuthenticatedWholeGraph(t *testing.T) {
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	for _, marker := range []string{`extra == 'feature'`, `extra == 'Feature'`, `extra == 'feature' and python_version >= '3.14'`} {
		root := toolTestWheel(t, "haa-tool-fixture", "child>=1; "+marker+"\nProvides-Extra: Feature", toolTestEntrypoints)
		lock, inputs, pages := toolTestSelection(t, root, toolTestWheel(t, "child", "", ""))
		request, _ := ParseUVRunRequirement("haa-tool-fixture[feature]==1.0.0")
		graph, err := lock.BindUVRunWheels(request, inputs, pages, target, DefaultWheelLimits())
		if err != nil || len(graph.Nodes()) != 2 || len(graph.Edges()) != 1 {
			t.Fatal("authenticated extra edge did not bind whole graph", err)
		}
		if _, err := lock.BindPipxQueryWheels(inputs, pages, target, DefaultWheelLimits()); err == nil {
			t.Fatal("unrequested extra changed the existing base lane")
		}
	}
	lock, inputs, pages := toolTestSelection(t, toolTestWheel(t, "haa-tool-fixture", "child>=1", toolTestEntrypoints), toolTestWheel(t, "child", "", ""))
	plain, _ := ParseUVRunRequirement("haa-tool-fixture==1.0.0")
	a, err := lock.BindUVRunWheels(plain, inputs, pages, target, DefaultWheelLimits())
	if err != nil {
		t.Fatal(err)
	}
	b, err := lock.BindPipxQueryWheels(inputs, pages, target, DefaultWheelLimits())
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("ordinary graph changed", err)
	}
}

func TestUVRunExtrasRejectUnboundOrUnsafeSelection(t *testing.T) {
	for _, fault := range []string{"unknown", "duplicate declaration", "undeclared", "missing wheel", "extra wheel", "constraint", "source", "bytes", "startup", "unsupported marker", "foreign root", "zero request"} {
		t.Run(fault, func(t *testing.T) {
			requires := "child>=1; extra == 'feature'\nProvides-Extra: feature"
			request, _ := ParseUVRunRequirement("haa-tool-fixture[feature]==1.0.0")
			var extra []wheelTestEntry
			switch fault {
			case "unknown":
				request, _ = ParseUVRunRequirement("haa-tool-fixture[unknown]==1.0.0")
			case "duplicate declaration":
				requires += "\nProvides-Extra: Feature"
			case "undeclared":
				requires = "child>=1; extra == 'feature'"
			case "extra wheel":
				requires = "child>=1; extra == 'other'\nProvides-Extra: feature"
			case "constraint":
				requires = "child>=2; extra == 'feature'\nProvides-Extra: feature"
			case "startup":
				extra = []wheelTestEntry{{name: "startup.pth", body: []byte("import os\n")}}
			case "unsupported marker":
				requires = "child>=1; extra == 'feature' or unknown == 'yes'\nProvides-Extra: feature"
			case "foreign root":
				request, _ = ParseUVRunRequirement("other[feature]==1.0.0")
			case "zero request":
				request = UVRunRequirement{}
			}
			lock, inputs, pages := toolTestSelection(t, toolTestWheel(t, "haa-tool-fixture", requires, toolTestEntrypoints, extra...), toolTestWheel(t, "child", "", ""))
			switch fault {
			case "missing wheel":
				inputs = inputs[:1]
			case "source":
				pages[1].files[0].yanked = true
			case "bytes":
				inputs[1].Reader = bytes.NewReader(make([]byte, inputs[1].Size))
			}
			graph, err := lock.BindUVRunWheels(request, inputs, pages, WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits())
			if err == nil || len(graph.Nodes()) != 0 {
				t.Fatal("unbound/unsafe extra selection returned a graph")
			}
		})
	}
}

func TestUVRunExtrasUnionBaseAndRequestedFeatures(t *testing.T) {
	for _, marker := range []string{`extra != 'feature'`, `extra == 'other'`, `extra == 'feature'`} {
		_, active, err := uvRunDependency("child>=1; "+marker, "3.14.7", []string{"feature", "other"})
		if err != nil || !active {
			t.Fatal("extra removed a base or requested dependency", marker, err)
		}
	}
}
