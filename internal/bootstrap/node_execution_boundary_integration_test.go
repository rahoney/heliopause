//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// This delegates to the same registered protected Host executor. The timer is
// only a test cancellation trigger, never guest completion authority.
type nodeBoundaryCapture struct {
	*hosttool.Executor
	cancel context.CancelCauseFunc
	cause  error
}

func (r *nodeBoundaryCapture) CaptureExecutionOutput(ctx context.Context, name string, args ...string) (domain.CapturedProcessOutput, error) {
	if r.cancel != nil {
		timer := time.AfterFunc(time.Second, func() { r.cancel(r.cause) })
		defer timer.Stop()
	}
	return r.Executor.CaptureExecutionOutput(ctx, name, args...)
}

func TestLinuxNodeExecutionBoundaryIntegration(t *testing.T) {
	if os.Getenv("HELOX_NODE_EXECUTION_BOUNDARY_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user controlled Node boundary qualification")
	}
	ctx, stop := context.WithTimeout(context.Background(), 4*time.Minute)
	defer stop()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	var previous string
	for _, mode := range []string{"argv", "term", "kill", "cancel", "stdout-overflow", "stderr-overflow", "outside-read", "network", "child", "comm-spoof", "readonly", "deadline"} {
		if !t.Run(mode, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			run, cancel := context.WithCancelCause(ctx)
			defer cancel(nil)
			if mode == "deadline" {
				var finish context.CancelFunc
				run, finish = context.WithTimeout(ctx, 30*time.Second)
				defer finish()
			}
			cause := errors.New("controlled Node caller cancellation")
			runner := &nodeBoundaryCapture{Executor: executor}
			if mode == "cancel" {
				runner.cancel, runner.cause = cancel, cause
			}
			launcher, err := hosttool.NewSystemObserverLauncher()
			if err != nil {
				t.Fatal(err)
			}
			observer, err := sandbox.NewObserverSupervisor(run, func(start context.Context, remote, output string) (sandbox.ObserverProcess, error) {
				return launcher.StartObserver(start, remote, output)
			})
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := observer.Close(); err != nil {
					t.Error(err)
				}
			}()
			backend, err := sandbox.NewLinuxBackendWithExecutor(root+"/intake", runner, observer.Observer())
			if err != nil {
				t.Fatal(err)
			}
			source, resolution := nodeBoundarySource(t, run, root+"/intake")
			arguments := []string{mode}
			literals := []string{"", "a b", "a\nb", "; exit 0", "$(touch forbidden)", "--", "--handoff-python"}
			if mode == "argv" {
				arguments = append(arguments, literals...)
			}
			execution, layout := nodeBoundaryExecution(t, run, root, backend, source, resolution, arguments)
			if execution.OperationID().String() == previous {
				t.Fatal("current fixture approval reused")
			}
			previous = execution.OperationID().String()
			result, executeErr := backend.ExecuteNodeExecution(run, execution, layout)
			var program *sandbox.NodeProgramError
			switch mode {
			case "argv":
				var actual []string
				code, exited := result.ExitCode()
				if executeErr != nil || !result.Complete() || !exited || code != 0 || json.Unmarshal(result.Stdout(), &actual) != nil || !slices.Equal(actual, literals) || len(result.Stderr()) != 0 {
					t.Fatal("literal argv boundary incomplete", executeErr)
				}
			case "readonly":
				code, exited := result.ExitCode()
				if executeErr != nil || !result.Complete() || !exited || code != 0 || string(result.Stdout()) != "readonly-denied\n" || len(result.Stderr()) != 0 {
					t.Fatal("readonly self-write was not denied with whole source retained", executeErr)
				}
			case "deadline":
				if result.Complete() || !errors.Is(executeErr, context.DeadlineExceeded) || errors.As(executeErr, &program) || !bytes.HasPrefix(result.Stdout(), []byte("ready\n")) {
					t.Fatal("actual caller deadline lost body readiness/cause or became complete", executeErr)
				}
			case "term", "kill":
				want := uint8(15)
				if mode == "kill" {
					want = 9
				}
				signal, observed := result.Signal()
				if !result.Complete() || !observed || signal != want || !errors.As(executeErr, &program) || string(result.Stdout()) != "signal\n" || len(result.Stderr()) != 0 {
					t.Fatal("selected Node kernel signal differs", executeErr)
				}
				if _, exited := result.ExitCode(); exited {
					t.Fatal("kernel signal became normal exit")
				}
			case "cancel":
				if result.Complete() || !errors.Is(executeErr, context.Canceled) || !errors.Is(executeErr, cause) || errors.As(executeErr, &program) || !bytes.HasPrefix(result.Stdout(), []byte("ready\n")) {
					t.Fatal("body cancellation lost cause or became complete", executeErr)
				}
			case "stdout-overflow", "stderr-overflow":
				if result.Complete() || !errors.Is(executeErr, hosttool.ErrExecutionOutputLimit) || errors.As(executeErr, &program) {
					t.Fatal("overflow became completed guest result", executeErr)
				}
				out, diagnostic := len(result.Stdout()), len(result.Stderr())
				if mode == "stdout-overflow" && (out != 1<<20 || diagnostic != 0) || mode == "stderr-overflow" && (diagnostic != 1<<20 || out != 0) {
					t.Fatal("independent stream bounds differ", out, diagnostic)
				}
			case "outside-read", "network", "child", "comm-spoof":
				prefix := map[string]string{"outside-read": "fs-ready\n", "network": "network-ready\n", "child": "child-ready\n", "comm-spoof": "spoof-ready\n"}[mode]
				if result.Complete() || executeErr == nil || !bytes.HasPrefix(result.Stdout(), []byte(prefix)) {
					t.Fatal("hostile body became complete or was not reached", executeErr)
				}
				detail := executeErr.Error()
				switch mode {
				case "outside-read":
					if !strings.Contains(detail, "filesystem-outside-workspace") {
						t.Fatal("outside open lacks actual forbidden-path fact", executeErr)
					}
				case "network":
					if !strings.Contains(detail, "network-attempt") {
						t.Fatal("guest loopback connection lacks untrusted network fact", executeErr)
					}
				case "child":
					unexpected := strings.Contains(detail, "process-exec-unexpected")
					clone := strings.Contains(detail, "open_provenance=CLONE_CHILD")
					artifact := strings.Contains(detail, "open_role=ARTIFACT")
					if !unexpected && (!clone || !artifact) {
						t.Fatal("child failure lacks untrusted exec or clone fact", executeErr)
					}
					t.Logf("node_boundary_child wait_delay=%t unexpected_exec=%t clone_fault=%t artifact_role=%t", errors.Is(executeErr, exec.ErrWaitDelay), unexpected, clone, artifact)
				case "comm-spoof":
					if !strings.Contains(detail, "open_role=ARTIFACT") || !strings.Contains(detail, "open_kernel_image=NODE") {
						t.Fatal("helper name spoof lacks retained kernel Node Artifact identity", executeErr)
					}
				}
			}
			t.Logf("node_boundary case=%s current_policy=ALLOW fixture_registry=CONTROLLED_TEST native_solver=NOT_RUN complete=%t stdout_bytes=%d stderr_bytes=%d", mode, result.Complete(), len(result.Stdout()), len(result.Stderr()))
		}) {
			return
		}
	}
}
