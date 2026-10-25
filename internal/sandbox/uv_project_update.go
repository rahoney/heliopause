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
	if !update.Valid() || update.HasRequestedResolution() || (update.Snapshot().Context().Mode() != domain.InstallPythonVenv && update.Snapshot().Context().Mode() != domain.InstallPythonProjectControls) || update.Snapshot().Source() != artifactpypi.PublicPyPIProfile().Source() {
		return errors.New("UV project requires an exact complete public Python update")
	}
	input, err := parseUVProjectOriginalControls(update.OriginalControls())
	if err != nil {
		return err
	}
	if update.Snapshot().Context().Mode() == domain.InstallPythonProjectControls && input.Intent() != artifactpypi.UVProjectLockUpgrade {
		return errors.New("UV project control-only update requires lock-upgrade intent")
	}
	selected := update.SelectedControls()
	if len(selected) != 6 {
		return errors.New("UV project selected controls are incomplete")
	}
	byName := map[string]domain.ProjectControlFile{}
	for _, c := range selected {
		if _, exists := byName[c.Name()]; exists || !c.Present() {
			return errors.New("UV project selected control is repeated or absent")
		}
		byName[c.Name()] = c
	}
	for _, c := range update.OriginalControls() {
		if c.Name() == uvProjectRuntimeControlName || c.Name() == artifactpypi.UVProjectIntentControl {
			if !bytes.Equal(c.Body(), byName[c.Name()].Body()) {
				return errors.New("UV project runtime or intent changed")
			}
		}
	}
	after, err := input.SelectManifest(byName[artifactpypi.UVProjectInputControl].Body())
	if err != nil {
		return err
	}
	lock, err := artifactpypi.ParseUVNativeProjectLock(after, byName[artifactpypi.UVNativeLockControl].Body())
	if err != nil {
		return err
	}
	export, err := lock.ParseTargetExport(byName[artifactpypi.UVProjectExportControl].Body())
	if err != nil {
		return err
	}
	if input.Intent() == artifactpypi.UVProjectAdd {
		if err := input.ValidateAddSelection(after, export.Packages()); err != nil {
			return err
		}
	}
	controls, err := export.Controls()
	if err != nil {
		return err
	}
	for _, c := range controls {
		got, ok := byName[c.Name()]
		if !ok || got.Present() != c.Present() || !bytes.Equal(got.Body(), c.Body()) {
			return errors.New("UV project native provenance or derived view changed")
		}
	}
	view, err := artifactpypi.ParseUVProjectLock(byName[artifactpypi.UVSelectionLockControlName].Body(), PinnedPythonRuntime().PythonVersion)
	if err != nil {
		return err
	}
	_, err = view.SelectResolvedWheels(update.Snapshot().Dependencies())
	return err
}
