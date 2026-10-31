package pypi

import (
	"bytes"
	"errors"

	"github.com/pelletier/go-toml/v2"
)

// ValidateUVProjectOriginalLock validates historical lock input without
// requiring its old direct requirements to equal an edited current manifest.
// The derived old-requirement view is discarded. Selected lock/export/current
// manifest validation must still use ParseUVNativeProjectLock unchanged.
func ValidateUVProjectOriginalLock(input UVProjectInput, body []byte) error {
	if !input.Valid() || !validUVProjectText(body, false) {
		return errors.New("UV original project lock is unavailable")
	}
	var wire uvNativeProjectLockWire
	if err := toml.NewDecoder(bytes.NewReader(body)).DisallowUnknownFields().Decode(&wire); err != nil {
		return err
	}
	if len(wire.Packages) == 0 || len(wire.Packages) > PublicPyPIProfile().ResourcePolicy().MaxGraphArtifacts()+1 {
		return errors.New("UV original project lock package count is unsupported")
	}
	roots := 0
	var root uvNativePackage
	for _, item := range wire.Packages {
		if item.Source.Virtual != nil {
			if *item.Source.Virtual != "." || item.Name != input.project || item.Version != input.version {
				return errors.New("UV original virtual project identity changed")
			}
			roots++
			root = item
		}
	}
	if roots != 1 {
		return errors.New("UV original project lock requires one virtual root")
	}
	requirements := []string{}
	if root.Metadata != nil {
		if len(root.Metadata.RequiresDist) > PublicPyPIProfile().ResourcePolicy().MaxGraphArtifacts() {
			return errors.New("UV original project requirements exceed bounds")
		}
		for _, item := range root.Metadata.RequiresDist {
			requirements = append(requirements, item.Name+item.Specifier)
		}
	}
	var manifest uvProjectInputWire
	if err := toml.NewDecoder(bytes.NewReader(input.Body())).DisallowUnknownFields().Decode(&manifest); err != nil || manifest.Project == nil {
		return errors.New("UV original project manifest is unavailable")
	}
	manifest.Project.Dependencies = &requirements
	view, err := toml.Marshal(manifest)
	if err != nil {
		return err
	}
	previous, err := ParseUVProjectInput(view, nil, false, UVProjectSync, "")
	if err != nil {
		return err
	}
	_, err = ParseUVNativeProjectLock(previous, body)
	return err
}
