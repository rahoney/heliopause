package sandbox

import (
	"context"
	"errors"
	"os/exec"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type programCapturedRunner interface {
	CaptureExecutionOutput(context.Context, string, ...string) (domain.CapturedProcessOutput, error)
}

// Internal runtime DATA only. The public family outcome retains its own
// redaction and may be downgraded by later input disposal/caller cancellation.
type capturedProgramExecution struct {
	complete bool
	terminal directExecTerminal
	output   domain.CapturedProcessOutput
}

// This is the existing captured Node protocol, shared without changing its
// order. Source verification runs before complete, while the lease is held;
// the lease is released before the caller tears down the input/observer.
func captureAdmittedProgram(run context.Context, capture programCapturedRunner, containerID, expectedMode string, command []string, incompleteErr error, makeProgramError func(error) error, verify func() error) (outcome capturedProgramExecution, resultErr error) {
	if incompleteErr == nil {
		return outcome, errors.New("program execution evidence is unavailable")
	}
	if run == nil || capture == nil || makeProgramError == nil || verify == nil || !exactObservationContainerID(containerID) || len(command) == 0 || (expectedMode != boundaryNodeHandoffMode && expectedMode != boundaryPythonHandoffMode && expectedMode != boundaryBunHandoffMode) {
		return outcome, incompleteErr
	}
	session := activeObserverSession(containerID)
	if session == nil {
		return outcome, incompleteErr
	}
	lease, err := acquireDirectExecLifecycle(run, session)
	if err != nil {
		return outcome, errors.Join(incompleteErr, err)
	}
	defer lease.release()
	nonce, err := newDirectExecAdmissionNonce()
	if err != nil {
		return outcome, errors.Join(incompleteErr, err)
	}
	admission := directExecAdmission{containerID: containerID, generation: session.generation, mode: expectedMode, nonce: nonce, session: session, lifecycle: lease}
	if err := controlDirectExecAdmission(run, "arm", admission); err != nil {
		return outcome, errors.Join(incompleteErr, err)
	}
	completed := false
	defer func() {
		if !completed {
			finish, done := context.WithTimeout(context.Background(), directExecAdmissionCompletionTimeout)
			defer done()
			resultErr = errors.Join(resultErr, finalizeDirectExecAdmission(finish, admission, false))
		}
	}()
	args := boundaryExecArguments(containerID, expectedMode, command...)
	_, mode, index, valid := directExecOrigin(args, "docker")
	if !valid || mode != expectedMode {
		return outcome, incompleteErr
	}
	prepared := append(append(append([]string{}, args[:index]...), nonce), args[index:]...)
	var commandErr error
	outcome.output, commandErr = capture.CaptureExecutionOutput(run, "docker", prepared...)
	if outcome.output == nil || run.Err() != nil || outcome.output.StdoutOverflow() || outcome.output.StderrOverflow() || errors.Is(commandErr, exec.ErrWaitDelay) {
		return outcome, errors.Join(incompleteErr, commandErr, run.Err(), context.Cause(run))
	}
	wait, stop := directExecAdmissionCompletionContext(run)
	defer stop()
	for !outcome.terminal.observed {
		outcome.terminal, err = directExecOutcome(wait, admission)
		if err != nil {
			return outcome, errors.Join(incompleteErr, commandErr, err)
		}
		if !outcome.terminal.observed {
			select {
			case <-wait.Done():
				return outcome, errors.Join(incompleteErr, commandErr, wait.Err(), context.Cause(wait))
			case <-time.After(directExecAdmissionPollInterval):
			}
		}
	}
	target, err := directExecTargetLoaded(wait, admission)
	if err != nil || !target.loaded || target.missed {
		return outcome, errors.Join(incompleteErr, commandErr, err)
	}
	programFailure := false
	terminalErr := programExecutionClientResult(outcome.terminal, commandErr, true, incompleteErr, func(cause error) error {
		programErr := makeProgramError(cause)
		if programErr != nil {
			programFailure = true
		}
		return programErr
	})
	if terminalErr != nil && !programFailure {
		return outcome, terminalErr
	}
	if err := verify(); err != nil {
		return outcome, errors.Join(terminalErr, incompleteErr, err)
	}
	if err := completeDirectExecAdmission(run, admission); err != nil {
		return outcome, errors.Join(terminalErr, incompleteErr, err)
	}
	completed = true
	outcome.complete = true
	return outcome, terminalErr
}

func programExecutionClientResult(terminal directExecTerminal, commandErr error, captureComplete bool, incompleteErr error, programError func(error) error) error {
	if incompleteErr == nil || programError == nil {
		return errors.New("program client result mapping is unavailable")
	}
	if !terminal.observed || !captureComplete {
		return errors.Join(incompleteErr, commandErr)
	}
	expected := int(terminal.exitCode)
	if !terminal.exited {
		expected = 128 + int(terminal.signal)
	}
	if commandErr == nil {
		if expected == 0 {
			return nil
		}
		return incompleteErr
	}
	var exited *exec.ExitError
	if !errors.As(commandErr, &exited) || expected == 0 || exited.ExitCode() != expected || errors.Is(commandErr, exec.ErrWaitDelay) {
		return errors.Join(incompleteErr, commandErr)
	}
	programErr := programError(commandErr)
	if programErr == nil {
		return errors.Join(incompleteErr, commandErr)
	}
	return programErr
}
