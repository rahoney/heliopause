//go:build linux

package releaseactivation

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

var ErrInstaller = errors.New("SELF_UPDATE_PROTECTED_INSTALLER_FAILED")

// Activate transfers only an already-quarantined sealed release. The old
// protected controller remains the caller; the candidate is never launched as
// root. Independent root verification and durable recovery remain mandatory.
// drain must finish the old command's observer/executor cleanup before the
// shared runtime lease is released. It cannot select a privileged operation.
func Activate(ctx context.Context, lease *RuntimeLease, manifest []byte, proofs [][]byte, open func(string) (io.ReadCloser, error), drain func() error) (resultErr error) {
	if ctx == nil || open == nil || drain == nil {
		return ErrProtocol
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	old, err := lease.Installed()
	if err != nil {
		return err
	}
	if lease.borrowed {
		return ErrTransition
	}
	if err := CheckRuntimeController(old); err != nil {
		return err
	}
	release, err := newWireRelease(manifest, proofs)
	if err != nil {
		return err
	}
	if _, err := release.verify(ctx, old); err != nil {
		return err
	}
	current, err := ReadInstalledRelease(ctx)
	if err != nil || current.record != old.record {
		return errors.Join(ErrUntrusted, err)
	}
	process, err := startInstalledInstaller(old)
	if err != nil {
		return err
	}
	activationSent := false
	// Close the connection before waiting, including every pre-ACTIVATE abort.
	// Do not SIGKILL the root transaction on cancellation or response loss.
	defer func() {
		waitErr := process.wait()
		if waitErr != nil && activationSent {
			waitErr = errors.Join(ErrTransition, waitErr)
		}
		resultErr = errors.Join(resultErr, waitErr)
	}()
	connection, err := connectInstalledInstaller(ctx, old.ClientGID(), process)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, connection.Close()) }()
	if err := connection.SetDeadline(time.Now().Add(30 * time.Minute)); err != nil {
		return ErrProtocol
	}
	stop := context.AfterFunc(ctx, func() { _ = connection.SetDeadline(time.Now()) })
	defer stop()
	activated, err := exchangeUpdate(ctx, connection, release, open, func() error {
		if err := drain(); err != nil {
			return err
		}
		return lease.Close()
	})
	activationSent = activated
	if err != nil && activated {
		return errors.Join(ErrTransition, err)
	}
	return err
}

// exchangeUpdate never interprets helper stdout or a candidate self-report.
// An activation write may have reached the server even when the write fails.
func exchangeUpdate(ctx context.Context, connection io.ReadWriter, release wireRelease, open func(string) (io.ReadCloser, error), handoff func() error) (activated bool, err error) {
	if err := writeWirePreamble(ctx, connection, release); err != nil {
		return false, err
	}
	if err := writeWirePayloads(ctx, connection, release, open); err != nil {
		return false, err
	}
	if err := readControl(ctx, connection, "PREPARED1\n"); err != nil {
		return false, err
	}
	if err := handoff(); err != nil {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := writeWireBytes(ctx, connection, []byte("ACTIVATE1\n")); err != nil {
		return true, err
	}
	return true, readControl(ctx, connection, "COMMITTED1\n")
}

type installerProcess struct {
	done     <-chan error
	finished bool
	result   error
}

func startInstalledInstaller(old InstalledRelease) (*installerProcess, error) {
	// sudo is the fixed OS authentication boundary, not an ambient PATH tool.
	// -n requires existing administrator authentication and never consumes
	// report/JSON input. The user can authenticate separately with sudo -v.
	const sudo = "/usr/bin/sudo"
	if err := protectedDirectory(filepath.Dir(sudo)); err != nil {
		return nil, err
	}
	info, err := os.Lstat(sudo)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0022 != 0 || info.Mode().Perm()&0111 == 0 {
		return nil, ErrInstaller
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != 0 || owner.Nlink != 1 {
		return nil, ErrInstaller
	}
	command := exec.Command(sudo, "-n", "--", nativeVersionPath(old.Version(), "haa-release-installer-linux-amd64"), "serve")
	command.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C", "HOME=/"}
	command.Stdin, command.Stdout, command.Stderr = nil, io.Discard, io.Discard
	if err := command.Start(); err != nil {
		return nil, errors.Join(ErrInstaller, err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	return &installerProcess{done: done}, nil
}

func (p *installerProcess) exited() bool {
	if p.finished {
		return true
	}
	select {
	case p.result = <-p.done:
		p.finished = true
	default:
	}
	return p.finished
}

func (p *installerProcess) wait() error {
	if !p.exited() {
		timer := time.NewTimer(45 * time.Second)
		defer timer.Stop()
		select {
		case p.result = <-p.done:
			p.finished = true
		case <-timer.C:
			return errors.Join(ErrInstaller, ErrTransition)
		}
	}
	if p.result != nil {
		return errors.Join(ErrInstaller, p.result)
	}
	return nil
}

func connectInstalledInstaller(ctx context.Context, gid uint32, process *installerProcess) (*net.UnixConn, error) {
	wait, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	for {
		if err := wait.Err(); err != nil {
			return nil, errors.Join(ErrInstaller, err)
		}
		if process.exited() {
			return nil, errors.Join(ErrInstaller, process.result)
		}
		connection, err := (&net.Dialer{Timeout: 100 * time.Millisecond}).DialContext(wait, "unix", installerSocketPath)
		if err == nil {
			unix, ok := connection.(*net.UnixConn)
			if !ok {
				_ = connection.Close()
				return nil, ErrProtocol
			}
			if err := verifyInstallerEndpoint(unix, gid); err != nil {
				return nil, errors.Join(err, unix.Close())
			}
			return unix, nil
		}
		// Only not-yet-created/stopped sockets are retryable. Permission,
		// identity and other query errors are never interpreted as absence.
		if !errors.Is(err, syscall.ENOENT) && !errors.Is(err, syscall.ECONNREFUSED) {
			return nil, errors.Join(ErrInstaller, err)
		}
		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-wait.Done():
			timer.Stop()
			return nil, errors.Join(ErrInstaller, wait.Err())
		case <-timer.C:
		}
	}
}

func verifyInstallerEndpoint(connection *net.UnixConn, gid uint32) error {
	if err := protectedDirectory(installerRunDirectory); err != nil {
		return err
	}
	before, err := os.Lstat(installerSocketPath)
	if err != nil || before.Mode()&os.ModeSocket == 0 || before.Mode().Perm() != 0660 {
		return ErrUntrusted
	}
	owner, ok := before.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != 0 || owner.Gid != gid || owner.Nlink != 1 {
		return ErrUntrusted
	}
	raw, err := connection.SyscallConn()
	if err != nil {
		return ErrCaller
	}
	var peer *syscall.Ucred
	var credentialErr error
	if err := raw.Control(func(fd uintptr) {
		peer, credentialErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil || credentialErr != nil || peer == nil || peer.Uid != 0 || peer.Gid != 0 || peer.Pid <= 0 {
		return ErrCaller
	}
	after, err := os.Lstat(installerSocketPath)
	if err != nil || !os.SameFile(before, after) {
		return ErrUntrusted
	}
	// sudo may fork a monitor; its PID is not the server's PID. The
	// root-reserved endpoint authenticates root, and Serve independently
	// verifies its actual old signed installer before creating this socket.
	return nil
}
