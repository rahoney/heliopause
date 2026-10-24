package npm

import (
	"errors"
	"slices"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// PNPMInspectionWorkspaceLimit is the fixed inspection workspace capacity.
// Both native and Artifact project volumes consume this one owner. It is not
// a release version, a user override or permission to exceed runtime budgets.
const PNPMInspectionWorkspaceLimit = 128 << 20

var errPNPMInspectionBudget = errors.New("pnpm complete inspection input exceeds its fixed workspace budget")

func (i PNPMInspectionInput) indexInventories(inventories []ArchiveInventory) (map[domain.AcquiredArtifact]ArchiveInventory, error) {
	if !i.Valid() || len(i.artifacts) == 0 || len(inventories) != len(i.artifacts) {
		return nil, errPNPMInspectionLayout
	}
	bySubject := make(map[domain.AcquiredArtifact]ArchiveInventory, len(inventories))
	for _, inventory := range inventories {
		if !inventory.Valid() || !slices.Contains(i.artifacts, inventory.artifact) {
			return nil, errPNPMInspectionLayout
		}
		if _, duplicate := bySubject[inventory.artifact]; duplicate {
			return nil, errPNPMInspectionLayout
		}
		bySubject[inventory.artifact] = inventory
	}
	return bySubject, nil
}

// CheckWorkspace admits only complete, exact static inputs and charges archive
// bytes, the unique expanded store and every logical peer instance. Native
// hardlink deduplication never increases this authorization. Generated metadata
// and bins still live under the hard volume ceiling; admission is not proof
// that native preparation can complete, nor a source or Policy approval.
func (i PNPMInspectionInput) CheckWorkspace(inventories []ArchiveInventory) error {
	bySubject, err := i.indexInventories(inventories)
	if err != nil {
		return err
	}
	type footprint struct {
		bytes   uint64
		entries int
	}
	byPackage := make(map[string]footprint, len(bySubject))
	var charged uint64
	uniqueEntries := 0
	charge := func(amount uint64) bool {
		if amount > PNPMInspectionWorkspaceLimit-charged {
			return false
		}
		charged += amount
		return true
	}
	for subject, inventory := range bySubject {
		if !charge(subject.SizeBytes()) || len(inventory.entries) > pnpmLayoutEntryLimit-uniqueEntries {
			return errPNPMInspectionBudget
		}
		uniqueEntries += len(inventory.entries)
		var expanded uint64
		for _, entry := range inventory.entries {
			if !entry.Directory {
				if !charge(uint64(entry.Size)) {
					return errPNPMInspectionBudget
				}
				expanded += uint64(entry.Size)
			}
		}
		byPackage[subject.Identity().Name()+"@"+subject.Identity().Version()] = footprint{expanded, len(inventory.entries)}
	}
	instanceEntries := 0
	for _, instance := range i.context.selection.lock.snapshots {
		footprint, present := byPackage[instance.Package]
		if !present || footprint.entries > pnpmLayoutEntryLimit-instanceEntries || !charge(footprint.bytes) {
			return errPNPMInspectionBudget
		}
		instanceEntries += footprint.entries
	}
	return nil
}
