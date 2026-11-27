//go:build linux

package hosttool

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// The systemd slice is pinned independently of Docker children. Its fixed,
// artifact-free keeper prevents the parent from disappearing between probes;
// the runtime CPUAccounting property keeps cpu.stat available through keeper
// termination so the last teardown is included in the final sample.
type observationKeeper interface {
	Start(context.Context, string) (string, string, uint64, error)
	Verify(context.Context, string, string) error
	Close(context.Context, string, string) (uint64, error)
}

type systemdObservationKeeper struct{ executor *Executor }

func observationKeeperService(parent string) string {
	return strings.TrimSuffix(parent, ".slice") + "keeper.service"
}

func (k systemdObservationKeeper) Start(ctx context.Context, parent string) (string, string, uint64, error) {
	if k.executor == nil || !validObservationParent(parent) || ctx == nil {
		return "", "", 0, errors.New("observation systemd keeper unavailable")
	}
	service := observationKeeperService(parent)
	if _, err := verifySystemExecutable("/usr/bin/sleep"); err != nil {
		return "", "", 0, errors.New("observation keeper executable is untrusted")
	}
	if err := k.executor.RunDiscard(ctx, "systemd-run", "--unit="+service, "--slice="+parent,
		"--property=Type=exec", "--property=CPUAccounting=yes", "/usr/bin/sleep", "infinity"); err != nil {
		return "", "", 0, errors.New("observation transaction slice creation failed")
	}
	cleanup := true
	defer func() {
		if cleanup {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), observationKeeperCleanupTimeout)
			defer cancel()
			_ = k.executor.RunDiscard(cleanupCtx, "systemctl", "stop", service, parent)
			_ = k.executor.RunDiscard(cleanupCtx, "systemctl", "revert", parent)
		}
	}()
	if err := k.executor.RunDiscard(ctx, "systemctl", "set-property", "--runtime", parent, "CPUAccounting=yes"); err != nil {
		return "", "", 0, errors.New("observation slice CPU accounting unavailable")
	}
	path, scope := observationKeeperPaths(parent)
	if err := k.Verify(ctx, parent, scope); err != nil {
		return "", "", 0, err
	}
	usage, err := readObservationCPU(path)
	if err != nil {
		return "", "", 0, err
	}
	cleanup = false
	return path, scope, usage, nil
}

const observationKeeperCleanupTimeout = 7 * time.Second

func validObservationParent(parent string) bool {
	if !strings.HasPrefix(parent, "haaobs") || !strings.HasSuffix(parent, ".slice") || len(parent) != len("haaobs")+24+len(".slice") {
		return false
	}
	for _, character := range strings.TrimSuffix(strings.TrimPrefix(parent, "haaobs"), ".slice") {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return false
		}
	}
	return true
}

func observationKeeperPaths(parent string) (string, string) {
	path := filepath.Join("/sys/fs/cgroup", parent)
	return path, filepath.Join(path, observationKeeperService(parent))
}

func (k systemdObservationKeeper) Verify(ctx context.Context, parent, scope string) error {
	if k.executor == nil || ctx == nil || !validObservationParent(parent) {
		return errors.New("observation keeper identity unavailable")
	}
	path, expectedScope := observationKeeperPaths(parent)
	if scope != expectedScope {
		return errors.New("observation keeper scope substituted")
	}
	parentBody, err := k.executor.Output(ctx, "systemctl", "show", parent,
		"-p", "ActiveState", "-p", "ControlGroup", "-p", "CPUAccounting")
	if err != nil || len(parentBody) > 1024 ||
		!exactSystemdProperty(parentBody, "ActiveState", "active") ||
		!exactSystemdProperty(parentBody, "ControlGroup", "/"+parent) ||
		!exactSystemdProperty(parentBody, "CPUAccounting", "yes") {
		return errors.New("observation transaction slice identity changed")
	}
	service := observationKeeperService(parent)
	body, err := k.executor.Output(ctx, "systemctl", "show", service,
		"-p", "ActiveState", "-p", "ControlGroup", "-p", "Slice", "-p", "MainPID")
	if err != nil || len(body) > 1024 ||
		!exactSystemdProperty(body, "ActiveState", "active") ||
		!exactSystemdProperty(body, "ControlGroup", "/"+parent+"/"+service) ||
		!exactSystemdProperty(body, "Slice", parent) {
		return errors.New("observation keeper systemd identity changed")
	}
	pidText, ok := systemdProperty(body, "MainPID")
	pid, parseErr := strconv.Atoi(pidText)
	if !ok || parseErr != nil || pid <= 1 {
		return errors.New("observation keeper process identity unavailable")
	}
	pidPath := filepath.Join("/proc", strconv.Itoa(pid))
	executable, err := os.Readlink(filepath.Join(pidPath, "exe"))
	if err != nil || executable != "/usr/bin/sleep" {
		return errors.New("observation keeper executable changed")
	}
	cgroup, err := os.ReadFile(filepath.Join(pidPath, "cgroup"))
	if err != nil || len(cgroup) > 1024 || strings.TrimSpace(string(cgroup)) != "0::/"+parent+"/"+service {
		return errors.New("observation keeper cgroup membership changed")
	}
	if info, err := os.Lstat(path); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("observation parent cgroup unavailable")
	}
	return nil
}

