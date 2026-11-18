//go:build linux

package hosttool

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const d3ObserverDirectory = "/run/heliopause-observer"

var d3ObserverEndpoints = [...]string{
	"gvisor-remote.sock", "haa-output.sock", "haa-control.sock", ".heliopause-supervisor.lock",
}

type d3ObserverRecord struct {
	PID       int          `json:"pid"`
	Start     string       `json:"start"`
	Boot      string       `json:"boot"`
	UID       uint32       `json:"uid"`
	Endpoints []d3Endpoint `json:"endpoints"`
}

type d3Endpoint struct {
	Name string `json:"name"`
	Dev  uint64 `json:"dev"`
	Ino  uint64 `json:"ino"`
}

func (e *Executor) PrepareRecoveryObserver(ctx context.Context) error {
	_, err := e.recovery.request.call(ctx, observationResourceRequest{
		Operation: "guard-observer-prepare", Owner: e.recovery.owner,
	})
	return err
}

func (e *Executor) CancelRecoveryObserver(ctx context.Context) error {
	_, err := e.recovery.request.call(ctx, observationResourceRequest{
		Operation: "guard-observer-abort", Owner: e.recovery.owner,
	})
	return err
}

func (e *Executor) RecordRecoveryObserver(ctx context.Context, process ObserverProcess) error {
	child, ok := process.(*observerProcess)
	if !ok || child == nil || child.command == nil || child.command.Process == nil || e.recovery == nil {
		return errD3Unconfirmed
	}
	_, err := e.recovery.request.call(ctx, observationResourceRequest{
		Operation: "guard-observer", Owner: e.recovery.owner, ObserverPID: child.command.Process.Pid,
	})
	if err == nil {
		e.recoveryObserverDone = process.Done()
	}
	return err
}

func (r *d3Recovery) recordObserver(ctx context.Context, generation uint32, peer d3Peer, request observationResourceRequest) error {
	if request.Transaction != "" || request.Container != "" || request.Role != "" || request.Profile != "" {
		return errD3Unconfirmed
	}
	var observerFD = -1
	err := r.change(generation, func(row *d3Record) error {
		if row.Owner != request.Owner || row.Observer != nil {
			return errD3Unconfirmed
		}
		if request.Operation == "guard-observer-prepare" || request.Operation == "guard-observer-abort" {
			preparing := request.Operation == "guard-observer-prepare"
			if row.ObserverPending == preparing || request.ObserverPID != 0 {
				return errD3Unconfirmed
			}
			for _, name := range d3ObserverEndpoints {
				if _, err := os.Lstat(filepath.Join(d3ObserverDirectory, name)); !errors.Is(err, os.ErrNotExist) {
					return errD3Unconfirmed
				}
			}
			row.ObserverPending = preparing
			return nil
		}
		if !row.ObserverPending || request.ObserverPID <= 1 {
			return errD3Unconfirmed
		}
		base := filepath.Join("/proc", strconv.Itoa(request.ObserverPID))
		start, err := d3ProcessStart(base)
		if err != nil || verifyD3ObserverProcess(base, start, int(peer.pid)) != nil {
			return errD3Unconfirmed
		}
		boot, err := d3BootID()
		if err != nil {
			return err
		}
		record := &d3ObserverRecord{PID: request.ObserverPID, Start: start, Boot: boot, UID: peer.uid}
		for _, name := range d3ObserverEndpoints {
			var s unix.Stat_t
			if err := unix.Lstat(filepath.Join(d3ObserverDirectory, name), &s); err != nil || !validD3ObserverEntry(name, s, peer.uid) {
				return errD3Unconfirmed
			}
			record.Endpoints = append(record.Endpoints, d3Endpoint{Name: name, Dev: s.Dev, Ino: s.Ino})
		}
		observerFD, err = unix.PidfdOpen(record.PID, 0)
		if err != nil || verifyD3ObserverProcess(base, start, int(peer.pid)) != nil {
			return errD3Unconfirmed
		}
		row.Observer = record
		row.ObserverPending = false
		return nil
	})
	if observerFD >= 0 {
		if err != nil {
			_ = unix.Close(observerFD)
		} else {
			go r.watchObserver(ctx, generation, observerFD)
		}
	}
	return err
}

