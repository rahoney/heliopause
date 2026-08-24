//go:build linux

package hosttool

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"

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
