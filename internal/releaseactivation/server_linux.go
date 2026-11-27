//go:build linux

package releaseactivation

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

const (
	installerRunDirectory = "/run/heliopause-install"
	installerSocketPath   = installerRunDirectory + "/installer.sock"
)

// Serve is a one-shot installation-role endpoint. All content comes through
// the fixed codec after kernel authorization of the old protected controller.
func Serve(ctx context.Context) (resultErr error) {
	if ctx == nil || runtime.GOARCH != "amd64" || os.Geteuid() != 0 || os.Getegid() != 0 {
		return ErrCaller
	}
	owner, err := acquireInstallerOwner(ctx)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, owner.Close()) }()
	managed, err := managedStatePresent()
	if err != nil {
		return err
	}
	if !managed {
		return ErrUnmanaged
	}
	old, err := ReadInstalledRelease(ctx)
	if err != nil {
		return err
	}
	if err := verifyInstallerController(old); err != nil {
		return err
	}
	listener, identity, err := listenInstaller(old.ClientGID())
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, closeInstallerListener(listener, identity)) }()
	if err := listener.SetDeadline(time.Now().Add(90 * time.Second)); err != nil {
		return ErrProtocol
	}
	stop := context.AfterFunc(ctx, func() { _ = listener.SetDeadline(time.Now()) })
	defer stop()
	connection, err := listener.AcceptUnix()
	if err != nil {
		return errors.Join(ErrProtocol, err, ctx.Err())
	}
	defer func() {
		if err := connection.Close(); err != nil {
			resultErr = errors.Join(resultErr, ErrProtocol, err)
		}
	}()
	if err := connection.SetDeadline(time.Now().Add(30 * time.Minute)); err != nil {
		return ErrProtocol
	}
	stopConnection := context.AfterFunc(ctx, func() { _ = connection.SetDeadline(time.Now()) })
	defer stopConnection()
	peer, err := authorizeOldController(connection, old)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, peer.Close()) }()
	stage, err := prepareWireRelease(ctx, connection, peer, old)
	if err != nil {
		return err
	}
	activation, err := prepareRootActivation(ctx, stage, peer)
	if activation == nil {
		// No journal was issued by this constructor. Query errors are not
		// absence; uncertain markers retain the stage for root recovery.
		if journalConfirmedAbsent() {
			err = errors.Join(err, stage.Close())
		}
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, activation.Close()) }()
	if err != nil {
		return err
	}
	prepared := true
	defer func() {
		if !prepared || activation.retired {
			return
		}
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := activation.refreshJournal(cleanup); err != nil {
			resultErr = errors.Join(resultErr, ErrTransition, err)
			return
		}
		if activation.journal.value.State == "PREPARED" {
			if err := activation.abortPrepared(cleanup); err != nil {
				resultErr = errors.Join(resultErr, err, activation.requireRecovery(cleanup))
			}
		}
	}()
	if err := writeWireBytes(ctx, connection, []byte("PREPARED1\n")); err != nil {
		return err
	}
	if err := readControl(ctx, connection, "ACTIVATE1\n"); err != nil {
		return err
	}
	if err := peer.alive(); err != nil {
		return err
	}
	if err := activation.activate(ctx); err != nil {
		return err
	}
	prepared = false
	return writeWireBytes(ctx, connection, []byte("COMMITTED1\n"))
}

func listenInstaller(gid uint32) (*net.UnixListener, os.FileInfo, error) {
	if os.Geteuid() != 0 || gid > 1<<31-1 {
		return nil, nil, ErrCaller
	}
	if err := protectedDirectory(filepath.Dir(installerRunDirectory)); err != nil {
		return nil, nil, err
	}
	if _, err := os.Lstat(installerRunDirectory); errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(installerRunDirectory, 0755); err != nil {
			return nil, nil, ErrUntrusted
		}
		if err := os.Chmod(installerRunDirectory, 0755); err != nil {
			return nil, nil, ErrUntrusted
		}
	} else if err != nil {
		return nil, nil, ErrUntrusted
	}
	if err := protectedDirectory(installerRunDirectory); err != nil {
		return nil, nil, err
	}
	if err := reclaimStoppedInstallerSocket(gid); err != nil {
		return nil, nil, err
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: installerSocketPath, Net: "unix"})
	if err != nil {
		return nil, nil, ErrBusy
	}
	listener.SetUnlinkOnClose(false)
	identity, err := os.Lstat(installerSocketPath)
	if err != nil {
		_ = listener.Close()
		return nil, nil, ErrCleanup
	}
	if err := os.Chmod(installerSocketPath, 0660); err != nil {
		return nil, nil, errors.Join(ErrUntrusted, closeInstallerListener(listener, identity))
	}
	if err := os.Chown(installerSocketPath, 0, int(gid)); err != nil {
		return nil, nil, errors.Join(ErrUntrusted, closeInstallerListener(listener, identity))
	}
	return listener, identity, nil
}

// The fixed root-reserved directory, socket type/owner/mode/group and an
// actual ECONNREFUSED are all required. Any live/unknown/foreign endpoint is
// retained. The caller owns the separate persistent installer lock.
func reclaimStoppedInstallerSocket(gid uint32) error {
	identity, err := os.Lstat(installerSocketPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || identity.Mode()&os.ModeSocket == 0 || identity.Mode().Perm() != 0660 {
		return ErrCleanup
	}
	owner, ok := identity.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != 0 || owner.Gid != gid || owner.Nlink != 1 {
		return ErrCleanup
	}
	connection, err := net.DialTimeout("unix", installerSocketPath, time.Second)
	if err == nil {
		_ = connection.Close()
		return ErrBusy
	}
	if !errors.Is(err, syscall.ECONNREFUSED) {
		return errors.Join(ErrCleanup, err)
	}
	current, err := os.Lstat(installerSocketPath)
	if err != nil || !os.SameFile(current, identity) {
		return ErrCleanup
	}
	if err := os.Remove(installerSocketPath); err != nil {
		return ErrCleanup
	}
	return nil
}

func closeInstallerListener(listener *net.UnixListener, identity os.FileInfo) error {
	var result error
	if listener != nil {
		if err := listener.Close(); err != nil {
			result = errors.Join(result, ErrCleanup, err)
		}
	}
	current, err := os.Lstat(installerSocketPath)
	if err != nil || identity == nil || !os.SameFile(current, identity) || current.Mode()&os.ModeSocket == 0 {
		return errors.Join(result, ErrCleanup)
	}
	if err := os.Remove(installerSocketPath); err != nil {
		return errors.Join(result, ErrCleanup, err)
	}
	return result
}
