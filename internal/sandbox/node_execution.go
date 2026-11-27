package sandbox

import (
	"context"
	"errors"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

var errNodeExecution = errors.New("node execution lacks complete bound runtime evidence")

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
func (b *Backend) ExecuteNodeExecution(ctx context.Context, execution domain.ExecutionContext, layout artifactnpm.NodeExecutionLayout) (NodeExecutionOutcome, error) {
	return b.executeJavaScriptExecution(ctx, execution, layout, false)
}

func (b *Backend) executeJavaScriptExecution(ctx context.Context, execution domain.ExecutionContext, layout artifactnpm.NodeExecutionLayout, bun bool) (outcome NodeExecutionOutcome, resultErr error) {
	failure, runtime, mode, makeProgramError := errNodeExecution, nodeImageReference, boundaryNodeHandoffMode, nodeProgramError
	command := nodeExecutionCommand(layout.EntryPath(), execution.Arguments())
	if bun {
		failure, runtime, mode, makeProgramError = errBunExecution, BunExecutionRuntimeIdentity(), boundaryBunHandoffMode, bunProgramError
		command = bunExecutionCommand(layout.EntryPath(), execution.Arguments())
	}
	defer func() {
		if ctx != nil && ctx.Err() != nil {
			outcome.complete = false
			resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx))
		}
	}()
	if b == nil || ctx == nil || !layout.MatchesExecution(execution) || execution.RuntimeIdentity() != runtime {
		return outcome, failure
	}
	wrapped, ok := b.runner.(*admissionAwareCommandRunner)
	if !ok || wrapped == nil {
		return outcome, failure
	}
	capture, ok := wrapped.runner.(programCapturedRunner)
	if !ok {
		return outcome, failure
	}
	inputs, err := b.prepareJavaScriptExecutionInputView(ctx, execution, layout, true, bun)
	if err != nil {
		return outcome, errors.Join(failure, err)
	}
	defer func() {
		if closeErr := inputs.close(); closeErr != nil {
			outcome.complete = false
			resultErr = errors.Join(resultErr, closeErr)
		}
	}()
	verify := func() error {
		if err := verifyNodeExecutionInputTree(inputs.run, b.runner.(streamingDockerOutput), inputs.reader.id, execution, layout); err != nil {
			return err
		}
		if inputs.bun != nil {
			if err := inputs.bun.volume.verifyTarget(inputs.run, b.runner, inputs.reader.id); err != nil {
				return err
			}
		}
		return inputs.volume.verifyAttachments(inputs.run, b.runner, map[string]bool{inputs.reader.id: true})
	}
	captured, runErr := captureAdmittedProgram(inputs.run, capture, inputs.reader.id, mode, command, failure, makeProgramError, verify)
	outcome.complete, outcome.terminal, outcome.output = captured.complete, captured.terminal, captured.output
	return outcome, runErr
}

func nodeExecutionClientResult(terminal directExecTerminal, commandErr error, captureComplete bool) error {
	return programExecutionClientResult(terminal, commandErr, captureComplete, errNodeExecution, nodeProgramError)
}

func nodeProgramError(cause error) error { return &NodeProgramError{cause: cause} }
