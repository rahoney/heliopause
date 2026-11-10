package pypi

import "github.com/rahoney/heliopause/internal/core/domain"

const PipxToolUpgradeWheelsControlName = "pipx-tool-upgrade-wheels.json"

// ReplayWheelControls binds an exact inert original census, not archive source,
// startup execution permission, Host ownership or inspection approval.
func (u PipxToolUpgradeInput) ReplayWheelControls(controls []UVInstalledWheelControls, target WheelTarget) ([]UVInstalledWheelControls, error) {
	if _, err := u.Control(); err != nil {
		return nil, err
	}
	return replayToolWheelControls(u.receipt, u.original, controls, target)
}

func (u PipxToolUpgradeInput) ReplayWheelsControl(controls []UVInstalledWheelControls, target WheelTarget) (domain.ProjectControlFile, error) {
	checked, err := u.ReplayWheelControls(controls, target)
	if err != nil {
		return domain.ProjectControlFile{}, err
	}
	return newToolReplayWheelsControl(PipxToolUpgradeWheelsControlName, checked)
}
