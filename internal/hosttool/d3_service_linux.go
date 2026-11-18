//go:build linux

package hosttool

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const d3RecoveryTimeout = 15 * time.Second

func openD3Recovery(executor *Executor, initialize bool) (*d3Recovery, error) {
	if err := verifyTrustedParents(filepath.Dir(d3Directory)); err != nil {
		return nil, err
	}
	journal, err := openD3Journal(d3Directory, initialize)
	if err != nil {
		return nil, err
	}
	return &d3Recovery{journal: journal, runner: executor}, nil
}

func (r *d3Recovery) hold(ctx context.Context, connection *net.UnixConn, reader *bufio.Reader, peer d3Peer) {
	owner, err := r.acquire(peer)
	if err != nil {
		return
	}
	var released bool
	if json.NewEncoder(connection).Encode(observationResourceResponse{OK: true, Owner: owner}) == nil {
		_ = connection.SetDeadline(time.Time{})
		stop := context.AfterFunc(ctx, func() { _ = connection.Close() })
		body, err := reader.ReadSlice('\n')
		stop()
		released = err == nil && string(body) == "{\"operation\":\"release\"}\n"
	}
	r.closeAdmission(peer)
	recoveryContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), d3RecoveryTimeout)
	defer cancel()
	err = r.recover(recoveryContext, false)
	if released && err == nil {
		_ = connection.SetDeadline(time.Now().Add(5 * time.Second))
		_ = json.NewEncoder(connection).Encode(observationResourceResponse{OK: true})
	}
}

func (r *d3Recovery) dockerRequest(ctx context.Context, generation uint32, request observationResourceRequest) error {
	if request.Role != "" || request.Profile != "" || request.Transaction != "" {
		return errD3Unconfirmed
	}
	if request.Operation == "guard-exec" {
		r.mu.Lock()
		defer r.mu.Unlock()
		row := r.journal.record
		if r.fault || r.observerDead || r.peer.pid == 0 || r.generation != generation || row == nil || row.Owner != request.Owner ||
			!slices.ContainsFunc(row.Containers, func(c d3Container) bool { return c.ID == request.Container }) {
			return errD3Unconfirmed
		}
		return nil
	}
	return r.change(generation, func(row *d3Record) error {
		if row.Owner != request.Owner || row.ObserverPending || r.observerDead && request.Operation != "guard-created" {
			return errD3Unconfirmed
		}
		switch request.Operation {
		case "guard-create":
			if request.Container != "" || row.Mutations >= 64 {
				return errD3Unconfirmed
			}
			row.Mutations++
		case "guard-created":
			if request.Container != "" || row.Mutations == 0 {
				return errD3Unconfirmed
			}
			row.Mutations--
		case "guard-start":
			if !validDockerContainerID(request.Container) {
				return errD3Unconfirmed
			}
			scope, exists, err := r.inspectContainer(ctx, row.Owner, request.Container)
			if err != nil || !exists {
				return errD3Unconfirmed
			}
			c := d3Container{ID: request.Container, Scope: scope}
			if slices.ContainsFunc(row.Containers, func(v d3Container) bool { return v.ID == c.ID && v.Scope != c.Scope }) {
				return errD3Unconfirmed
			}
			if !slices.Contains(row.Containers, c) {
				row.Containers = append(row.Containers, c)
			}
		default:
			return errD3Unconfirmed
		}
		return nil
	})
}

func (r *d3Recovery) observationRequest(ctx context.Context, e *observationResourceEngine, generation uint32, request observationResourceRequest) (ObservationResourceLease, error) {
	if request.Operation != "create" && request.Operation != "close" {
		return e.apply(ctx, generation, request)
	}
	parent, err := observationParent(request.Transaction)
	if err != nil {
		return ObservationResourceLease{}, err
	}
	if request.Operation == "create" {
		err = r.change(generation, func(row *d3Record) error {
			if r.observerDead || row.Mutations >= 64 || slices.Contains(row.Keepers, parent) {
				return errD3Unconfirmed
			}
			row.Keepers = append(row.Keepers, parent)
			row.Mutations++
			return nil
		})
		if err != nil {
			return ObservationResourceLease{}, err
		}
	}
	lease, applyErr := e.apply(ctx, generation, request)
	// Most failed transactions are still removed by an authoritative close.
	// Keep the intent when the helper cannot prove that state transition.
	if applyErr != nil {
		return lease, applyErr
	}
	err = r.change(generation, func(row *d3Record) error {
		if request.Operation == "create" {
			if row.Mutations == 0 {
				return errD3Unconfirmed
			}
			row.Mutations--
		} else {
			row.Keepers = slices.DeleteFunc(row.Keepers, func(v string) bool { return v == parent })
		}
		return nil
	})
	return lease, err
}

