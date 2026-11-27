package npm

import (
	"bytes"
	"errors"
	"path"
	"slices"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errExecutionEntry = errors.New("npm execution entry requires an exact unambiguous declared bin")

// ExecutionEntry binds a command to one regular member of an exact acquired
// inventory. It is structural data, not byte verification, runtime admission,
// a Policy decision or permission to execute. Its zero value is invalid.
type ExecutionEntry struct {
	inventory ArchiveInventory
	command   string
	target    ArchiveEntry
}

// SelectExecutionEntry reuses the layout bin parser. An empty command selects
// only a single bin whose name equals the package basename. Different command
// names and multiple bins require an explicit command, even for target aliases.
func SelectExecutionEntry(inventory ArchiveInventory, command string) (ExecutionEntry, error) {
	bins, err := pnpmInventoryBins(inventory)
	if err != nil || len(bins) == 0 {
		return ExecutionEntry{}, errExecutionEntry
	}
	if command == "" {
		if len(bins) != 1 {
			return ExecutionEntry{}, errExecutionEntry
		}
		command = path.Base(inventory.Artifact().Identity().Name())
	}
	target, present := bins[command]
	if !present {
		return ExecutionEntry{}, errExecutionEntry
	}
	for _, member := range inventory.entries {
		if member.Path == target && !member.Directory {
			return ExecutionEntry{inventory, command, member}, nil
		}
	}
	return ExecutionEntry{}, errExecutionEntry
}

func (e ExecutionEntry) Valid() bool {
	return e.inventory.Valid() && e.command != "" && e.target.Path != ""
}

func (e ExecutionEntry) Artifact() domain.AcquiredArtifact { return e.inventory.Artifact() }
func (e ExecutionEntry) Command() string                   { return e.command }
func (e ExecutionEntry) Target() ArchiveEntry              { return e.target }

// Matches requires the whole original inventory, including its acquisition
// handle, manifest and every member's digest, size and mode. A copied package
// name or target digest alone cannot substitute for that binding.
func (e ExecutionEntry) Matches(inventory ArchiveInventory) bool {
	return e.Valid() && inventory.Matches(e.inventory.Artifact()) &&
		bytes.Equal(inventory.manifest, e.inventory.manifest) && slices.Equal(inventory.entries, e.inventory.entries)
}
