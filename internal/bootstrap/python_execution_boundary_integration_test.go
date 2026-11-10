//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestLinuxPythonExecutionBoundaryIntegration(t *testing.T) {
	if os.Getenv("HELOX_PYTHON_EXECUTION_BOUNDARY_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user controlled Python boundary qualification")
	}
	ctx, stop := context.WithTimeout(context.Background(), 8*time.Minute)
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
	seen := map[string]bool{}
	for _, mode := range []string{"argv", "exit37", "term", "kill", "cancel", "stdout-overflow", "stderr-overflow", "deadline"} {
		if !t.Run(mode, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			run, cancel := context.WithCancelCause(ctx)
			defer cancel(nil)
			cause := errors.New("controlled Python caller cancellation")
			runner := &nodeBoundaryCapture{Executor: executor}
			if mode == "cancel" {
				runner.cancel, runner.cause = cancel, cause
			}
			launcher, err := hosttool.NewSystemObserverLauncher()
			if err != nil {
				t.Fatal(err)
			}
			// Caller cancellation does not end the trusted observer cleanup lifetime.
			observer, err := sandbox.NewObserverSupervisor(ctx, func(start context.Context, remote, output string) (sandbox.ObserverProcess, error) {
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
			backend, err := sandbox.NewLinuxPyPIDynamicBackendWithExecutor(root+"/intake", runner, observer.Observer(), newObservationResourceAdapter())
			if err != nil {
				t.Fatal(err)
			}
			source, resolution := pythonBoundarySource(t, root+"/intake")
			arguments := []string{mode}
			literals := []string{"", "a b", "a\nb", "; exit 0", "$(touch forbidden)", "--", "--handoff-python"}
			if mode == "argv" {
				arguments = append(arguments, literals...)
			}
			execution, entry := pythonBoundaryExecution(t, run, root, backend, source, resolution, arguments)
			if seen[execution.OperationID().String()] || seen[execution.EntryArtifact().ContentHandle()] {
				t.Fatal("current fixture approval or acquisition reused")
			}
			seen[execution.OperationID().String()], seen[execution.EntryArtifact().ContentHandle()] = true, true
			if mode == "deadline" {
				var finish context.CancelFunc
				run, finish = context.WithTimeout(run, 30*time.Second)
				defer finish()
			}
			result, executeErr := backend.ExecutePythonExecution(run, execution, entry)
			var program *sandbox.PythonProgramError
			switch mode {
			case "argv":
				var actual []string
				code, exited := result.ExitCode()
				if executeErr != nil || !result.Complete() || !exited || code != 0 || json.Unmarshal(result.Stdout(), &actual) != nil || !slices.Equal(actual, literals) || len(result.Stderr()) != 0 {
					t.Fatal("literal Python argv boundary incomplete", executeErr)
				}
			case "exit37":
				code, exited := result.ExitCode()
				if !result.Complete() || !exited || code != 37 || !errors.As(executeErr, &program) || len(result.Stdout()) != 0 || len(result.Stderr()) != 0 {
					t.Fatal("actual guest exit37 differs", executeErr)
				}
			case "term", "kill":
				want := uint8(15)
				if mode == "kill" {
					want = 9
				}
				signal, observed := result.Signal()
				if !result.Complete() || !observed || signal != want || !errors.As(executeErr, &program) || string(result.Stdout()) != "signal\n" || len(result.Stderr()) != 0 {
					t.Fatal("selected Python kernel signal differs", executeErr)
				}
				if _, exited := result.ExitCode(); exited {
					t.Fatal("kernel signal became normal exit")
				}
			case "cancel":
				if result.Complete() || !errors.Is(executeErr, context.Canceled) || !errors.Is(executeErr, cause) || errors.As(executeErr, &program) || !bytes.HasPrefix(result.Stdout(), []byte("ready\n")) {
					t.Fatal("body cancellation lost original cause/readiness or became complete", executeErr)
				}
			case "deadline":
				if result.Complete() || !errors.Is(executeErr, context.DeadlineExceeded) || errors.As(executeErr, &program) || !bytes.HasPrefix(result.Stdout(), []byte("ready\n")) {
					t.Fatal("actual caller deadline lost readiness/cause or became complete", executeErr)
				}
			case "stdout-overflow", "stderr-overflow":
				if result.Complete() || !errors.Is(executeErr, hosttool.ErrExecutionOutputLimit) || errors.As(executeErr, &program) {
					t.Fatal("overflow became completed guest result", executeErr)
				}
				out, diagnostic := len(result.Stdout()), len(result.Stderr())
				if mode == "stdout-overflow" && (out != 1<<20 || diagnostic != 0) || mode == "stderr-overflow" && (diagnostic != 1<<20 || out != 0) {
					t.Fatal("independent Python stream bounds differ", out, diagnostic)
				}
			}
			parts := strings.Split(execution.EntryArtifact().ContentHandle(), ":")
			if len(parts) != 3 {
				t.Fatal("fixture intake handle differs")
			}
			body, err := os.ReadFile(filepath.Join(source.root, parts[1], "wheel.whl"))
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(body)
			if hex.EncodeToString(sum[:]) != source.digest.String() {
				t.Fatal("original fixture wheel mutated")
			}
			t.Logf("python_boundary case=%s current_policy=ALLOW fixture_source=CONTROLLED_TEST native_solver=NOT_RUN complete=%t stdout_bytes=%d stderr_bytes=%d", mode, result.Complete(), len(result.Stdout()), len(result.Stderr()))
		}) {
			return
		}
	}
}
