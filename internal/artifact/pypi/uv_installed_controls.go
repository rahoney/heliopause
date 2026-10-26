package pypi

import (
	"encoding/json"
	"errors"
)

// UVInstalledWheelControls are untrusted data from a native UV target. They
// select archive candidates only; subsequent archive binding must authenticate
// each one before a single-root graph or complete snapshot can be returned.
type UVInstalledWheelControls struct {
	DistInfo        string
	Metadata, Wheel []byte
}

// SelectInstalledWheelControls reuses the wheel metadata grammar and the native
// snapshot selector without importing anything from the installed environment.
func (l UVLock) SelectInstalledWheelControls(controls []UVInstalledWheelControls, target WheelTarget) ([]UVLockedWheel, error) {
	return selectUVInstalledWheelControls(l.packages, l.python, controls, target, false)
}

func selectUVInstalledWheelControls(packages []UVLockedPackage, python string, controls []UVInstalledWheelControls, target WheelTarget, allowEmpty bool) ([]UVLockedWheel, error) {
	if controls == nil || (!allowEmpty && len(packages) == 0) || len(controls) != len(packages) {
		return nil, errors.New("uv installed controls are incomplete")
	}
	snapshot := make([]uvInstalledWheelSnapshot, 0, len(controls))
	var total int
	for _, control := range controls {
		for _, body := range [][]byte{control.Metadata, control.Wheel} {
			if len(body) == 0 || len(body) > defaultWheelMaxMetadata || len(body) > maxReportBytes-total {
				return nil, errors.New("uv installed controls exceed metadata bounds")
			}
			total += len(body)
		}
		value, err := readUVInstalledWheelControl(control)
		if err != nil {
			return nil, err
		}
		snapshot = append(snapshot, value)
	}
	body, err := json.Marshal(snapshot)
	if err != nil {
		return nil, errors.New("uv installed snapshot encoding failed")
	}
	return selectUVInstalledWheels(packages, python, body, target, allowEmpty)
}

type uvInstalledWheelSnapshot struct {
	Name    string   `json:"name"`
	Version string   `json:"version"`
	Tags    []string `json:"wheel_tags"`
}

// The existing per-control header grammar is shared without granting archive
// authentication. Callers retain their own aggregate and selection boundaries.
func readUVInstalledWheelControl(control UVInstalledWheelControls) (uvInstalledWheelSnapshot, error) {
	metadata, err := parseRFC822Metadata(control.Metadata, defaultWheelMaxMetadata)
	if err != nil || len(metadata.all("name")) != 1 || len(metadata.all("version")) != 1 {
		return uvInstalledWheelSnapshot{}, errors.New("uv installed metadata identity is invalid")
	}
	rawName, _ := metadata.first("name")
	name, err := NormalizeProjectName(rawName)
	version, _ := metadata.first("version")
	if err != nil || control.DistInfo != wheelDistInfo(name, version) {
		return uvInstalledWheelSnapshot{}, errors.New("uv installed dist-info identity differs from metadata")
	}
	wheel, err := parseRFC822Metadata(control.Wheel, defaultWheelMaxMetadata)
	if err != nil || len(wheel.all("wheel-version")) != 1 {
		return uvInstalledWheelSnapshot{}, errors.New("uv installed WHEEL header is invalid")
	}
	wheelVersion, _ := wheel.first("wheel-version")
	if !supportedWheelVersion(wheelVersion) {
		return uvInstalledWheelSnapshot{}, errors.New("uv installed WHEEL version is unsupported")
	}
	return uvInstalledWheelSnapshot{name, version, wheel.all("tag")}, nil
}
