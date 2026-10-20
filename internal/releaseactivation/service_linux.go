//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rahoney/heliopause/internal/hosttool"
	"golang.org/x/sys/unix"
)

type serviceTools interface {
	RunOutput(context.Context, io.Writer, string, ...string) error
	RunDiscard(context.Context, string, ...string) error
	Close() error
}

type serviceController struct{ tools serviceTools }

func newServiceController(ctx context.Context) (*serviceController, error) {
	tools, err := hosttool.NewSystemInstallExecutor(ctx)
	if err != nil {
		return nil, err
	}
	return &serviceController{tools}, nil
}

func (c *serviceController) Close() error {
	if c == nil || c.tools == nil {
		return nil
	}
	return c.tools.Close()
}

type machineOutput struct {
	buffer bytes.Buffer
	limit  int
}

func (o *machineOutput) Write(body []byte) (int, error) {
	if len(body) > o.limit-o.buffer.Len() {
		return 0, ErrTransition
	}
	return o.buffer.Write(body)
}

func (c *serviceController) output(ctx context.Context, limit int, tool string, args ...string) ([]byte, error) {
	if c == nil || c.tools == nil || ctx == nil || limit <= 0 {
		return nil, ErrTransition
	}
	wait, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	body := &machineOutput{limit: limit}
	if err := c.tools.RunOutput(wait, body, tool, args...); err != nil {
		return nil, errors.Join(ErrTransition, err, wait.Err())
	}
	if err := wait.Err(); err != nil {
		return nil, errors.Join(ErrTransition, err)
	}
	return body.buffer.Bytes(), nil
}

func (c *serviceController) command(ctx context.Context, args ...string) error {
	if c == nil || c.tools == nil || ctx == nil {
		return ErrTransition
	}
	wait, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := c.tools.RunDiscard(wait, "systemctl", args...); err != nil {
		return errors.Join(ErrTransition, err, wait.Err())
	}
	return wait.Err()
}

const servicePropertyList = "Id,LoadState,FragmentPath,DropInPaths,NeedDaemonReload,User,Group,ActiveState,SubState,MainPID,ControlPID,ControlGroup,Result"

type serviceState struct {
	properties map[string]string
	pid        int
}

func parseServiceState(body []byte) (serviceState, error) {
	state := serviceState{properties: map[string]string{}}
	if len(body) == 0 || len(body) > 16<<10 || bytes.IndexByte(body, 0) >= 0 {
		return state, ErrTransition
	}
	required := map[string]bool{}
	for _, key := range strings.Split(servicePropertyList, ",") {
		required[key] = true
	}
	for _, line := range strings.Split(strings.TrimSuffix(string(body), "\n"), "\n") {
		key, value, found := strings.Cut(line, "=")
		if !found || !required[key] {
			return state, ErrTransition
		}
		if _, duplicate := state.properties[key]; duplicate {
			return state, ErrTransition
		}
		state.properties[key] = value
	}
	if len(state.properties) != len(required) {
		return state, ErrTransition
	}
	for key, want := range map[string]string{"Id": policyUnitName, "LoadState": "loaded", "FragmentPath": policyUnitPath, "DropInPaths": "", "NeedDaemonReload": "no", "User": "root", "Group": "root", "ControlPID": "0"} {
		if state.properties[key] != want {
			return state, ErrTransition
		}
	}
	pid, err := strconv.ParseUint(state.properties["MainPID"], 10, 31)
	if err != nil || strconv.FormatUint(pid, 10) != state.properties["MainPID"] {
		return state, ErrTransition
	}
	state.pid = int(pid)
	return state, nil
}

func (c *serviceController) query(ctx context.Context) (serviceState, error) {
	body, err := c.output(ctx, 16<<10, "systemctl", "show", "--no-pager", "--property="+servicePropertyList, "--", policyUnitName)
	if err != nil {
		return serviceState{}, err
	}
	return parseServiceState(body)
}

type serviceProcess struct {
	pidfd   *os.File
	pid     int
	version string
	gid     uint32
}

func (p *serviceProcess) Close() error {
	if p == nil || p.pidfd == nil {
		return nil
	}
	file := p.pidfd
	p.pidfd = nil
	if err := file.Close(); err != nil {
		return ErrUntrusted
	}
	return nil
}

func (p *serviceProcess) exited() (bool, error) {
	if p == nil || p.pidfd == nil {
		return false, ErrUntrusted
	}
	items := []unix.PollFd{{Fd: int32(p.pidfd.Fd()), Events: unix.POLLIN}}
	if _, err := unix.Poll(items, 0); err != nil {
		return false, ErrUntrusted
	}
	if items[0].Revents & ^int16(unix.POLLIN|unix.POLLHUP) != 0 {
		return false, ErrUntrusted
	}
	return items[0].Revents&(unix.POLLIN|unix.POLLHUP) != 0, nil
}

