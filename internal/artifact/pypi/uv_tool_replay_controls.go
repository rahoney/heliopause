package pypi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const UVToolUpgradeWheelsControlName = "uv-tool-upgrade-wheels.json"

// ReplayWheelsControl binds the exact bounded replay WHEEL data to the original
// METADATA digest. It confers no source authenticity or ownership authority.
func (u UVToolUpgradeInput) ReplayWheelsControl(controls []UVInstalledWheelControls, target WheelTarget) (domain.ProjectControlFile, error) {
	checked, err := u.ReplayWheelControls(controls, target)
	if err != nil {
		return domain.ProjectControlFile{}, err
	}
	type wheel struct {
		Directory string `json:"directory"`
		Metadata  string `json:"metadata"`
		Wheel     []byte `json:"wheel"`
	}
	files := make([]wheel, 0, len(checked))
	for _, c := range checked {
		sum := sha256.Sum256(c.Metadata)
		files = append(files, wheel{c.DistInfo, hex.EncodeToString(sum[:]), c.Wheel})
	}
	body, err := json.Marshal(struct {
		Version int     `json:"version"`
		Wheels  []wheel `json:"wheels"`
	}{1, files})
	if err != nil || len(body) > upgradeInputLimit {
		return domain.ProjectControlFile{}, errors.New("tool replay wheel control exceeds byte bounds")
	}
	return domain.NewProjectControlFile(UVToolUpgradeWheelsControlName, body, true)
}

// ReplayWheelControls returns a defensive, canonical copy of inert controls for
// the exact M1 original. It authenticates neither the old archive nor ownership
// and never authorizes the query to execute installed package/startup code.
func (u UVToolUpgradeInput) ReplayWheelControls(controls []UVInstalledWheelControls, target WheelTarget) ([]UVInstalledWheelControls, error) {
	if _, err := u.Control(); err != nil {
		return nil, err
	}
	metadata := u.original.MetadataFiles()
	if len(metadata) == 0 || len(controls) != len(metadata) {
		return nil, errors.New("tool replay requires complete original wheel controls")
	}
	python := strings.Split(u.original.Python(), ".")
	if len(python) != 3 || target.Python != "cp"+python[0]+python[1] || target.ABI != target.Python || target.Platform == "" {
		return nil, errors.New("tool replay target differs from original runtime")
	}
	expected := map[string][]byte{}
	for _, file := range metadata {
		expected[file.Directory] = file.Body
	}
	seen := map[string]UVInstalledWheelControls{}
	total := len(u.receipt.Body())
	for _, source := range controls {
		// Reject raw input before allocating any copy of its payload.
		for _, body := range [][]byte{source.Metadata, source.Wheel} {
			if len(body) == 0 || len(body) > defaultWheelMaxMetadata || len(body) > upgradeInputLimit-total {
				return nil, errors.New("tool replay controls exceed metadata bounds")
			}
			total += len(body)
		}
		control := UVInstalledWheelControls{DistInfo: source.DistInfo, Metadata: bytes.Clone(source.Metadata), Wheel: bytes.Clone(source.Wheel)}
		if _, ok := seen[control.DistInfo]; ok || !bytes.Equal(expected[control.DistInfo], control.Metadata) {
			return nil, errors.New("tool replay controls duplicate or change original metadata")
		}
		parsed, err := readUVInstalledWheelControl(control)
		if err != nil || len(parsed.Tags) == 0 || len(parsed.Tags) > maxUVWheelAlternatives {
			return nil, errors.New("tool replay WHEEL controls are invalid")
		}
		compatible := false
		tags := map[string]bool{}
		for _, tag := range parsed.Tags {
			if len(tag) > 160 || tags[tag] || !validExpandedWheelTag(tag) {
				return nil, errors.New("tool replay wheel tag is invalid or duplicated")
			}
			tags[tag] = true
			parts := strings.Split(tag, "-")
			compatible = compatible || wheelTagsCompatible([]string{parts[0]}, []string{parts[1]}, []string{parts[2]}, target)
		}
		if !compatible {
			return nil, errors.New("tool replay wheel is incompatible with original runtime")
		}
		seen[control.DistInfo] = control
	}
	result := make([]UVInstalledWheelControls, 0, len(metadata))
	for _, file := range metadata {
		control, ok := seen[file.Directory]
		if !ok {
			return nil, errors.New("tool replay original distribution is missing")
		}
		result = append(result, control)
	}
	return result, nil
}