func (k systemdObservationKeeper) Close(ctx context.Context, parent, scope string) (uint64, error) {
	if err := k.Verify(ctx, parent, scope); err != nil {
		return 0, err
	}
	path, _ := observationKeeperPaths(parent)
	service := observationKeeperService(parent)
	if err := k.executor.RunDiscard(ctx, "systemctl", "stop", service); err != nil {
		return 0, errors.New("observation keeper termination failed")
	}
	body, err := k.executor.Output(ctx, "systemctl", "show", service, "-p", "ActiveState", "-p", "MainPID")
	if err != nil || len(body) > 1024 || !exactSystemdProperty(body, "ActiveState", "inactive") || !exactSystemdProperty(body, "MainPID", "0") {
		return 0, errors.New("observation keeper drainage unproven")
	}
	if err := verifyDrainedScope(scope); err != nil {
		return 0, err
	}
	usage, err := readObservationCPU(path)
	if err != nil {
		return 0, err
	}
	if err := k.executor.RunDiscard(ctx, "systemctl", "stop", parent); err != nil {
		return 0, errors.New("observation transaction slice removal failed")
	}
	if err := k.executor.RunDiscard(ctx, "systemctl", "revert", parent); err != nil {
		return 0, errors.New("observation transaction slice property removal failed")
	}
	for {
		if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
			return usage, nil
		} else if err != nil {
			return 0, errors.New("observation transaction cgroup removal unproven")
		}
		select {
		case <-ctx.Done():
			return 0, errors.New("observation transaction cgroup removal timed out")
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// recover resumes after a keeper stop whose slice cleanup did not finish.
func (k systemdObservationKeeper) recover(ctx context.Context, parent string) error {
	if !validObservationParent(parent) {
		return errD3Unconfirmed
	}
	path, scope := observationKeeperPaths(parent)
	if k.Verify(ctx, parent, scope) == nil {
		_, err := k.Close(ctx, parent, scope)
		return err
	}
	service := observationKeeperService(parent)
	keeper, err := k.executor.Output(ctx, "systemctl", "show", service, "-p", "LoadState", "-p", "ActiveState", "-p", "ControlGroup", "-p", "MainPID", "-p", "Job")
	if err != nil || len(keeper) > 1024 || !exactSystemdProperty(keeper, "ActiveState", "inactive") ||
		!exactSystemdProperty(keeper, "MainPID", "0") && !exactSystemdProperty(keeper, "LoadState", "not-found") ||
		!exactSystemdProperty(keeper, "ControlGroup", "") || !exactSystemdProperty(keeper, "Job", "") {
		return errD3Unconfirmed
	}
	slice, err := k.executor.Output(ctx, "systemctl", "show", parent, "-p", "Id", "-p", "LoadState", "-p", "ActiveState", "-p", "SubState", "-p", "ControlGroup", "-p", "CPUAccounting", "-p", "Job", "-p", "FragmentPath", "-p", "DropInPaths", "-p", "Transient", "-p", "NeedDaemonReload")
	if err != nil || len(slice) > 1024 || !exactSystemdProperty(slice, "Id", parent) || !exactSystemdProperty(slice, "Job", "") || verifyDrainedScope(path) != nil {
		return errD3Unconfirmed
	}
	if _, statErr := os.Lstat(path); errors.Is(statErr, os.ErrNotExist) && absentObservationSlice(slice) {
		return nil
	}
	if !exactSystemdProperty(slice, "CPUAccounting", "yes") ||
		!exactSystemdProperty(slice, "ControlGroup", "/"+parent) {
		return errD3Unconfirmed
	}
	if err := k.executor.RunDiscard(ctx, "systemctl", "stop", parent); err != nil {
		return err
	}
	if err := k.executor.RunDiscard(ctx, "systemctl", "revert", parent); err != nil {
		return err
	}
	_, err = os.Lstat(path)
	if !errors.Is(err, os.ErrNotExist) {
		return errD3Unconfirmed
	}
	return nil
}

func absentObservationSlice(body []byte) bool {
	if !exactSystemdProperty(body, "ActiveState", "inactive") || !exactSystemdProperty(body, "ControlGroup", "") {
		return false
	}
	if exactSystemdProperty(body, "LoadState", "not-found") {
		return true
	}
	// systemd can load an implicit slice definition on lookup after its
	// cgroup and runtime properties are gone.
	return exactSystemdProperty(body, "LoadState", "loaded") &&
		exactSystemdProperty(body, "SubState", "dead") &&
		exactSystemdProperty(body, "FragmentPath", "") &&
		exactSystemdProperty(body, "DropInPaths", "") &&
		exactSystemdProperty(body, "Transient", "no") &&
		exactSystemdProperty(body, "NeedDaemonReload", "no")
}

func systemdProperty(body []byte, key string) (string, bool) {
	value, found := "", false
	for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
		name, candidate, ok := strings.Cut(line, "=")
		if ok && name == key {
			if found {
				return "", false
			}
			value, found = candidate, true
		}
	}
	return value, found
}

func exactSystemdProperty(body []byte, key, value string) bool {
	got, ok := systemdProperty(body, key)
	return ok && got == value
}
