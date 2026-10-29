package application_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// Runtime doubles exercise orchestration only. Public real runtime qualification
// is in the separate Bootstrap realcorpus caller.
type executionRuntimeFixture struct {
	complete, exited, signaled bool
	code, signal               uint8
	err                        error
	stdout, stderr             []byte
	requests                   []domain.ExecutionContext
	after                      func()
}

func (f *executionRuntimeFixture) Execute(_ context.Context, request domain.ExecutionContext) (application.ExecutionRuntimeOutcome, error) {
	f.requests = append(f.requests, request)
	if f.after != nil {
		f.after()
	}
	return f, f.err
}
func (f *executionRuntimeFixture) Complete() bool          { return f.complete }
func (f *executionRuntimeFixture) ExitCode() (uint8, bool) { return f.code, f.exited }
func (f *executionRuntimeFixture) Signal() (uint8, bool)   { return f.signal, f.signaled }
func (f *executionRuntimeFixture) Stdout() []byte          { return f.stdout }
func (f *executionRuntimeFixture) Stderr() []byte          { return f.stderr }

func TestExecutionWorkflowUsesOnlyFreshCurrentPreparation(t *testing.T) {
	preparation, _, _, request := executionPreparationFixture(t)
	runtime := &executionRuntimeFixture{complete: true, exited: true, stdout: []byte("private-output"), stderr: []byte("private-stderr")}
	service, err := application.NewExecutionService(preparation, runtime)
	if err != nil {
		t.Fatal(err)
	}
	var previous domain.OperationID
	for range 2 {
		result, err := service.Run(context.Background(), request)
		if err != nil || !result.RuntimeAttempted() || !result.Complete() || result.FailureCode() != "" || !result.Preparation().Valid() {
			t.Fatal("fresh workflow did not complete", err)
		}
		bound := runtime.requests[len(runtime.requests)-1]
		prepared := result.Preparation().ExecutionContext()
		if bound.OperationID() == previous || bound.OperationID() != prepared.OperationID() || bound.SourceFingerprint() != prepared.SourceFingerprint() || !slices.Equal(bound.Arguments(), request.Arguments()) || bound.EntryArtifact() != prepared.EntryArtifact() || !reflect.DeepEqual(bound.Set(), prepared.Set()) {
			t.Fatal("runtime received cached or substituted preparation")
		}
		previous = bound.OperationID()
		out := result.Stdout()
		out[0] = 'X'
		errout := result.Stderr()
		errout[0] = 'X'
		if !bytes.Equal(result.Stdout(), runtime.stdout) || !bytes.Equal(result.Stderr(), runtime.stderr) {
			t.Fatal("private stream accessor aliases runtime memory")
		}
		if _, err := json.Marshal(result); err == nil {
			t.Fatal("private execution result persisted")
		}
		for _, format := range []string{"%v", "%+v", "%#v"} {
			printed := fmt.Sprintf(format, result)
			if bytes.Contains([]byte(printed), []byte("private-")) {
				t.Fatal("result formatter leaked request or program output")
			}
		}
	}
	if len(runtime.requests) != 2 {
		t.Fatal("runtime invocation count changed")
	}
}

func TestExecutionWorkflowDoesNotRunFailedPreparation(t *testing.T) {
	for _, mode := range []string{"resolve error", "records error", "stage error"} {
		t.Run(mode, func(t *testing.T) {
			preparation, p, _, request := executionPreparationFixture(t)
			p.mode = mode
			runtime := &executionRuntimeFixture{complete: true, exited: true}
			service, err := application.NewExecutionService(preparation, runtime)
			if err != nil {
				t.Fatal(err)
			}
			result, err := service.Run(context.Background(), request)
			if !errors.Is(err, p.err) || result.Complete() || result.FailureCode() != "PREPARATION_FAILED" || result.RuntimeAttempted() || len(runtime.requests) != 0 {
				t.Fatal("failed preparation reached runtime or lost cause", err)
			}
		})
	}
}

