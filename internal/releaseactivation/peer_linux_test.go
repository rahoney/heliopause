//go:build linux

package releaseactivation

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestControllerGroupsUseBoundedKernelStatusOnly(t *testing.T) {
	for _, test := range []struct {
		name, text string
		valid      bool
	}{
		{"normal", "Name:\tignored\nGroups:\t998 1000\nState:\tR\n", true},
		{"empty", "Groups:\n", true},
		{"missing", "Name:\tGroups: 0\n", false},
		{"duplicate-group", "Groups:\t1000 1000\n", false},
		{"duplicate-line", "Groups:\t1000\nGroups:\t998\n", false},
		{"non-number", "Groups:\troot\n", false},
		{"excess-groups", "Groups:\t" + strings.Repeat("1000 ", 65) + "\n", false},
		{"excess-line", "Name:\t" + strings.Repeat("x", 64<<10) + "\nGroups:\t1000\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "status")
			if err := os.WriteFile(path, []byte(test.text), 0600); err != nil {
				t.Fatal(err)
			}
			file, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			groups, err := kernelGroups(file)
			if test.valid && err != nil {
				t.Fatal(err)
			}
			if !test.valid && !errors.Is(err, ErrCaller) {
				t.Fatal("invalid kernel status accepted", err)
			}
			if test.name == "normal" && (len(groups) != 2 || groups[0] != 998 || groups[1] != 1000) {
				t.Fatal("kernel group membership changed")
			}
		})
	}
}

func TestControllerPidfdTracksKnownProcessExit(t *testing.T) {
	if os.Getenv("HELOX_PIDFD_TEST_CHILD") == "1" {
		_, err := io.Copy(io.Discard, os.Stdin)
		if err != nil {
			t.Fatal(err)
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestControllerPidfdTracksKnownProcessExit$", "-test.count=1")
	command.Env = append(os.Environ(), "HELOX_PIDFD_TEST_CHILD=1")
	pipe, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		_ = pipe.Close()
		if !waited {
			_ = command.Wait()
		}
	}()
	fd, err := unix.PidfdOpen(command.Process.Pid, 0)
	if err != nil {
		t.Fatal(err)
	}
	peer := &controllerPeer{pidfd: os.NewFile(uintptr(fd), "known-child"), pid: int32(command.Process.Pid)}
	defer peer.Close()
	if err := peer.alive(); err != nil {
		t.Fatal("known live child was not recognized", err)
	}
	if err := pipe.Close(); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err != nil {
		t.Fatal(err)
	}
	waited = true
	if err := peer.alive(); !errors.Is(err, ErrCaller) {
		t.Fatal("exited/reaped child was treated as live")
	}
	if err := peer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := peer.Close(); err != nil {
		t.Fatal("pidfd Close is not idempotent", err)
	}
	if err := peer.alive(); !errors.Is(err, ErrCaller) {
		t.Fatal("closed identity was treated as live")
	}
	if err := CheckRuntimeController(InstalledRelease{}); !errors.Is(err, ErrCaller) {
		t.Fatal("empty installation approved runtime caller")
	}
}
