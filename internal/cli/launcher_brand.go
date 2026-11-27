package cli

import (
	"fmt"
	"strings"
)

type launcherBrandLine struct {
	text, style                  string
	gradientStart, gradientWidth int
}

// launcherBrand is display-only: these fixed product letters never consume input
// or change terminal fonts. Keep four menu rows and the navigation footer visible.
func launcherBrand(cells, height, reservedRows int, home bool) []launcherBrandLine {
	var words []string
	available := height - reservedRows - 5 // Four menu rows and one footer.
	if home {
		switch {
		case cells >= 67 && height >= 24 && available >= 15:
			words = []string{"HELIOPAUSE", "ARTIFACT AIRLOCK"}
		case cells >= 111 && height >= 18 && available >= 9:
			words = []string{"HELIOPAUSE ARTIFACT AIRLOCK"}
		case cells >= 43 && height >= 28 && available >= 21:
			words = []string{"HELIOPAUSE", "ARTIFACT", "AIRLOCK"}
		}
	}
	var lines []launcherBrandLine
	if len(words) == 0 {
		for _, line := range launcherWrap("Heliopause Artifact Airlock", cells) {
			lines = append(lines, launcherBrandLine{text: line, style: "1;36"})
		}
		if height >= 18 {
			lines = append(lines, launcherBrandLine{text: "helox", style: "2"})
		}
		return lines
	}
	// Flat, simple door mark: outline, split and two short handles.
	// It shares the existing five-row wordmark height.
	emblem := [5]string{
		"    ▄▀▀▀▀▀▀▀▀▀▄    ",
		"   ▟     ║     ▙   ",
		"  █  ═══ ║ ═══  █  ",
		"   ▜     ║     ▛   ",
		"    ▀▄▄▄▄▄▄▄▄▄▀    ",
	}
	panel := func(text, style string) {
		padding := cells - 4 - len([]rune(text))
		start := 2 + padding/2
		end := start + len([]rune(text))
		row := []rune("│ " + strings.Repeat(" ", padding/2) + text + strings.Repeat(" ", padding-padding/2) + " │")
		// Stable stars only in empty margins; never overwrite lettering or the door.
		for col := 2; col < cells-2; col++ {
			if col >= start && col < end {
				continue
			}
			switch (col*37 + len(lines)*61) % 97 {
			case 0:
				row[col] = '✦'
			case 19:
				row[col] = '+'
			case 41:
				row[col] = '·'
			case 67:
				row[col] = '.'
			}
		}
		lines = append(lines, launcherBrandLine{text: string(row), style: style})
	}
	lines = append(lines, launcherBrandLine{text: "╭─ · " + strings.Repeat("─", cells-6) + "╮", style: "36;40"})
	for i, word := range words {
		rows := launcherWordmark(word)
		width := len([]rune(rows[0]))
		withEmblem := i == 0 && width+len([]rune(emblem[0]))+3 <= cells-4
		for row, text := range rows {
			if withEmblem {
				text = emblem[row] + "   " + text
			}
			panel(text, "1;37;40")
			line := &lines[len(lines)-1]
			line.gradientStart = 2 + (cells-4-len([]rune(text)))/2 + len([]rune(text)) - width
			line.gradientWidth = width
		}
		panel("", "40")
	}
	panel("Heliopause Artifact Airlock", "1;93;40")
	lines = append(lines, launcherBrandLine{text: "╰─ helox " + strings.Repeat("─", cells-10) + "╯", style: "36;40"})
	return lines
}

// Keep layout plain; add the horizontal RGB gradient only at the paint boundary.
// presentation.paint remains the single owner of color/NO_COLOR preferences.
func (line launcherBrandLine) paint(p *presentation) string {
	if line.gradientWidth < 2 || p.paint("x", line.style) == "x" {
		return p.paint(line.text, line.style)
	}
	pixels := []rune(line.text)
	end := line.gradientStart + line.gradientWidth
	var out strings.Builder
	for _, pixel := range pixels[:line.gradientStart] {
		style := line.style
		if pixel == '═' {
			style = "1;93;40"
		} else if pixel == '║' {
			style = "90;40"
		}
		out.WriteString(p.paint(string(pixel), style))
	}
	for col, pixel := range pixels[line.gradientStart:end] {
		r := 246 + (93-246)*col/(line.gradientWidth-1)
		g := 119 + (225-119)*col/(line.gradientWidth-1)
		b := 210 + (246-210)*col/(line.gradientWidth-1)
		weight := "1"
		if pixel == '░' {
			weight = "2"
		}
		out.WriteString(p.paint(string(pixel), fmt.Sprintf("%s;38;2;%d;%d;%d;40", weight, r, g, b)))
	}
	out.WriteString(p.paint(string(pixels[end:]), line.style))
	return out.String()
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
	// A one-cell down/right shadow stays inside the existing five-row grid.
	// Work bottom-up so the shadow never becomes another shadow's source.
	for row := len(rows) - 1; row > 0; row-- {
		pixels, above := []rune(rows[row]), []rune(rows[row-1])
		for col := 1; col < len(pixels); col++ {
			if pixels[col] == ' ' && above[col-1] == '█' {
				pixels[col] = '░'
			}
		}
		rows[row] = string(pixels)
	}
	return rows
}
