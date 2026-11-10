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

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
)

// This exercises trusted controller roots, not Artifact Node admission. The
// kernel fact and captured Host-client bytes are independently reconciled.
func TestLinuxDirectExecKernelTerminalIntegration(t *testing.T) {
	if os.Getenv("HELOX_KERNEL_TERMINAL_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor kernel terminal observer")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close()
	supervisor := integrationObserverSupervisor(t)
	defer supervisor.Close()
	for _, test := range []struct {
		name, script string
		code         uint8
		signal       uint8
	}{
		{"zero", "printf out; printf err >&2; exit 0", 0, 0},
		{"nonzero", "printf out; printf err >&2; exit 37", 37, 0},
		{"reserved125", "printf out; printf err >&2; exit 125", 125, 0},
		{"reserved126", "printf out; printf err >&2; exit 126", 126, 0},
		{"reserved127", "printf out; printf err >&2; exit 127", 127, 0},
		{"exit255", "printf out; printf err >&2; exit 255", 255, 0},
		{"signalTERM", "printf out; printf err >&2; kill -TERM $$", 0, 15},
		{"signalKILL", "printf out; printf err >&2; kill -KILL $$", 0, 9},
	} {
		t.Run(test.name, func(t *testing.T) {
			sessionID, err := domain.NewSandboxSessionID()
			if err != nil {
				t.Fatal(err)
			}
			body, createErr := executor.Output(ctx, "docker", createArguments(sessionID)...)
			container := strings.TrimSpace(string(body))
			if !exactObservationContainerID(container) {
				t.Fatal("kernel terminal container identity unavailable", createErr)
			}
			var trace TraceReader
			defer func() {
				cleanup, done := context.WithTimeout(context.Background(), cleanupTimeout)
				defer done()
				if _, err := executor.Output(cleanup, "docker", "rm", "--force", container); err != nil {
					t.Error("kernel terminal container cleanup failed", err)
				}
				if trace != nil {
					collect, finish := context.WithTimeout(context.Background(), cleanupTimeout)
					defer finish()
					_, limitation, diagnostic := collectTraceDiagnostic(collect, trace)
					if limitation != "" || len(retainedSuspiciousTraceFacts(diagnostic.KindCounts)) != 0 {
						t.Error("kernel terminal trace incomplete", diagnostic.String())
					}
				}
			}()
			if createErr != nil {
				t.Fatal(createErr)
			}
			trace, err = startTrace(ctx, supervisor.Observer(), container, "npm-lifecycle")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := executor.Output(ctx, "docker", "start", container); err != nil {
				t.Fatal(err)
			}
			if err := awaitBoundaryHelper(ctx, admissionAwareRunner(executor), container); err != nil {
				t.Fatal(err)
			}
			if err := awaitMountAnchors(ctx, supervisor.Observer(), container); err != nil {
				t.Fatal(err)
			}
			if err := verifyPNPMInstanceNetwork(ctx, executor, container); err != nil {
				t.Fatal(err)
			}
			session := activeObserverSession(container)
			if session == nil {
				t.Fatal("kernel terminal admission session absent")
			}
			lease, err := acquireDirectExecLifecycle(ctx, session)
			if err != nil {
				t.Fatal(err)
			}
			defer lease.release()
			nonce, err := newDirectExecAdmissionNonce()
			if err != nil {
				t.Fatal(err)
			}
			admission := directExecAdmission{containerID: container, generation: session.generation, mode: boundaryLaunchMode, nonce: nonce, session: session, lifecycle: lease}
			if err := controlDirectExecAdmission(ctx, "arm", admission); err != nil {
				t.Fatal(err)
			}
			arguments := boundaryExecArguments(container, boundaryLaunchMode, "/bin/sh", "-ceu", test.script)
			_, _, index, valid := directExecOrigin(arguments, "docker")
			if !valid {
				t.Fatal("kernel terminal fixture has no canonical direct origin")
			}
			prepared := append(append(append([]string{}, arguments[:index]...), nonce), arguments[index:]...)
			output, commandErr := executor.RunCaptured(ctx, "docker", prepared...)
			if string(output.Stdout()) != "out" || string(output.Stderr()) != "err" || output.StdoutOverflow() || output.StderrOverflow() {
				t.Fatal("kernel terminal captured streams changed")
			}
			if test.code == 0 && test.signal == 0 {
				if commandErr != nil {
					t.Fatal(commandErr)
				}
			} else {
				var exited *exec.ExitError
				if !errors.As(commandErr, &exited) {
					t.Fatal("trusted Docker client lost nonzero process error", commandErr)
				}
			}
			wait, stop := context.WithTimeout(ctx, directExecAdmissionCompletionTimeout)
			defer stop()
			var terminal directExecTerminal
			for !terminal.observed {
				terminal, err = directExecOutcome(wait, admission)
				if err != nil {
					t.Fatal("kernel terminal query failed", err)
				}
				if !terminal.observed {
					select {
					case <-wait.Done():
						t.Fatal("kernel terminal event absent", wait.Err())
					case <-time.After(directExecAdmissionPollInterval):
					}
				}
			}
			if terminal.exited != (test.signal == 0) || terminal.exitCode != test.code || terminal.signal != test.signal || terminal.coreDump {
				t.Fatal("kernel terminal outcome did not match actual controller", terminal)
			}
			target, err := directExecTargetLoaded(ctx, admission)
			if err != nil || !target.loaded || target.missed {
				t.Fatal("accepted target image was not independently observed", err)
			}
			if err := completeDirectExecAdmission(ctx, admission); err != nil {
				t.Fatal("kernel terminal slot completion failed", err)
			}
			t.Logf("kernel_terminal case=%s exited=%t exit_code=%d signal=%d stdout_bytes=%d stderr_bytes=%d artifact_user_program_run=false", test.name, terminal.exited, terminal.exitCode, terminal.signal, len(output.Stdout()), len(output.Stderr()))
		})
	}
}
