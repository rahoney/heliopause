package sandbox

import (
	"bytes"
	"errors"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// ValidateUVProjectUpdate rechecks frozen native provenance at the approved
// cache boundary. It grants no inspection, Evidence or Policy authority.
func ValidateUVProjectUpdate(update domain.ProjectDependencyUpdate) error {
	if len(update.OriginalControls()) != 6 || len(update.SelectedControls()) != 6 {
		return errors.New("UV project public cache requires exactly six controls")
	}
	_, err := ParseUVProjectUpdate(update)
	return err
}

// ParseUVProjectUpdate is shared by inspection and approved cache readback.
// Native target proofs remain data and require authenticated metadata readback.
func ParseUVProjectUpdate(update domain.ProjectDependencyUpdate) (artifactpypi.UVProjectExport, error) {
	invalid := errors.New("UV project requires an exact complete public Python update")
	if !update.Valid() || update.HasRequestedResolution() || (update.Snapshot().Context().Mode() != domain.InstallPythonVenv && update.Snapshot().Context().Mode() != domain.InstallPythonProjectControls) || update.Snapshot().Source() != artifactpypi.PublicPyPIProfile().Source() {
		return artifactpypi.UVProjectExport{}, invalid
	}
	originalControls := update.OriginalControls()
	// Native adoption adds one existing ownership proof around the six-control
	// project. It must remain byte-identical on both sides of this update.
	if len(originalControls) == 7 {
		filtered := make([]domain.ProjectControlFile, 0, 6)
		found := false
		for _, original := range originalControls {
			if original.Name() == "pypi-update-native-ownership.json" {
				matches := 0
				for _, selected := range update.SelectedControls() {
					if selected.Name() == original.Name() && selected.Present() && original.Present() && selected.Digest() == original.Digest() {
						matches++
					}
				}
				if matches != 1 || found {
					return artifactpypi.UVProjectExport{}, invalid
				}
				found = true
			} else {
				filtered = append(filtered, original)
			}
		}
		if !found {
			return artifactpypi.UVProjectExport{}, invalid
		}
		originalControls = filtered
	}
	input, err := parseUVProjectOriginalControls(originalControls)
	if err != nil {
		return artifactpypi.UVProjectExport{}, err
	}
	if update.Snapshot().Context().Mode() == domain.InstallPythonProjectControls && input.Intent() != artifactpypi.UVProjectLockUpgrade {
		return artifactpypi.UVProjectExport{}, errors.New("UV project control-only update requires lock-upgrade intent")
	}
	selected := update.SelectedControls()
	if len(selected) != len(update.OriginalControls()) {
		return artifactpypi.UVProjectExport{}, invalid
	}
	byName := map[string]domain.ProjectControlFile{}
	for _, c := range selected {
		if _, exists := byName[c.Name()]; exists || !c.Present() {
			return artifactpypi.UVProjectExport{}, invalid
		}
		byName[c.Name()] = c
	}
	var intent domain.ProjectControlFile
	for _, c := range update.OriginalControls() {
		if c.Name() == uvProjectRuntimeControlName && !bytes.Equal(c.Body(), byName[c.Name()].Body()) {
			return artifactpypi.UVProjectExport{}, invalid
		}
		if c.Name() == artifactpypi.UVProjectIntentControl {
			intent = c
		}
	}
	after, err := input.SelectManifest(byName[artifactpypi.UVProjectInputControl].Body())
	if err != nil {
		return artifactpypi.UVProjectExport{}, err
	}
	lockBody := byName[artifactpypi.UVNativeLockControl].Body()
	if input.Selection().Locked || input.Selection().Frozen {
		original, present := input.NativeLock()
		if !present || !bytes.Equal(original, lockBody) {
			return artifactpypi.UVProjectExport{}, errors.New("UV fixed lock bytes changed")
		}
	}
	lock, err := artifactpypi.ParseUVNativeProjectLock(after, lockBody)
	if err != nil {
		return artifactpypi.UVProjectExport{}, err
	}
	export, err := lock.ParseTargetExport(byName[artifactpypi.UVProjectExportControl].Body())
	if err != nil {
		return artifactpypi.UVProjectExport{}, err
	}
	if export.NeedsMetadataProjection() {
		export, err = export.RestoreTarget(byName[artifactpypi.UVProjectIntentControl])
		if err != nil {
			return artifactpypi.UVProjectExport{}, err
		}
	} else if !bytes.Equal(intent.Body(), byName[intent.Name()].Body()) {
		return artifactpypi.UVProjectExport{}, invalid
	}
	if input.Intent() == artifactpypi.UVProjectAdd {
		if err := input.ValidateAddSelection(after, export.Packages()); err != nil {
			return artifactpypi.UVProjectExport{}, err
		}
	}
	controls, err := export.Controls()
	if err != nil {
		return artifactpypi.UVProjectExport{}, err
	}
	for _, c := range controls {
		got, ok := byName[c.Name()]
		if !ok || got.Present() != c.Present() || !bytes.Equal(got.Body(), c.Body()) {
			return artifactpypi.UVProjectExport{}, errors.New("UV project native provenance or derived view changed")
		}
	}
	if _, err := export.SelectResolvedWheels(update.Snapshot().Dependencies()); err != nil {
		return artifactpypi.UVProjectExport{}, err
	}
	return export, nil
}
