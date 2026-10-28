package npm

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

var errNodeExecutionEntry = errors.New("npm Node entry bytes or interpreter differ from the selected bin")

// NodeExecutionEntry retains matched file bytes with a closed Node entrypoint
// declaration beside the original inventory binding. Composition must bind the
// pinned runtime separately. This is neither source
// authentication nor an inspection result, Policy decision or execution permit.
// Its zero value is invalid.
type NodeExecutionEntry struct {
	entry ExecutionEntry
	body  []byte
}

// BindNodeTarget copies bounded input before checking its original size/digest
// and the existing closed Node entrypoint rules. No file is opened or executed.
// The caller must still verify the whole acquired set and introduction bytes.
func (e ExecutionEntry) BindNodeTarget(body []byte) (NodeExecutionEntry, error) {
	if !e.Valid() || len(body) > ArchiveFileLimit || int64(len(body)) != e.target.Size {
		return NodeExecutionEntry{}, errNodeExecutionEntry
	}
	owned := bytes.Clone(body)
	sum := sha256.Sum256(owned)
	if hex.EncodeToString(sum[:]) != e.target.Digest.String() || !pnpmNodeBinTarget(owned, e.target.Path) {
		return NodeExecutionEntry{}, errNodeExecutionEntry
	}
	return NodeExecutionEntry{e, owned}, nil
}

func (e NodeExecutionEntry) Valid() bool {
	return e.entry.Valid() && int64(len(e.body)) == e.entry.Target().Size
}

func (e NodeExecutionEntry) Entry() ExecutionEntry { return e.entry }
func (e NodeExecutionEntry) Bytes() []byte         { return bytes.Clone(e.body) }

func (e NodeExecutionEntry) Matches(inventory ArchiveInventory) bool {
	return e.Valid() && e.entry.Matches(inventory)
}