// RecoverProtectedExecution is the fixed post-stop entry point of the existing
// network-policy unit. It opens only the protected journal and installed tools.
func RecoverProtectedExecution(ctx context.Context) error {
	if ctx == nil {
		return errD3Unconfirmed
	}
	bound, cancel := context.WithTimeout(ctx, d3RecoveryTimeout)
	defer cancel()
	if os.Geteuid() != 0 {
		return errD3Unconfirmed
	}
	recovery, err := openD3Recovery(nil, false)
	if err != nil {
		return errors.Join(errD3Unconfirmed, err)
	}
	defer recovery.journal.lock.Close()
	var stopErr error
	if record := recovery.journal.record; record != nil {
		// Verified cgroup identities are available even with Docker down.
		stopErr = errors.Join(stopRecordedD3Containers(record), stopD3ObserverProcess(bound, record.Observer))
	}
	config, err := loadPolicyServiceConfig()
	if err != nil {
		return errors.Join(stopErr, err)
	}
	socketErr := recovery.retireHelperSockets(config)
	if recovery.journal.record == nil {
		return errors.Join(stopErr, socketErr)
	}
	executor, err := newD3Executor(bound)
	if err != nil {
		return errors.Join(stopErr, socketErr, err)
	}
	defer executor.Close()
	recovery.runner = executor
	return errors.Join(stopErr, socketErr, recovery.recover(bound, true))
}

func (r *d3Recovery) publishSocket(path string, group uint32) (*net.UnixListener, os.FileInfo, error) {
	dir, err := os.MkdirTemp(filepath.Dir(path), ".listener-")
	if err != nil {
		return nil, nil, err
	}
	defer os.Remove(dir)
	staged := filepath.Join(dir, "sock")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: staged, Net: "unix"})
	if err != nil {
		return nil, nil, err
	}
	info, err := os.Lstat(staged)
	if err != nil || os.Chmod(staged, 0o660) != nil || os.Chown(staged, 0, int(group)) != nil {
		_ = listener.Close()
		return nil, nil, errD3Unconfirmed
	}
	if err := r.saveSocket(path, info); err != nil {
		_ = listener.Close()
		return nil, nil, err
	}
	if err := unix.Renameat2(unix.AT_FDCWD, staged, unix.AT_FDCWD, path, unix.RENAME_NOREPLACE); err != nil {
		_ = listener.Close()
		return nil, nil, err
	}
	listener.SetUnlinkOnClose(false)
	return listener, info, nil
}

func (r *d3Recovery) saveSocket(path string, file os.FileInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if file == nil || file.Mode()&os.ModeSocket == 0 {
		return errD3Unconfirmed
	}
	stat, ok := file.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 || len(r.journal.sockets) >= 2 {
		return errD3Unconfirmed
	}
	r.journal.sockets = append(r.journal.sockets, d3Endpoint{Name: path, Dev: stat.Dev, Ino: stat.Ino})
	return r.write(r.journal.record)
}

func (r *d3Recovery) retireHelperSockets(config policyServiceConfig) error {
	for _, endpoint := range r.journal.sockets {
		if endpoint.Name != config.SocketPath && endpoint.Name != observationResourceSocket {
			return errD3Unconfirmed
		}
		current, err := os.Lstat(endpoint.Name)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil || verifyPolicySocket(endpoint.Name, config.ClientGID) != nil {
			return errD3Unconfirmed
		}
		stat, ok := current.Sys().(*syscall.Stat_t)
		if !ok || stat.Dev != endpoint.Dev || stat.Ino != endpoint.Ino {
			return errD3Unconfirmed
		}
		if err := os.Remove(endpoint.Name); err != nil {
			return errD3Unconfirmed
		}
	}
	if len(r.journal.sockets) == 0 {
		return nil
	}
	r.journal.sockets = nil
	return r.write(r.journal.record)
}

// A signed candidate started by the previous native updater may receive a
// legacy service file. Refuse that pairing before opening recovery authority.
func verifyD3Unit(ctx context.Context, executor *Executor) error {
	body, err := os.ReadFile("/proc/self/cgroup")
	if err != nil || len(body) > 1024 {
		return errD3Unconfirmed
	}
	group, found := strings.CutPrefix(strings.TrimSpace(string(body)), "0::/system.slice/")
	if !found || group != "haa-network-policy-helper.service" && group != "helox-network-policy.service" {
		// Existing explicitly supervised foreground qualification has its
		// own launcher. It does not qualify the native independent stop hook.
		return nil
	}
	return verifyManagedD3Unit(ctx, executor, group)
}

func verifyManagedD3Unit(ctx context.Context, executor *Executor, unit string) error {
	executable, err := os.Readlink("/proc/self/exe")
	if err != nil || strings.ContainsAny(executable, " \t\n\r;{}") {
		return errD3Unconfirmed
	}
	body, err := executor.Output(ctx, "systemctl", "show", unit, "-p", "MainPID", "-p", "NeedDaemonReload", "-p", "ExecStopPost")
	if err != nil || len(body) > 4096 || !exactSystemdProperty(body, "MainPID", strconv.Itoa(os.Getpid())) ||
		!exactSystemdProperty(body, "NeedDaemonReload", "no") {
		return errD3Unconfirmed
	}
	action, ok := systemdProperty(body, "ExecStopPost")
	prefix := "{ path=" + executable + " ; argv[]=" + executable + " recover ; ignore_errors=no ;"
	if !ok || !strings.HasPrefix(action, prefix) || strings.Count(action, "{") != 1 ||
		strings.Count(action, "}") != 1 || !strings.HasSuffix(action, "}") {
		return errors.New("native helper unit does not have the required recovery stop action")
	}
	return nil
}
