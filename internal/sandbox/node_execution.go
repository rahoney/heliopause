package sandbox

import (
	"context"
	"errors"
	"os/exec"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

var errNodeExecution = errors.New("node execution lacks complete bound runtime evidence")

type nodeCapturedRunner interface {
	CaptureExecutionOutput(context.Context, string, ...string) (domain.CapturedProcessOutput, error)
}

// NodeExecutionOutcome is private-memory runtime DATA. It grants neither
// inspection coverage nor Policy approval or generated-file promotion.
type NodeExecutionOutcome struct {
	complete bool
	terminal directExecTerminal
	output   domain.CapturedProcessOutput
}

func (o NodeExecutionOutcome) Complete() bool { return o.complete }
func (o NodeExecutionOutcome) ExitCode() (uint8, bool) {
	return o.terminal.exitCode, o.complete && o.terminal.observed && o.terminal.exited
}
func (o NodeExecutionOutcome) Signal() (uint8, bool) {
	return o.terminal.signal, o.complete && o.terminal.observed && !o.terminal.exited
}
func (o NodeExecutionOutcome) Stdout() []byte {
	if o.output == nil {
		return nil
	}
	return o.output.Stdout()
}
func (o NodeExecutionOutcome) Stderr() []byte {
	if o.output == nil {
		return nil
	}
	return o.output.Stderr()
}
func (NodeExecutionOutcome) String() string     { return "private Node execution outcome" }
func (o NodeExecutionOutcome) GoString() string { return o.String() }
func (NodeExecutionOutcome) MarshalJSON() ([]byte, error) {
	return nil, errors.New("node runtime output is private memory data")
}

// NodeProgramError represents a completed guest nonzero/signal separately
// from incomplete runtime evidence. The original Host process error survives.
type NodeProgramError struct{ cause error }

func (*NodeProgramError) Error() string    { return "node program did not exit successfully" }
func (e *NodeProgramError) Unwrap() error  { return e.cause }
func (*NodeProgramError) GoString() string { return "private Node program error" }

func nodeExecutionCommand(entry string, arguments []string) []string {
	command := []string{"/usr/local/bin/node", "--", nodeExecutionProject + "/" + entry}
	return append(command, arguments...)
}

// ExecuteNodeExecution runs the current approved complete readonly tree as
// ARTIFACT. Success additionally requires independent target/terminal facts,
// bounded capture, complete observation and disposal of the whole guest.
func (b *Backend) ExecuteNodeExecution(ctx context.Context, execution domain.ExecutionContext, layout artifactnpm.NodeExecutionLayout) (outcome NodeExecutionOutcome, resultErr error) {
	defer func() {
		if ctx != nil && ctx.Err() != nil {
			outcome.complete = false
			resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx))
		}
	}()
	if b == nil || ctx == nil || !layout.MatchesExecution(execution) || execution.RuntimeIdentity() != nodeImageReference {
		return outcome, errNodeExecution
	}
	wrapped, ok := b.runner.(*admissionAwareCommandRunner)
	if !ok || wrapped == nil {
		return outcome, errNodeExecution
	}
	capture, ok := wrapped.runner.(nodeCapturedRunner)
	if !ok {
		return outcome, errNodeExecution
	}
	inputs, err := b.prepareNodeExecutionInputView(ctx, execution, layout, true)
	if err != nil {
		return outcome, errors.Join(errNodeExecution, err)
	}
	defer func() {
		if closeErr := inputs.close(); closeErr != nil {
			outcome.complete = false
			resultErr = errors.Join(resultErr, closeErr)
		}
	}()
	run := inputs.run
	session := activeObserverSession(inputs.reader.id)
	if session == nil {
		return outcome, errNodeExecution
	}
	lease, err := acquireDirectExecLifecycle(run, session)
	if err != nil {
		return outcome, errors.Join(errNodeExecution, err)
	}
	defer lease.release()
	nonce, err := newDirectExecAdmissionNonce()
	if err != nil {
		return outcome, errors.Join(errNodeExecution, err)
	}
	admission := directExecAdmission{containerID: inputs.reader.id, generation: session.generation, mode: boundaryNodeHandoffMode, nonce: nonce, session: session, lifecycle: lease}
	if err := controlDirectExecAdmission(run, "arm", admission); err != nil {
		return outcome, errors.Join(errNodeExecution, err)
	}
	completed := false
	defer func() {
		if !completed {
			finish, done := context.WithTimeout(context.Background(), directExecAdmissionCompletionTimeout)
			defer done()
			resultErr = errors.Join(resultErr, finalizeDirectExecAdmission(finish, admission, false))
		}
	}()
	args := boundaryExecArguments(inputs.reader.id, boundaryNodeHandoffMode, nodeExecutionCommand(layout.EntryPath(), execution.Arguments())...)
	_, mode, index, valid := directExecOrigin(args, "docker")
	if !valid || mode != boundaryNodeHandoffMode {
		return outcome, errNodeExecution
	}
	prepared := append(append(append([]string{}, args[:index]...), nonce), args[index:]...)
	var commandErr error
	outcome.output, commandErr = capture.CaptureExecutionOutput(run, "docker", prepared...)
	if outcome.output == nil || run.Err() != nil || outcome.output.StdoutOverflow() || outcome.output.StderrOverflow() || errors.Is(commandErr, exec.ErrWaitDelay) {
		return outcome, errors.Join(errNodeExecution, commandErr, run.Err(), context.Cause(run))
	}
	wait, stop := directExecAdmissionCompletionContext(run)
	defer stop()
	for !outcome.terminal.observed {
		outcome.terminal, err = directExecOutcome(wait, admission)
		if err != nil {
			return outcome, errors.Join(errNodeExecution, commandErr, err)
		}
		if !outcome.terminal.observed {
			select {
			case <-wait.Done():
				return outcome, errors.Join(errNodeExecution, commandErr, wait.Err(), context.Cause(wait))
			case <-time.After(directExecAdmissionPollInterval):
			}
		}
	}
	target, err := directExecTargetLoaded(wait, admission)
	if err != nil || !target.loaded || target.missed {
		return outcome, errors.Join(errNodeExecution, commandErr, err)
	}
	terminalErr := nodeExecutionClientResult(outcome.terminal, commandErr, true)
	var programErr *NodeProgramError
	if terminalErr != nil && !errors.As(terminalErr, &programErr) {
		return outcome, terminalErr
	}
	if err := verifyNodeExecutionInputTree(run, b.runner.(streamingDockerOutput), inputs.reader.id, execution, layout); err != nil {
		return outcome, errors.Join(terminalErr, errNodeExecution, err)
	}
	if err := inputs.volume.verifyAttachments(run, b.runner, map[string]bool{inputs.reader.id: true}); err != nil {
		return outcome, errors.Join(terminalErr, errNodeExecution, err)
	}
	if err := completeDirectExecAdmission(run, admission); err != nil {
		return outcome, errors.Join(terminalErr, errNodeExecution, err)
	}
	completed = true
	outcome.complete = true
	return outcome, terminalErr
}

func nodeExecutionClientResult(terminal directExecTerminal, commandErr error, captureComplete bool) error {
	if !terminal.observed || !captureComplete {
		return errors.Join(errNodeExecution, commandErr)
	}
	expected := int(terminal.exitCode)
	if !terminal.exited {
		expected = 128 + int(terminal.signal)
	}
	if commandErr == nil {
		if expected == 0 {
			return nil
		}
		return errNodeExecution
	}
	var exited *exec.ExitError
	if !errors.As(commandErr, &exited) || expected == 0 || exited.ExitCode() != expected || errors.Is(commandErr, exec.ErrWaitDelay) {
		return errors.Join(errNodeExecution, commandErr)
	}
	return &NodeProgramError{cause: commandErr}
}
