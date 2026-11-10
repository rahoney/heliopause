package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestBunExecutionCommandKeepsArgumentsOutsideItsFixedRuntimeOptions(t *testing.T) {
	arguments := []string{"", "a b", "a\nb", "--install=force", "--eval", "--handoff-node"}
	command := bunExecutionCommand("node_modules/tool/bin/cli.js", arguments)
	if len(command) != 6+len(arguments) || !slices.Equal(command[:5], []string{bunToolGuestPath, "--no-install", "--no-env-file", "--no-macros", "--"}) || command[5] != nodeExecutionProject+"/node_modules/tool/bin/cli.js" || !slices.Equal(command[6:], arguments) {
		t.Fatal("program arguments changed the fixed Bun runtime recipe")
	}
	boundary := boundaryExecArguments("0123456789abcdef", boundaryBunHandoffMode, command...)
	container, mode, index, ok := directExecOrigin(boundary, "docker")
	if !ok || container != "0123456789abcdef" || mode != boundaryBunHandoffMode || !slices.Equal(boundary[index:], command) {
		t.Fatal("Bun command lost its separate admission")
	}
}

func TestBunExecutionCannotTurnIncompleteInputOrCancellationIntoZeroSuccess(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("private Bun cancellation")
	cancel(cause)
	var backend *Backend
	outcome, err := backend.ExecuteBunExecution(ctx, domain.ExecutionContext{}, artifactnpm.NodeExecutionLayout{})
	if outcome.Complete() || !errors.Is(err, errBunExecution) || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal("Bun preflight lost original cause", err)
	}
	if _, present := outcome.ExitCode(); present {
		t.Fatal("incomplete Bun outcome became default zero exit")
	}
	if _, err := json.Marshal(outcome); err == nil || fmt.Sprintf("%+v", outcome) != "private Bun execution outcome" {
		t.Fatal("Bun runtime exposed private output")
	}
}

func TestBunExecutionTopologyPreservesNodeAndRequiresSeparateNativeToolMount(t *testing.T) {
	node, ok := observerExpectedTopology(nodeExecutionProfile)
	bun, bunOK := observerExpectedTopology(bunExecutionProfile)
	if !ok || !bunOK || len(node) != len(bun) {
		t.Fatal("missing bounded program topology")
	}
	changed := 0
	for i := range node {
		if node[i].Mountpoint == "/haa-runtime" {
			if node[i].FSType != "tmpfs" || bun[i].FSType != "9p" || !node[i].NoSUID || bun[i].NoSUID {
				t.Fatal("Bun supply replaced the ordinary Node helper boundary")
			}
			node[i].FSType, node[i].NoSUID = bun[i].FSType, bun[i].NoSUID
			changed++
		}
		if node[i] != bun[i] {
			t.Fatal("Bun runtime changed the readonly input or base image boundary")
		}
	}
	if changed != 1 || !bun[len(bun)-1].ReadOnly || BunExecutionRuntimeIdentity() == nodeImageReference {
		t.Fatal("Bun program lost its distinct identity or complete readonly input")
	}
}
