package npm

import "github.com/rahoney/heliopause/internal/core/domain"

// BunInspectionWorkspaceLimit is the fixed native inspection volume capacity.
// Per-client accounting differs; the ceiling is not a user override.
const BunInspectionWorkspaceLimit = PNPMInspectionWorkspaceLimit

// BunInspectionBudget reuses the existing engine and charges the additional
// native file-cache copy retained during registry-cache preparation.
type BunInspectionBudget struct{ shared *npmInspectionBudget }

func (i BunInspectionInput) indexInventories(inventories []ArchiveInventory) (map[domain.AcquiredArtifact]ArchiveInventory, error) {
	if !i.Valid() {
		return nil, errBunInspectionLayout
	}
	return indexNPMProjectInventories(i.artifacts, inventories, errBunInspectionLayout)
}

func (i BunInspectionInput) BeginWorkspace() (*BunInspectionBudget, error) {
	if !i.Valid() || len(i.artifacts) == 0 {
		return nil, errBunInspectionLayout
	}
	instances := make([]string, 0, len(i.context.selection.lock.packages))
	for _, instance := range i.context.selection.lock.packages {
		instances = append(instances, instance.Name+"@"+instance.Version)
	}
	budget, err := beginNPMInspectionBudget(i.artifacts, instances)
	if err != nil {
		return nil, err
	}
	return &BunInspectionBudget{budget}, nil
}

func (b *BunInspectionBudget) Add(inventory ArchiveInventory) error {
	if b == nil || b.shared == nil {
		return errBunInspectionLayout
	}
	// Validate membership and charge the original cache plus logical copies
	// before accounting for the second, temporary cache. Failures stay sticky.
	if err := b.shared.Add(inventory); err != nil {
		return err
	}
	for _, entry := range inventory.entries {
		if !entry.Directory && !b.shared.charge(uint64(entry.Size)) {
			b.shared.failure = errPNPMInspectionBudget
			return b.shared.failure
		}
	}
	return nil
}

func (b *BunInspectionBudget) Complete() error {
	if b == nil || b.shared == nil {
		return errBunInspectionLayout
	}
	return b.shared.Complete()
}

// CheckWorkspace charges every compressed archive, two unique expanded caches
// and every logical placement before preparation. It grants no execution.
func (i BunInspectionInput) CheckWorkspace(inventories []ArchiveInventory) error {
	bySubject, err := i.indexInventories(inventories)
	if err != nil {
		return err
	}
	budget, err := i.BeginWorkspace()
	if err != nil {
		return err
	}
	for _, inventory := range bySubject {
		if err := budget.Add(inventory); err != nil {
			return err
		}
	}
	return budget.Complete()
}
