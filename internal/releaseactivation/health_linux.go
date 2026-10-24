//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const ProtectedHealthArgument = "--internal-protected-update-health"

var ErrHealth = errors.New("SELF_UPDATE_COMPATIBILITY_HEALTH_FAILED")

type borrowedRuntimeKey struct{}
type borrowedRuntime struct{ installed InstalledRelease }

type healthMessage struct {
	Old            registration `json:"old"`
	ManifestSHA256 string       `json:"manifest_sha256"`
	Mode           string       `json:"mode"`
}

func parseHealthMessage(body []byte) (healthMessage, error) {
	var message healthMessage
	if len(body) == 0 || len(body) > 16<<10 || json.Unmarshal(body, &message) != nil {
		return message, ErrHealth
	}
	canonical, err := json.Marshal(message)
	if err != nil || !bytes.Equal(body, canonical) {
		return message, ErrHealth
	}
	old, err := json.Marshal(message.Old)
	if err != nil {
		return message, ErrHealth
	}
	if _, err := parseRegistration(old); err != nil {
		return message, errors.Join(ErrHealth, err)
	}
	if !sha256Text(message.ManifestSHA256) || (message.Mode != "CANDIDATE" && message.Mode != "ROLLBACK") {
		return message, ErrHealth
	}
	return message, nil
}

func readControl(ctx context.Context, reader io.Reader, expected string) error {
	if ctx == nil || reader == nil {
		return ErrProtocol
	}
	body := make([]byte, len(expected))
	if _, err := io.ReadFull(protectedContextReader{ctx, reader}, body); err != nil {
		return errors.Join(ErrProtocol, err, ctx.Err())
	}
	if string(body) != expected {
		return ErrProtocol
	}
	return nil
}

func unixFileConnection(file *os.File) (*net.UnixConn, error) {
	connection, err := net.FileConn(file)
	if err != nil {
		return nil, ErrHealth
	}
	local, ok := connection.(*net.UnixConn)
	if !ok {
		_ = connection.Close()
		return nil, ErrHealth
	}
	return local, nil
}

