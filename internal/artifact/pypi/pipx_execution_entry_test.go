package pypi

import "testing"

func TestPipxConsoleChoiceUsesPrimaryOrSoleDeclaredOriginal(t *testing.T) {
	for _, row := range []struct{ points, command, expected string }{
		{"[console_scripts]\nexample=example:main\nother=example:other\n", "", "example"},
		{"[console_scripts]\nother=example:main\n", "", "other"},
		{"[console_scripts]\nother=example:main\n", "example", "other"},
		{"[console_scripts]\nother=example:main\n", "other", "other"},
		{"[console_scripts]\none=example:main\ntwo=example:other\n", "two", "two"},
	} {
		_, _, inventory := consoleExecutionWheel(t, row.points, "")
		entry, err := SelectPipxConsoleExecutionEntry(inventory, row.command)
		if err != nil || entry.Command() != row.expected || !entry.Matches(inventory) {
			t.Fatal("pipx source choice differs", row.command, err)
		}
	}
	_, _, inventory := consoleExecutionWheel(t, "[console_scripts]\nother=example:main\n", "")
	if _, err := SelectConsoleExecutionEntry(inventory, ""); err == nil {
		t.Fatal("pipx default widened the generic execution contract")
	}
}

func TestPipxConsoleChoiceRejectsAmbiguousMissingAndModuleRoles(t *testing.T) {
	for _, row := range []struct{ points, command string }{
		{"[console_scripts]\none=example:main\ntwo=example:other\n", ""},
		{"[console_scripts]\nother=example:main\n", "missing"},
		{"[console_scripts]\nother=example:main\n", "../other"},
		{"[console_scripts]\nexample=example:main\n[pipx.run]\nexample=example:other\n", ""},
		{"[console_scripts]\nother=example:main\n[pipx.run]\nother=example:other\n", ""},
	} {
		_, _, inventory := consoleExecutionWheel(t, row.points, "")
		if entry, err := SelectPipxConsoleExecutionEntry(inventory, row.command); err == nil || entry.Valid() {
			t.Fatal("unsupported pipx choice accepted", row.command)
		}
	}
}
