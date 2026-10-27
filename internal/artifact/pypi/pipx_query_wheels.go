package pypi

import (
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// BindPipxQueryWheels authenticates the complete original wheel input before a
// native pipx metadata query can start an installed interpreter. It reuses the
// existing startup-surface restriction without a fabricated native receipt.
// The returned graph grants no execution admission, Policy or Host ownership.
func (l UVLock) BindPipxQueryWheels(inputs []UVWheelInput, pages []SimpleProject, target WheelTarget, limits WheelLimits) (domain.LockedDependencyGraph, error) {
	root, err := RequestedProject(l.reference)
	if err != nil || root == "" || len(l.packages) == 0 {
		return domain.LockedDependencyGraph{}, errors.New("pipx pre-query input requires a complete single-root selection")
	}
	checked, err := bindUVSelectedWheelsWithSurface(l.packages, l.python, root, inputs, pages, target, limits, validatePipxMetadataSurface)
	if err != nil {
		return domain.LockedDependencyGraph{}, err
	}
	return BuildLockedGraph(l.reference, checked)
}
