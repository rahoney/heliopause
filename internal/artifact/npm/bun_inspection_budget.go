package npm

import "github.com/rahoney/heliopause/internal/core/domain"

// BunInspectionWorkspaceLimit is the same fixed cache plus logical-copy budget
// as the existing npm inspection engine. It is not a user override.
const BunInspectionWorkspaceLimit = PNPMInspectionWorkspaceLimit

// BunInspectionBudget uses the existing measured-input accounting engine.
type BunInspectionBudget = npmInspectionBudget

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
	return beginNPMInspectionBudget(i.artifacts, instances)
}

// CheckWorkspace charges every compressed archive, unique expanded cache and
// logical tree placement before native preparation. It grants no execution.
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