func (r *d3Recovery) watchObserver(ctx context.Context, generation uint32, fd int) {
	defer unix.Close(fd)
	poll := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	for ctx.Err() == nil {
		n, err := unix.Poll(poll, 25)
		if err == unix.EINTR || n == 0 && err == nil {
			continue
		}
		// Exit and an unreadable pidfd both revoke new starts. Read-only
		// observations and exact cleanup can still finish.
		r.mu.Lock()
		if r.generation == generation && r.journal.record != nil {
			r.observerDead = true
			stop, cancel := context.WithTimeout(context.WithoutCancel(ctx), 7*time.Second)
			stopErr := errors.Join(stopRecordedD3Containers(r.journal.record), r.stopContainers(stop, r.journal.record))
			cancel()
			if stopErr != nil {
				_, _ = os.Stderr.WriteString("protected observer loss: runtime termination unconfirmed\n")
			}
		}
		r.mu.Unlock()
		return
	}
}

func verifyD3ObserverProcess(proc, start string, parent int) error {
	first, err := d3ProcessStart(proc)
	if err != nil || first != start {
		return errD3Unconfirmed
	}
	config, err := loadSystemConfig()
	if err != nil {
		return err
	}
	identity, err := verifyExecutable(config.ObserverHelperPath, "")
	if err != nil {
		return err
	}
	info, err := os.Stat(filepath.Join(proc, "exe"))
	if err != nil || !os.SameFile(identity.info, info) {
		return errD3Unconfirmed
	}
	if parent != 0 {
		body, err := os.ReadFile(filepath.Join(proc, "status"))
		if err != nil || len(body) > 8192 {
			return errD3Unconfirmed
		}
		want := "PPid:\t" + strconv.Itoa(parent) + "\n"
		if !strings.Contains("\n"+string(body), "\n"+want) {
			return errD3Unconfirmed
		}
	}
	again, err := d3ProcessStart(proc)
	if err != nil || again != start {
		return errD3Unconfirmed
	}
	return nil
}

func validD3ObserverEntry(name string, stat unix.Stat_t, uid uint32) bool {
	kind := uint32(unix.S_IFSOCK)
	if name == ".heliopause-supervisor.lock" {
		kind = unix.S_IFREG
	}
	return stat.Mode&unix.S_IFMT == kind && stat.Mode&0o022 == 0 && stat.Uid == uid && stat.Nlink == 1
}

func stopD3Observer(ctx context.Context, record *d3ObserverRecord) error {
	if record == nil {
		return nil
	}
	if err := stopD3ObserverProcess(ctx, record); err != nil {
		return fmt.Errorf("stop recorded observer process: %w", err)
	}
	file, err := os.Open("/proc/net/unix")
	if err != nil {
		return fmt.Errorf("open observer socket census: %w", err)
	}
	body, err := io.ReadAll(io.LimitReader(file, 1<<20))
	if errors.Join(err, file.Close()) != nil || len(body) == 1<<20 {
		return fmt.Errorf("read bounded observer socket census: %w", errD3Unconfirmed)
	}
	for i, endpoint := range record.Endpoints {
		if endpoint.Name != d3ObserverEndpoints[i] || endpoint.Ino == 0 {
			return errD3Unconfirmed
		}
		path := filepath.Join(d3ObserverDirectory, endpoint.Name)
		for _, line := range strings.Split(string(body), "\n") {
			parts := strings.Fields(line)
			if len(parts) == 8 && parts[7] == path {
				return fmt.Errorf("recorded observer endpoint %s is still active: %w", endpoint.Name, errD3Unconfirmed)
			}
		}
		if err := removeD3ObserverEndpoint(path, endpoint, record.UID); err != nil {
			return fmt.Errorf("retire recorded observer endpoint %s: %w", endpoint.Name, err)
		}
	}
	return nil
}

