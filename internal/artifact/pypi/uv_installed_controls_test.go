package pypi

import (
	"bytes"
	"testing"
)

func uvControlFixture() []UVInstalledWheelControls {
	return []UVInstalledWheelControls{
		{"parent-1.0.dist-info", []byte("Metadata-Version: 2.4\nName: parent\nVersion: 1.0\n"), []byte("Wheel-Version: 1.0\nTag: py3-none-any\n")},
		{"child-1.0.dist-info", []byte("Metadata-Version: 2.4\nName: child\nVersion: 1.0\n"), []byte("Wheel-Version: 1.0\nTag: py3-none-any\n")},
	}
}

func TestUVInstalledControlsSelectUnapprovedArchives(t *testing.T) {
	lock, inputs, pages := uvTestSelection(t, "child==1.0")
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	selected, err := lock.SelectInstalledWheelControls(uvControlFixture(), target)
	if err != nil || len(selected) != 2 {
		t.Fatalf("native controls did not select complete set: %v", err)
	}
	// A successful native selection cannot authenticate tampered archive bytes.
	inputs[0].Reader = bytes.NewReader(bytes.Repeat([]byte("x"), int(inputs[0].Size)))
	if _, err := lock.BindWheels(inputs, pages, target, DefaultWheelLimits()); err == nil {
		t.Fatal("installed metadata replaced archive authentication")
	}
}

func TestUVInstalledControlsRejectAmbiguousOrUnsupportedData(t *testing.T) {
	lock, _, _ := uvTestSelection(t, "child==1.0")
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	for name, mutate := range map[string]func([]UVInstalledWheelControls) []UVInstalledWheelControls{
		"missing distribution":   func(c []UVInstalledWheelControls) []UVInstalledWheelControls { return c[:1] },
		"duplicate distribution": func(c []UVInstalledWheelControls) []UVInstalledWheelControls { c[1] = c[0]; return c },
		"wrong directory": func(c []UVInstalledWheelControls) []UVInstalledWheelControls {
			c[0].DistInfo = "other-1.0.dist-info"
			return c
		},
		"duplicate name": func(c []UVInstalledWheelControls) []UVInstalledWheelControls {
			c[0].Metadata = append(c[0].Metadata, []byte("Name: child\n")...)
			return c
		},
		"duplicate version": func(c []UVInstalledWheelControls) []UVInstalledWheelControls {
			c[0].Metadata = append(c[0].Metadata, []byte("Version: 2.0\n")...)
			return c
		},
		"malformed metadata": func(c []UVInstalledWheelControls) []UVInstalledWheelControls {
			c[0].Metadata = []byte("not a header")
			return c
		},
		"metadata byte bound": func(c []UVInstalledWheelControls) []UVInstalledWheelControls {
			c[0].Metadata = bytes.Repeat([]byte("x"), defaultWheelMaxMetadata+1)
			return c
		},
		"missing wheel": func(c []UVInstalledWheelControls) []UVInstalledWheelControls { c[0].Wheel = nil; return c },
		"duplicate wheel version": func(c []UVInstalledWheelControls) []UVInstalledWheelControls {
			c[0].Wheel = append(c[0].Wheel, []byte("Wheel-Version: 2.0\n")...)
			return c
		},
		"unknown wheel version": func(c []UVInstalledWheelControls) []UVInstalledWheelControls {
			c[0].Wheel = []byte("Wheel-Version: 99\nTag: py3-none-any\n")
			return c
		},
		"wrong platform": func(c []UVInstalledWheelControls) []UVInstalledWheelControls {
			c[0].Wheel = []byte("Wheel-Version: 1.0\nTag: cp314-cp314-win_amd64\n")
			return c
		},
		"duplicate tag": func(c []UVInstalledWheelControls) []UVInstalledWheelControls {
			c[0].Wheel = append(c[0].Wheel, []byte("Tag: py3-none-any\n")...)
			return c
		},
	} {
		t.Run(name, func(t *testing.T) {
			if selected, err := lock.SelectInstalledWheelControls(mutate(uvControlFixture()), target); err == nil || len(selected) != 0 {
				t.Fatal("invalid native controls produced a selected set")
			}
		})
	}
}
