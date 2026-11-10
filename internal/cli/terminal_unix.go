//go:build linux || darwin

package cli

import (
	"golang.org/x/sys/unix"
	"os"
)

func terminalWidth(file *os.File) (int, bool) {
	if file == nil {
		return 0, false
	}
	size, err := unix.IoctlGetWinsize(int(file.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 0, false
	}
	width := int(size.Col)
	if width < 20 {
		width = 20
	}
	if width > 160 {
		width = 160
	}
	return width, true
}
