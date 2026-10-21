package cli

import "strings"

type launcherBrandLine struct {
	text, style string
}

// launcherBrand is display-only: these fixed product letters never consume input
// or change terminal fonts. Keep four menu rows and the navigation footer visible.
func launcherBrand(cells, height, reservedRows int, home bool) []launcherBrandLine {
	var words []string
	available := height - reservedRows - 5 // Four menu rows and one footer.
	if home {
		switch {
		case cells >= 111 && height >= 18 && available >= 9:
			words = []string{"HELIOPAUSE ARTIFACT AIRLOCK"}
		case cells >= 67 && height >= 24 && available >= 15:
			words = []string{"HELIOPAUSE", "ARTIFACT AIRLOCK"}
		case cells >= 43 && height >= 28 && available >= 21:
			words = []string{"HELIOPAUSE", "ARTIFACT", "AIRLOCK"}
		}
	}
	var lines []launcherBrandLine
	if len(words) == 0 {
		for _, line := range launcherWrap("Heliopause Artifact Airlock", cells) {
			lines = append(lines, launcherBrandLine{line, "1;36"})
		}
		if height >= 18 {
			lines = append(lines, launcherBrandLine{"helox", "2"})
		}
		return lines
	}
	panel := func(text, style string) {
		padding := cells - len([]rune(text))
		lines = append(lines, launcherBrandLine{strings.Repeat(" ", padding/2) + text + strings.Repeat(" ", padding-padding/2), style})
	}
	panel("", "44")
	for i, word := range words {
		style := "1;96;44"
		if i > 0 {
			style = "1;97;44"
		}
		for _, row := range launcherWordmark(word) {
			panel(row, style)
		}
		panel("", "44")
	}
	panel("Heliopause Artifact Airlock", "1;93;44")
	panel("helox", "97;44")
	return lines
}

// Only the letters in the fixed product name, at three cells by five rows.
// Spaces keep word boundaries readable even with colors disabled.
func launcherWordmark(word string) [5]string {
	glyphs := map[rune][5]string{
		'H': {"█ █", "█ █", "███", "█ █", "█ █"},
		'E': {"███", "█  ", "██ ", "█  ", "███"},
		'L': {"█  ", "█  ", "█  ", "█  ", "███"},
		'I': {"███", " █ ", " █ ", " █ ", "███"},
		'O': {"███", "█ █", "█ █", "█ █", "███"},
		'P': {"███", "█ █", "███", "█  ", "█  "},
		'A': {"███", "█ █", "███", "█ █", "█ █"},
		'U': {"█ █", "█ █", "█ █", "█ █", "███"},
		'S': {"███", "█  ", "███", "  █", "███"},
		'R': {"██ ", "█ █", "██ ", "█ █", "█ █"},
		'T': {"███", " █ ", " █ ", " █ ", " █ "},
		'F': {"███", "█  ", "██ ", "█  ", "█  "},
		'C': {"███", "█  ", "█  ", "█  ", "███"},
		'K': {"█ █", "█ █", "██ ", "█ █", "█ █"},
		' ': {"   ", "   ", "   ", "   ", "   "},
	}
	var rows [5]string
	for i, letter := range word {
		for row, pixels := range glyphs[letter] {
			if i > 0 {
				rows[row] += " "
			}
			rows[row] += pixels
		}
	}
	return rows
}
