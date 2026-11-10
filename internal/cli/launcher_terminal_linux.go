//go:build linux

package cli

import "golang.org/x/sys/unix"

func launcherGetTermios(fd int) (*unix.Termios, error) {
	return unix.IoctlGetTermios(fd, unix.TCGETS)
}

func launcherSetTermios(fd int, state *unix.Termios) error {
	return unix.IoctlSetTermios(fd, unix.TCSETS, state)
}
