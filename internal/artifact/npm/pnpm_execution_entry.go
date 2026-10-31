package npm

import "path"

// SelectPNPMExecutionEntry applies the pinned client's sole-bin or package
// basename default. Exact member binding remains with the existing selector.
func SelectPNPMExecutionEntry(inventory ArchiveInventory, command string) (ExecutionEntry, error) {
	if command == "" {
		bins, err := pnpmInventoryBins(inventory)
		if err != nil || len(bins) == 0 {
			return ExecutionEntry{}, errExecutionEntry
		}
		command = path.Base(inventory.Artifact().Identity().Name())
		if len(bins) == 1 {
			for name := range bins {
				command = name
			}
		}
	}
	return SelectExecutionEntry(inventory, command)
}
