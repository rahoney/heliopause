package pypi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func pipxReplayControlsFixture(t *testing.T) (PipxToolUpgradeInput, []UVInstalledWheelControls) {
	t.Helper()
	receipt, files := pipxToolUpgradeFixture(t, nil)
	files[0].Body = append(files[0].Body, []byte("Requires-Dist: tool-dep>=1\n")...)
	files = append(files, InstalledMetadataFile{Directory: "tool_dep-1.0.0.dist-info", Body: []byte("Metadata-Version: 2.4\nName: tool-dep\nVersion: 1.0.0\n")})
	input, err := NewPipxToolUpgradeInput(receipt, files, runtimeidentity.PythonVersion)
	if err != nil {
		t.Fatal(err)
	}
	var controls []UVInstalledWheelControls
	for _, file := range files {
		controls = append(controls, UVInstalledWheelControls{DistInfo: file.Directory, Metadata: bytes.Clone(file.Body), Wheel: []byte("Wheel-Version: 1.0\nTag: py3-none-any\n")})
	}
	return input, controls
}

func TestPipxReplayControlsBindCompleteOriginalAndCanonicalWheelReceipt(t *testing.T) {
	input, controls := pipxReplayControlsFixture(t)
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	reversed := []UVInstalledWheelControls{controls[1], controls[0]}
	checked, err := input.ReplayWheelControls(reversed, target)
	if err != nil || !reflect.DeepEqual(checked, controls) {
		t.Fatal("original census changed by caller order", err)
	}
	control, err := input.ReplayWheelsControl(reversed, target)
	if err != nil || control.Name() != PipxToolUpgradeWheelsControlName || !control.Present() {
		t.Fatal(err)
	}
	var wire struct {
		Version int `json:"version"`
		Wheels  []struct {
			Directory string `json:"directory"`
			Metadata  string `json:"metadata"`
			Wheel     []byte `json:"wheel"`
		} `json:"wheels"`
	}
	if err := json.Unmarshal(control.Body(), &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Version != 1 || len(wire.Wheels) != 2 {
		t.Fatal("incomplete canonical wheel control")
	}
	for i, row := range wire.Wheels {
		sum := sha256.Sum256(controls[i].Metadata)
		if row.Directory != controls[i].DistInfo || row.Metadata != hex.EncodeToString(sum[:]) || !bytes.Equal(row.Wheel, controls[i].Wheel) {
			t.Fatal("original metadata or WHEEL binding changed")
		}
	}
	controls[0].Metadata[0] = '!'
	controls[0].Wheel[0] = '!'
	if checked[0].Metadata[0] == '!' || checked[0].Wheel[0] == '!' {
		t.Fatal("replay aliases caller bytes")
	}
	if _, err := input.ReplayWheelsControl(controls, target); err == nil {
		t.Fatal("changed original controls reused")
	}
}

func TestPipxReplayControlsRejectMissingChangedAndIncompatibleCensus(t *testing.T) {
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	for _, fault := range []string{"missing", "extra", "duplicate", "changed metadata", "bad WHEEL", "wrong ABI", "oversize", "wrong runtime"} {
		t.Run(fault, func(t *testing.T) {
			input, controls := pipxReplayControlsFixture(t)
			platform := target
			switch fault {
			case "missing":
				controls = controls[:1]
			case "extra":
				controls = append(controls, controls[0])
			case "duplicate":
				controls[1] = controls[0]
			case "changed metadata":
				controls[1].Metadata[0] = '!'
			case "bad WHEEL":
				controls[1].Wheel = []byte("Wheel-Version: 99.0\nTag: py3-none-any\n")
			case "wrong ABI":
				controls[1].Wheel = []byte("Wheel-Version: 1.0\nTag: cp314-cp314-win_amd64\n")
			case "oversize":
				controls[1].Wheel = bytes.Repeat([]byte("x"), defaultWheelMaxMetadata+1)
			case "wrong runtime":
				platform.Python = "cp313"
			}
			if result, err := input.ReplayWheelControls(controls, platform); err == nil || len(result) != 0 {
				t.Fatal("invalid replay controls accepted")
			}
		})
	}
	input, controls := pipxReplayControlsFixture(t)
	controls[1].Wheel = nil
	if _, err := input.ReplayWheelsControl(controls, target); err == nil {
		t.Fatal("invalid WHEEL gained canonical control")
	}
	if result, err := (PipxToolUpgradeInput{}).ReplayWheelControls(controls, target); err == nil || len(result) != 0 {
		t.Fatal("zero input reached replay")
	}
}
