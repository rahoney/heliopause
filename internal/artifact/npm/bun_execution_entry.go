package npm

import (
	"bytes"
	"errors"
	"path"
	"slices"
	"strings"
)

var errBunExecutionEntry = errors.New("npm Bun entry bytes or declaration differ from the selected bin")

// BunExecutionEntry binds measured JS/TS bytes to the complete selected original.
// Its declaration selects a pinned runtime only after current whole approval;
// it is neither an inspection result nor runtime admission.
type BunExecutionEntry struct {
	entry ExecutionEntry
	body  []byte
}

func (e ExecutionEntry) BindBunTarget(body []byte) (BunExecutionEntry, error) {
	owned, ok := e.bindTargetBytes(body)
	if !ok || !bunExecutionBinTarget(owned, e.target.Path) {
		return BunExecutionEntry{}, errBunExecutionEntry
	}
	return BunExecutionEntry{e, owned}, nil
}

func bunExecutionBinTarget(body []byte, name string) bool {
	if bytes.HasPrefix(body, []byte("#!")) {
		line, _, _ := bytes.Cut(body[:min(len(body), 512)], []byte{'\n'})
		if len(line) == 512 {
			return false
		}
		return bunExecutionShebang(body) || pnpmNodeBinTarget(body, name)
	}
	return slices.Contains([]string{".js", ".cjs", ".mjs", ".ts", ".tsx", ".mts", ".cts"}, path.Ext(name))
}

func bunExecutionShebang(body []byte) bool {
	if !bytes.HasPrefix(body, []byte("#!")) {
		return false
	}
	line, _, _ := bytes.Cut(body[:min(len(body), 512)], []byte{'\n'})
	if len(line) == 512 {
		return false
	}
	fields := strings.Fields(strings.TrimSpace(strings.TrimPrefix(string(line), "#!")))
	return slices.Equal(fields, []string{"/usr/bin/env", "bun"}) || slices.Equal(fields, []string{"/usr/bin/env", "-S", "bun"}) ||
		slices.Equal(fields, []string{"/usr/bin/bun"}) || slices.Equal(fields, []string{"/usr/local/bin/bun"})
}

func (e BunExecutionEntry) Valid() bool {
	return e.entry.Valid() && int64(len(e.body)) == e.entry.Target().Size
}

func (e BunExecutionEntry) Entry() ExecutionEntry { return e.entry }
func (e BunExecutionEntry) Bytes() []byte         { return bytes.Clone(e.body) }
func (e BunExecutionEntry) BunShebang() bool      { return e.Valid() && bunExecutionShebang(e.body) }
func (e BunExecutionEntry) NodeCompatible() bool {
	return e.Valid() && pnpmNodeBinTarget(e.body, e.entry.Target().Path)
}
func (e BunExecutionEntry) Matches(inventory ArchiveInventory) bool {
	return e.Valid() && e.entry.Matches(inventory)
}
