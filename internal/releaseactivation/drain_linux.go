//go:build linux

package releaseactivation

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/rahoney/heliopause/internal/releaseinstall"
)

func (c *serviceController) drained(ctx context.Context, peer *controllerPeer, service *serviceProcess) error {
	if ctx == nil || os.Geteuid() != 0 || peer == nil || service == nil {
		return ErrUntrusted
	}
	if err := peer.alive(); err != nil {
		return err
	}
	if err := c.resourcesDrained(ctx, peer.uid, int(peer.pid), service.pid); err != nil {
		return err
	}
	if exited, err := service.exited(); err != nil || exited {
		return ErrTransition
	}
	return peer.alive()
}

func (c *serviceController) resourcesDrained(ctx context.Context, uid uint32, controllerPID, servicePID int) error {
	if ctx == nil || os.Geteuid() != 0 || uid == 0 {
		return ErrUntrusted
	}
	containers, err := c.output(ctx, 1<<20, "docker", "ps", "--all", "--format", "{{.Names}}")
	if err != nil {
		return err
	}
	if err := noHAAContainers(containers); err != nil {
		return err
	}
	slices, err := c.output(ctx, 1<<20, "systemctl", "list-units", "--all", "--plain", "--no-legend", "--no-pager", "--type=slice", "--state=active,activating,deactivating,failed", "haaobs*.slice")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(slices)) != "" {
		return ErrBusy
	}
	if err := observerEndpointsAbsent(uid); err != nil {
		return err
	}
	if err := protectedConsumersAbsent(ctx, controllerPID, servicePID); err != nil {
		return err
	}
	return nil
}

func noHAAContainers(body []byte) error {
	if len(body) > 1<<20 || strings.ContainsAny(string(body), "\x00\r") {
		return ErrTransition
	}
	for _, line := range strings.Split(strings.TrimSuffix(string(body), "\n"), "\n") {
		if strings.HasPrefix(line, "heliopause-") {
			return ErrBusy
		}
		if strings.TrimSpace(line) != line || strings.ContainsAny(line, " \t") {
			return ErrTransition
		}
	}
	return nil
}

func observerEndpointsAbsent(uid uint32) error {
	const directory = "/run/heliopause-observer"
	if err := protectedDirectory("/run"); err != nil {
		return err
	}
	identity, err := os.Lstat(directory)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || !identity.IsDir() || identity.Mode()&os.ModeSymlink != 0 || identity.Mode().Perm() != 0700 {
		return ErrTransition
	}
	owner, ok := identity.Sys().(*syscall.Stat_t)
	if !ok || (owner.Uid != uid && owner.Uid != 0) {
		return ErrTransition
	}
	for _, name := range []string{"gvisor-remote.sock", "haa-output.sock", "haa-control.sock", ".heliopause-supervisor.lock"} {
		if _, err := os.Lstat(filepath.Join(directory, name)); !errors.Is(err, os.ErrNotExist) {
			return ErrBusy
		}
	}
	return nil
}

func protectedConsumersAbsent(ctx context.Context, controllerPID, servicePID int) (resultErr error) {
	if ctx == nil || controllerPID < 0 || servicePID < 0 {
		return ErrUntrusted
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	proc, err := os.Open("/proc")
	if err != nil {
		return ErrTransition
	}
	defer func() {
		if err := proc.Close(); err != nil {
			resultErr = errors.Join(resultErr, ErrTransition)
		}
	}()
	names, err := proc.Readdirnames(65537)
	if err != nil && !errors.Is(err, io.EOF) || len(names) > 65536 {
		return ErrTransition
	}
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return err
		}
		pid, err := strconv.Atoi(name)
		if err != nil || pid <= 0 {
			continue
		}
		if pid == os.Getpid() || pid == controllerPID || pid == servicePID {
			continue
		}
		path, err := os.Readlink("/proc/" + name + "/exe")
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return ErrTransition
			}
			if _, queryErr := os.Lstat("/proc/" + name); errors.Is(queryErr, os.ErrNotExist) {
				continue
			}
			status, queryErr := readKernelStatus(pid)
			if queryErr != nil {
				return queryErr
			}
			if kernelProcessWithoutExecutable(status) {
				continue
			}
			return ErrTransition
		}
		if protectedConsumerPath(path) {
			return ErrBusy
		}
	}
	return nil
}

func protectedConsumerPath(path string) bool {
	return strings.HasPrefix(path, releaseinstall.SystemInstallRoot+"/versions/") || strings.HasPrefix(path, "/usr/libexec/heliopause/")
}

func kernelProcessWithoutExecutable(body []byte) bool {
	// A live userspace executable may disappear during exec; that race remains
	// UNKNOWN. Only explicit kernel-thread/zombie/dead facts permit no exe.
	state, kernel := "", ""
	seen := map[string]bool{}
	for _, line := range strings.Split(string(body), "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found || (key != "State" && key != "Kthread") {
			continue
		}
		if seen[key] {
			return false
		}
		seen[key] = true
		if key == "State" {
			state = strings.TrimSpace(value)
		} else {
			kernel = strings.TrimSpace(value)
		}
	}
	return kernel == "1" || strings.HasPrefix(state, "Z (") || strings.HasPrefix(state, "X (")
}
