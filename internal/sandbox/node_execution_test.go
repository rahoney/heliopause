package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
)

func TestNodeExecutionRejectsIncompleteInputAndPreservesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("private cancellation")
	cancel(cause)
	var backend *Backend
	outcome, err := backend.ExecuteNodeExecution(ctx, domain.ExecutionContext{}, artifactnpm.NodeExecutionLayout{})
	if outcome.Complete() || !errors.Is(err, errNodeExecution) || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal("preflight lost primary/caller cause", err)
	}
	var missing context.Context
	if result, err := backend.ExecuteNodeExecution(missing, domain.ExecutionContext{}, artifactnpm.NodeExecutionLayout{}); err == nil || result.Complete() {
		t.Fatal("missing current execution admitted")
	}
}

func TestNodeExecutionCommandKeepsPrivateArgumentsAsVector(t *testing.T) {
	args := []string{"", "a b", "a\nb", "; exit 0", "$(touch forbidden)", "--", "--handoff-python"}
	command := nodeExecutionCommand("node_modules/fixture/bin/cli.js", args)
	if len(command) != 3+len(args) || command[0] != "/usr/local/bin/node" || command[1] != "--" || command[2] != nodeExecutionProject+"/node_modules/fixture/bin/cli.js" || !reflect.DeepEqual(command[3:], args) {
		t.Fatal("Node command changed vector boundaries")
	}
	boundary := boundaryExecArguments("0123456789abcdef", boundaryNodeHandoffMode, command...)
	container, mode, index, ok := directExecOrigin(boundary, "docker")
	if !ok || container != "0123456789abcdef" || mode != boundaryNodeHandoffMode || !reflect.DeepEqual(boundary[index:], command) {
		t.Fatal("Node command lacks exact admitted origin")
	}
}

func TestNodeExecutionOutcomeCannotBecomeDefaultZeroSuccessOrPublicOutput(t *testing.T) {
	var outcome NodeExecutionOutcome
	if _, ok := outcome.ExitCode(); ok || outcome.Complete() {
		t.Fatal("empty result became completed zero exit")
	}
	if _, ok := outcome.Signal(); ok {
		t.Fatal("empty result became signal")
	}
	if _, err := json.Marshal(outcome); err == nil {
		t.Fatal("runtime bytes serializable")
	}
	for _, format := range []string{"%v", "%+v", "%#v"} {
		if got := fmt.Sprintf(format, outcome); got != "private Node execution outcome" {
			t.Fatal("private result formatter exposed fields", got)
		}
	}
}

func TestNodeExecutionClientStatusRequiresIndependentKernelTerminal(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("POSIX trusted fixture unavailable")
	}
	for _, code := range []string{"37", "125", "126", "127", "255"} {
		commandErr := exec.Command("/bin/sh", "-c", "exit "+code).Run()
		var hostExit *exec.ExitError
		if !errors.As(commandErr, &hostExit) {
			t.Fatal("trusted fixture did not exit", commandErr)
		}
		terminal := directExecTerminal{observed: true, exited: true, exitCode: uint8(hostExit.ExitCode())}
		result := nodeExecutionClientResult(terminal, commandErr, true)
		var program *NodeProgramError
		if !errors.As(result, &program) || !errors.Is(result, commandErr) {
			t.Fatal("completed nonzero lost driver cause", result)
		}
		if err := nodeExecutionClientResult(directExecTerminal{}, commandErr, true); !errors.Is(err, errNodeExecution) {
			t.Fatal("Host status became guest fact")
		}
		if err := nodeExecutionClientResult(terminal, nil, true); !errors.Is(err, errNodeExecution) {
			t.Fatal("kernel/client disagreement became success")
		}
		if err := nodeExecutionClientResult(terminal, errors.Join(commandErr, hosttool.ErrExecutionOutputLimit), false); !errors.Is(err, errNodeExecution) {
			t.Fatal("overflow became completed program error")
		}
		if err := nodeExecutionClientResult(terminal, errors.Join(commandErr, exec.ErrWaitDelay), false); !errors.Is(err, errNodeExecution) {
			t.Fatal("pipe wait failure became complete")
		}
	}
	if err := nodeExecutionClientResult(directExecTerminal{observed: true, exited: true}, nil, true); err != nil {
		t.Fatal(err)
	}
}

func TestNodeExecutionProfileUsesOnlyCurrentReadonlyTopology(t *testing.T) {
	normal, _ := observerExpectedTopology("npm-lifecycle")
	profile, ok := observerExpectedTopology(nodeExecutionProfile)
	if !ok || !validObserverProfile(nodeExecutionProfile) || len(profile) != len(normal)+1 {
		t.Fatal("Node profile lacks closed topology")
	}
	input := profile[len(profile)-1]
	if input.Mountpoint != nodeExecutionRoot || !input.ReadOnly || input.FSType != "9p" {
		t.Fatal("Node input is not current readonly tree")
	}
	after, _ := observerExpectedTopology("npm-lifecycle")
	if !reflect.DeepEqual(normal, after) || strings.Contains(boundaryOriginMode(boundaryNodeHandoffMode), "elf") {
		t.Fatal("old profile/mode changed")
	}
}

func TestNodeExecutionWorkingDirectoryDoesNotChangeOldContainerDefaults(t *testing.T) {
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	original := createArguments(session)
	changed, err := nodeExecutionWorkingDirectory(original)
	if err != nil {
		t.Fatal(err)
	}
	index := -1
	for i, arg := range changed {
		if arg == "--workdir" {
			index = i
			break
		}
	}
	if index < 0 || changed[index+1] != "/" || !reflect.DeepEqual(append(append([]string{}, changed[:index]...), changed[index+2:]...), original) {
		t.Fatal("execution cwd changed unrelated runtime limits")
	}
	if _, err := nodeExecutionWorkingDirectory(changed); err == nil {
		t.Fatal("ambiguous working directory accepted")
	}
}
