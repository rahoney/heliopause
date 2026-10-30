package sandbox

import (
	"context"
	"errors"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const bunExecutionProfile = "bun-execution"

var errBunExecution = errors.New("bun execution lacks complete bound runtime evidence")

func BunExecutionRuntimeIdentity() string { return "bun-program:" + BunResolverRuntimeIdentity() }

type bunExecutionObserver interface {
	StartBunExecutionProfile(context.Context, string) (TraceReader, error)
}

type bunExecutionSupply struct {
	volume nativeRuntimeVolume
	tool   *bunPreparedTool
}

func bunExecutionExpectedTopology() ([]observerMountExpectation, bool) {
	topology, ok := nodeExecutionInputExpectedTopology(true)
	if !ok {
		return nil, false
	}
	for i := range topology {
		if topology[i].Mountpoint == "/haa-runtime" {
			topology[i].FSType, topology[i].NoSUID = "9p", false
		}
	}
	return topology, true
}

func (o *SharedObserver) StartBunExecutionProfile(ctx context.Context, container string) (TraceReader, error) {
	topology, ok := bunExecutionExpectedTopology()
	if !ok {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	return o.startProfile(ctx, container, bunExecutionProfile, topology)
}

func bunExecutionCommand(entry string, arguments []string) []string {
	command := []string{bunToolGuestPath, "--no-install", "--no-env-file", "--no-macros", "--", nodeExecutionProject + "/" + entry}
	return append(command, arguments...)
}

// BunExecutionOutcome uses the existing independent target/terminal and bounded
// private output owner; its runtime identity and admission remain Bun-specific.
type BunExecutionOutcome struct{ NodeExecutionOutcome }

func (BunExecutionOutcome) String() string     { return "private Bun execution outcome" }
func (o BunExecutionOutcome) GoString() string { return o.String() }
func (BunExecutionOutcome) MarshalJSON() ([]byte, error) {
	return nil, errors.New("bun runtime output is private memory data")
}

type BunProgramError struct{ cause error }

func (*BunProgramError) Error() string    { return "bun program did not exit successfully" }
func (e *BunProgramError) Unwrap() error  { return e.cause }
func (*BunProgramError) GoString() string { return "private Bun program error" }
func bunProgramError(cause error) error   { return &BunProgramError{cause} }

func (b *Backend) ExecuteBunExecution(ctx context.Context, execution domain.ExecutionContext, layout artifactnpm.NodeExecutionLayout) (BunExecutionOutcome, error) {
	outcome, err := b.executeJavaScriptExecution(ctx, execution, layout, true)
	return BunExecutionOutcome{outcome}, err
}
