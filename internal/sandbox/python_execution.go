package sandbox

import (
	"context"
	"errors"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

var errPythonExecution = errors.New("python execution lacks complete bound runtime evidence")

// Private-memory runtime DATA, never inspection/Policy or output promotion.
type PythonExecutionOutcome struct{ captured capturedProgramExecution }

func (o PythonExecutionOutcome) Complete() bool { return o.captured.complete }
func (o PythonExecutionOutcome) ExitCode() (uint8, bool) {
	return o.captured.terminal.exitCode, o.Complete() && o.captured.terminal.observed && o.captured.terminal.exited
}
func (o PythonExecutionOutcome) Signal() (uint8, bool) {
	return o.captured.terminal.signal, o.Complete() && o.captured.terminal.observed && !o.captured.terminal.exited
}
func (o PythonExecutionOutcome) Stdout() []byte {
	if o.captured.output == nil {
		return nil
	}
	return o.captured.output.Stdout()
}
func (o PythonExecutionOutcome) Stderr() []byte {
	if o.captured.output == nil {
		return nil
	}
	return o.captured.output.Stderr()
}
func (PythonExecutionOutcome) String() string     { return "private Python execution outcome" }
func (o PythonExecutionOutcome) GoString() string { return o.String() }
func (PythonExecutionOutcome) MarshalJSON() ([]byte, error) {
	return nil, errors.New("python runtime output is private memory data")
}

// PythonProgramError is a completed guest failure, distinct from missing
// operational evidence. The original Host process cause is retained.
type PythonProgramError struct{ cause error }

func (*PythonProgramError) Error() string    { return "python program did not exit successfully" }
func (e *PythonProgramError) Unwrap() error  { return e.cause }
func (*PythonProgramError) GoString() string { return "private Python program error" }

const pythonExecutionProgram = `import importlib
import sys
expected = ['/usr/local/lib/python314.zip', '/usr/local/lib/python3.14', '/usr/local/lib/python3.14/lib-dynload']
if not (sys.flags.isolated and sys.flags.no_site and sys.flags.dont_write_bytecode) or 'site' in sys.modules or sys.path != expected:
    raise RuntimeError('console interpreter startup differs from pinned isolated runtime')
module, attr, command = sys.argv[1:4]
sys.path.append('/haa-site')
sys.argv = [command, *sys.argv[4:]]
target = importlib.import_module(module)
for part in attr.split('.'):
    target = getattr(target, part)
raise SystemExit(target())
`

func pythonExecutionCommand(entry artifactpypi.ConsoleExecutionEntry, arguments []string) []string {
	point := entry.EntryPoint()
	command := []string{"/usr/local/bin/python3.14", "-I", "-S", "-B", "-c", pythonExecutionProgram, point.Module, point.Attr, entry.Command()}
	return append(command, arguments...)
}

// ExecutePythonExecution consumes the existing current whole readonly input
// lifecycle. A valid guest nonzero never bypasses final CPU/trace/cleanup.
func (b *PythonDynamicBackend) ExecutePythonExecution(ctx context.Context, execution domain.ExecutionContext, entry artifactpypi.ConsoleExecutionEntry) (outcome PythonExecutionOutcome, resultErr error) {
	defer func() {
		if ctx != nil && ctx.Err() != nil {
			outcome.captured.complete = false
			resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx))
		}
	}()
	if b == nil || ctx == nil {
		return outcome, errPythonExecution
	}
	wrapped, ok := b.runner.(*admissionAwareCommandRunner)
	if !ok || wrapped == nil {
		return outcome, errPythonExecution
	}
	capture, ok := wrapped.runner.(programCapturedRunner)
	if !ok {
		return outcome, errPythonExecution
	}
	if _, ok := b.observer.(pythonExecutionObserver); !ok {
		return outcome, errPythonExecution
	}
	var programErr error
	operationalErr := b.withPythonExecutionInputs(ctx, execution, entry, func(t *pythonTransactionExecutor) error {
		for _, wheel := range t.manifest.inspections {
			if err := artifactpypi.ValidatePythonExecutionStartup(wheel); err != nil {
				return err
			}
		}
		t.execution = true
		reader, err := t.startPhase(t.requestCtx, phaseObservation, 1)
		if err != nil {
			return err
		}
		if err := t.verifyAnchorAndVolume(t.requestCtx, map[string]bool{reader.id: true}); err != nil {
			return err
		}
		verify := func() error {
			installed, err := verifyClosureInstallation(t.requestCtx, b.runner, reader.id, t.manifest, t.policy.RuntimeTmpfs())
			if err != nil || installed.identity != t.installed.identity {
				return errors.Join(errPythonExecution, err)
			}
			return t.verifyAnchorAndVolume(t.requestCtx, map[string]bool{reader.id: true})
		}
		captured, err := captureAdmittedProgram(t.requestCtx, capture, reader.id, boundaryPythonHandoffMode, pythonExecutionCommand(entry, execution.Arguments()), errPythonExecution, func(cause error) error { return &PythonProgramError{cause: cause} }, verify)
		outcome.captured = captured
		var failed *PythonProgramError
		if err != nil && (!captured.complete || !errors.As(err, &failed)) {
			return err
		}
		programErr = err
		// captureAdmittedProgram has released its lease before teardown.
		return t.terminatePhase(reader.id)
	})
	if operationalErr != nil {
		outcome.captured.complete = false
		return outcome, errors.Join(errPythonExecution, programErr, operationalErr)
	}
	if !outcome.Complete() {
		return outcome, errPythonExecution
	}
	return outcome, programErr
}