func socketCredentials(connection *net.UnixConn) (*unix.Ucred, error) {
	raw, err := connection.SyscallConn()
	if err != nil {
		return nil, ErrHealth
	}
	var credentials *unix.Ucred
	var credentialErr error
	if err := raw.Control(func(fd uintptr) {
		credentials, credentialErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil || credentialErr != nil || credentials == nil {
		return nil, ErrHealth
	}
	return credentials, nil
}

// RunProtectedHealth is reachable only from the CLI's fixed internal entry.
// A flag alone grants nothing: root-created FD3, the actual parent and signed
// old installer FD4 are checked before a private borrowed context is issued.
func RunProtectedHealth(ctx context.Context, check func(context.Context, bool) error) (resultErr error) {
	if ctx == nil || check == nil || os.Geteuid() == 0 {
		return ErrHealth
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Do not take ownership of arbitrary process descriptors when a user
	// supplies the internal flag without the inherited installation channel.
	credentials, err := unix.GetsockoptUcred(3, unix.SOL_SOCKET, unix.SO_PEERCRED)
	if err != nil || credentials == nil || credentials.Uid != 0 || credentials.Gid != 0 || credentials.Pid <= 0 || int(credentials.Pid) != os.Getppid() {
		return ErrHealth
	}
	var inherited unix.Stat_t
	flags, err := unix.FcntlInt(4, unix.F_GETFL, 0)
	if err != nil || flags&unix.O_ACCMODE != unix.O_RDONLY || unix.Fstat(4, &inherited) != nil || inherited.Mode&unix.S_IFMT != unix.S_IFREG || inherited.Uid != 0 || inherited.Nlink != 1 || inherited.Mode&07777 != 0555 {
		return ErrHealth
	}
	channelFile := os.NewFile(3, "protected-health-channel")
	installerFile := os.NewFile(4, "protected-health-issuer")
	defer func() {
		if err := channelFile.Close(); err != nil {
			resultErr = errors.Join(resultErr, ErrHealth, err)
		}
		if err := installerFile.Close(); err != nil {
			resultErr = errors.Join(resultErr, ErrHealth, err)
		}
	}()
	connection, err := unixFileConnection(channelFile)
	if err != nil {
		return err
	}
	defer func() {
		if err := connection.Close(); err != nil {
			resultErr = errors.Join(resultErr, ErrHealth, err)
		}
	}()
	stop := context.AfterFunc(ctx, func() { _ = connection.SetDeadline(time.Now()) })
	defer stop()
	if err := connection.SetDeadline(time.Now().Add(10 * time.Minute)); err != nil {
		return ErrHealth
	}
	creator, err := socketCredentials(connection)
	if err != nil || creator.Uid != 0 || creator.Gid != 0 || creator.Pid <= 0 || int(creator.Pid) != os.Getppid() {
		return ErrHealth
	}
	if err := readControl(ctx, connection, "HAAHEALTH1\n"); err != nil {
		return err
	}
	body, err := readWireFrame(ctx, protectedContextReader{ctx, connection}, 16<<10)
	if err != nil {
		return err
	}
	message, err := parseHealthMessage(body)
	if err != nil {
		return err
	}
	old, err := readRecordedRelease(ctx, message.Old)
	if err != nil {
		return err
	}
	issuerPath := nativeVersionPath(old.Version(), "haa-release-installer-linux-amd64")
	issuer, err := installerFile.Stat()
	if err != nil {
		return ErrHealth
	}
	expected, err := os.Stat(issuerPath)
	if err != nil || !os.SameFile(issuer, expected) {
		return ErrHealth
	}
	owner, ok := issuer.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != 0 || owner.Nlink != 1 || issuer.Mode().Perm() != 0555 {
		return ErrHealth
	}
	current, err := ReadInstalledRelease(ctx)
	if err != nil {
		return err
	}
	if current.manifest.SHA256() != message.ManifestSHA256 || current.ClientUID() != old.ClientUID() || current.ClientGID() != old.ClientGID() {
		return ErrHealth
	}
	if current.manifest.RuntimeLockSHA256() != old.manifest.RuntimeLockSHA256() || message.Mode == "ROLLBACK" && current.manifest.SHA256() != old.manifest.SHA256() {
		return ErrHealth
	}
	if err := CheckRuntimeController(current); err != nil {
		return err
	}
	borrowed := context.WithValue(ctx, borrowedRuntimeKey{}, borrowedRuntime{current})
	if err := check(borrowed, message.Mode == "CANDIDATE"); err != nil {
		return errors.Join(ErrHealth, err)
	}
	if os.Getppid() != int(creator.Pid) {
		return ErrHealth
	}
	if err := writeWireBytes(ctx, connection, []byte("HEALTHY1\n")); err != nil {
		return err
	}
	return readControl(ctx, connection, "RELEASE1\n")
}

type healthWorkspace struct {
	path     string
	identity os.FileInfo
	uid      uint32
	closed   bool
	closeErr error
}

func healthWorkspacePath(stage string) (string, error) {
	if len(stage) > 128 || filepath.Base(stage) != stage || len(stage) <= len(".prepared-") || stage[:len(".prepared-")] != ".prepared-" || strings.ContainsAny(stage, "\x00\r\n\\") {
		return "", ErrUntrusted
	}
	return filepath.Join(stateRoot, "health-"+stage), nil
}

func createHealthWorkspace(stage string, installed InstalledRelease) (*healthWorkspace, error) {
	if os.Geteuid() != 0 || installed.ClientUID() == 0 {
		return nil, ErrCaller
	}
	path, err := healthWorkspacePath(stage)
	if err != nil {
		return nil, err
	}
	if err := protectedDirectory(stateRoot); err != nil {
		return nil, err
	}
	if err := os.Mkdir(path, 0700); err != nil {
		return nil, ErrUntrusted
	}
	identity, err := os.Lstat(path)
	if err != nil {
		return nil, errors.Join(ErrUntrusted, ErrCleanup)
	}
	workspace := &healthWorkspace{path: path, identity: identity}
	if err := os.Chmod(path, 0700); err != nil {
		return workspace, ErrUntrusted
	}
	if err := os.Chown(path, int(installed.ClientUID()), int(installed.ClientGID())); err != nil {
		return workspace, ErrUntrusted
	}
	workspace.uid = installed.ClientUID()
	if err := syncProtectedDirectory(stateRoot); err != nil {
		return workspace, err
	}
	return workspace, nil
}

// Close is called only after independent resource drainage, including failure
// paths. Unknown/in-use workspaces remain reconstructible from the journal.
func (w *healthWorkspace) Close() error {
	if w == nil {
		return nil
	}
	if w.closed {
		return w.closeErr
	}
	w.closed = true
	current, err := os.Lstat(w.path)
	if err != nil || !os.SameFile(current, w.identity) || !current.IsDir() || current.Mode()&os.ModeSymlink != 0 {
		w.closeErr = ErrCleanup
		return w.closeErr
	}
	owner, ok := current.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != w.uid {
		w.closeErr = ErrCleanup
		return w.closeErr
	}
	if err := os.RemoveAll(w.path); err != nil {
		w.closeErr = ErrCleanup
		return w.closeErr
	}
	w.closeErr = syncProtectedDirectory(stateRoot)
	return w.closeErr
}

func verifyInstallerController(installed InstalledRelease) error {
	if os.Geteuid() != 0 || os.Getegid() != 0 || installed.record.Schema != registrationSchema || len(installed.proof) == 0 {
		return ErrCaller
	}
	actual, err := os.Stat("/proc/self/exe")
	if err != nil {
		return ErrCaller
	}
	expected, err := os.Stat(nativeVersionPath(installed.Version(), "haa-release-installer-linux-amd64"))
	if err != nil || !os.SameFile(actual, expected) {
		return ErrCaller
	}
	return nil
}

func verifyHealthChild(pid int, installed InstalledRelease) error {
	if pid <= 0 {
		return ErrHealth
	}
	actual, err := os.Stat("/proc/" + strconv.Itoa(pid) + "/exe")
	if err != nil {
		return ErrHealth
	}
	expected, err := os.Stat(installed.ClientPath())
	if err != nil || !os.SameFile(actual, expected) {
		return ErrHealth
	}
	status, err := readKernelStatus(pid)
	if err != nil {
		return err
	}
	if !ordinaryKernelCredentials(status, installed.ClientUID(), installed.ClientGID()) {
		return ErrHealth
	}
	return nil
}

func ordinaryKernelCredentials(body []byte, uid, gid uint32) bool {
	if uid == 0 {
		return false
	}
	seen := map[string]bool{}
	for _, line := range bytes.Split(body, []byte{'\n'}) {
		key, value, found := bytes.Cut(line, []byte{':'})
		name := string(key)
		if !found || (name != "Uid" && name != "Gid") {
			continue
		}
		if seen[name] {
			return false
		}
		seen[name] = true
		want := uid
		if name == "Gid" {
			want = gid
		}
		fields := bytes.Fields(value)
		if len(fields) != 4 {
			return false
		}
		for _, field := range fields {
			if string(field) != strconv.FormatUint(uint64(want), 10) {
				return false
			}
		}
	}
	return len(seen) == 2
}

// runCompatibilityHealth never trusts child output as installation authority.
// Its caller retains EX and independently rechecks bytes/service/resources.
func runCompatibilityHealth(ctx context.Context, old, current InstalledRelease, peer *controllerPeer, stage string, full bool) (_ *healthWorkspace, resultErr error) {
	if ctx == nil || peer == nil {
		return nil, ErrHealth
	}
	if err := verifyInstallerController(old); err != nil {
		return nil, err
	}
	if peer.uid != current.ClientUID() || peer.gid != current.ClientGID() || current.ClientUID() != old.ClientUID() || current.ClientGID() != old.ClientGID() {
		return nil, ErrCaller
	}
	if err := peer.alive(); err != nil {
		return nil, err
	}
	workspace, err := createHealthWorkspace(stage, current)
	if err != nil {
		return workspace, err
	}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(ErrHealth, resultErr)
		}
	}()
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return workspace, ErrHealth
	}
	parentFile, childFile := os.NewFile(uintptr(fds[0]), "health-parent"), os.NewFile(uintptr(fds[1]), "health-child")
	defer func() {
		for _, file := range []*os.File{parentFile, childFile} {
			if file != nil {
				if err := file.Close(); err != nil {
					resultErr = errors.Join(resultErr, ErrHealth, err)
				}
			}
		}
	}()
	connection, err := unixFileConnection(parentFile)
	if err != nil {
		return workspace, err
	}
	defer func() {
		if err := connection.Close(); err != nil {
			resultErr = errors.Join(resultErr, ErrHealth, err)
		}
	}()
	closeErr := parentFile.Close()
	parentFile = nil
	if closeErr != nil {
		return workspace, errors.Join(ErrHealth, closeErr)
	}
	stop := context.AfterFunc(ctx, func() { _ = connection.SetDeadline(time.Now()) })
	defer stop()
	if err := connection.SetDeadline(time.Now().Add(10 * time.Minute)); err != nil {
		return workspace, ErrHealth
	}
	issuer, _, err := openProtectedFile(nativeVersionPath(old.Version(), "haa-release-installer-linux-amd64"))
	if err != nil {
		return workspace, err
	}
	defer func() {
		if err := issuer.Close(); err != nil {
			resultErr = errors.Join(resultErr, ErrHealth, err)
		}
	}()
	wait, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	child := exec.CommandContext(wait, current.ClientPath(), ProtectedHealthArgument)
	child.Env = []string{"HOME=" + workspace.path, "XDG_CONFIG_HOME=" + filepath.Join(workspace.path, "config"), "XDG_CACHE_HOME=" + filepath.Join(workspace.path, "cache"), "PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
	child.ExtraFiles = []*os.File{childFile, issuer}
	child.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: current.ClientUID(), Gid: current.ClientGID(), Groups: append([]uint32(nil), peer.groups...)}}
	child.Stdout, child.Stderr = io.Discard, io.Discard
	if err := child.Start(); err != nil {
		return workspace, errors.Join(ErrHealth, err)
	}
	waited := false
	defer func() {
		if !waited {
			cancel()
			resultErr = errors.Join(resultErr, child.Wait())
		}
	}()
	closeErr = childFile.Close()
	childFile = nil
	if closeErr != nil {
		return workspace, errors.Join(ErrHealth, closeErr)
	}
	if err := verifyHealthChild(child.Process.Pid, current); err != nil {
		return workspace, err
	}
	mode := "ROLLBACK"
	if full {
		mode = "CANDIDATE"
	}
	message, err := json.Marshal(healthMessage{old.record, current.manifest.SHA256(), mode})
	if err != nil {
		return workspace, ErrHealth
	}
	if err := writeWireBytes(wait, connection, []byte("HAAHEALTH1\n")); err != nil {
		return workspace, err
	}
	if err := writeWireFrame(wait, connection, message); err != nil {
		return workspace, err
	}
	if err := readControl(wait, connection, "HEALTHY1\n"); err != nil {
		return workspace, err
	}
	if err := verifyHealthChild(child.Process.Pid, current); err != nil {
		return workspace, err
	}
	if err := writeWireBytes(wait, connection, []byte("RELEASE1\n")); err != nil {
		return workspace, err
	}
	err = child.Wait()
	waited = true
	if err != nil {
		return workspace, errors.Join(ErrHealth, err, wait.Err())
	}
	if err := peer.alive(); err != nil {
		return workspace, err
	}
	return workspace, nil
}
