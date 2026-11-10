package npm

import (
	"bytes"
	"strings"
	"testing"
)

func TestBunExecutionEntryBindsClosedDeclarationsWithoutWideningNode(t *testing.T) {
	for _, test := range []struct {
		name, file, body string
		valid, bun, node bool
	}{
		{"plain JS", "cli.js", "console.log('ok')\n", true, false, true},
		{"Node declaration", "bin/tool", "#!/usr/bin/env node\n", true, false, true},
		{"Bun declaration", "bin/tool", "#!/usr/bin/env bun\n", true, true, false},
		{"split Bun", "cli.ts", "#!/usr/bin/env -S bun\nconst x: number = 1;\n", true, true, false},
		{"absolute Bun", "bin/tool", "#!/usr/local/bin/bun\n", true, true, false},
		{"system Bun", "bin/tool", "#!/usr/bin/bun\n", true, true, false},
		{"TS without declaration", "cli.ts", "const x: number = 1;\n", true, false, false},
		{"TSX", "cli.tsx", "const x: number = 1;\n", true, false, false},
		{"module TS", "cli.mts", "export const x: number = 1;\n", true, false, false},
		{"common TS", "cli.cts", "const x: number = 1;\n", true, false, false},
		{"TS Node declaration", "cli.ts", "#!/usr/bin/env node\n", true, false, true},
		{"shell", "cli.ts", "#!/bin/sh\necho no\n", false, false, false},
		{"python", "cli.js", "#!/usr/bin/env python\n", false, false, false},
		{"Bun argument", "cli.ts", "#!/usr/bin/env bun --eval bad\n", false, false, false},
		{"Bun loader", "cli.ts", "#!/usr/bin/env -S bun --preload ./bad.ts\n", false, false, false},
		{"foreign Bun path", "cli.ts", "#!/tmp/bun\n", false, false, false},
		{"unknown file", "cli.txt", "console.log('ok')\n", false, false, false},
		{"hidden long argument", "cli.ts", "#!/usr/bin/env bun" + strings.Repeat(" ", 512) + "--eval bad\n", false, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			inventory, selected := nodeEntryFixture(t, test.file, []byte(test.body))
			proof, err := selected.BindBunTarget([]byte(test.body))
			if !test.valid {
				if err == nil || proof.Valid() || proof.Matches(inventory) || proof.BunShebang() || proof.NodeCompatible() {
					t.Fatal("unsupported declaration or interpreter admitted")
				}
				return
			}
			if err != nil || !proof.Matches(inventory) || !bytes.Equal(proof.Bytes(), []byte(test.body)) || proof.BunShebang() != test.bun || proof.NodeCompatible() != test.node || proof.Entry().Target() != selected.Target() {
				t.Fatal("selected original or declaration changed", err)
			}
			old, nodeErr := selected.BindNodeTarget([]byte(test.body))
			if test.node != (nodeErr == nil && old.Valid()) {
				t.Fatal("existing Node entry contract changed")
			}
		})
	}
}

func TestBunExecutionEntryRejectsSubstitutionAndOwnsBytes(t *testing.T) {
	body := []byte("#!/usr/bin/env bun\nconsole.log('ok')\n")
	inventory, selected := nodeEntryFixture(t, "cli.ts", body)
	for _, changed := range [][]byte{nil, body[:len(body)-1], append(bytes.Clone(body), 'x'), bytes.Repeat([]byte{'x'}, ArchiveFileLimit+1)} {
		if proof, err := selected.BindBunTarget(changed); err == nil || proof.Valid() {
			t.Fatal("changed size/digest or oversize input admitted")
		}
	}
	proof, err := selected.BindBunTarget(body)
	if err != nil {
		t.Fatal(err)
	}
	body[0] = '?'
	proof.Bytes()[0] = '!'
	if !proof.BunShebang() || proof.Bytes()[0] != '#' || !proof.Matches(inventory) || proof.Matches(ArchiveInventory{}) || proof.Matches(executionEntryInventory(t, "tiny", `,"bin":"cli.js"`)) {
		t.Fatal("caller or different original substituted retained proof")
	}
	if proof, err := (ExecutionEntry{}).BindBunTarget(nil); err == nil || proof.Valid() {
		t.Fatal("zero entry produced a proof")
	}
}
