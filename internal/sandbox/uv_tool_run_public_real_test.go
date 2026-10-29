//go:build linux && realcorpus

package sandbox

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
)

// The shared pipx-named helpers below hold only authenticated wheel DATA.
// No pipx ZIP, interpreter query, receipt or runtime identity is introduced.
type uvPublicRawMirror struct {
	session pipxNativeSession
	volume  *pipxWheelVolume
}

func (m *uvPublicRawMirror) prepare(ctx context.Context, input *pipxNativeInput) error {
	id, err := domain.NewSandboxSessionID()
	if err != nil {
		return err
	}
	m.session.transaction = id.String()
	m.volume, err = createPipxWheelVolume(ctx, m.session.runner, id.String(), input)
	if err != nil {
		return err
	}
	rw, err := m.volume.preparationMount()
	if err != nil {
		return err
	}
	writer, err := m.session.dataKeeper(ctx, "0:0", "--mount", rw)
	if err != nil {
		return err
	}
	if err := m.volume.seed(ctx, m.session.runner, writer.id); err != nil {
		return err
	}
	ro, err := m.volume.readerMount()
	if err != nil {
		return err
	}
	anchor, err := m.session.dataKeeper(ctx, "1000:1000", "--mount", ro)
	if err != nil {
		return err
	}
	if err := m.volume.bindAnchor(ctx, m.session.runner, anchor.id); err != nil {
		return err
	}
	if err := m.session.removeKeeper(ctx, writer); err != nil {
		return err
	}
	return m.volume.seal(ctx, m.session.runner)
}

func (m *uvPublicRawMirror) close() error {
	ctx, cancel := resolverCleanupContext()
	defer cancel()
	var result error
	for i := len(m.session.keepers) - 1; i >= 0; i-- {
		result = errors.Join(result, m.session.removeKeeper(ctx, m.session.keepers[i]))
	}
	if m.volume != nil {
		result = errors.Join(result, m.volume.close(ctx, m.session.runner))
	}
	return result
}

func publicUVToolRunInput(t *testing.T) *pipxNativeInput {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	resolver, err := NewLinuxUVResolverWithExecutorAndPolicy(executor, supervisor.Observer(), integrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Error(err)
		}
	}()
	requirement, err := artifactpypi.ParseUpgradeRequirement("cowsay==6.1")
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := resolver.resolveNativeSelection(ctx, requirement.Reference(), requirement.Requirement())
	if err != nil {
		t.Fatal("public UV proposal failed", err)
	}
	lock, err := artifactpypi.ParseUVLock(requirement.Reference(), proposal.lock, PinnedPythonRuntime().PythonVersion)
	if err != nil {
		t.Fatal(err)
	}
	runtime := PinnedPythonRuntime()
	target := artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}
	selected, err := lock.SelectInstalledWheelControls(proposal.controls, target)
	if err != nil {
		t.Fatal(err)
	}
	var frozen *pipxNativeInput
	if err := resolver.reader.withUVArchiveInputs(ctx, selected, func(inputs []artifactpypi.UVWheelInput, pages []artifactpypi.SimpleProject) error {
		var err error
		frozen, err = freezePipxNativeInput(ctx, lock, inputs, pages)
		return err
	}); err != nil {
		t.Fatal("whole public original/source/startup binding failed", err)
	}
	if frozen == nil || !frozen.valid() || len(frozen.graph.Nodes()) != 1 || len(frozen.files) != 1 ||
		frozen.files[0].digest != "274b1e6fc1b966d53976333eb90ac94cb07a450a700b455af9fbdf882244b30a" {
		t.Fatal("public frozen source changed")
	}
	return frozen
}

func TestUVActualPublicToolRunSourceBinding(t *testing.T) {
	if os.Getenv("HELOX_UV_TOOL_RUN_PUBLIC_BINDING_INTEGRATION") != "1" ||
		os.Getenv("HELOX_UV_NATIVE_RUNTIME") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user public UV query binding qualification")
	}
	// Resolve/source-read sessions are completely closed before the native
	// query fixture starts. Owned frozen bytes survive their reader cleanup.
	frozen := publicUVToolRunInput(t)
	withActualUVNativeRuntimeInput(t, frozen, func(ctx context.Context, runner CommandRunner, id string) {
		before, err := frozen.manifestID()
		if err != nil || before != frozen.id {
			t.Fatal("public source changed before native query", err)
		}
		args := []string{
			uvToolGuestPath, "tool", "run", "--isolated", "--from", "cowsay==6.1",
			"--offline", "--no-index", "--find-links", pipxNativeWheels,
			"--no-build", "--no-config", "--no-cache", "--no-python-downloads",
			"--no-managed-python", "--python", "/usr/local/bin/python",
			"--link-mode", "copy", "--no-sources", "/dev/null",
		}
		output, runErr := runner.Output(ctx, "docker", boundaryExecArguments(id, boundaryLaunchMode, args...)...)
		var exited *exec.ExitError
		if !errors.As(runErr, &exited) || exited.ExitCode() == 0 || len(output) != 0 ||
			!strings.Contains(string(exited.Stderr), "/dev/null") ||
			!strings.Contains(string(exited.Stderr), "Permission denied") {
			t.Fatal("public native preparation did not reach fixed non-executable terminal", runErr)
		}
		after, err := frozen.manifestID()
		if err != nil || after != before {
			t.Fatal("whole original changed during public native query", err)
		}
		t.Logf("uv_public_query_binding phase=prepared source=ACTUAL_PUBLIC_INDEPENDENT whole_original_startup=BOUND nodes=%d wheels=%d readonly_source=SEALED selected_program=NOT_RUN Evidence=NOT_RUN Policy=NOT_RUN native_terminal=EACCES wheel_sha256=%s",
			len(frozen.graph.Nodes()), len(frozen.files), frozen.files[0].digest)
	})
}