func d3BootID() (string, error) {
	body, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	id := strings.TrimSpace(string(body))
	if err != nil || len(id) != 36 {
		return "", errD3Unconfirmed
	}
	return id, nil
}

func stopD3ObserverProcess(ctx context.Context, record *d3ObserverRecord) error {
	if record == nil {
		return nil
	}
	if record.PID <= 1 || len(record.Endpoints) != len(d3ObserverEndpoints) {
		return errD3Unconfirmed
	}
	base := filepath.Join("/proc", strconv.Itoa(record.PID))
	boot, err := d3BootID()
	if err != nil || len(record.Boot) != 36 {
		return errD3Unconfirmed
	}
	if boot != record.Boot {
		return nil
	}
	fd, err := unix.PidfdOpen(record.PID, 0)
	if errors.Is(err, unix.ESRCH) {
		return nil
	}
	if err != nil {
		return errD3Unconfirmed
	}
	defer unix.Close(fd)
	poll := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	if n, err := unix.Poll(poll, 0); err == nil && n == 1 && poll[0].Revents&unix.POLLIN != 0 {
		return nil
	}
	current, err := d3ProcessStart(base)
	if err == nil && current != record.Start {
		return nil
	}
	if err != nil || verifyD3ObserverProcess(base, record.Start, 0) != nil || unix.PidfdSendSignal(fd, unix.SIGTERM, nil, 0) != nil {
		return errD3Unconfirmed
	}
	wait, cancel := context.WithTimeout(ctx, observerReadinessTimeout)
	defer cancel()
	for wait.Err() == nil {
		if n, err := unix.Poll(poll, 25); err == nil && n == 1 && poll[0].Revents&unix.POLLIN != 0 {
			return nil
		}
	}
	return errD3Unconfirmed
}

func removeD3ObserverEndpoint(path string, endpoint d3Endpoint, uid uint32) (result error) {
	var s unix.Stat_t
	err := unix.Lstat(path, &s)
	if errors.Is(err, unix.ENOENT) {
		return nil
	}
	if err != nil || !validD3ObserverEntry(endpoint.Name, s, uid) || s.Dev != endpoint.Dev || s.Ino != endpoint.Ino {
		return errD3Unconfirmed
	}
	// A restricted service may have separate mounts for the policy and
	// observer directories. Pin a private directory within the observer mount.
	target, err := os.MkdirTemp(filepath.Dir(path), ".retired-observer-")
	if err != nil {
		return errors.Join(errD3Unconfirmed, err)
	}
	fd, err := unix.Open(target, unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return errors.Join(errD3Unconfirmed, err)
	}
	defer unix.Close(fd)
	var dir unix.Stat_t
	if err := unix.Fstat(fd, &dir); err != nil || dir.Mode != unix.S_IFDIR|0o700 || dir.Uid != uint32(os.Geteuid()) {
		return errors.Join(errD3Unconfirmed, err)
	}
	defer func() {
		var current unix.Stat_t
		err := unix.Lstat(target, &current)
		if err != nil || current.Dev != dir.Dev || current.Ino != dir.Ino {
			result = errors.Join(result, errD3Unconfirmed, err)
			return
		}
		// Use rmdir so a retained endpoint or replacement is never recursed into.
		result = errors.Join(result, unix.Rmdir(target))
	}()
	const moved = "endpoint"
	if err := unix.Renameat2(unix.AT_FDCWD, path, fd, moved, unix.RENAME_NOREPLACE); err != nil {
		return errors.Join(errD3Unconfirmed, err)
	}
	if err := unix.Fstatat(fd, moved, &s, unix.AT_SYMLINK_NOFOLLOW); err != nil || !validD3ObserverEntry(endpoint.Name, s, uid) || s.Dev != endpoint.Dev || s.Ino != endpoint.Ino {
		restoreErr := unix.Renameat2(fd, moved, unix.AT_FDCWD, path, unix.RENAME_NOREPLACE)
		return errors.Join(errD3Unconfirmed, err, restoreErr)
	}
	return unix.Unlinkat(fd, moved, 0)
}
