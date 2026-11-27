//go:build linux

package hosttool

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// PolicyPeerAuthorizer accepts only one configured ordinary-user identity
// executing the configured, protected Heliopause client binary.
type PolicyPeerAuthorizer struct {
	uid    uint32
	client identity
}

func NewPolicyPeerAuthorizer(clientPath string, uid uint32) (*PolicyPeerAuthorizer, error) {
	if clientPath == "" {
		return nil, errors.New("network policy client identity is required")
	}
	client, err := verifyExecutable(clientPath, "")
	if err != nil {
		return nil, errors.New("network policy client identity is invalid")
	}
	return &PolicyPeerAuthorizer{uid: uid, client: client}, nil
}

// AuthorizePeer verifies both kernel peer credentials and the peer process's
// executable inode. A claimed UID or request-body path is never trusted.
func (a *PolicyPeerAuthorizer) AuthorizePeer(connection *net.UnixConn) (uint32, error) {
	if a == nil || connection == nil {
		return 0, errors.New("network policy peer is unavailable")
	}
	var credentials *unix.Ucred
	raw, err := connection.SyscallConn()
	if err != nil {
		return 0, errors.New("inspect network policy peer")
	}
	controlErr := raw.Control(func(fd uintptr) {
		credentials, err = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	})
	if controlErr != nil || err != nil || credentials == nil || credentials.Pid <= 0 || credentials.Uid != a.uid {
		return 0, errors.New("network policy peer credential is unauthorized")
	}
	path, readErr := os.Readlink(filepath.Join("/proc", strconv.Itoa(int(credentials.Pid)), "exe"))
	if readErr != nil || !filepath.IsAbs(path) {
		return 0, errors.New("network policy peer executable is unavailable")
	}
	current, verifyErr := verifyExecutable(path, a.client.digest)
	if verifyErr != nil || !os.SameFile(a.client.info, current.info) {
		return 0, errors.New("network policy peer executable is unauthorized")
	}
	return credentials.Uid, nil
}

func exactD3Peer(a *PolicyPeerAuthorizer, connection *net.UnixConn) (d3Peer, error) {
	raw, err := connection.SyscallConn()
	if err != nil {
		return d3Peer{}, err
	}
	var creds *unix.Ucred
	var socketErr error
	if err := raw.Control(func(fd uintptr) {
		creds, socketErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil || socketErr != nil || creds == nil || creds.Pid <= 1 {
		return d3Peer{}, errD3Unconfirmed
	}
	base := filepath.Join("/proc", strconv.Itoa(int(creds.Pid)))
	start, err := d3ProcessStart(base)
	if err != nil {
		return d3Peer{}, err
	}
	uid, err := a.AuthorizePeer(connection)
	again, readErr := d3ProcessStart(base)
	if err != nil || readErr != nil || start != again || uid != creds.Uid {
		return d3Peer{}, errD3Unconfirmed
	}
	return d3Peer{uid: uid, pid: creds.Pid, start: start}, nil
}

func d3ProcessStart(proc string) (string, error) {
	record, err := os.ReadFile(filepath.Join(proc, "stat"))
	if err != nil || len(record) > 4096 {
		return "", errD3Unconfirmed
	}
	_, fields, ok := strings.Cut(string(record), ") ")
	if !ok {
		return "", errD3Unconfirmed
	}
	if end := strings.LastIndex(fields, ") "); end >= 0 {
		fields = fields[end+2:]
	}
	f := strings.Fields(fields)
	if len(f) <= 19 || f[0] == "Z" || f[0] == "X" {
		return "", errD3Unconfirmed
	}
	if start, err := strconv.ParseUint(f[19], 10, 64); err != nil || start == 0 {
		return "", errD3Unconfirmed
	}
	return f[19], nil
}

type d3PolicyAuthorizer struct {
	peer     *PolicyPeerAuthorizer
	recovery *d3Recovery
}

func (a d3PolicyAuthorizer) AuthorizePeer(connection *net.UnixConn) (uint32, error) {
	peer, err := exactD3Peer(a.peer, connection)
	if err != nil {
		return 0, err
	}
	return a.recovery.authorize(peer)
}
