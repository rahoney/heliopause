package npm

import "testing"

func TestPNPMExecutionEntryUsesClientDefaultAndExactOriginalMember(t *testing.T) {
	for _, test := range []struct {
		name, pkg, bins, command, want string
	}{
		{"sole-different", "tool", `"other":"cli.js"`, "", "other"},
		{"multiple-basename", "@scope/tool", `"other":"other.js","tool":"cli.js"`, "", "tool"},
		{"explicit", "tool", `"other":"other.js","tool":"cli.js"`, "other", "other"},
		{"ambiguous", "tool", `"first":"other.js","second":"cli.js"`, "", ""},
		{"undeclared", "tool", `"other":"cli.js"`, "missing", ""},
		{"missing-target", "tool", `"other":"missing.js"`, "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			inventory := executionEntryInventory(t, test.pkg, `,"bin":{`+test.bins+`}`)
			entry, err := SelectPNPMExecutionEntry(inventory, test.command)
			if test.want == "" {
				if err == nil || entry.Valid() {
					t.Fatal("ambiguous or foreign entry accepted", err)
				}
				return
			}
			if err != nil || !entry.Matches(inventory) || entry.Command() != test.want {
				t.Fatal("client choice lost original member binding", err)
			}
		})
	}
	inventory := executionEntryInventory(t, "tool", `,"bin":{"other":"cli.js"}`)
	if _, err := SelectExecutionEntry(inventory, ""); err == nil {
		t.Fatal("PNPM default broadened generic npm selection")
	}
}
