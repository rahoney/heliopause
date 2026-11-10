package npm

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func nodeEntryFixture(t *testing.T, name string, body []byte) (ArchiveInventory, ExecutionEntry) {
	t.Helper()
	original := executionEntryInventory(t, "tiny", fmt.Sprintf(`,"bin":%q`, name))
	files := original.Entries()
	for i := range files {
		if files[i].Path == "cli.js" {
			files[i].Path, files[i].Size, files[i].Digest = name, int64(len(body)), inventoryDigest(body)
		}
	}
	inventory, err := NewArchiveInventory(original.Artifact(), files, original.Manifest())
	if err != nil {
		t.Fatal(err)
	}
	entry, err := SelectExecutionEntry(inventory, "")
	if err != nil {
		t.Fatal(err)
	}
	return inventory, entry
}

func TestNodeExecutionEntryUsesOriginalBytesAndClosedInterpreter(t *testing.T) {
	for _, test := range []struct {
		name, file, body string
		accepted         bool
	}{
		{"js", "cli.js", "console.log('ok')\n", true},
		{"empty js", "cli.js", "", true},
		{"cjs", "script.cjs", "module.exports = {}\n", true},
		{"mjs", "script.mjs", "export default {}\n", true},
		{"env node", "bin/tool", "#!/usr/bin/env node\nconsole.log('ok')\n", true},
		{"env split node", "bin/tool", "#!/usr/bin/env -S node\n", true},
		{"pinned path", "bin/tool", "#!/usr/local/bin/node\n", true},
		{"node alias", "bin/tool", "#!/usr/bin/node\n", true},
		{"unknown extension", "script.txt", "console.log('ok')\n", false},
		{"shell in js", "cli.js", "#!/bin/sh\necho unsafe\n", false},
		{"python", "bin/tool", "#!/usr/bin/env python\n", false},
		{"bun", "bin/tool", "#!/usr/bin/env bun\n", false},
		{"env extra args", "bin/tool", "#!/usr/bin/env node --eval evil\n", false},
		{"loader override", "cli.js", "#!/usr/bin/env -S node --loader ./evil.js\n", false},
		{"interpreter syntax", "bin/tool", "#!/usr/bin/env node;sh\n", false},
		{"unbounded header", "bin/tool", "#!" + strings.Repeat(" ", 512) + "/usr/bin/node\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			inventory, selected := nodeEntryFixture(t, test.file, []byte(test.body))
			entry, err := selected.BindNodeTarget([]byte(test.body))
			if !test.accepted {
				if err == nil || entry.Valid() || entry.Matches(inventory) {
					t.Fatal("unsupported interpreter or target admitted")
				}
				return
			}
			if err != nil || !entry.Valid() || !entry.Matches(inventory) || !bytes.Equal(entry.Bytes(), []byte(test.body)) {
				t.Fatal("exact Node binding failed", err)
			}
			if entry.Entry().Artifact() != inventory.Artifact() || entry.Entry().Target() != selected.Target() || entry.Entry().Command() != "tiny" {
				t.Fatal("original acquired subject or selected member changed")
			}
		})
	}
}

func TestNodeExecutionEntryRejectsChangedBytesAndKeepsPrivateCopy(t *testing.T) {
	body := []byte("console.log('ok')\n")
	inventory, selected := nodeEntryFixture(t, "cli.js", body)
	for _, candidate := range [][]byte{
		[]byte("console.log('no')\n"),
		body[:len(body)-1],
		append(append([]byte{}, body...), '\n'),
		nil,
		make([]byte, ArchiveFileLimit+1),
	} {
		if entry, err := selected.BindNodeTarget(candidate); err == nil || entry.Valid() {
			t.Fatal("different size/digest or oversize bytes accepted")
		}
	}
	entry, err := selected.BindNodeTarget(body)
	if err != nil {
		t.Fatal(err)
	}
	body[0] = '!'
	returned := entry.Bytes()
	returned[0] = '?'
	member := entry.Entry().Target()
	member.Path = "changed.js"
	if !bytes.Equal(entry.Bytes(), []byte("console.log('ok')\n")) || entry.Entry().Target().Path != "cli.js" || !entry.Matches(inventory) {
		t.Fatal("caller or returned copy changed retained bytes/facts")
	}
	other := executionEntryInventory(t, "tiny", `,"bin":"cli.js"`)
	if entry.Matches(other) || entry.Matches(ArchiveInventory{}) {
		t.Fatal("different or zero whole inventory substituted")
	}
	var zero NodeExecutionEntry
	if zero.Valid() || zero.Matches(inventory) || len(zero.Bytes()) != 0 {
		t.Fatal("zero Node binding accepted")
	}
	if result, err := (ExecutionEntry{}).BindNodeTarget([]byte("code")); err == nil || result.Valid() {
		t.Fatal("zero selection accepted")
	}
}