func pinServiceProcess(pid int, version string, gid uint32) (_ *serviceProcess, resultErr error) {
	if pid <= 0 || !exactVersion(version) {
		return nil, ErrUntrusted
	}
	fd, err := unix.PidfdOpen(pid, 0)
	if err != nil {
		return nil, ErrUntrusted
	}
	process := &serviceProcess{os.NewFile(uintptr(fd), "protected-service"), pid, version, gid}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, process.Close())
		}
	}()
	if exited, err := process.exited(); err != nil || exited {
		return nil, ErrUntrusted
	}
	executable := nativeVersionPath(version, "haa-network-policy-helper-linux-amd64")
	file, expected, err := openProtectedFile(executable)
	if err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, ErrUntrusted
	}
	actual, err := os.Stat("/proc/" + strconv.Itoa(pid) + "/exe")
	if err != nil {
		return nil, ErrUntrusted
	}
	if !os.SameFile(expected, actual) {
		return nil, ErrUntrusted
	}
	status, err := readKernelStatus(pid)
	if err != nil {
		return nil, err
	}
	if !rootKernelCredentials(status) {
		return nil, ErrUntrusted
	}
	if exited, err := process.exited(); err != nil || exited {
		return nil, ErrUntrusted
	}
	return process, nil
}

func readKernelStatus(pid int) ([]byte, error) {
	file, err := os.Open("/proc/" + strconv.Itoa(pid) + "/status")
	if err != nil {
		return nil, ErrTransition
	}
	body, readErr := io.ReadAll(io.LimitReader(file, 64<<10+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || len(body) == 0 || len(body) > 64<<10 {
		return nil, ErrTransition
	}
	return body, nil
}

func rootKernelCredentials(body []byte) bool {
	seen := map[string]bool{}
	for _, line := range strings.Split(string(body), "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found || (key != "Uid" && key != "Gid") {
			continue
		}
		if seen[key] {
			return false
		}
		seen[key] = true
		fields := strings.Fields(value)
		if len(fields) != 4 {
			return false
		}
		for _, field := range fields {
			if field != "0" {
				return false
			}
		}
	}
	return len(seen) == 2
}

var errServiceNotReady = errors.New("SELF_UPDATE_SERVICE_NOT_READY")

func (c *serviceController) active(ctx context.Context, version string, gid uint32) (_ *serviceProcess, resultErr error) {
	unit, err := nativePolicyUnit(version)
	if err != nil {
		return nil, err
	}
	actual, err := readProtected(ctx, policyUnitPath, 16<<10)
	if err != nil || !bytes.Equal(actual, unit) {
		return nil, errors.Join(ErrUntrusted, err)
	}
	state, err := c.query(ctx)
	if err != nil {
		return nil, err
	}
	if state.pid <= 0 || state.properties["ActiveState"] != "active" || state.properties["SubState"] != "running" || state.properties["Result"] != "success" || state.properties["ControlGroup"] != policyControlGroup {
		return nil, ErrTransition
	}
	process, err := pinServiceProcess(state.pid, version, gid)
	if err != nil {
		return nil, err
	}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, process.Close())
		}
	}()
	for _, path := range []string{policySocketPath, resourceSocketPath} {
		if err := verifyServiceSocket(ctx, path, gid, state.pid); err != nil {
			return nil, err
		}
	}
	if err := serviceCgroupMembers(state.pid); err != nil {
		return nil, err
	}
	confirmed, err := c.query(ctx)
	if err != nil || confirmed.pid != state.pid || confirmed.properties["ActiveState"] != "active" || confirmed.properties["SubState"] != "running" || confirmed.properties["ControlGroup"] != policyControlGroup {
		return nil, errors.Join(ErrTransition, err)
	}
	if exited, err := process.exited(); err != nil || exited {
		return nil, ErrTransition
	}
	return process, nil
}

