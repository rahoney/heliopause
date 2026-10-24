//go:build linux || darwin

package cli

import (
	"testing"

	"golang.org/x/sys/unix"
)

func launcherTermiosRestored(want, got unix.Termios, darwin bool) bool {
	if darwin {
		// Darwin sets PENDIN when restoring canonical input. This transient
		// kernel state does not change the saved terminal configuration.
		if want.Lflag&unix.ICANON != 0 && got.Lflag&unix.ICANON != 0 {
			want.Lflag &^= unix.PENDIN
			got.Lflag &^= unix.PENDIN
		}
		// tcsetattr defines zero input speed as the current output speed.
		if want.Ispeed == 0 {
			want.Ispeed = want.Ospeed
		}
		if got.Ispeed == 0 {
			got.Ispeed = got.Ospeed
		}
	}
	return want == got
}

func TestLauncherTermiosContract(t *testing.T) {
	base := unix.Termios{
		Iflag:  unix.ICRNL | unix.IXON,
		Oflag:  unix.OPOST,
		Cflag:  unix.CS8 | unix.CREAD,
		Lflag:  unix.ICANON | unix.ECHO | unix.ISIG | unix.IEXTEN,
		Ispeed: 9600, Ospeed: 9600,
	}
	base.Cc[unix.VMIN], base.Cc[unix.VTIME], base.Cc[unix.VEOF] = 1, 0, 4
	tests := []struct {
		name          string
		change        func(*unix.Termios, *unix.Termios)
		linux, darwin bool
	}{
		{"identical", func(_, _ *unix.Termios) {}, true, true},
		{"pending-input", func(_, got *unix.Termios) { got.Lflag |= unix.PENDIN }, false, true},
		{"pending-input-cleared", func(want, _ *unix.Termios) { want.Lflag |= unix.PENDIN }, false, true},
		{"input-speed-alias", func(want, _ *unix.Termios) { want.Ispeed = 0 }, false, true},
		{"restored-input-speed-alias", func(_, got *unix.Termios) { got.Ispeed = 0 }, false, true},
		{"combined-normalization", func(want, got *unix.Termios) { want.Ispeed = 0; got.Lflag |= unix.PENDIN }, false, true},
		{"input-flags", func(_, got *unix.Termios) { got.Iflag ^= unix.IXON }, false, false},
		{"output-flags", func(_, got *unix.Termios) { got.Oflag ^= unix.OPOST }, false, false},
		{"control-flags", func(_, got *unix.Termios) { got.Cflag ^= unix.CREAD }, false, false},
		{"canonical-mode", func(_, got *unix.Termios) { got.Lflag ^= unix.ICANON }, false, false},
		{"echo", func(_, got *unix.Termios) { got.Lflag ^= unix.ECHO }, false, false},
		{"signals", func(_, got *unix.Termios) { got.Lflag ^= unix.ISIG }, false, false},
		{"extended-input", func(_, got *unix.Termios) { got.Lflag ^= unix.IEXTEN }, false, false},
		{"minimum-read", func(_, got *unix.Termios) { got.Cc[unix.VMIN]++ }, false, false},
		{"read-timeout", func(_, got *unix.Termios) { got.Cc[unix.VTIME]++ }, false, false},
		{"end-of-file", func(_, got *unix.Termios) { got.Cc[unix.VEOF]++ }, false, false},
		{"input-speed", func(_, got *unix.Termios) { got.Ispeed = 19200 }, false, false},
		{"output-speed", func(_, got *unix.Termios) { got.Ospeed = 19200 }, false, false},
		{"unequal-speed-alias", func(want, got *unix.Termios) { want.Ispeed = 0; got.Ispeed = 19200 }, false, false},
		{"noncanonical-pending-input", func(want, got *unix.Termios) {
			want.Lflag &^= unix.ICANON
			got.Lflag &^= unix.ICANON
			got.Lflag |= unix.PENDIN
		}, false, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			want, got := base, base
			test.change(&want, &got)
			beforeWant, beforeGot := want, got
			for _, platform := range []struct {
				name             string
				darwin, expected bool
			}{
				{"linux", false, test.linux}, {"darwin", true, test.darwin},
			} {
				t.Run(platform.name, func(t *testing.T) {
					if result := launcherTermiosRestored(want, got, platform.darwin); result != platform.expected {
						t.Errorf("restored=%v want=%v; before=%+v after=%+v", result, platform.expected, want, got)
					}
					if want != beforeWant || got != beforeGot {
						t.Fatal("comparison mutated saved terminal snapshots")
					}
				})
			}
		})
	}
}
