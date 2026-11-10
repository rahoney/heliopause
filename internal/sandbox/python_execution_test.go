package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestPythonExecutionRejectsUnavailableContextAndPreservesCause(t *testing.T) {
	var backend *PythonDynamicBackend
	for _, ctx := range []context.Context{nil, context.Background()} {
		outcome, err := backend.ExecutePythonExecution(ctx, domain.ExecutionContext{}, artifactpypi.ConsoleExecutionEntry{})
		if err == nil || outcome.Complete() {
			t.Fatal("unconfigured backend was complete")
		}
	}
	cause := errors.New("original Python caller cause")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	outcome, err := backend.ExecutePythonExecution(ctx, domain.ExecutionContext{}, artifactpypi.ConsoleExecutionEntry{})
	if outcome.Complete() || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal("caller cause lost", err)
	}
}
func TestPythonExecutionKeepsArgumentsOutOfFixedProgram(t *testing.T) {
	args := []string{"literal;$(not-a-shell)", "a b", "--help"}
	command := pythonExecutionCommand(artifactpypi.ConsoleExecutionEntry{}, args)
	if len(command) != 12 || !reflect.DeepEqual(command[:6], []string{"/usr/local/bin/python3.14", "-I", "-S", "-B", "-c", pythonExecutionProgram}) || !reflect.DeepEqual(command[9:], args) || strings.Contains(command[5], args[0]) {
		t.Fatal("private argv entered code")
	}
	command[9] = "changed"
	if args[0] != "literal;$(not-a-shell)" {
		t.Fatal("argument input aliased")
	}
}
func TestPythonExecutionOutcomeZeroAndPublicFormattingAreNotSuccessOrOutput(t *testing.T) {
	var outcome PythonExecutionOutcome
	if outcome.Complete() {
		t.Fatal("default complete")
	}
	if _, ok := outcome.ExitCode(); ok {
		t.Fatal("default exit")
	}
	if _, ok := outcome.Signal(); ok {
		t.Fatal("default signal")
	}
	if _, err := json.Marshal(outcome); err == nil {
		t.Fatal("runtime JSON exposed")
	}
	if fmt.Sprintf("%#v", outcome) != "private Python execution outcome" {
		t.Fatal("private representation exposed")
	}
}
func TestPythonExecutionProfileKeepsOldInspectionTopologyAndBudget(t *testing.T) {
	old, ok := pythonClosureExpectedTopology("pypi-wheel", pythonClosureObservation)
	current, valid := observerExpectedTopology(pythonExecutionProfile)
	if !ok || !valid || !validObserverProfile(pythonExecutionProfile) || isPythonObserverProfile(pythonExecutionProfile) || !reflect.DeepEqual(old, current) {
		t.Fatal("wrong profile ownership")
	}
	for _, mount := range current {
		if mount.Mountpoint == pythonSitePath && !mount.ReadOnly {
			t.Fatal("writable selected input")
		}
	}
	if traceBudgetForProfile(pythonExecutionProfile) != traceBudgetForProfile("pypi-wheel") {
		t.Fatal("new budget grant")
	}
	after, _ := pythonClosureExpectedTopology("pypi-wheel", pythonClosureObservation)
	if !reflect.DeepEqual(old, after) {
		t.Fatal("inspection topology mutated")
	}
	observer := new(SharedObserver)
	if _, err := observer.StartPythonExecutionProfileWithBudget(context.Background(), strings.Repeat("a", 64), nil); err == nil {
		t.Fatal("unaccounted stream accepted")
	}
}
