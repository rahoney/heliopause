//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestLinuxPythonExecutionHostileIntegration(t *testing.T) {
	if os.Getenv("HELOX_PYTHON_EXECUTION_HOSTILE_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user controlled hostile Python qualification")
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
	for _, mode := range []string{"outside-read", "network", "clone-exec", "comm-spoof", "readonly"} {
		if !t.Run(mode, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			launcher, err := hosttool.NewSystemObserverLauncher()
			if err != nil {
				t.Fatal(err)
			}
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
			backend, err := sandbox.NewLinuxPyPIDynamicBackendWithExecutor(root+"/intake", executor, observer.Observer(), newObservationResourceAdapter())
			if err != nil {
				t.Fatal(err)
			}
			source, resolution := pythonBoundarySourceWithModule(t, root+"/intake", pythonHostileModule)
			execution, entry := pythonBoundaryExecution(t, ctx, root, backend, source, resolution, []string{mode})
			if seen[execution.OperationID().String()] || seen[execution.EntryArtifact().ContentHandle()] {
				t.Fatal("current hostile fixture approval/acquisition reused")
			}
			seen[execution.OperationID().String()], seen[execution.EntryArtifact().ContentHandle()] = true, true
			result, executeErr := backend.ExecutePythonExecution(ctx, execution, entry)
			var program *sandbox.PythonProgramError
			prefix := map[string]string{"outside-read": "fs-ready\n", "network": "network-ready\n", "clone-exec": "clone-ready\n", "comm-spoof": "spoof-ready\n", "readonly": "readonly-denied\n"}[mode]
			if mode == "readonly" {
				code, exited := result.ExitCode()
				if executeErr != nil || !result.Complete() || !exited || code != 0 || string(result.Stdout()) != "readonly-denied\n" || len(result.Stderr()) != 0 {
					t.Fatal("readonly write was not denied with complete unchanged input", executeErr)
				}
			} else if result.Complete() || executeErr == nil || errors.As(executeErr, &program) || !bytes.HasPrefix(result.Stdout(), []byte(prefix)) {
				t.Fatal("hostile Python body not reached or became completed guest result", executeErr)
			}
			detail := ""
			if executeErr != nil {
				detail = executeErr.Error()
			}
			switch mode {
			case "outside-read":
				if !strings.Contains(detail, "filesystem-outside-workspace") {
					t.Fatal("actual forbidden filesystem kind missing", executeErr)
				}
			case "network":
				if !strings.Contains(detail, "network-attempt") {
					t.Fatal("actual network attempt kind missing", executeErr)
				}
			case "clone-exec":
				unexpected := strings.Contains(detail, "process-exec-unexpected")
				clone := strings.Contains(detail, "open_provenance=CLONE_CHILD")
				artifact := strings.Contains(detail, "open_role=ARTIFACT")
				if !unexpected && (!clone || !artifact) {
					t.Fatal("cloned Python exec lacks untrusted exec or immutable clone/Artifact facts", executeErr)
				}
				t.Logf("python_hostile_clone_exec child_body_observed=%t unexpected_exec=%t clone_fault=%t artifact_role=%t", bytes.Contains(result.Stdout(), []byte("child-body\n")), unexpected, clone, artifact)
			case "comm-spoof":
				if !strings.Contains(detail, "open_image=PYTHON") || !strings.Contains(detail, "open_executable_pinned=true") || !strings.Contains(detail, "open_role=ARTIFACT") {
					t.Fatal("helper-looking comm changed immutable Python Artifact identity", executeErr)
				}
			}
			parts := strings.Split(execution.EntryArtifact().ContentHandle(), ":")
			if len(parts) != 3 {
				t.Fatal("hostile fixture intake handle differs")
			}
			body, err := os.ReadFile(filepath.Join(source.root, parts[1], "wheel.whl"))
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(body)
			if hex.EncodeToString(sum[:]) != source.digest.String() {
				t.Fatal("original hostile fixture wheel mutated")
			}
			t.Logf("python_hostile case=%s current_policy=ALLOW fixture_source=CONTROLLED_TEST native_solver=NOT_RUN complete=%t source_unchanged=true stdout_bytes=%d stderr_bytes=%d", mode, result.Complete(), len(result.Stdout()), len(result.Stderr()))
		}) {
			return
		}
	}
}
