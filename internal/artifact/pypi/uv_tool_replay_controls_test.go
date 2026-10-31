package pypi

import (
	"bytes"
	"strings"
	"testing"
)

func toolReplayControlsFixture(t *testing.T) (UVToolUpgradeInput, []UVInstalledWheelControls) {
	t.Helper()
	receipt, files := uvToolUpgradeFixture(t)
	request, _ := ParseUpgradeRequirement("haa-tool-fixture>=1.1,<2")
	input, err := NewUVToolUpgradeInput(receipt, request, files, "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	return input, []UVInstalledWheelControls{{DistInfo: files[0].Directory, Metadata: bytes.Clone(files[0].Body), Wheel: []byte("Wheel-Version: 1.0\nTag: py3-none-any\n")}}
}

func TestUVToolReplayControlsBindExactOriginalWithoutTrustingArchiveClaims(t *testing.T) {
	input, controls := toolReplayControlsFixture(t)
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	checked, err := input.ReplayWheelControls(controls, target)
	if err != nil || len(checked) != 1 {
		t.Fatal(err)
	}
	if checked[0].DistInfo != input.Original().Installed()[0].Directory() {
		t.Fatal("original directory identity changed")
	}
	controls[0].Metadata[0] = 'x'
	controls[0].Wheel[0] = 'x'
	if checked[0].Metadata[0] == 'x' || checked[0].Wheel[0] == 'x' {
		t.Fatal("caller changed replay data")
	}
}

func TestUVToolReplayControlsRejectPartialChangedAndUnsupportedOriginal(t *testing.T) {
	for name, mutate := range map[string]func([]UVInstalledWheelControls) []UVInstalledWheelControls{
		"missing": func(c []UVInstalledWheelControls) []UVInstalledWheelControls { return nil },
		"extra":   func(c []UVInstalledWheelControls) []UVInstalledWheelControls { return append(c, c[0]) },
		"different metadata": func(c []UVInstalledWheelControls) []UVInstalledWheelControls {
			c[0].Metadata = bytes.ReplaceAll(c[0].Metadata, []byte("1.0.0"), []byte("1.1.0"))
			return c
		},
		"different directory": func(c []UVInstalledWheelControls) []UVInstalledWheelControls {
			c[0].DistInfo = "../other.dist-info"
			return c
		},
		"missing WHEEL": func(c []UVInstalledWheelControls) []UVInstalledWheelControls { c[0].Wheel = nil; return c },
		"future WHEEL": func(c []UVInstalledWheelControls) []UVInstalledWheelControls {
			c[0].Wheel = bytes.ReplaceAll(c[0].Wheel, []byte("1.0"), []byte("99.0"))
			return c
		},
		"no tag": func(c []UVInstalledWheelControls) []UVInstalledWheelControls {
			c[0].Wheel = []byte("Wheel-Version: 1.0\n")
			return c
		},
		"duplicate tag": func(c []UVInstalledWheelControls) []UVInstalledWheelControls {
			c[0].Wheel = append(c[0].Wheel, []byte("Tag: py3-none-any\n")...)
			return c
		},
		"tag path": func(c []UVInstalledWheelControls) []UVInstalledWheelControls {
			c[0].Wheel = []byte("Wheel-Version: 1.0\nTag: ../../outside\n")
			return c
		},
		"different platform": func(c []UVInstalledWheelControls) []UVInstalledWheelControls {
			c[0].Wheel = []byte("Wheel-Version: 1.0\nTag: cp314-cp314-win_amd64\n")
			return c
		},
		"oversize WHEEL": func(c []UVInstalledWheelControls) []UVInstalledWheelControls {
			c[0].Wheel = bytes.Repeat([]byte("x"), defaultWheelMaxMetadata+1)
			return c
		},
	} {
		t.Run(name, func(t *testing.T) {
			input, controls := toolReplayControlsFixture(t)
			if result, err := input.ReplayWheelControls(mutate(controls), WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}); err == nil || len(result) != 0 {
				t.Fatal("invalid replay controls accepted")
			}
		})
	}
	input, controls := toolReplayControlsFixture(t)
	if _, err := input.ReplayWheelControls(controls, WheelTarget{"cp313", "cp313", "manylinux_2_36_x86_64"}); err == nil {
		t.Fatal("different Python target accepted")
	}
}

func TestUVToolReplayControlsEnforceCombinedBoundBeforeTransport(t *testing.T) {
	receipt, _ := uvToolUpgradeFixture(t)
	request, _ := ParseUpgradeRequirement("haa-tool-fixture")
	var files []InstalledMetadataFile
	var controls []UVInstalledWheelControls
	for _, project := range []string{"haa-tool-fixture", "child-one", "child-two", "child-three"} {
		dir := strings.ReplaceAll(project, "-", "_") + "-1.0.0.dist-info"
		body := []byte("Metadata-Version: 2.4\nName: " + project + "\nVersion: 1.0.0\n")
		header := []byte("Wheel-Version: 1.0\nTag: py3-none-any\n\n")
		wheel := append(header, bytes.Repeat([]byte("x"), defaultWheelMaxMetadata-len(header))...)
		c := UVInstalledWheelControls{DistInfo: dir, Metadata: body, Wheel: wheel}
		if _, err := readUVInstalledWheelControl(c); err != nil {
			t.Fatal("individually valid bounded fixture", err)
		}
		files = append(files, InstalledMetadataFile{Directory: dir, Body: body})
		controls = append(controls, c)
	}
	input, err := NewUVToolUpgradeInput(receipt, request, files, "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := input.ReplayWheelControls(controls, WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}); err == nil {
		t.Fatal("combined replay metadata bound ignored")
	}
}

func TestUVToolReplayRejectsOversizedWheelBeforeCopy(t *testing.T) {
	input, controls := toolReplayControlsFixture(t)
	controls[0].Wheel = bytes.Repeat([]byte("x"), defaultWheelMaxMetadata+1)
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	result := testing.Benchmark(func(b *testing.B) {
		for range b.N {
			if _, err := input.ReplayWheelControls(controls, target); err == nil {
				b.Fatal("oversized replay accepted")
			}
		}
	})
	t.Logf("oversize_replay_rejection bytes_per_op=%d input_bytes=%d", result.AllocedBytesPerOp(), len(controls[0].Wheel))
	// The tiny original control has a bounded small allocation. An over-limit
	// WHEEL must not allocate another copy of its megabyte payload before refusal.
	if result.AllocedBytesPerOp() > 128<<10 {
		t.Fatal("over-limit WHEEL copied before rejection")
	}
}

func TestUVToolReplayControlBindsOriginalWheelBytes(t *testing.T) {
	input, controls := toolReplayControlsFixture(t)
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	first, err := input.ReplayWheelsControl(controls, target)
	if err != nil {
		t.Fatal(err)
	}
	controls[0].Wheel = append(controls[0].Wheel, []byte("Generator: changed-inert-metadata\n")...)
	second, err := input.ReplayWheelsControl(controls, target)
	if err != nil || first.Name() != UVToolUpgradeWheelsControlName || first.Digest() == second.Digest() {
		t.Fatal("changed original WHEEL did not change provenance", err)
	}
	controls[0].Metadata[0] = 'x'
	if _, err := input.ReplayWheelsControl(controls, target); err == nil {
		t.Fatal("unbound original metadata accepted")
	}
}