func verifyServiceSocket(ctx context.Context, path string, gid uint32, pid int) (resultErr error) {
	if ctx == nil {
		return ErrUntrusted
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := protectedDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	identity, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return errServiceNotReady
	}
	if err != nil || identity.Mode()&os.ModeSocket == 0 || identity.Mode().Perm() != 0660 {
		return ErrUntrusted
	}
	owner, ok := identity.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != 0 || owner.Gid != gid || owner.Nlink != 1 {
		return ErrUntrusted
	}
	dialer := net.Dialer{Timeout: time.Second}
	connection, err := dialer.DialContext(ctx, "unix", path)
	if err != nil {
		return errors.Join(ErrTransition, err, ctx.Err())
	}
	defer func() {
		if err := connection.Close(); err != nil {
			resultErr = errors.Join(resultErr, ErrUntrusted, err)
		}
	}()
	local, ok := connection.(*net.UnixConn)
	if !ok {
		return ErrUntrusted
	}
	raw, err := local.SyscallConn()
	if err != nil {
		return ErrUntrusted
	}
	var credentials *unix.Ucred
	var credentialErr error
	if err := raw.Control(func(fd uintptr) {
		credentials, credentialErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil || credentialErr != nil || credentials == nil || credentials.Uid != 0 || credentials.Gid != 0 || int(credentials.Pid) != pid {
		return ErrUntrusted
	}
	current, err := os.Lstat(path)
	if err != nil || !os.SameFile(identity, current) {
		return ErrUntrusted
	}
	return nil
}

func serviceCgroupMembers(pid int) error {
	path := "/sys/fs/cgroup" + policyControlGroup
	var filesystem unix.Statfs_t
	if unix.Statfs("/sys/fs/cgroup", &filesystem) != nil || filesystem.Type != unix.CGROUP2_SUPER_MAGIC {
		return ErrTransition
	}
	body, err := os.ReadFile(filepath.Join(path, "cgroup.procs"))
	if errors.Is(err, os.ErrNotExist) && pid == 0 {
		if _, queryErr := os.Lstat(path); errors.Is(queryErr, os.ErrNotExist) {
			return nil
		}
		return ErrTransition
	}
	if err != nil || len(body) > 64<<10 {
		return ErrTransition
	}
	fields := strings.Fields(string(body))
	if pid == 0 && len(fields) != 0 {
		return ErrBusy
	}
	if pid != 0 && (len(fields) != 1 || fields[0] != strconv.Itoa(pid)) {
		return ErrBusy
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return ErrTransition
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return ErrBusy
		}
	}
	return nil
}

func (c *serviceController) stopped(ctx context.Context, process *serviceProcess) error {
	state, err := c.query(ctx)
	if err != nil {
		return err
	}
	if state.pid != 0 || (state.properties["ActiveState"] != "inactive" && state.properties["ActiveState"] != "failed") || state.properties["ControlGroup"] != "" && state.properties["ControlGroup"] != policyControlGroup {
		return ErrTransition
	}
	if state.properties["Result"] != "success" || state.properties["ActiveState"] == "inactive" && state.properties["SubState"] != "dead" || state.properties["ActiveState"] == "failed" && state.properties["SubState"] != "failed" {
		return ErrTransition
	}
	if process != nil {
		exited, err := process.exited()
		if err != nil || !exited {
			return ErrTransition
		}
	}
	if err := serviceCgroupMembers(0); err != nil {
		return err
	}
	for _, path := range []string{policySocketPath, resourceSocketPath} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			return ErrTransition
		}
	}
	return nil
}

func (c *serviceController) stop(ctx context.Context, process *serviceProcess) error {
	if process == nil {
		return ErrUntrusted
	}
	unit, err := nativePolicyUnit(process.version)
	if err != nil {
		return err
	}
	actual, err := readProtected(ctx, policyUnitPath, 16<<10)
	if err != nil || !bytes.Equal(unit, actual) {
		return errors.Join(ErrUntrusted, err)
	}
	state, err := c.query(ctx)
	if err != nil || state.pid != process.pid {
		return errors.Join(ErrTransition, err)
	}
	if exited, err := process.exited(); err != nil || exited {
		return ErrTransition
	}
	commandErr := c.command(ctx, "stop", "--", policyUnitName)
	return errors.Join(commandErr, c.stopped(ctx, process))
}

func (c *serviceController) start(ctx context.Context, version string, gid uint32) (*serviceProcess, error) {
	unit, err := nativePolicyUnit(version)
	if err != nil {
		return nil, err
	}
	actual, err := readProtected(ctx, policyUnitPath, 16<<10)
	if err != nil || !bytes.Equal(unit, actual) {
		return nil, errors.Join(ErrUntrusted, err)
	}
	if err := c.command(ctx, "daemon-reload"); err != nil {
		return nil, err
	}
	if err := c.stopped(ctx, nil); err != nil {
		return nil, err
	}
	if err := c.command(ctx, "start", "--", policyUnitName); err != nil {
		return nil, err
	}
	wait, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	for {
		process, err := c.active(wait, version, gid)
		if !errors.Is(err, errServiceNotReady) {
			return process, err
		}
		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-wait.Done():
			timer.Stop()
			return nil, errors.Join(ErrTransition, wait.Err())
		case <-timer.C:
		}
	}
}
