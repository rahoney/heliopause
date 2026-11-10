//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestProtectedServiceFactsInDisposableNamespace(t *testing.T) {
	if os.Getenv("HELOX_SERVICE_NAMESPACE") != "1" {
		t.Skip("explicit disposable root service qualification")
	}
	if os.Getenv("HELOX_SERVICE_CHILD") == "1" {
		for _, path := range []string{policySocketPath, resourceSocketPath} {
			listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			if err := os.Chmod(path, 0660); err != nil {
				t.Fatal(err)
			}
			if err := os.Chown(path, 0, 1000); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := io.Copy(io.Discard, os.Stdin); err != nil {
			t.Fatal(err)
		}
		return
	}
	if os.Geteuid() != 0 {
		t.Fatal("root namespace required")
	}
	for _, kind := range []string{"mnt", "net"} {
		self, err := os.Readlink("/proc/self/ns/" + kind)
		if err != nil {
			t.Fatal(err)
		}
		initial, err := os.Readlink("/proc/1/ns/" + kind)
		if err != nil || self == initial {
			t.Fatal("separate namespaces required")
		}
	}
	if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/var/lib", "/etc", "/usr/lib", "/run"} {
		if err := syscall.Mount("tmpfs", path, "tmpfs", syscall.MS_NODEV|syscall.MS_NOSUID, "mode=0755,size=134217728"); err != nil {
			t.Fatal(err)
		}
	}
	journal := runtimeConfigFixture(t)
	for _, path := range []string{stateRoot, "/etc/heliopause", "/etc/systemd/system", filepath.Dir(policySocketPath), filepath.Dir(nativeVersionPath(journal.Old.Version, "helox-linux-amd64"))} {
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
		for current := path; current != filepath.Dir(current); current = filepath.Dir(current) {
			if err := os.Chmod(current, 0755); err != nil {
				t.Fatal(err)
			}
		}
	}
	for index, path := range configPaths() {
		if err := os.WriteFile(path, journal.Configs[index].Body, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0644); err != nil {
			t.Fatal(err)
		}
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	helper := nativeVersionPath(journal.Old.Version, "haa-network-policy-helper-linux-amd64")
	if err := os.WriteFile(helper, binary, 0555); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(helper, 0555); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, helper, "-test.run=^TestProtectedServiceFactsInDisposableNamespace$", "-test.v")
	child.Env = []string{"HELOX_SERVICE_NAMESPACE=1", "HELOX_SERVICE_CHILD=1"}
	var output bytes.Buffer
	child.Stdout = &output
	child.Stderr = &output
	input, err := child.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		if !waited {
			_ = input.Close()
			cancel()
			_ = child.Wait()
		}
	}()
	for {
		ready := true
		for _, path := range []string{policySocketPath, resourceSocketPath} {
			identity, err := os.Lstat(path)
			if err != nil {
				ready = false
				continue
			}
			owner, ok := identity.Sys().(*syscall.Stat_t)
			if !ok || owner.Gid != 1000 || identity.Mode().Perm() != 0660 {
				ready = false
			}
		}
		if ready {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("root test listener did not start")
		case <-time.After(5 * time.Millisecond):
		}
	}
	process, err := pinServiceProcess(child.Process.Pid, journal.Old.Version, 1000)
	if err != nil {
		t.Fatal(err)
	}
	defer process.Close()
	if exited, err := process.exited(); err != nil || exited {
		t.Fatal("live root process rejected", err)
	}
	for _, path := range []string{policySocketPath, resourceSocketPath} {
		if err := verifyServiceSocket(ctx, path, 1000, child.Process.Pid); err != nil {
			t.Fatal("actual kernel socket peer rejected", err)
		}
		if err := verifyServiceSocket(ctx, path, 1001, child.Process.Pid); err == nil {
			t.Fatal("foreign socket group admitted")
		}
		if err := verifyServiceSocket(ctx, path, 1000, os.Getpid()); err == nil {
			t.Fatal("foreign socket PID admitted")
		}
	}
	failure := errors.New("injected fixed stop failure")
	tools := &serviceToolFixture{body: strings.ReplaceAll(serviceStateFixture(), "MainPID=123", "MainPID="+strconv.Itoa(child.Process.Pid)), discardErr: failure}
	controller := &serviceController{tools}
	if err := controller.stop(ctx, process); !errors.Is(err, failure) {
		t.Fatal("stop failure or actual live process was hidden", err)
	}
	if !strings.Contains(strings.Join(tools.calls, "\n"), "systemctl stop -- "+policyUnitName) {
		t.Fatal("stop did not select fixed unit")
	}
	if _, err := controller.start(ctx, journal.Old.Version, 1000); !errors.Is(err, failure) {
		t.Fatal("reload failure was hidden", err)
	}
	if err := input.Close(); err != nil {
		t.Fatal(err)
	}
	err = child.Wait()
	waited = true
	if err != nil {
		t.Fatal("root child wait failed", err, output.String())
	}
	if exited, err := process.exited(); err != nil || !exited {
		t.Fatal("actual reaped root process not confirmed exited", err)
	}
	if err := observerEndpointsAbsent(1000); err != nil {
		t.Fatal(err)
	}
	// The namespace shares the host's proc view. Do not call a global absence
	// assertion against unrelated host services as a synthetic positive fixture.
	derived, err := derivedRuntimeConfigs(journal)
	if err != nil {
		t.Fatal(err)
	}
	if err := replaceProtected(ctx, policyConfigPath, journal.Configs[0].Body, derived[0], 0644, 0); err != nil {
		t.Fatal(err)
	}
	if err := switchRuntimeConfigs(ctx, journal, true); err != nil {
		t.Fatal("partial switch exact restore failed", err)
	}
	if err := switchRuntimeConfigs(ctx, journal, false); err != nil {
		t.Fatal("full fixed switch failed", err)
	}
	if err := verifyRuntimeConfigs(ctx, journal, false); err != nil {
		t.Fatal(err)
	}
	unknown := []byte("root administrator changed this config")
	if err := os.WriteFile(hostConfigPath, unknown, 0644); err != nil {
		t.Fatal(err)
	}
	if err := switchRuntimeConfigs(ctx, journal, true); err == nil {
		t.Fatal("unknown config was restored by guessing")
	}
	actual, err := os.ReadFile(hostConfigPath)
	if err != nil || !bytes.Equal(actual, unknown) {
		t.Fatal("unknown config was overwritten")
	}
	t.Log("PASS: actual root process/pidfd/socket PID and GID, fixed command fault causes, exact config partial restore and unknown retention; no real systemd/crypto activation")
}