func TestExecutionWorkflowReconcilesTerminalWithoutErasingCause(t *testing.T) {
	cause := errors.New("original runtime fault")
	for _, row := range []struct {
		name     string
		runtime  executionRuntimeFixture
		complete bool
	}{
		{"zero", executionRuntimeFixture{complete: true, exited: true}, true},
		{"nonzero", executionRuntimeFixture{complete: true, exited: true, code: 37, err: cause}, true},
		{"signal", executionRuntimeFixture{complete: true, signaled: true, signal: 15, err: cause}, true},
		{"missing", executionRuntimeFixture{complete: true}, false},
		{"both", executionRuntimeFixture{complete: true, exited: true, signaled: true, signal: 15, err: cause}, false},
		{"zero signal", executionRuntimeFixture{complete: true, signaled: true, err: cause}, false},
		{"zero with error", executionRuntimeFixture{complete: true, exited: true, err: cause}, false},
		{"nonzero no error", executionRuntimeFixture{complete: true, exited: true, code: 37}, false},
		{"signal no error", executionRuntimeFixture{complete: true, signaled: true, signal: 15}, false},
		{"incomplete cause", executionRuntimeFixture{err: cause}, false},
		{"incomplete nil", executionRuntimeFixture{}, false},
	} {
		t.Run(row.name, func(t *testing.T) {
			preparation, _, _, request := executionPreparationFixture(t)
			service, err := application.NewExecutionService(preparation, &row.runtime)
			if err != nil {
				t.Fatal(err)
			}
			result, err := service.Run(context.Background(), request)
			if result.Complete() != row.complete || row.runtime.err != nil && !errors.Is(err, cause) || !row.complete && err == nil {
				t.Fatal("terminal/cause contract lost", err)
			}
			_, exited := result.ExitCode()
			_, signaled := result.Signal()
			if !row.complete && (exited || signaled) {
				t.Fatal("incomplete result exposed completed terminal")
			}
		})
	}
}

func TestExecutionWorkflowPreservesLateCallerCancellation(t *testing.T) {
	preparation, _, _, request := executionPreparationFixture(t)
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("caller custom cause")
	original := errors.New("original provider failure")
	runtime := &executionRuntimeFixture{complete: true, exited: true, code: 37, err: original, after: func() { cancel(cause) }}
	service, err := application.NewExecutionService(preparation, runtime)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Run(ctx, request)
	if result.Complete() || result.FailureCode() != "CANCELED" || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, original) {
		t.Fatal("late cancellation lost original cause or became complete", err)
	}
}

func TestExecutionWorkflowRejectsInvalidRequestBeforeResolution(t *testing.T) {
	preparation, p, _, request := executionPreparationFixture(t)
	runtime := &executionRuntimeFixture{}
	service, err := application.NewExecutionService(preparation, runtime)
	if err != nil {
		t.Fatal(err)
	}
	var missing context.Context
	for _, input := range []struct {
		ctx     context.Context
		request application.ExecutionPreparationRequest
	}{
		{missing, request}, {context.Background(), application.ExecutionPreparationRequest{}},
	} {
		result, err := service.Run(input.ctx, input.request)
		if err == nil || result.Complete() || result.FailureCode() != "REQUEST_INVALID" {
			t.Fatal("invalid execution accepted")
		}
	}
	ctx, stop := context.WithCancel(context.Background())
	stop()
	if result, err := service.Run(ctx, request); result.Complete() || !errors.Is(err, context.Canceled) || p.resolves != 0 || len(runtime.requests) != 0 {
		t.Fatal("pre-canceled execution reached resolution/runtime", err)
	}
	if service, err := application.NewExecutionService(nil, runtime); service != nil || err == nil {
		t.Fatal("missing preparation accepted")
	}
	if service, err := application.NewExecutionService(preparation, nil); service != nil || err == nil {
		t.Fatal("missing runtime accepted")
	}
}
