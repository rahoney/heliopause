//go:build linux

package releaseactivation

import (
	"bufio"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/rahoney/heliopause/internal/hosttool"
	"golang.org/x/sys/unix"
)

var ErrCaller = errors.New("SELF_UPDATE_PROTECTED_CONTROLLER_REQUIRED")

// CheckRuntimeController prevents a stale/development executable from using a
// managed installation's newly rebound helpers. It grants no Policy approval.
func CheckRuntimeController(installed InstalledRelease) error {
	if installed.record.Schema != registrationSchema || len(installed.proof) == 0 || os.Geteuid() == 0 || uint32(os.Geteuid()) != installed.ClientUID() || uint32(os.Getegid()) != installed.ClientGID() {
		return ErrCaller
	}
	self, err := os.Stat("/proc/self/exe")
	if err != nil {
		return ErrCaller
	}
	current, err := os.Lstat(installed.ClientPath())
	if err != nil || !os.SameFile(self, current) {
		return ErrCaller
	}
	return nil
}

type controllerPeer struct {
	mu       sync.Mutex
	pidfd    *os.File
	pid      int32
	uid, gid uint32
	groups   []uint32
	closed   bool
	closeErr error
}

func authorizeOldController(connection *net.UnixConn, installed InstalledRelease) (*controllerPeer, error) {
	if os.Geteuid() != 0 || connection == nil || installed.record.Schema != registrationSchema || installed.ClientUID() == 0 || len(installed.proof) == 0 {
		return nil, ErrCaller
	}
	credentials, err := peerCredentials(connection)
	if err != nil {
		return nil, err
	}
	if credentials.Pid <= 0 || credentials.Uid != installed.ClientUID() || credentials.Gid != installed.ClientGID() {
		return nil, ErrCaller
	}
	fd, err := unix.PidfdOpen(int(credentials.Pid), 0)
	if err != nil {
		return nil, ErrCaller
	}
	peer := &controllerPeer{pidfd: os.NewFile(uintptr(fd), "protected-controller"), pid: credentials.Pid, uid: credentials.Uid, gid: credentials.Gid}
	fail := func(cause error) (*controllerPeer, error) { return nil, errors.Join(cause, peer.Close()) }
	if err := peer.alive(); err != nil {
		return fail(err)
	}
	// Independently verified signed bytes were supplied by ReadInstalledRelease.
	// The existing authorizer supplies the kernel executable/inode boundary.
	authorizer, err := hosttool.NewPolicyPeerAuthorizer(installed.ClientPath(), installed.ClientUID())
	if err != nil {
		return fail(ErrCaller)
	}
	uid, err := authorizer.AuthorizePeer(connection)
	if err != nil || uid != peer.uid {
		return fail(ErrCaller)
	}
	status, err := os.Open(filepath.Join("/proc", strconv.Itoa(int(peer.pid)), "status"))
	if err != nil {
		return fail(ErrCaller)
	}
	groups, readErr := kernelGroups(status)
	closeErr := status.Close()
	if readErr != nil || closeErr != nil {
		return fail(ErrCaller)
	}
	peer.groups = groups
	if err := peer.alive(); err != nil {
		return fail(err)
	}
	return peer, nil
}

func peerCredentials(connection *net.UnixConn) (*unix.Ucred, error) {
	raw, err := connection.SyscallConn()
	if err != nil {
		return nil, ErrCaller
	}
	var credentials *unix.Ucred
	var socketErr error
	if err := raw.Control(func(fd uintptr) {
		credentials, socketErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil || socketErr != nil || credentials == nil {
		return nil, ErrCaller
	}
	return credentials, nil
}

func kernelGroups(file *os.File) ([]uint32, error) {
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 64<<10)
	var result []uint32
	found := false
	for scanner.Scan() {
		values, match := strings.CutPrefix(scanner.Text(), "Groups:")
		if !match {
			continue
		}
		if found {
			return nil, ErrCaller
		}
		found = true
		fields := strings.Fields(values)
		if len(fields) > 64 {
			return nil, ErrCaller
		}
		seen := map[uint32]bool{}
		for _, field := range fields {
			value, err := strconv.ParseUint(field, 10, 31)
			if err != nil || seen[uint32(value)] {
				return nil, ErrCaller
			}
			seen[uint32(value)] = true
			result = append(result, uint32(value))
		}
	}
	if scanner.Err() != nil || !found {
		return nil, ErrCaller
	}
	return result, nil
}

func (p *controllerPeer) alive() error {
	if p == nil {
		return ErrCaller
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.pidfd == nil {
		return ErrCaller
	}
	fds := []unix.PollFd{{Fd: int32(p.pidfd.Fd()), Events: unix.POLLIN}}
	count, err := unix.Poll(fds, 0)
	if err != nil || count != 0 || fds[0].Revents != 0 {
		return ErrCaller
	}
	return nil
}

func (p *controllerPeer) Close() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return p.closeErr
	}
	p.closed = true
	if p.pidfd != nil && p.pidfd.Close() != nil {
		p.closeErr = ErrCaller
	}
	p.pidfd = nil
	return p.closeErr
}
