package pypi

// Pipx defaults to the primary name, then to a sole declared console. The
// generic selector keeps its stricter default rule; both native preparation
// and current approved binding use this client-specific choice.
func SelectPipxConsoleExecutionEntry(inventory ConsoleExecutionInventory, command string) (ConsoleExecutionEntry, error) {
	if !inventory.Valid() {
		return ConsoleExecutionEntry{}, errConsoleExecutionEntry
	}
	primary := inventory.Artifact().Identity().Name()
	if command == "" {
		command = primary
	}
	selected, err := SelectConsoleExecutionEntry(inventory, command)
	if err != nil && command == primary {
		var sole string
		count := 0
		for _, entry := range inventory.entries {
			if entry.Group == "console_scripts" {
				sole = entry.Name
				count++
			}
		}
		if count == 1 {
			selected, err = SelectConsoleExecutionEntry(inventory, sole)
		}
	}
	if err != nil {
		return ConsoleExecutionEntry{}, err
	}
	for _, entry := range inventory.entries {
		if entry.Group == "pipx.run" && (entry.Name == command || entry.Name == selected.Command()) {
			return ConsoleExecutionEntry{}, errConsoleExecutionEntry
		}
	}
	return selected, nil
}
