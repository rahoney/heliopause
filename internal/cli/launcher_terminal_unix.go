//go:build linux || darwin

package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

type unixLauncherTerminal struct {
	input, output *os.File
	original      *unix.Termios
	decoder       launcherKeyDecoder
	pending       []byte
	escapeStarted time.Time
}

func launcherGeometry(input, output *os.File) (int, int, bool) {
	if _, ok := terminalWidth(input); !ok {
		return 0, 0, false
	}
	width, height, err := launcherWindowSize(output)
	return width, height, err == nil
}

func launcherWindowSize(output *os.File) (int, int, error) {
	size, err := unix.IoctlGetWinsize(int(output.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 0, 0, fmt.Errorf("read launcher terminal size: %w", err)
	}
	return min(int(size.Col), 160), min(int(size.Row), 80), nil
}

func openLauncherTerminal(input, output *os.File) (launcherTerminal, error) {
	if input.Fd() > math.MaxInt32 {
		return nil, errors.New("launcher input descriptor is unsupported")
	}
	original, err := launcherGetTermios(int(input.Fd()))
	if err != nil {
		return nil, fmt.Errorf("read launcher terminal state: %w", err)
	}
	raw := *original
	raw.Iflag &^= unix.BRKINT | unix.ICRNL | unix.INPCK | unix.ISTRIP | unix.IXON
	raw.Oflag &^= unix.OPOST
	raw.Cflag &^= unix.CSIZE | unix.PARENB
	raw.Cflag |= unix.CS8
	// Preserve ISIG: the process entrypoint's existing SIGINT context also
	// cancels an explicit, synchronous doctor request while input is idle.
	raw.Lflag &^= unix.ECHO | unix.ICANON | unix.IEXTEN
	raw.Cc[unix.VMIN], raw.Cc[unix.VTIME] = 0, 0
	if err := launcherSetTermios(int(input.Fd()), &raw); err != nil {
		return nil, errors.Join(fmt.Errorf("prepare launcher terminal: %w", err), launcherSetTermios(int(input.Fd()), original))
	}
	return &unixLauncherTerminal{input: input, output: output, original: original}, nil
}

func (t *unixLauncherTerminal) Size() (int, int, error) {
	return launcherWindowSize(t.output)
}

func (t *unixLauncherTerminal) Close() error {
	if t.original == nil {
		return nil
	}
	err := launcherSetTermios(int(t.input.Fd()), t.original)
	if err != nil {
		return fmt.Errorf("restore launcher terminal state: %w", err)
	}
	t.original = nil
	return nil
}

func (t *unixLauncherTerminal) Next(ctx context.Context) (launcherKey, error) {
	for {
		if err := ctx.Err(); err != nil {
			return launcherIdle, err
		}
		if t.decoder.pending() && time.Since(t.escapeStarted) >= 60*time.Millisecond {
			return t.decoder.expire(), nil
		}
		if len(t.pending) > 0 {
			value := t.pending[0]
			t.pending = t.pending[1:]
			wasPending := t.decoder.pending()
			key, ready := t.decoder.feed(value)
			if !wasPending && t.decoder.pending() {
				t.escapeStarted = time.Now()
			}
			if ready {
				return key, nil
			}
			continue
		}
		poll := []unix.PollFd{{Fd: int32(t.input.Fd()), Events: unix.POLLIN}}
		count, err := unix.Poll(poll, 25)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return launcherIdle, fmt.Errorf("poll launcher input: %w", err)
		}
		if count == 0 {
			return launcherIdle, nil
		}
		if poll[0].Revents&unix.POLLNVAL != 0 {
			return launcherIdle, os.ErrClosed
		}
		var body [32]byte
		n, err := unix.Read(int(t.input.Fd()), body[:])
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return launcherIdle, fmt.Errorf("read launcher input: %w", err)
		}
		if n == 0 {
			return launcherIdle, io.EOF
		}
		t.pending = body[:n]
	}
}
